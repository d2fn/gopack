package main

import (
	"fmt"
	"github.com/pelletier/go-toml"
	"log"
	"os"
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

var (
	Scms = map[string]Scm{
		GitTag: Git{},
		HgTag:  Hg{},
		SvnTag: Svn{}}

	HiddenDirs = map[string]string{
		GitTag: HiddenGit,
		HgTag:  HiddenHg,
		SvnTag: HiddenSvn}
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

	// does this dep need to be fetched
	fetch bool

	// what is the scm for this dep (hg, git, bzr etc)
	Scm string
	// whence the Scm should clone/checkout
	Source string
}

func NewDependency(repo string) *Dep {
	return &Dep{Import: repo}
}

func (d *Dependencies) IncludesDependency(importPath string) (*Node, bool) {
	node := d.ImportGraph.Search(importPath)
	return node, node != nil
}

func (d *Dep) Fetch(all bool) bool {
	d.fetch = all || (d.CheckoutFlag != CommitFlag && d.CheckoutFlag != TagFlag)
	return d.fetch
}

func (d *Dep) setCheckout(t *toml.TomlTree, key string, flag uint8) {
	s := t.Get(key)
	if s != nil {
		d.CheckoutSpec = s.(string)
		d.CheckoutFlag |= flag
	}
}

func (d *Dep) setScm(t *toml.TomlTree) {
	scm := t.Get("scm")
	if scm != nil {
		d.Scm = scm.(string)
	} else {
		d.Scm = "go"
	}
}

func (d *Dep) setSource(t *toml.TomlTree) {
	source := t.Get("source")
	if source != nil {
		d.Source = source.(string)
	} else {
		d.Source = ""
	}
}

func (d *Dep) Validate() (err error) {
	f := d.CheckoutFlag
	if f&(f-1) != 0 {
		err = fmt.Errorf("%s - only one of branch/commit/tag may be specified\n", d.Import)
	}

	if d.Scm != "go" && d.Source == "" {
		err = fmt.Errorf("%s - Scm set to an SCM system, but no source set.", d.Import)
	}

	if d.Scm == "go" && d.Source != "" {
		err = fmt.Errorf("%s - Source set, but no scm", d.Import)
	}
	return err
}

func (d *Dependencies) VisitDeps(fn func(dep *Dep)) {
	for _, dep := range d.DepList {
		fn(dep)
	}
}

func (d *Dependencies) AnyDepsNeedFetching() bool {
	for _, dep := range d.DepList {
		if dep.fetch {
			return true
		}
	}
	return false
}

func (d *Dependencies) AllDepsNeedFetching() bool {
	for _, dep := range d.DepList {
		if !dep.fetch {
			return false
		}
	}
	return true
}

func (d *Dependencies) String() string {
	return fmt.Sprintf("imports = %s, keys = %s", d.Imports, d.Keys)
}

func (d *Dependencies) PrintDependencyTree() {
	d.ImportGraph.PreOrderVisit(
		func(n *Node, depth int) {
			fmt.Printf("depth = %d\n", depth)
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

func (d *Dependencies) Install(repo string) {
	var importName string

	for e := d.ImportGraph.Leafs.Front(); e != nil; e = e.Next() {
		importName = e.Value.(string)

		if importName != repo {
			run("install", importName)
		}
	}
}

func (d *Dep) String() string {
	if d.CheckoutType() != "" {
		return fmt.Sprintf("import = %s, %s = %s, scm = %s", d.Import, d.CheckoutType(), d.CheckoutSpec, d.Scm)
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

	scm, err := NewScm(d)

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

func (d *Dep) LoadTransitiveDeps(importGraph *Graph) (*Dependencies, error) {
	configPath := path.Join(d.Src(), "gopack.config")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil
	}
	config := NewConfig(d.Src())
	return config.LoadDependencyModel(importGraph)
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
