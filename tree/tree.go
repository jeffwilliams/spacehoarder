package tree

type Tree interface {
	GetParent() Tree
	GetChild(i int) Tree
	NumChildren() int
}

// WalkDirection is the direction to process children in a tree walk.
type WalkDirection int

const (
	// Forward indicates the walk should process children first to last (0..n)
	Forward WalkDirection = iota
	// Reverse indicates the walk should process children last to first (n..0)
	Reverse
)

// WalkOrder specifies whether the walk is a pre-order or post-order walk (i.e. if the parent
// is processed before children or after children)
type WalkOrder int

const (
	PreOrder WalkOrder = iota
	PostOrder
)

// Visitor is the visitor function for a tree walk.
// If cont is false on return, the walk terminates. If skipChildren is true
// on return the children and their descendants of the current node are
// skipped.
type Visitor func(t Tree, depth int) (continu bool)

// Walk walks the tree of which `tree` is a member node. This function walks the node
// and it's children, but also flows up to the parent if this is not the root of the tree.
// It's like continuing a tree walk from the root of the tree that was interrupted at the
// specified node.
//
// `dir` specifies whether the walk of children is done from the last to first,
// or first to last. As well WalkOrder specifies whether the parent is printed before or after children.
// skip: if true, skip the current node and it's children are not walked
func Walk(tree Tree, visitor Visitor, dir WalkDirection, order WalkOrder, depth int, skip bool) {
	defer endWalk()

	walk(tree, visitor, dir, order, depth, skip)

	// Now continue the walk of the tree from the sibling before/after this node.
	walkSiblings(tree, visitor, dir, order, depth)
}

type endWalkType struct{}

func visit(visitor Visitor, tree Tree, depth int) {
	if !visitor(tree, depth) {
		panic(endWalkType{})
	}
}

func endWalk() {
	v := recover()
	if v != nil {
		if _, ok := v.(endWalkType); !ok {
			panic(v)
		}
	}
}

func walk(tree Tree, visitor Visitor, dir WalkDirection, order WalkOrder, depth int, skip bool) {
	if tree == nil {
		return
	}

	i := 0
	inc := 1
	end := tree.NumChildren()

	if dir == Reverse {
		i = end - 1
		inc = -1
		end = -1
	}

	// Visit this node and it's decendants, if desired
	if !skip {
		if order == PreOrder {
			visit(visitor, tree, depth)
		}

		for ; i != end; i += inc {
			ch := tree.GetChild(i)
			walk(ch, visitor, dir, order, depth+1, false)
		}

		if order == PostOrder {
			visit(visitor, tree, depth)
		}
	}

}

// Walk the siblings of `tree` which has depth `depth`.
func walkSiblings(tree Tree, visitor Visitor, dir WalkDirection, order WalkOrder, depth int) {
	if tree.GetParent() == nil {
		return
	}

	i := 0
	inc := 1
	end := tree.GetParent().NumChildren()

	if dir == Reverse {
		i = end - 1
		inc = -1
		end = -1
	}

	ignore := true

	for ; i != end; i += inc {
		ch := tree.GetParent().GetChild(i)

		if ignore {
			if ch == tree {
				ignore = false
			}
			continue
		}

		walk(ch, visitor, dir, order, depth, false)
	}

	if tree.GetParent() != nil && order == PostOrder {
		visit(visitor, tree.GetParent(), depth-1)
	}
	walkSiblings(tree.GetParent(), visitor, dir, order, depth-1)
}

// Next returns the next element in a depth-first tree walk.
func Next(tree Tree, dir WalkDirection, order WalkOrder) Tree {
	/*
	   Overview:

	   PreOrder:

	   - if the node `tree` is not a leaf:
	   	- return the first child if dir is forward, or last child if dir is reverse
	   - if the node is `tree` is a leaf:
	   	- return next sibling/non-direct-ancestor: walk upwards recursively to find the first parent node
	       that has other children after the child we walked from, and then return the other
	       child in the correct direction.

	   PostOrder

	   - If this node has siblings in the correct direction (a later sibling for forward, or an earlier
	   	sibling for reverse), find the lowest leaf in the correct direction of that sibling.
	   - If no unprocessed siblings, return the parent node.
	*/

	if tree == nil {
		return nil
	}

	isLeaf := func(tree Tree) bool {
		return tree.NumChildren() == 0
	}

	// Find the first child after `ch` under `tree`
	childAfter := func(tree, ch Tree) Tree {
		i := 0
		inc := 1
		end := tree.NumChildren()

		if dir == Reverse {
			i = end - 1
			inc = -1
			end = -1
		}

		ignore := true
		for ; i != end; i += inc {
			c := tree.GetChild(i)

			if ignore {
				if c == ch {
					ignore = false
				}
				continue
			}

			return c
		}
		return nil
	}

	leafUnder := func(tree Tree) Tree {
		n := tree
		for !isLeaf(n) {
			if dir == Forward {
				n = n.GetChild(0)
			} else {
				n = n.GetChild(n.NumChildren() - 1)
			}
			if order == PreOrder {
				return n
			}
		}
		return n
	}

	// Calculate next for pre-order walk
	nextPreOrder := func(tree Tree) Tree {
		if !isLeaf(tree) {
			if dir == Forward {
				return tree.GetChild(0)
			} else {
				return tree.GetChild(tree.NumChildren() - 1)
			}
		} else {
			n := tree
			for ; n != nil; n = n.GetParent() {
				parent := n.GetParent()
				if parent == nil {
					// Done walk
					return nil
				}
				nxt := childAfter(parent, n)
				if nxt != nil {
					return nxt
				}
			}
			return nil
		}
	}

	nextPostOrder := func(tree Tree) Tree {
		parent := tree.GetParent()
		if parent == nil {
			// Done walk
			return nil
		}

		nxt := childAfter(parent, tree)
		if nxt == nil {
			return parent
		}

		return leafUnder(nxt)
	}

	if order == PreOrder {
		return nextPreOrder(tree)
	} else {
		return nextPostOrder(tree)
	}

}
