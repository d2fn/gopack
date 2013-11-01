package main

// LOL so we're gonna try and avoid THIS situation http://golang.org/src/cmd/go/vcs.go#L331

import (
	"fmt"
	"os"
	"os/exec"
)

type Scm interface {
	Init(d *Dep) error
	Checkout(d *Dep) error
}

type Git struct {
}

type Hg struct {
}

type Svn struct {
}

// The Go provider embeds another provider and only implements Init so that
// deps that don't specify a provider keep working like they did before
type Go struct {
	Scm
}

func (g Git) Init(d *Dep) error {
	path := fmt.Sprintf("%s/%s/src/%s", pwd, VendorDir, d.Import)
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("Error creating import dir %s", err)
	} else {
		cmd := exec.Command("git", "clone", d.Source, path)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Error cloning repo %s", err)
		}
	}
	return nil
}

func (g Git) Checkout(d *Dep) error {
	cmd := exec.Command("git", "checkout", d.CheckoutSpec)
	return cmd.Run()
}

func (h Hg) Checkout(d *Dep) error {
	var cmd *exec.Cmd

	if d.CheckoutFlag == CommitFlag {
		cmd = exec.Command("hg", "update", "-c", d.CheckoutSpec)
	} else {
		cmd = exec.Command("hg", "checkout", d.CheckoutSpec)
	}

	return cmd.Run()
}

// TODO someone should vet this that knows hg
func (h Hg) Init(d *Dep) error {
	path := fmt.Sprintf("%s/s", VendorDir, d.Import)
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	} else {
		cmd := exec.Command("hg", "clone", d.Source, path)
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// FIXME someone that has an SVN repo accessible, please
func (s Svn) Init(d *Dep) error {
	failf("SVN repos not yet fully supported")
	return nil
}

func (s Svn) Checkout(d *Dep) error {
	var cmd *exec.Cmd

	switch d.CheckoutFlag {
	case CommitFlag:
		cmd = exec.Command("svn", "up", "-r", d.CheckoutSpec)
	case BranchFlag:
		cmd = exec.Command("svn", "switch", "^/branches/"+d.CheckoutSpec)
	case TagFlag:
		cmd = exec.Command("svn", "switch", "^/tags/"+d.CheckoutSpec)
	}

	return cmd.Run()
}

func (g Go) Init(d *Dep) error {
	cmd := exec.Command("go", "get", "-d", "-u", d.Import)
	return cmd.Run()
}
