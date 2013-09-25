package main

import (
	"fmt"
	"github.com/pelletier/go-toml"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	ImportProp = "import"
	BranchProp = "branch"
	CommitProp = "commit"
	TagProp    = "tag"
	BranchFlag = 1 << 0
	CommitFlag = 1 << 1
	TagFlag    = 1 << 2
)

type Dependencies struct {
	Imports     []string
	Keys        []string
	DepList     []*Dep
	ImportGraph *Graph
}

type Dep struct {
	Import string
	// which of BranchFlag, CommitFlag, TagFlag is this repo
	CheckoutFlag uint8
	// the name of the thing to checkout whether it be a commit, branch, or tag
	CheckoutSpec string
}

func NewDependency(repo string) *Dep {
	return &Dep{Import: repo}
}

func LoadDependencyModel(depsTree *toml.TomlTree, importGraph *Graph) *Dependencies {
	if depsTree == nil {
		return nil
	}

	deps := new(Dependencies)

	deps.Imports = make([]string, len(depsTree.Keys()))
	deps.Keys = make([]string, len(depsTree.Keys()))
	deps.DepList = make([]*Dep, len(depsTree.Keys()))
	deps.ImportGraph = importGraph

	for i, k := range depsTree.Keys() {
		depTree := depsTree.Get(k).(*toml.TomlTree)
		d := NewDependency(depTree.Get("import").(string))

		d.setCheckout(depTree, "branch", BranchFlag)
		d.setCheckout(depTree, "commit", CommitFlag)
		d.setCheckout(depTree, "tag", TagFlag)

		d.CheckValidity()

		deps.Keys[i] = k
		deps.Imports[i] = d.Import
		deps.DepList[i] = d

		deps.ImportGraph.Insert(d)
	}
	return deps
}

func (d *Dependencies) IncludesDependency(importPath string) (*Node, bool) {
	node := d.ImportGraph.Search(importPath)
	return node, node != nil
}

func (d *Dep) setCheckout(t *toml.TomlTree, key string, flag uint8) {
	s := t.Get(key)
	if s != nil {
		d.CheckoutSpec = s.(string)
		d.CheckoutFlag |= flag
	}
}

func (d *Dep) CheckValidity() {
	f := d.CheckoutFlag
	if f&(f-1) != 0 {
		failf("%s - only one of branch/commit/tag may be specified\n", d.Import)
	}
}

func (d *Dependencies) VisitDeps(fn func(dep *Dep)) {
	for _, dep := range d.DepList {
		fn(dep)
	}
}

func (d *Dependencies) String() string {
	return fmt.Sprintf("imports = %s, keys = %s", d.Imports, d.Keys)
}

func (d *Dependencies) PrintDependencyTree() {
	d.ImportGraph.PreOrderVisit(
		func(n *Node, depth int) {
			indent := strings.Repeat(" ", depth*2)
			dep := n.Dependency
			bullet := "+-"
			if n.Leaf {
				bullet = "-"
			}
			if dep == nil {
				fmt.Printf("%s%s %s\n", indent, bullet, n.Key)
			} else {
				fmt.Printf("%s%s %s @ %s\n", indent, bullet, dep.Import, dep.CheckoutSpec)
			}
		})
}

func (d *Dep) String() string {
	if d.CheckoutType() != "" {
		return fmt.Sprintf("import = %s, %s = %s", d.Import, d.CheckoutType(), d.CheckoutSpec)
	} else {
		return fmt.Sprintf("import = %s", d.Import)
	}
}

func (d *Dep) CheckoutType() string {
	switch d.CheckoutFlag {
	case BranchFlag:
		return "branch"
	case TagFlag:
		return "tag"
	case CommitFlag:
		return "commit"
	}
	return ""
}

func (d *Dep) Src() string {
	return fmt.Sprintf("%s/%s/src/%s", pwd, VendorDir, d.Import)
}

// switch the dep to the appropriate branch or tag
func (d *Dep) switchToBranchOrTag() error {
	err := d.cdSrc()
	if err != nil {
		return err
	}

	scm, err := d.Scm()

	if err != nil {
		log.Println(err)
	} else {
		err = scm.Checkout(d)

		if err != nil {
			log.Printf("error checking out %s on %s\n", d.CheckoutSpec, d.Import)
		}
	}

	return cdHome()
}

// Tell the scm where the dependency is hosted.
func (d *Dep) Scm() (Scm, error) {
	parts := strings.Split(d.Import, "/")
	initPath := d.Src()
	scms := map[string]Scm{".git": Git{}, ".hg": Hg{}, ".svn": Svn{}}

	// Traverse the source tree backwards until
	// it finds the right directory
	// or it arrives to the base of the import.
	for _, _ = range parts {
		for key, scm := range scms {
			if d.scmPath(path.Join(initPath, key)) {
				return scm, nil
			}
		}

		initPath = path.Join(initPath, "..")
	}

	return nil, fmt.Errorf("unknown scm for %s", d.Import)
}

func (d *Dep) scmPath(scmPath string) bool {
	stat, err := os.Stat(scmPath)
	if err != nil {
		return false
	}

	return stat.IsDir()
}

func (d *Dep) cdSrc() error {
	err := os.Chdir(d.Src())
	if err != nil {
		log.Print(err)
		log.Printf("couldn't cd to src dir for %s\n", d.Import)
		return err
	}
	return nil
}

func cdHome() error {
	return os.Chdir(pwd)
}

// update the git repo for this dep
func (d *Dep) goGetUpdate() error {
	cmd := exec.Command("go", "get", "-u", d.Import)
	return cmd.Run()
}

func (d *Dep) LoadTransitiveDeps(importGraph *Graph) *Dependencies {
	configPath := path.Join(d.Src(), "gopack.config")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}
	config := NewConfig(d.Src())
	return LoadDependencyModel(config.DepsTree, importGraph)
}

func (d *Dependencies) Validate(p *ProjectStats) []*ProjectError {
	errors := []*ProjectError{}
	includedDeps := make(map[string]*Dep)

	for path, s := range p.ImportStatsByPath {
		node, found := d.IncludesDependency(path)
		if s.Remote {
			if found {
				includedDeps[node.Dependency.Import] = node.Dependency
			} else {
				// report a validation error with the locations in source
				// where an import is used but unmanaged in gopack.config
				errors = append(errors, UnmanagedImportError(s))
			}
		}
	}

	for _, dep := range d.DepList {
		_, found := includedDeps[dep.Import]
		if !found && !p.IsImportUsed(dep.Import) {
			errors = append(errors, UnusedDependencyError(dep.Import))
		}
	}
	return errors
}

func ShowValidationErrors(errors []*ProjectError) {
	for _, e := range errors {
		fmt.Errorf("%s\n", e.String())
	}
}
