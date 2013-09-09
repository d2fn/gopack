package main

import (
	"os"
	"path"
	"testing"
)

func createFixtureConfig(dir string, config string) {
	file, err := os.Create(path.Join(dir, "gopack.config"))
	check(err)
	defer file.Close()

	_, err = file.WriteString(config)
	file.Sync()
}

func TestNewConfig(t *testing.T) {
	setupTestPwd()
	setupEnv()

	fixture := `
repo = "github.com/d2fn/gopack"

[deps.testgopack]
  import = "github.com/calavera/testGoPack"
`
	createFixtureConfig(pwd, fixture)
	config := NewConfig(pwd)

	if config.Repository == "" {
		t.Error("Expected repository to not be empty.")
	}

	if config.DepsTree == nil {
		t.Error("Expected dependency tree to not be empty.")
	}
}

func TestInitRepoWithoutRepo(t *testing.T) {
	setupTestPwd()
	setupEnv()

	fixture := `
[deps.testgopack]
  import = "github.com/calavera/testGoPack"
`
	createFixtureConfig(pwd, fixture)

	graph := NewGraph()
	config := NewConfig(pwd)
	config.InitRepo(graph)

  src := path.Join(pwd, VendorDir, "src")
	_, err := os.Stat(src)

	if !os.IsNotExist(err) {
		t.Errorf("Expected vendor to not exist in %s\n", pwd)
	}
}

func TestInitRepo(t *testing.T) {
	setupTestPwd()
	setupEnv()

	fixture := `repo = "github.com/d2fn/gopack"`
	createFixtureConfig(pwd, fixture)

	graph := NewGraph()
	config := NewConfig(pwd)
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


