package dirtree

import (
	"sort"

	"github.com/jeffwilliams/spacehoarder/tree"
	"github.com/jeffwilliams/squarify"
)

// A node within a Dirtree. Note that the order of Children is not preserved when using operations below.
type Node struct {
	Parent   *Node
	Dir      Directory
	Children []*Node
	UserData interface{}
	// SortChildren specifies whether the children of this node should be sorted from biggest to smallest.
	SortChildren bool
}

func (n *Node) sortChildren() {
	if n.SortChildren {
		sort.SliceStable(n.Children, func(i, j int) bool {
			return n.Children[i].Dir.Size > n.Children[j].Dir.Size
		})
	}
}

// Add adds a child node to this node, returning it.
func (n *Node) Add(child *Node) *Node {
	n.add(child, true)
	n.sortChildren()
	return child
}

// Add the specified node, but optionally don't update the ancestor node directory sizes.
func (n *Node) add(child *Node, updateSize bool) {
	n.Children = append(n.Children, child)
	child.Parent = n
	child.SortChildren = n.SortChildren
	n.sortChildren()
	if updateSize {
		n.addSize(child.Dir.Size)
	}
}

// Del removes the specified child node from this node.
func (n *Node) Del(child *Node) {
	n.del(child, true)
	n.sortChildren()
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
	n.sortChildren()
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
	n.sortChildren()
}

// Visitor is the visitor function for a pre-order tree walk.
// if cont is false on return, the walk terminates. If skipChildren is true
// on return the children and their descendants of the current node are
// skipped.
type Visitor func(n *Node, depth int) (cont, skipChildren bool)

// Walk performs a pre-order tree walk.
func (n *Node) Walk(visitor Visitor, depth int) bool {
	if n == nil {
		return false
	}

	cont, skipCh := visitor(n, depth)

	if !cont {
		return false
	}

	if skipCh {
		return true
	}

	if n.Children == nil {
		return true
	}

	for _, v := range n.Children {
		cont = v.Walk(visitor, depth+1)
		if !cont {
			return false
		}
	}

	return true
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
	if i < 0 || i > len(n.Children) {
		return nil
	}
	return n.Children[i]
}

// Needed to implement tree.Tree
func (n *Node) GetChild(i int) tree.Tree {
	if i < 0 || i > len(n.Children) {
		return nil
	}
	return n.Children[i]
}

// Needed to implement tree.Tree
func (n *Node) GetParent() tree.Tree {
	if n.Parent == nil {
		return nil
	}
	return n.Parent
}

func (n *Node) Depth() int {
	if n.Parent != nil {
		return n.Parent.Depth() + 1
	} else {
		return 0
	}
}

/*
func (n *Node) Walk(visitor Visitor, depth int) {
	visitor(n, depth)

	for _, ch := range n.Children {
		ch.Walk(visitor, depth+1)
	}
}
*/
type applyContext struct {
	curNode *Node
	work    []*Node
}

// Dirtree is a directory tree where each node has a Size property that's the size of the contents of the directory
// and all descendent directories.
type Dirtree struct {
	Root         *Node
	applyCtx     applyContext
	SortChildren bool
}

// New creates a new, empty Dirtree
func New() *Dirtree {
	return &Dirtree{}
}

func (t *Dirtree) Apply(op OpData) (added *Node) {
	if t.applyCtx.work == nil {
		// Directories to process
		t.applyCtx.work = make([]*Node, 0, 1000)
	}

	push := func(op OpData) {
		node := &Node{Dir: Directory{Path: op.Path, Basename: op.Basename}}
		added = node

		// Push is used to add a child to the current tree node and also
		// to add the root to the tree. We distinguish by checking if
		// curNode is nil.
		if t.applyCtx.curNode == nil {
			if t.Root != nil {
				panic("Apply: curNode is nil but tree Root is not nil")
			}
			t.Root = node
			if t.SortChildren {
				t.Root.SortChildren = true
			}
		} else {
			t.applyCtx.curNode.Add(node)
		}

		t.applyCtx.work = append(t.applyCtx.work, node)
	}

	pop := func() {
		t.applyCtx.curNode = t.applyCtx.work[len(t.applyCtx.work)-1]
		t.applyCtx.work = t.applyCtx.work[0 : len(t.applyCtx.work)-1]
	}

	addSize := func(op OpData) {
		size := t.applyCtx.curNode.Dir.Size + op.Size
		t.applyCtx.curNode.UpdateSize(size)
	}

	switch op.Op {
	case Push:
		push(op)
	case Pop:
		pop()
	case AddSize:
		addSize(op)
	}
	return
}

func (t *Dirtree) ApplyAll(ops chan OpData) {
	for op := range ops {
		t.Apply(op)
	}
}
