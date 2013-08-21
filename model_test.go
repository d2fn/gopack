package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func setupTestPwd() {
	dir, _ := ioutil.TempDir("", "gopack-config-")
	os.Setenv("GOPACK_APP_CONFIG", dir)
	setPwd()
}

func createScmDep(project, scm string) *Dep {
	dep := &Dep{Import: project}
	scmPath := path.Join(dep.Src(), scm)
	err := os.MkdirAll(scmPath, 0700)

	if err != nil {
		panic(err)
	}

	return dep
}

func TestGit(t *testing.T) {
	setupTestPwd()

	dep := createScmDep("github.com/d2fn/gopack", ".git")

	scm, err := dep.Scm()
	if scm != "git" {
		t.Error("Expected scm to be git but it was %s.\n%v", scm, err)
	}
}

func TestHg(t *testing.T) {
	setupTestPwd()

	dep := createScmDep("code.google.com/p/go", ".hg")

	scm, err := dep.Scm()
	if scm != "hg" {
		t.Errorf("Expected scm to be hg but it was %s.\n%v", scm, err)
	}
}

func TestUnknownScm(t *testing.T) {
	setupTestPwd()

	dep := createScmDep("foo/bar/baz", ".svn")

	scm, err := dep.Scm()
	if err == nil {
		t.Errorf("Expected unknown scm but it was %s", scm)
	}
}
