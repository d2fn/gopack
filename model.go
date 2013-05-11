package main

import (
	"fmt"
	"github.com/kylelemons/go-gypsy/yaml"
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
	f, err := yaml.ReadFile("./gopack.yml")
	if err != nil {
		log.Fatal(err)
	}
	child, err := yaml.Child(f.Root, "deps")
	if err != nil {
		log.Fatal(err)
	}
	depMap := child.(yaml.Map)
	deps.Imports = make([]string, len(depMap))
	deps.Keys = make([]string, len(depMap))
	deps.DepList = make([]*Dep, len(depMap))
	i := 0
	for depKey, child := range depMap {
		d := new(Dep)
		depProps := child.(yaml.Map)
		d.Import = string(depProps["import"].(yaml.Scalar))
		d.setCheckout(depProps, "branch", BranchFlag)
		d.setCheckout(depProps, "commit", CommitFlag)
		d.setCheckout(depProps, "tag", TagFlag)
		deps.Keys[i] = depKey
		deps.Imports[i] = d.Import
		deps.DepList[i] = d
		i = i + 1
	}
	return deps
}

func (d *Dep) setCheckout(n yaml.Map, key string, flag uint8) {
	s, found := n[key]
	if found {
		d.CheckoutSpec = string(s.(yaml.Scalar))
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
