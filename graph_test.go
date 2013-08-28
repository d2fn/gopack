package main

import (
	"strings"
	"testing"
)

func TestSingleNode(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com"}

	graph.Insert(dep)
	root := graph.Nodes["github.com"]
	if root == nil {
		t.Error("Expected root not to be nil")
	}

	if root.Dependency != dep {
		t.Errorf("Expected Node.Dependency to be %v", dep)
	}

	if root.Leaf == false {
		t.Errorf("Expected root to be also a leaf")
	}
}

func TestGraphWithSeveralRoots(t *testing.T) {
	graph := NewGraph()
	dep1 := &Dep{Import: "github.com"}
	dep2 := &Dep{Import: "code.google.com"}

	graph.Insert(dep1)
	graph.Insert(dep2)

	root := graph.Nodes["github.com"]
	if root == nil {
		t.Error("Expected root not to be nil")
	}

	root = graph.Nodes["code.google.com"]
	if root == nil {
		t.Error("Expected root not to be nil")
	}
}

func TestDeepGraph(t *testing.T) {
	graph := NewGraph()
	dep1 := &Dep{Import: "github.com/d2fn/gopack"}
	dep2 := &Dep{Import: "code.google.com/p/go.net"}

	graph.Insert(dep1)
	graph.Insert(dep2)

	testTree(dep1, graph, t)
	testTree(dep2, graph, t)
}

func testTree(dep *Dep, graph *Graph, t *testing.T) {
	nodes := graph.Nodes
	keys := strings.Split(dep.Import, "/")

	for idx, key := range keys {
		node := nodes[key]
		if node == nil {
			t.Error("Expected node to not be nil")
		}

		if idx < len(keys)-1 {
			if node.Leaf == true {
				t.Error("Expected leaf to not be a leaf")
			}

			if node.Dependency != nil {
				t.Errorf("Expected node to not store the dependency")
			}

			nodes = node.Nodes
		} else {
			if node.Leaf == false {
				t.Error("Expected node to be a leaf")
			}

			if node.Dependency != dep {
				t.Errorf("Expected node to store the dependency")
			}
		}
	}
}

func TestSearchFailsWithNoNodes(t *testing.T) {
	graph := NewGraph()
	node := graph.Search("github.com/d2fn/gopack")

	if node != nil {
		t.Error("Expected search to fail when there are no nodes")
	}
}

func TestSearchFailsWithDifferentNodes(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack"}
	graph.Insert(dep)

	node := graph.Search("github.com/dotcloud/docker")
	if node != nil {
		t.Error("Expected search to fail when the dependency doesn't exist")
	}
}

func TestSearchWorksWithBareNames(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack"}
	graph.Insert(dep)

	node := graph.Search("github.com/d2fn/gopack")
	if node == nil {
		t.Error("Expected search to succeed importing bare repos")
	}
}

func TestSearchWorksWithExtendedNames(t *testing.T) {
	graph := NewGraph()
	dep := &Dep{Import: "github.com/d2fn/gopack"}
	graph.Insert(dep)

	node := graph.Search("github.com/d2fn/gopack/graph")
	if node.Dependency != dep {
		t.Error("Expected search to succeed importing extended repos")
	}
}
