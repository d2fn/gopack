package main

import (
	"log"
	"fmt"
	"github.com/pelletier/go-toml"
	"os"
	"os/exec"
	"strings"
)

type Dependencies struct {
	Imports []string
	Keys    []string
	DepList []*Dep
}

type Dep struct {
	Import string
	Branch string
	Tag string
}

func LoadDependencyModel() *Dependencies {
	deps := new(Dependencies)
	t, err := toml.LoadFile("./goop.config")
	if err != nil {
		log.Fatal(err)
	}
	depsTree := t.Get("dependencies").(*toml.TomlTree)
	deps.Imports = make([]string, len(depsTree.Keys()))
	deps.Keys = make([]string, len(depsTree.Keys()))
	deps.DepList = make([]*Dep, len(depsTree.Keys()))
	for i, k := range depsTree.Keys() {
		depTree := depsTree.Get(k).(*toml.TomlTree)
		dep := new(Dep)
		dep.Import = depTree.Get("import").(string)
		if !strings.HasPrefix(dep.Import, "github.com") {
			log.Fatal("don't know how to manage this dependency, not a known git repo: %s", dep.Import)
		}
		b := depTree.Get("branch")
		t := depTree.Get("tag")
		if t != nil && b != nil {
			log.Fatal("both branch and tag specified for import of %s\n", dep.Import)
		}
		if b != nil {
			dep.Branch = b.(string)
		} else if t != nil {
			dep.Tag = t.(string)
		}
		deps.Keys[i] = k
		deps.Imports[i] = dep.Import
		deps.DepList[i] = dep

	}
	return deps
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
	return fmt.Sprintf("import = %s, branch = %s, tag = %s", d.Import, d.Branch, d.Tag)
}

func (d *Dep) checkoutName() string {
	if d.Tag != "" {
		return d.Tag
	} else if d.Branch != "" {
		return d.Branch
	}
	return "master"
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
	cmd := exec.Command("git", "checkout", d.checkoutName())
	err = cmd.Run()
	if err != nil {
		log.Println("error checking out %s on %s", d.checkoutName(), d.Import)
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

