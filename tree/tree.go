package tree

type Tree interface {
	GetParent() Tree
	GetChild(i int) Tree
	NumChildren() int
}

type WalkDirection int

const (
	Forward WalkDirection = iota
	Reverse
)

type WalkOrder int

const (
	PreOrder WalkOrder = iota
	PostOrder
)

// Visitor is the visitor function for a tree walk.
// if cont is false on return, the walk terminates. If skipChildren is true
// on return the children and their descendants of the current node are
// skipped.
type Visitor func(t Tree, depth int) (continu, skipChildren bool)

// Walk walks the tree of which `tree` is a member node. This function walks the node
// and it's children, but also flows up to the parent if this is not the root of the tree.
// It's like continuing a tree walk from the root of the tree that was interrupted at the
// specified node.
//
// `dir` specifies whether the walk of children is done from the last to first,
// or first to last. As well WalkOrder specifies whether the parent is printed before or after children.
// skip: if true, skip the current node and it's children are not walked
func Walk(tree Tree, visitor Visitor, dir WalkDirection, order WalkOrder, depth int, skip bool) {
	walk(tree, visitor, dir, order, depth, skip)

	// Now continue the walk of the tree from the sibling before/after this node.
	walkSiblings(tree, visitor, dir, order, depth)
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
			visitor(tree, depth)
		}

		for ; i != end; i += inc {
			ch := tree.GetChild(i)
			walk(ch, visitor, dir, order, depth+1, false)
		}

		if order == PostOrder {
			visitor(tree, depth)
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
		visitor(tree.GetParent(), depth-1)
	}
	walkSiblings(tree.GetParent(), visitor, dir, order, depth-1)
}

/*
func Next(tree Tree, visitor Visitor, dir WalkDirection, order WalkOrder, depth int, skip bool) {
	if tree == nil {
		return
	}

	// Get first child.
	if order == PreOrder && tree.NumChildren() > 0 {
		return tree.GetChild(0)
	}

	// Nope, get next sibling
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

		if ch == tree {
			ignore = false
			continue
		}
		if ignore {
			if ch == tree {
				ignore = false
			}
			continue
		}

		walk(ch, visitor, dir, order, depth, false)
	}

}
*/
