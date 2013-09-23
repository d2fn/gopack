package main

import (
	"os/exec"
)

type Scm interface {
	Checkout(d *Dep) error
}

type Git struct {
}

type Hg struct {
}

type Svn struct {
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
