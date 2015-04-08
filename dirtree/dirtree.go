package dirtree

import (
  "errors"
  "fmt"
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
func (d *Dirtree) SetRootCopy(root *Node) {
  d.origToCopy[root] = d.Root()
  d.Root().Dir.Basename = root.Dir.Basename
  d.Root().Dir.Path = root.Dir.Path
}

// AddCopy adds a copy of the specified node that exists in an original Dirtree to this copy Dirtree. 
// This function is used when the current tree is a copy of a different, original tree.
// Given origNode, which is a node in the original tree, add a copy of that node and it's descendants to this tree.
func (d *Dirtree) AddCopy(origNode *Node) (*Node, error) {
  n, err := d.addCopy(origNode)
  if err != nil {
    return nil, err
  }

fmt.Println("Adding copy",origNode.Dir.Basename)

  parentCopy := d.origToCopy[origNode.Parent]
  parentCopy.addSize(origNode.Dir.Size)

  return n, err
}

// Private addCopy function. This one recursively adds descendants but doesn't update the Size of ancestors.
func (d *Dirtree) addCopy(origNode *Node) (*Node, error) {
  parentCopy, ok := d.origToCopy[origNode.Parent]

  if !ok {
    return nil, errors.New("AddCopy can't find copy of parent for original node")
  }

  node := &Node{Parent: parentCopy, Dir: origNode.Dir, Children: []*Node{}}
  parentCopy.add(node, false)

  d.origToCopy[origNode] = node

  for _, ch := range origNode.Children {
fmt.Println("--> Adding child copy",ch.Dir.Basename)
    d.addCopy(ch)
  }

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
func (d *Dirtree) UpdateCopy(origNode *Node) error {
  node, ok := d.origToCopy[origNode]
  if !ok {
    return errors.New("UpdateCopy can't find copy of parent for original node")
  }

  delta := origNode.Dir.Size - node.Dir.Size

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
