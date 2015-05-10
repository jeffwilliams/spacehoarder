package dirtree

import (
  "errors"
  "github.com/jeffwilliams/spacehoarder/squarify"
)

// A node within a Dirtree. Note that the order of Children is not preserved when using operations below.
type Node struct {
  Parent *Node
  Dir Directory
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
  for i,v := range n.Children {
    if v == child {
      last := len(n.Children)-1

      if i != last {
        // Move last node to i 
        n.Children[i] = n.Children[last]
      }

      // Strip off last node.
      n.Children[last] = nil
      n.Children = n.Children[0:len(n.Children)-1]

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
  for _,v := range n.Children {
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
  root Node

  // If this tree is a copy of an original tree, this maps a pointer to the original node to the matching copy in this tree.
  origToCopy map[*Node]*Node
}

// New creates a new, empty Dirtree
func New() *Dirtree {
  return &Dirtree{origToCopy: make(map[*Node]*Node)}
}

// Root returns the root node of the Dirtree.
func (d *Dirtree) Root() *Node {
  return &d.root
}

// If this Dirtree is going to be a copy of another Dirtree, then SetRootCopy must be called
// with the root of the original tree after initialization, so that the AddCopy method is able
// to locate the parents of nodes.
func (d *Dirtree) SetRootCopy(root *Node, size int64) {
  d.origToCopy[root] = d.Root()
  d.Root().Dir.Basename = root.Dir.Basename
  d.Root().Dir.Path = root.Dir.Path
  d.Root().Dir.Size = size
}

// AddCopy adds a copy of the specified node that exists in an original Dirtree to this copy Dirtree. 
// This function is used when the current tree is a copy of a different, original tree.
// Given origNode, which is a node in the original tree, add a copy of that node and it's descendants to this tree.
//
// There are several preconditions that are required for this function to operate correctly:
//
//    * The node being added must already be parented in the original tree
//    * There must be an AddCopy for each node Added to the original tree. In other words, you cannot build a subtree of Nodes
//      and then Add that tree to the original tree. This is enforced so that the node being passed to AddCopy can still 
//      mutate in the original tree before this AddCopy method is called; if we instead added all chidren of this node
//      we may then later get duplicate Add requests for the children.
//    * size should be set to the size of the node when it was added to the tree.
func (d *Dirtree) AddCopy(origNode *Node, size int64) (*Node, error) {
  n, err := d.addCopy(origNode, size)
  if err != nil {
    return nil, err
  }

  parentCopy := d.origToCopy[origNode.Parent]
  parentCopy.addSize(size)

  return n, err
}

// Private addCopy function. This one recursively adds descendants but doesn't update the Size of ancestors.
func (d *Dirtree) addCopy(origNode *Node, size int64) (*Node, error) {
  parentCopy, ok := d.origToCopy[origNode.Parent]

  if !ok {
    return nil, errors.New("AddCopy can't find copy of parent for original node")
  }

  node := &Node{Parent: parentCopy, Dir: origNode.Dir, Children: []*Node{}}
  node.Dir.Size = size
  parentCopy.add(node, false)

  d.origToCopy[origNode] = node

  return node, nil
}

// This function is used when the current tree is a copy of a different, original tree.
// Given origNode, which is a node in the original tree, add a copy of that node to this tree.
func (d *Dirtree) DelCopy(origNode *Node) error {
  parentCopy, ok := d.origToCopy[origNode.Parent]
  if !ok {
    return errors.New("DelCopy can't find copy of parent for original node")
  }

  nodeCopy, ok := d.origToCopy[origNode]
  if !ok {
    return errors.New("DelCopy can't find copy of original node")
  }

  parentCopy.Del(nodeCopy)
  d.delCopyFromMap(origNode)

  return nil
}

// Update the Dir property of the copy of origNode in this tree
func (d *Dirtree) UpdateCopy(origNode *Node, size int64) error {
  node, ok := d.origToCopy[origNode]
  if !ok {
    return errors.New("UpdateCopy can't find copy of parent for original node")
  }

  delta := size - node.Dir.Size

  node.Dir = origNode.Dir

  if node.Parent != nil {
    node.Parent.addSize(delta)
  }

  return nil
}

// Delete the copy of the specified node and all of it's descendants copies from the origToCopy map.
func (d *Dirtree) delCopyFromMap(origNode *Node) {
  delete(d.origToCopy, origNode)

  for _,v := range origNode.Children {
    d.delCopyFromMap(v)
  }
}
