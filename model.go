package main

import (
	"fmt"
	"github.com/pelletier/go-toml"
	"log"
	"os"
	"os/exec"
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
	Imports []string
	Keys    []string
	DepList []*Dep
}

type Dep struct {
	Import string
	// which of BranchFlag, CommitFlag, TagFlag is this repo
	CheckoutFlag uint8
	// the name of the thing to checkout whether it be a commit, branch, or tag
	CheckoutSpec string
}

func LoadDependencyModel() *Dependencies {
	deps := new(Dependencies)
	t, err := toml.LoadFile("./gopack.config")
	if err != nil {
		log.Fatal(err)
	}
	depsTree := t.Get("deps").(*toml.TomlTree)
	deps.Imports = make([]string, len(depsTree.Keys()))
	deps.Keys = make([]string, len(depsTree.Keys()))
	deps.DepList = make([]*Dep, len(depsTree.Keys()))
	for i, k := range depsTree.Keys() {
		depTree := depsTree.Get(k).(*toml.TomlTree)
		d := new(Dep)
		d.Import = depTree.Get("import").(string)
		if !strings.HasPrefix(d.Import, "github.com") {
			log.Fatal("don't know how to manage this dependency, not a known git repo: %s", d.Import)
		}
		d.setCheckout(depTree, "branch", BranchFlag)
		d.setCheckout(depTree, "commit", CommitFlag)
		d.setCheckout(depTree, "tag",    TagFlag)
		deps.Keys[i] = k
		deps.Imports[i] = d.Import
		deps.DepList[i] = d

	}
	return deps
}

func (d *Dep) setCheckout(t *toml.TomlTree, key string, flag uint8) {
	s := t.Get(key)
	if s != nil {
		d.CheckoutSpec = s.(string)
		d.CheckoutFlag |= flag
	}
}

func (d *Dep) isValid() error {
	// check that the checkout flag is a power of 2 (only one flag is checked)
	f := d.CheckoutFlag
	if f&(f-1) != 0 {
		return fmt.Errorf("%s - only one of branch/commit/tag may be specified", d.Import)
	}
	return nil
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
	return fmt.Sprintf("import = %s, %s = %s", d.Import, d.CheckoutType(), d.CheckoutSpec)
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
	cmd := exec.Command("git", "checkout", d.CheckoutSpec)
	err = cmd.Run()
	if err != nil {
		log.Println("error checking out %s on %s", d.CheckoutSpec, d.Import)
	}
	return cdHome()
}

func (d *Dep) cdSrc() error {
	err := os.Chdir(d.Src())
	if err != nil {
		log.Print(err)
		log.Println("couldn't cd to src dir for %s", d.Import)
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
