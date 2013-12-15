package main

// LOL so we're gonna try and avoid THIS situation http://golang.org/src/cmd/go/vcs.go#L331

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	GitTag    = "git"
	HgTag     = "hg"
	SvnTag    = "svn"
	HiddenGit = ".git"
	HiddenHg  = ".hg"
	HiddenSvn = ".svn"
)

type Scm interface {
	Init(d *Dep) error
	Checkout(d *Dep) error
	Fetch(path string) error
	DownloadCommand(source, path string) *exec.Cmd
}

func dependencyPath(importPath string) string {
	return path.Join(pwd, VendorDir, "src", importPath)
}

func scmStageDir(depPath, scmDir string) string {
	return path.Join(depPath, scmDir)
}

func downloadDependency(d *Dep, depPath, scmType string, scm Scm) (err error) {
	_, err = os.Stat(scmStageDir(depPath, scmType))

	if os.IsExist(err) {
		err = scm.Fetch(depPath)
	} else if err != nil {
		err = fmt.Errorf("Error while examining dependency path for %s: %s", d.Import, err)
	} else {
		fmtcolor(Gray, "downloading %s\n", d.Source)

		cmd := scm.DownloadCommand(d.Source, depPath)

		if err = cmd.Run(); err != nil {
			return fmt.Errorf("Error downloading dependency: %s", err)
		}
	}

	return
}

func initScm(d *Dep, scmType string, scm Scm) error {
	path := dependencyPath(d.Import)

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("Error creating import dir %s", err)
	} else {
		downloadDependency(d, path, scmType, scm)
	}
	return nil
}

func runInPath(path string, fn func() error) error {
	err := os.Chdir(path)
	if err != nil {
		return err
	}
	defer os.Chdir(pwd)

	return fn()
}

type Git struct{}

func (g Git) Init(d *Dep) error {
	return initScm(d, HiddenGit, g)
}

func (g Git) DownloadCommand(source, path string) *exec.Cmd {
	return exec.Command("git", "clone", source, path)
}

func (g Git) Checkout(d *Dep) error {
	cmd := exec.Command("git", "checkout", d.CheckoutSpec)
	return cmd.Run()
}

func (g Git) Fetch(path string) error {
	return runInPath(path, func() error {
		return exec.Command("git", "fetch").Run()
	})
}

type Hg struct{}

func (h Hg) Init(d *Dep) error {
	return initScm(d, HiddenHg, h)
}

func (h Hg) DownloadCommand(source, path string) *exec.Cmd {
	return exec.Command("hg", "clone", source, path)
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

func (h Hg) Fetch(path string) error {
	return runInPath(path, func() error {
		return exec.Command("hg", "pull").Run()
	})
}

type Svn struct {
}

func (s Svn) Init(d *Dep) error {
	return initScm(d, HiddenSvn, s)
}

func (s Svn) DownloadCommand(source, path string) *exec.Cmd {
	return exec.Command("svn", "checkout", source, path)
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

func (s Svn) Fetch(path string) error {
	return runInPath(path, func() error {
		return exec.Command("svn", "update").Run()
	})
}

// The Go scm embeds another scm and only implements Init so that
// deps that don't specify a scm keep working like they did before
type Go struct {
	Scm
}

func (g Go) Init(d *Dep) error {
	return g.DownloadCommand(d.Import, "").Run()
}

func (g Go) DownloadCommand(source, path string) *exec.Cmd {
	return exec.Command("go", "get", "-d", "-u", source)
}

func NewScm(d *Dep) (Scm, error) {
	switch d.Scm {
	case GitTag:
		return Scms[GitTag], nil
	case HgTag:
		return Scms[HgTag], nil
	case SvnTag:
		return Scms[SvnTag], nil
	}

	scm := scmInSource(d)

	if d.Scm == "go" {
		return Go{scm}, nil
	} else if scm != nil {
		return scm, nil
	}

	return nil, fmt.Errorf("unknown scm for %s", d.Import)
}

// Traverse the source tree backwards until
// it finds the right directory
// or it arrives to the base of the import.
func scmInSource(d *Dep) Scm {
	parts := strings.Split(d.Import, "/")
	initPath := d.Src()

	for _, _ = range parts {
		for key, scm := range Scms {
			if d.scmPath(path.Join(initPath, HiddenDirs[key])) {
				return scm
			}
		}
		initPath = path.Join(initPath, "..")
	}

	return nil
}
