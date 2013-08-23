package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func setupTestPwd() {
	dir, _ := ioutil.TempDir("", "gopack-config-")
	os.Setenv("GOPACK_APP_CONFIG", dir)
	setPwd()
}

func createScmDep(project, scm string) *Dep {
	dep := &Dep{Import: project}
	scmPath := path.Join(dep.Src(), scm)
	err := os.MkdirAll(scmPath, 0700)
	check(err)

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

func TestTransitiveDependencies(t *testing.T) {
	setupTestPwd()
	setupEnv()

	file, err := os.Create(path.Join(pwd, "gopack.config"))
	check(err)
	_, err = file.WriteString("[deps.testgopack]\n")
	_, err = file.WriteString("  import = \"github.com/calavera/testGoPack\"\n")
	file.Sync()
	file.Close()

	dependencies := LoadDependencyModel(pwd, NewGraph())
	loadTransitiveDependencies(dependencies)

	dep := path.Join(pwd, VendorDir, "src", "github.com", "calavera", "testGoPack")
	if _, err := os.Stat(dep); os.IsNotExist(err) {
		t.Errorf("Expected dependency github.com/calavera/testGoPack to be in vendor %s\n", pwd)
	}

	dep = path.Join(pwd, VendorDir, "src", "github.com", "d2fn", "gopack")
	if _, err = os.Stat(dep); os.IsNotExist(err) {
		t.Errorf("Expected dependency github.com/d2fn/gopack to be in vendor %s\n", pwd)
	}
}
