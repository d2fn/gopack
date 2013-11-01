package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/pelletier/go-toml"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Config struct {
	Checksum []byte
	// Path to the configuration file.
	Path string
	// Name of your repository "github.com/d2fn/gopack" for instance.
	Repository string
	// Dependencies tree
	DepsTree *toml.TomlTree
}

func NewConfig(dir string) *Config {
	config := &Config{Path: fmt.Sprintf("%s/gopack.config", dir)}

	t, err := toml.LoadFile(config.Path)
	if err != nil {
		fail(err)
	}

	if deps := t.Get("deps"); deps != nil {
		config.DepsTree = deps.(*toml.TomlTree)
	}

	if repo := t.Get("repo"); repo != nil {
		config.Repository = repo.(string)
	}

	return config
}

func (c *Config) InitRepo(importGraph *Graph) {
	if c.Repository != "" {
		src := fmt.Sprintf("%s/%s/src", pwd, VendorDir)
		os.MkdirAll(src, 0755)

		dir := filepath.Dir(c.Repository)
		base := fmt.Sprintf("%s/%s", src, dir)
		os.MkdirAll(base, 0755)

		repo := fmt.Sprintf("%s/%s", src, c.Repository)
		err := os.Symlink(pwd, repo)
		if err != nil && !os.IsExist(err) {
			fail(err)
		}

		dependency := NewDependency(c.Repository)
		importGraph.Insert(dependency)
	}
}

func (c *Config) modifiedChecksum() bool {
	dat, err := ioutil.ReadFile(c.checksumPath())
	return (err != nil && os.IsNotExist(err)) || !bytes.Equal(dat, c.checksum())
}

func (c *Config) WriteChecksum() {
	os.MkdirAll(filepath.Join(pwd, GopackDir), 0755)
	err := ioutil.WriteFile(c.checksumPath(), c.checksum(), 0644)

	if err != nil {
		fail(err)
	}
}

func (c *Config) checksumPath() string {
	return filepath.Join(pwd, GopackChecksum)
}

func (c *Config) checksum() []byte {
	if c.Checksum == nil {
		dat, err := ioutil.ReadFile(c.Path)
		if err != nil {
			fail(err)
		}

		h := md5.New()
		h.Write(dat)
		c.Checksum = h.Sum(nil)
	}
	return c.Checksum
}

func (c *Config) LoadDependencyModel(importGraph *Graph) (deps *Dependencies) {
	depsTree := c.DepsTree

	if depsTree == nil {
		return
	}

	deps = new(Dependencies)

	deps.Imports = make([]string, len(depsTree.Keys()))
	deps.Keys = make([]string, len(depsTree.Keys()))
	deps.DepList = make([]*Dep, len(depsTree.Keys()))
	deps.ImportGraph = importGraph

	modifiedChecksum := c.modifiedChecksum()

	for i, k := range depsTree.Keys() {
		depTree := depsTree.Get(k).(*toml.TomlTree)
		d := NewDependency(depTree.Get("import").(string))

		d.setCheckout(depTree, "branch", BranchFlag)
		d.setCheckout(depTree, "commit", CommitFlag)
		d.setCheckout(depTree, "tag", TagFlag)

		d.CheckValidity()
		d.fetch = modifiedChecksum || d.CheckoutFlag == BranchFlag

		deps.Keys[i] = k
		deps.Imports[i] = d.Import
		deps.DepList[i] = d

		deps.ImportGraph.Insert(d)
	}

	return
}
