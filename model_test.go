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

func createPath(path string) {
	err := os.MkdirAll(path, 0700)
	check(err)
}

func createScmDep(scm string, project string, paths ...string) *Dep {
	dep := &Dep{Import: project}
	scmPath := path.Join(dep.Src(), scm)
	createPath(scmPath)

	for _, p := range paths {
		createPath(path.Join(scmPath, p))
	}

	return dep
}

func TestGit(t *testing.T) {
	setupTestPwd()

	dep := createScmDep(".git", "github.com/d2fn/gopack")

	scm, err := dep.Scm()
	if scm != "git" {
		t.Error("Expected scm to be git but it was %s.\n%v", scm, err)
	}
}

func TestHg(t *testing.T) {
	setupTestPwd()

	dep := createScmDep(".hg", "code.google.com/p/go")

	scm, err := dep.Scm()
	if scm != "hg" {
		t.Errorf("Expected scm to be hg but it was %s.\n%v", scm, err)
	}
}

func TestUnknownScm(t *testing.T) {
	setupTestPwd()

	dep := createScmDep(".svn", "foo/bar/baz")

	scm, err := dep.Scm()
	if err == nil {
		t.Errorf("Expected unknown scm but it was %s", scm)
	}
}

func TestSubPackages(t *testing.T) {
	setupTestPwd()

	dep := createScmDep(".hg", "code.google.com/p/go", "path/filepath", "io")
	dep.Import = "code.google.com/p/go/path"

	scm, err := dep.Scm()
	if scm != "hg" {
		t.Errorf("Expected scm to be hg but it was %s.\n%v", scm, err)
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

	config := NewConfig(pwd)
	dependencies := LoadDependencyModel(config.DepsTree, NewGraph())
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

func TestRepositoryLink(t *testing.T) {
	setupTestPwd()
	setupEnv()

	file, err := os.Create(path.Join(pwd, "gopack.config"))
	check(err)
	_, err = file.WriteString("repo = \"github.com/d2fn/gopack\"\n")
	file.Sync()
	file.Close()

	LoadConfiguration(pwd, NewGraph())

	dep := path.Join(pwd, VendorDir, "src", "github.com", "d2fn", "gopack")
	stat, err := os.Stat(dep)

	if os.IsNotExist(err) || (stat.Mode()&os.ModeSymlink != 0) {
		t.Errorf("Expected repository github.com/d2fn/gopack to be linked in vendor %s\n", pwd)
	}
}
