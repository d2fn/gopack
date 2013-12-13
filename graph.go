package main

import (
	"container/list"
	"strings"
)

type Graph struct {
	Nodes map[string]*Node
	Leafs *list.List
}

type Node struct {
	Key        string
	Dependency *Dep
	Leaf       bool
	Nodes      map[string]*Node
}

func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
		Leafs: list.New()}
}

func (graph *Graph) Insert(dependency *Dep) {
	keys := strings.Split(dependency.Import, "/")
	graph.Nodes[keys[0]] = graph.deepInsert(graph.Nodes, keys, dependency)
}

func (graph *Graph) Search(importPath string) *Node {
	keys := strings.Split(importPath, "/")

	nodes := graph.Nodes
	for _, key := range keys {
		node := nodes[key]
		if node == nil {
			return nil
		}

		if node.Leaf {
			return node
		}

		nodes = node.Nodes
	}

	return nil
}

func (graph *Graph) deepInsert(nodes map[string]*Node, keys []string, dependency *Dep) *Node {
	node, found := nodes[keys[0]]
	if found == false {
		node = &Node{Key: keys[0], Nodes: make(map[string]*Node)}
	}

	newKeys := keys[1:]
	if len(newKeys) == 0 {
		node.Dependency = dependency
		node.Leaf = true

		graph.Leafs.PushBack(node.Dependency.Import)
	} else {
		node.Nodes[newKeys[0]] = graph.deepInsert(node.Nodes, newKeys, dependency)
	}

	return node
}

func (graph *Graph) PreOrderVisit(fn func(n *Node, depth int)) {
	for _, node := range graph.Nodes {
		node.PreOrderVisit(fn, 0)
	}
}

func (parent *Node) PreOrderVisit(fn func(n *Node, depth int), depth int) {
	for _, node := range parent.Nodes {
		fn(node, depth)
		if !node.Leaf {
			node.PreOrderVisit(fn, depth+1)
		}
	}
}
