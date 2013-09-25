package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func createSourceFixture(dir, name, fixture string) {
	os.MkdirAll(dir, 0755)
	err := ioutil.WriteFile(path.Join(dir, name), []byte(fixture), 0644)
	check(err)
}

func TestAnalyzeSourceTree(t *testing.T) {
	setupTestPwd()
	createSourceFixture(pwd, "foo.go", `package main
import "github.com/pelletier/go-toml"
`)

	stats, err := AnalyzeSourceTree(pwd)
	if err != nil {
		t.Fatal(err)
	}

	if stats.ImportStatsByPath["github.com/pelletier/go-toml"] == nil {
		t.Error("Expected to have go-toml in the dependencies tree")
	}
}

func TestAnalyzeSourceTreeIgnoresGopack(t *testing.T) {
	setupTestPwd()

	createSourceFixture(pwd, "foo.go", `package main
import "github.com/pelletier/go-toml"
`)

	createSourceFixture(path.Join(pwd, GopackDir, "src"), "foo.go", `package main
import "github.com/pelletier/go-toml"
`)

	stats, err := AnalyzeSourceTree(pwd)
	if err != nil {
		t.Fatal(err)
	}

	i := stats.ImportStatsByPath["github.com/pelletier/go-toml"]
	if len(i.ReferencePositions) == 2 {
		t.Error("Expected to ignore the gopack directory")
	}
}

func TestReferenceDifferentDependencies(t *testing.T) {
	setupTestPwd()

	createSourceFixture(pwd, "foo.go", `package main
import "github.com/pelletier/go-toml"
`)

	createSourceFixture(pwd, "bar.go", `package main
import "github.com/gorilla/mux"
`)

	stats, err := AnalyzeSourceTree(pwd)
	if err != nil {
		t.Fatal(err)
	}

	istats := stats.ImportStatsByPath["github.com/pelletier/go-toml"]
	if !istats.Remote {
		t.Errorf("Expected reference to be remote\n")
	}

	if len(istats.ReferencePositions) != 1 {
		t.Errorf("Expected to have 1 reference to github.com/pelletier/go-toml\n")
	}

	istats = stats.ImportStatsByPath["github.com/gorilla/mux"]
	if !istats.Remote {
		t.Errorf("Expected reference to be remote\n")
	}

	if len(istats.ReferencePositions) != 1 {
		t.Errorf("Expected to have 1 reference to github.com/gorilla/mux\n")
	}
}

func TestReferenceSameDependencies(t *testing.T) {
	setupTestPwd()

	createSourceFixture(pwd, "foo.go", `package main
import "github.com/pelletier/go-toml"
`)

	createSourceFixture(pwd, "bar.go", `package main
import "fmt"
import "github.com/pelletier/go-toml"
`)

	stats, err := AnalyzeSourceTree(pwd)
	if err != nil {
		t.Fatal(err)
	}

	istats := stats.ImportStatsByPath["github.com/pelletier/go-toml"]
	if !istats.Remote {
		t.Errorf("Expected reference to be remote\n")
	}

	if len(istats.ReferencePositions) != 2 {
		t.Errorf("Expected to have 2 references to github.com/pelletier/go-toml\n")
	}

	expected := fmt.Sprintf("* %s/bar.go:3\n* %s/foo.go:2", pwd, pwd)

	list := istats.ReferenceList()
	if list != expected {
		t.Errorf("Expected reference list to be %s but it was %s\n", expected, list)
	}
}

func TestReferenceLocalDependencies(t *testing.T) {
	setupTestPwd()

	createSourceFixture(pwd, "bar.go", `package main
import "fmt"
`)

	stats, err := AnalyzeSourceTree(pwd)
	if err != nil {
		t.Fatal(err)
	}

	istats := stats.ImportStatsByPath["fmt"]
	if istats.Remote {
		t.Errorf("Expected reference to not be remote\n")
	}
}

func TestUsedDependencies(t *testing.T) {
	setupTestPwd()

	createSourceFixture(pwd, "bar.go", `package main
import "fmt"
`)

	stats, err := AnalyzeSourceTree(pwd)
	if err != nil {
		t.Fatal(err)
	}

	if !stats.IsImportUsed("fmt") {
		t.Errorf("Expected fmt to be used\n")
	}

	if stats.IsImportUsed("os") {
		t.Errorf("Expected os to not be used\n")
	}
}

func TestGetStatsSummary(t *testing.T) {
	setupTestPwd()

	createSourceFixture(pwd, "foo.go", `package main
import "github.com/pelletier/go-toml"
`)

	createSourceFixture(pwd, "bar.go", `package main
import "fmt"
import "./foo"
import "github.com/pelletier/go-foo"
import "github.com/pelletier/go-toml"
`)

	stats, err := AnalyzeSourceTree(pwd)
	if err != nil {
		t.Fatal(err)
	}

	s := stats.GetSummary()

	checkSumaryItem(t, s.Get(0), "github.com/pelletier/go-toml", "R	github.com/pelletier/go-toml	2")
	checkSumaryItem(t, s.Get(1), "github.com/pelletier/go-foo", "R	github.com/pelletier/go-foo	1")
	checkSumaryItem(t, s.Get(2), "./foo", "L	./foo	1")
	checkSumaryItem(t, s.Get(3), "fmt", "S	fmt	1")
}

func checkSumaryItem(t *testing.T, item SummaryItem, path, legend string) {
	if item.Path != path {
		t.Errorf("Expected item to be %s\n", path)
	}

	actual := item.Legend()
	if actual != legend {
		t.Errorf("Expected legend to be %s, but was %s\n", legend, actual)
	}
}
