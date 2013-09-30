package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func createFixtureConfig(dir string, config string) {
	err := ioutil.WriteFile(path.Join(dir, "gopack.config"), []byte(config), 0644)
	check(err)
}

func setupTestConfig(fixture string) *Config {
	setupTestPwd()
	setupEnv()

	createFixtureConfig(pwd, fixture)
	return NewConfig(pwd)
}

func TestNewConfig(t *testing.T) {
	config := setupTestConfig(`
repo = "github.com/d2fn/gopack"

[deps.testgopack]
  import = "github.com/calavera/testGoPack"
`)

	if config.Repository == "" {
		t.Error("Expected repository to not be empty.")
	}

	if config.DepsTree == nil {
		t.Error("Expected dependency tree to not be empty.")
	}
}

func TestInitRepoWithoutRepo(t *testing.T) {
	config := setupTestConfig(`
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
`)

	graph := NewGraph()
	config.InitRepo(graph)

	src := path.Join(pwd, VendorDir, "src")
	_, err := os.Stat(src)

	if !os.IsNotExist(err) {
		t.Errorf("Expected vendor to not exist in %s\n", pwd)
	}
}

func TestInitRepo(t *testing.T) {
	config := setupTestConfig(`repo = "github.com/d2fn/gopack"`)

	graph := NewGraph()
	config.InitRepo(graph)

	dep := path.Join(pwd, VendorDir, "src", "github.com", "d2fn", "gopack")
	stat, err := os.Stat(dep)

	if os.IsNotExist(err) || (stat.Mode()&os.ModeSymlink != 0) {
		t.Errorf("Expected repository %s to be linked in vendor %s\n", config.Repository, pwd)
	}

	if graph.Search(config.Repository) == nil {
		t.Errorf("Expected repository %s to be in the dependencies graph\n", config.Repository)
	}
}

func TestWriteChecksum(t *testing.T) {
	config := setupTestConfig(`
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
`)

	config.WriteChecksum()

	path := path.Join(pwd, GopackChecksum)
	_, err := ioutil.ReadFile(path)
	if err != nil && os.IsNotExist(err) {
		t.Errorf("Expected checksum file %s to exist", path)
	}
}

func TestFetchDependenciesWithoutChecksum(t *testing.T) {
	config := setupTestConfig(`
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
`)

	if !config.FetchDependencies() {
		t.Errorf("Expected fetch dependencies to be true when there is no checksum")
	}
}

func TestFetchDependenciesWithoutChanges(t *testing.T) {
	config := setupTestConfig(`
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
`)
	config.WriteChecksum()

	if config.FetchDependencies() {
		t.Errorf("Expected fetch dependencies to be false when the checksum doesn't change")
	}
}

func TestFetchDependenciesWithChanges(t *testing.T) {
	config := setupTestConfig(`
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
`)

	config.WriteChecksum()
	config.Checksum = nil

	fixture := `
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
[deps.foo]
  import = "github.com/calavera/foo"
`
	createFixtureConfig(pwd, fixture)

	if !config.FetchDependencies() {
		t.Errorf("Expected fetch dependencies to be true when the checksum changes")
	}
}
