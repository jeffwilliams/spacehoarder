package dirtree

import (
	"log"
	"sort"

	"github.com/jeffwilliams/spacehoarder/tree"
	"github.com/jeffwilliams/squarify"
)

// A node within a Dirtree. Note that the order of Children is not preserved when using operations below.
type Node struct {
	Parent   *Node
	Info     PathInfo
	Children []*Node
	UserData interface{}
	// SortChildren specifies whether the children of this node should be sorted from biggest to smallest.
	SortChildren bool
}

func (n *Node) sortChildren() {
	if n.SortChildren {
		sort.SliceStable(n.Children, func(i, j int) bool {
			return n.Children[i].Info.Size > n.Children[j].Info.Size
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
		n.addSize(child.Info.Size, true)
	}
}

// Delete all children
func (n *Node) DelAll() {
	n.Children = n.Children[0:0]
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
				n.addSize(-v.Info.Size, true)
			}
			break
		}
	}
	n.sortChildren()
}

// UpdateSize updates the size of the directory in the node, and updates the size of the ancestors as well.
func (n *Node) UpdateSize(size int64, sizeAccurate bool) {
	delta := size - n.Info.Size
	n.addSize(delta, sizeAccurate)
}

// Add size bytes to the size of this node and all ancestors.
func (n *Node) addSize(size int64, sizeAccurate bool) {
	n.Info.Size += size
	if n.Info.SizeAccurate {
		n.Info.SizeAccurate = sizeAccurate
	}
	if n.Parent != nil {
		n.Parent.addSize(size, sizeAccurate)
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
	return float64(n.Info.Size)
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
	if n != nil && n.Parent != nil {
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
type ApplyContext struct {
	curNode *Node
	work    []*Node
}

func NewApplyContext(root *Node) *ApplyContext {
	return &ApplyContext{curNode: root, work: make([]*Node, 0, 1000)}
}

// Dirtree is a directory tree where each node has a Size property that's the size of the contents of the directory
// and all descendent directories.
type Dirtree struct {
	Root         *Node
	applyCtx     *ApplyContext
	SortChildren bool
}

// New creates a new, empty Dirtree
func New() *Dirtree {
	return &Dirtree{}
}

func (t *Dirtree) Apply(op OpData) (added *Node) {
	if t.applyCtx == nil {
		// Directories to process
		t.applyCtx = NewApplyContext(nil)
	}
	return t.ApplyCtx(t.applyCtx, op)
}

func (t *Dirtree) ApplyCtx(ctx *ApplyContext, op OpData) (added *Node) {
	push := func(op OpData) {
		node := &Node{Info: PathInfo{Path: op.Path, Basename: op.Basename, SizeAccurate: true, Type: op.Type, Size: op.Size}}
		added = node

		log.Printf("Dirtree.ApplyCtx: push operation. Current Tree Node = %v. Operation data = %v\n", ctx.curNode, op)
		// Push is used to add a child to the current tree node and also
		// to add the root to the tree. We distinguish by checking if
		// curNode is nil.
		if ctx.curNode == nil {
			if t.Root != nil {
				panic("Apply: curNode is nil but tree Root is not nil")
			}
			t.Root = node
			if t.SortChildren {
				t.Root.SortChildren = true
			}
		} else {
			if op.Path != ctx.curNode.Info.Path {
				log.Printf("Dirtree.ApplyCtx: push operation: adding op under current node\n")
				ctx.curNode.Add(node)
			} else {
				node = ctx.curNode
			}
		}

		if op.Type != PathTypeFile {
			ctx.work = append(ctx.work, node)
		}
	}

	pop := func() {
		ctx.curNode = ctx.work[len(ctx.work)-1]
		ctx.work = ctx.work[0 : len(ctx.work)-1]
		log.Printf("Dirtree.ApplyCtx: pop operation. Current Tree Node after = %v\n", ctx.curNode)
	}

	addSize := func(op OpData) {
		log.Printf("Dirtree.ApplyCtx: addSize operation. Current Tree Node = %v. Operation data = %v\n", ctx.curNode, op)
		size := ctx.curNode.Info.Size + op.Size
		ctx.curNode.UpdateSize(size, op.SizeAccurate)
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
