package dirtree

import (
	"github.com/jeffwilliams/squarify"
)

// A node within a Dirtree. Note that the order of Children is not preserved when using operations below.
type Node struct {
	Parent   *Node
	Dir      Directory
	Children []*Node
}

// Add adds a child node to this node.
func (n *Node) Add(child *Node) {
	n.add(child, true)
}

// Add the specified node, but optionally don't update the ancestor node directory sizes.
func (n *Node) add(child *Node, updateSize bool) {
	n.Children = append(n.Children, child)
	child.Parent = n
	if updateSize {
		n.addSize(child.Dir.Size)
	}
}

// Del removes the specified child node from this node.
func (n *Node) Del(child *Node) {
	n.del(child, true)
}

// Delete the specified node, but optionally don't update the ancestor node directory sizes.
func (n *Node) del(child *Node, updateSize bool) {
	for i, v := range n.Children {
		if v == child {
			last := len(n.Children) - 1

			if i != last {
				// Move last node to i
				n.Children[i] = n.Children[last]
			}

			// Strip off last node.
			n.Children[last] = nil
			n.Children = n.Children[0 : len(n.Children)-1]

			if updateSize {
				n.addSize(-v.Dir.Size)
			}
			break
		}
	}
}

// UpdateSize updates the size of the directory in the node, and updates the size of the ancestors as well.
func (n *Node) UpdateSize(size int64) {
	delta := size - n.Dir.Size
	n.addSize(delta)
}

// Add size bytes to the size of this node and all ancestors.
func (n *Node) addSize(size int64) {
	n.Dir.Size += size
	if n.Parent != nil {
		n.Parent.addSize(size)
	}
}

func (n *Node) Walk(visitor func(n *Node)) {
	visitor(n)
	for _, v := range n.Children {
		v.Walk(visitor)
	}
}

// Needed to implement TreeSizer
func (n *Node) Size() float64 {
	return float64(n.Dir.Size)
}

// Needed to implement TreeSizer
func (n *Node) NumChildren() int {
	return len(n.Children)
}

// Needed to implement TreeSizer
func (n *Node) Child(i int) squarify.TreeSizer {
	return n.Children[i]
}

// Dirtree is a directory tree where each node has a Size property that's the size of the contents of the directory
// and all descendent directories.
type Dirtree struct {
	Root *Node
}

// New creates a new, empty Dirtree
func New() *Dirtree {
	return &Dirtree{}
}
