package main

import (
	"fmt"
	"github.com/pelletier/go-toml"
	"os"
	"path/filepath"
)

type Config struct {
	// Name of your repository "github.com/d2fn/gopack" for instance.
	Repository string
	// Dependencies tree
	DepsTree *toml.TomlTree
}

func NewConfig(dir string) *Config {
	path := fmt.Sprintf("%s/gopack.config", dir)

	t, err := toml.LoadFile(path)
	if err != nil {
		fail(err)
	}

	config := new(Config)
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
		if !os.IsExist(err) {
			fail(err)
		}

		dependency := NewDependency(c.Repository)
		importGraph.Insert(dependency)
	}
}
