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
	Imports   []string
	Keys      []string
	DepList   []*Dep
	ImportMap map[string]*Dep
}

type Dep struct {
	Import string
	// which of BranchFlag, CommitFlag, TagFlag is this repo
	CheckoutFlag uint8
	// the name of the thing to checkout whether it be a commit, branch, or tag
	CheckoutSpec string
}

func LoadDependencyModel(dir string) *Dependencies {
	deps := new(Dependencies)
	path := fmt.Sprintf("%s/gopack.config", dir)
	t, err := toml.LoadFile(path)
	if err != nil {
		fail(err)
	}
	depsTree := t.Get("deps").(*toml.TomlTree)
	deps.Imports = make([]string, len(depsTree.Keys()))
	deps.Keys = make([]string, len(depsTree.Keys()))
	deps.DepList = make([]*Dep, len(depsTree.Keys()))
	deps.ImportMap = make(map[string]*Dep)
	for i, k := range depsTree.Keys() {
		depTree := depsTree.Get(k).(*toml.TomlTree)
		d := new(Dep)
		d.Import = depTree.Get("import").(string)
		d.setCheckout(depTree, "branch", BranchFlag)
		d.setCheckout(depTree, "commit", CommitFlag)
		d.setCheckout(depTree, "tag", TagFlag)
		d.CheckValidity()
		deps.Keys[i] = k
		deps.Imports[i] = d.Import
		deps.DepList[i] = d
		deps.ImportMap[d.Import] = d
	}
	return deps
}

func (d *Dependencies) IncludesDependency(importPath string) bool {
	_, found := d.ImportMap[importPath]
	return found
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

func (d *Dep) String() string {
	if d.CheckoutType() != "" {
		return fmt.Sprintf("import = %s, %s = %s", d.Import, d.CheckoutType(), d.CheckoutSpec)
	} else {
		return fmt.Sprintf("import = %s", d.Import)
	}
}

func (d *Dep) CheckoutType() string {
	if d.CheckoutFlag == BranchFlag {
		return "branch"
	}
	if d.CheckoutFlag == CommitFlag {
		return "commit"
	}
	if d.CheckoutFlag == TagFlag {
		return "tag"
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

	switch {
	case scm == "git":
		d.gitCheckout()
	case scm == "hg":
		d.hgCheckout()
	default:
		log.Println(err)
	}

	return cdHome()
}

// Tell the scm where the dependency is hosted.
func (d *Dep) Scm() (string, error) {
	parts := strings.Split(d.Import, "/")
	initPath := d.Src()

	// Traverse the source tree backwards until
	// it finds the right directory
	// or it arrives to the base of the import.
	for _, _ = range parts {
		if d.scmPath(path.Join(initPath, ".git")) {
			return "git", nil
		}

		if d.scmPath(path.Join(initPath, ".hg")) {
			return "hg", nil
		}

		initPath = path.Join(initPath, "..")
	}

	return "", fmt.Errorf("unknown scm for %s", d.Import)
}

func (d *Dep) scmPath(scmPath string) bool {
	stat, err := os.Stat(scmPath)
	if err != nil {
		return false
	}

	return stat.IsDir()
}

func (d *Dep) gitCheckout() {
	cmd := exec.Command("git", "checkout", d.CheckoutSpec)
	err := cmd.Run()
	if err != nil {
		log.Println("error checking out %s on %s", d.CheckoutSpec, d.Import)
	}
}

func (d *Dep) hgCheckout() {
	var cmd *exec.Cmd

	if d.CheckoutFlag == CommitFlag {
		cmd = exec.Command("hg", "update", "-c", d.CheckoutSpec)
	} else {
		cmd = exec.Command("hg", "checkout", d.CheckoutSpec)
	}

	err := cmd.Run()
	if err != nil {
		log.Printf("error checking out %s on %s\n", d.CheckoutSpec, d.Import)
	}
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

func (d *Dep) LoadTransitiveDeps() *Dependencies {
	return new(Dependencies)
}

func (d *Dependencies) Validate(p *ProjectStats) []*ProjectError {
	errors := []*ProjectError{}
	for path, s := range p.ImportStatsByPath {
		if s.Remote && !d.IncludesDependency(path) {
			// report a validation error with the locations in source
			// where an import is used but unmanaged in gopack.config
			errors = append(errors, UnmanagedImportError(s))
		}
	}
	for _, dep := range d.DepList {
		if !p.IsImportUsed(dep.Import) {
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
