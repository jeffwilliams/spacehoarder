package dirtree

import (
  "testing"
  "fmt"
)

func sameElems(a,b []*Node) bool {
  for _,i := range a {
    found := false
    for _,j := range b {
      if i == j {
        found = true
        break
      }
    }
    if ! found {
      return false
    }
  }
  return true
}

func hasBasenames(a []*Node, names []string) bool {
  for _,j := range names {
    found := false
    for _,i := range a {
      if i.Dir.Basename == j {
        found = true
        break
      }
    }
    if ! found {
      return false
    }
  }
  return true
}

func dumpOrigMap(t *Dirtree) {
  for k,v := range t.origToCopy {
    fmt.Printf("%p --> %p\n", k, v)
  }
}

func childWithBasename(n *Node, name string) *Node {
  for _,v := range n.Children {
    if v.Dir.Basename == name {
      return v
    }
  }
  return nil
}

func TestNodeAdd(t *testing.T) {
  p := &Node{Dir: Directory{Size: 20}}

  n1 := &Node{}

  p.Add(n1)

  if len(p.Children) != 1 {
    t.Fatal("Adding node failed")
  }

  if n1.Parent != p {
    t.Fatal("Adding node didn't reparent")
  }

  if p.Children[0] != n1 {
    t.Fatal("Adding node failed")
  }

  if p.Dir.Size != 20 {
    t.Fatal("Adding node didn't update root size correctly. Root size:", p.Dir.Size)
  }

  n2 := &Node{Dir: Directory{Size: 10}}

  p.Add(n2)

  if len(p.Children) != 2 {
    t.Fatal("Adding node failed")
  }

  if n2.Parent != p {
    t.Fatal("Adding node didn't reparent")
  }

  if ! sameElems(p.Children, []*Node{n1,n2}) {
    t.Fatal("Adding node failed")
  }

  if p.Dir.Size != 30 {
    t.Fatal("Adding node didn't update root size correctly. Root size:", p.Dir.Size)
  }
}

func TestNodeDel(t *testing.T) {
  p := &Node{}

  n1 := &Node{Dir: Directory{Size: 10}}
  n2 := &Node{Dir: Directory{Size: 20}}
  n3 := &Node{Dir: Directory{Size: 30}}

  // Make sure delete from empty parent doesn't panic
  p.Del(n1)

  p.Add(n1)

  p.Del(n1)

  if len(p.Children) != 0 {
    t.Fatal("Deleting child from parent with single child failed. Num children=", len(p.Children))
  }

  if p.Dir.Size != 0 {
    t.Fatal("Deleting child didn't update root size correctly. Root size is ", p.Dir.Size)
  }

  p.Add(n1)
  p.Add(n2)

  p.Del(n1)

  if len(p.Children) != 1 {
    t.Fatal("Deleting child from parent with two children failed. Num children=", len(p.Children))
  }

  if ! sameElems(p.Children, []*Node{n2}) {
    t.Fatal("Deleting child from parent with two children failed. Wrong child deleted.")
  }

  if p.Dir.Size != 20 {
    t.Fatal("Deleting child didn't update root size correctly. Root size is ", p.Dir.Size)
  }

  p.Del(n2)
  if len(p.Children) != 0 {
    t.Fatal("Deleting child failed. Num children=", len(p.Children))
  }

  p.Add(n1)
  p.Add(n2)

  p.Del(n2)

  if len(p.Children) != 1 {
    t.Fatal("Deleting child from parent with two children failed. Num children=", len(p.Children))
  }

  if ! sameElems(p.Children, []*Node{n1}) {
    t.Fatal("Deleting child from parent with two children failed. Wrong child deleted.")
  }

  if p.Dir.Size != 10 {
    t.Fatal("Deleting child didn't update root size correctly. Root size is ", p.Dir.Size)
  }

  p.Del(n1)
  if len(p.Children) != 0 {
    t.Fatal("Deleting child failed. Num children=", len(p.Children))
  }

  p.Add(n1)
  p.Add(n2)
  p.Add(n3)

  p.Del(n2)

  if len(p.Children) != 2 {
    t.Fatal("Deleting child from parent with three children failed. Num children=", len(p.Children))
  }

  if ! sameElems(p.Children, []*Node{n1, n3}) {
    t.Fatal("Deleting child from parent with two children failed. Wrong child deleted.")
  }

  if p.Dir.Size != 40 {
    t.Fatal("Deleting child didn't update root size correctly. Root size is ", p.Dir.Size)
  }
}

func TestAddCopy(t *testing.T) {
  p := &Node{}
  tree := New()
  tree.SetRootCopy(p, p.Dir.Size)

  n1 := &Node{Dir: Directory{Basename: "n1", Size: 10}}
  n2 := &Node{Dir: Directory{Basename: "n2", Size: 20}}
  n3 := &Node{Dir: Directory{Basename: "n3", Size: 30}}

  p.Add(n1)
  tree.AddCopy(n1, n1.Dir.Size)

  if p.Dir.Size != 10 {
    t.Fatal("Adding child didn't update root size correctly. Root size is ", p.Dir.Size)
  }

  p.Add(n2)
  tree.AddCopy(n2, n2.Dir.Size)

  if p.Dir.Size != 30 {
    t.Fatal("Adding child didn't update root size correctly. Root size is ", p.Dir.Size)
  }

  p.Add(n3)
  tree.AddCopy(n3, n3.Dir.Size)

  if p.Dir.Size != 60 {
    t.Fatal("Adding child didn't update root size correctly. Root size is ", p.Dir.Size)
  }

  if len(tree.Root().Children) != 3 {
    t.Fatal("Copying children failed: root has",len(tree.Root().Children),"children")
  }

  if ! hasBasenames(tree.Root().Children, []string{"n1","n2","n3"}) {
    t.Fatal("Copying children failed: root has",len(tree.Root().Children),"children")
  }
}

func TestDelCopy(t *testing.T) {
  p := &Node{}
  tree := New()
  tree.SetRootCopy(p, p.Dir.Size)

  n1 := &Node{Dir: Directory{Basename: "n1", Size: 10}}
  n2 := &Node{Dir: Directory{Basename: "n2", Size: 20}}
  n3 := &Node{Dir: Directory{Basename: "n3", Size: 30}}

  p.Add(n1)
  tree.AddCopy(n1, n1.Dir.Size)

  p.Add(n2)
  tree.AddCopy(n2, n2.Dir.Size)

  p.Add(n3)
  tree.AddCopy(n3, n3.Dir.Size)

  if tree.Root().Dir.Size != 60 {
    t.Fatal("Adding child didn't update root size correctly. Root size is ", tree.Root().Dir.Size)
  }

  err := tree.DelCopy(n1)
  if err != nil {
    t.Fatal("Deleting copied children failed: ",err)
  }

  if len(tree.Root().Children) != 2 {
    t.Fatal("Deleting copied children failed: root has",len(tree.Root().Children),"children")
  }

  if ! hasBasenames(tree.Root().Children, []string{"n2","n3"}) {
    t.Fatal("Deleting copyied children failed: root has",len(tree.Root().Children),"children")
  }

  if tree.Root().Dir.Size != 50 {
    t.Fatal("Deleting child didn't update root size correctly. Root size is ", tree.Root().Dir.Size)
  }

  tree.DelCopy(n3)

  if len(tree.Root().Children) != 1 {
    t.Fatal("Deleting copied children failed: root has",len(tree.Root().Children),"children")
  }

  if ! hasBasenames(tree.Root().Children, []string{"n2"}) {
    t.Fatal("Deleting copyied children failed: root has",len(tree.Root().Children),"children")
  }

  if tree.Root().Dir.Size != 20 {
    t.Fatal("Adding child didn't update root size correctly. Root size is ", tree.Root().Dir.Size)
  }

  tree.DelCopy(n2)

  if len(tree.Root().Children) != 0 {
    t.Fatal("Deleting copied children failed: root has",len(tree.Root().Children),"children")
  }

  if tree.Root().Dir.Size != 0 {
    t.Fatal("Adding child didn't update root size correctly. Root size is ", p.Dir.Size)
  }
}

func TestAddSubtreeCopy(t *testing.T) {
  p := &Node{}
  tree := New()
  tree.SetRootCopy(p, 0)

  n1 := &Node{Dir: Directory{Basename: "n1", Size: 10}}

  n2 := &Node{Dir: Directory{Basename: "n2", Size: 2}}

  n3 := &Node{Dir: Directory{Basename: "n3", Size: 2}}

  n1a := &Node{Dir: Directory{Basename: "n1a", Size: 10}}
  n1b := &Node{Dir: Directory{Basename: "n1b", Size: 10}}

  n1a1 := &Node{Dir: Directory{Basename: "n1a1", Size: 3}}
  n1b1 := &Node{Dir: Directory{Basename: "n1b1", Size: 4}}

  p.Add(n1)
  tree.AddCopy(n1, n1.Dir.Size)
  p.Add(n2)
  tree.AddCopy(n2, n2.Dir.Size)
  p.Add(n3)
  tree.AddCopy(n3, n3.Dir.Size)

  n1.Add(n1a)
  tree.AddCopy(n1a, n1a.Dir.Size)
  n1.Add(n1b)
  tree.AddCopy(n1b, n1b.Dir.Size)

  n1a.Add(n1a1)
  tree.AddCopy(n1a1, n1a1.Dir.Size)
  n1b.Add(n1b1)
  tree.AddCopy(n1b1, n1b1.Dir.Size)

  if p.Dir.Size != 41 {
    t.Fatal("Tree setup didn't calculate size correctly. Root size is ", p.Dir.Size)
  }

  if tree.Root().Dir.Size != 41 {
    t.Fatal("Adding child didn't update root size correctly. Root size is ", tree.Root().Dir.Size)
  }

  if len(tree.Root().Children) != 3 {
    t.Fatal("Copying children failed: root has",len(tree.Root().Children),"children")
  }

  if len(tree.origToCopy) != 8 {
    t.Fatal("Copying children failed: origToCopy map has wrong number of elements: ",len(tree.origToCopy))
  }

  if _, ok := tree.origToCopy[n1]; !ok {
    t.Fatal("Copying children failed: missing subtree node")
  }
  if _, ok := tree.origToCopy[n2]; !ok {
    t.Fatal("Copying children failed: missing subtree node")
  }
  if _, ok := tree.origToCopy[n3]; !ok {
    t.Fatal("Copying children failed: missing subtree node")
  }
  if _, ok := tree.origToCopy[n1a]; !ok {
    t.Fatal("Copying children failed: missing subtree node")
  }
  if _, ok := tree.origToCopy[n1b]; !ok {
    t.Fatal("Copying children failed: missing subtree node")
  }
  if _, ok := tree.origToCopy[n1a1]; !ok {
    t.Fatal("Copying children failed: missing subtree node")
  }
  if _, ok := tree.origToCopy[n1b1]; !ok {
    t.Fatal("Copying children failed: missing subtree node")
  }

}

func TestDelSubtreeCopy(t *testing.T) {
  p := &Node{}
  tree := New()
  tree.SetRootCopy(p, 0)

  n1 := &Node{Dir: Directory{Basename: "n1", Size: 10}}
  n2 := &Node{Dir: Directory{Basename: "n2"}}
  n3 := &Node{Dir: Directory{Basename: "n3"}}

  n1a := &Node{Dir: Directory{Basename: "n1a"}}
  n1b := &Node{Dir: Directory{Basename: "n1b"}}

  n1a1 := &Node{Dir: Directory{Basename: "n1a1"}}
  n1b1 := &Node{Dir: Directory{Basename: "n1b1"}}

  p.Add(n1)
  tree.AddCopy(n1, n1.Dir.Size)
  p.Add(n2)
  tree.AddCopy(n2, n2.Dir.Size)
  p.Add(n3)
  tree.AddCopy(n3, n3.Dir.Size)

  n1.Add(n1a)
  tree.AddCopy(n1a, n1a.Dir.Size)
  n1.Add(n1b)
  tree.AddCopy(n1b, n1b.Dir.Size)

  n1a.Add(n1a1)
  tree.AddCopy(n1a1, n1a1.Dir.Size)
  n1b.Add(n1b1)
  tree.AddCopy(n1b1, n1b1.Dir.Size)

  tree.DelCopy(n1)

  if len(tree.origToCopy) != 3 {
    t.Fatal("Deleting copied children failed: origToCopy map has wrong number of elements: ",len(tree.origToCopy))
  }

  if len(tree.Root().Children) != 2 {
    t.Fatal("Root node has wrong number of children: ",len(tree.Root().Children))
  }

  if ! hasBasenames(tree.Root().Children, []string{"n2","n3"}) {
    t.Fatal("Deleting copyied children failed: root has",len(tree.Root().Children),"children")
  }

  if tree.Root().Dir.Size != 0 {
    t.Fatal("Adding child didn't update root size correctly. Root size is ", tree.Root().Dir.Size)
  }

}

func TestUpdateCopy(t *testing.T) {
  p := &Node{}
  tree := New()
  tree.SetRootCopy(p, 0)

  n1 := &Node{Dir: Directory{Basename: "n1", Size: 10}}
  n2 := &Node{Dir: Directory{Basename: "n2"}}
  n3 := &Node{Dir: Directory{Basename: "n3"}}

  n1a := &Node{Dir: Directory{Basename: "n1a", Size: 20}}
  n1b := &Node{Dir: Directory{Basename: "n1b", Size: 20}}

  n1a1 := &Node{Dir: Directory{Basename: "n1a1", Size: 5}}
  n1b1 := &Node{Dir: Directory{Basename: "n1b1", Size: 5}}

  p.Add(n1)
  n1c,_ := tree.AddCopy(n1, n1.Dir.Size)
  p.Add(n2)
  tree.AddCopy(n2, n2.Dir.Size)
  p.Add(n3)
  tree.AddCopy(n3, n3.Dir.Size)

  n1.Add(n1a)
  tree.AddCopy(n1a, n1a.Dir.Size)
  n1.Add(n1b)
  tree.AddCopy(n1b, n1b.Dir.Size)

  n1a.Add(n1a1)
  tree.AddCopy(n1a1, n1a1.Dir.Size)
  n1b.Add(n1b1)
  tree.AddCopy(n1b1, n1b1.Dir.Size)

  if p.Dir.Size != 60 {
    t.Fatal("Adding node didn't update root size correctly. Root size:", p.Dir.Size)
  }
  if n1.Dir.Size != 60 {
    t.Fatal("Adding node didn't update size correctly. Size:", n1.Dir.Size)
  }
  if n1a.Dir.Size != 25 {
    t.Fatal("Adding node didn't update size correctly. Size:", n1a.Dir.Size)
  }
  if n1b.Dir.Size != 25 {
    t.Fatal("Adding node didn't update size correctly. Size:", n1b.Dir.Size)
  }

  n1a1.Dir.Size = 11
  tree.UpdateCopy(n1a1, n1a1.Dir.Size)

  n1ac := childWithBasename(n1c, "n1a")
  n1bc := childWithBasename(n1c, "n1b")
  n1a1c := childWithBasename(n1ac, "n1a1")
  n1b1c := childWithBasename(n1bc, "n1b1")

  if tree.Root().Dir.Size != 66 {
    t.Fatal("Adding node didn't update root size correctly. Root size:", tree.Root().Dir.Size)
  }
  if n1c.Dir.Size != 66 {
    t.Fatal("Adding node didn't update size correctly. Size:", n1c.Dir.Size)
  }
  if n1ac.Dir.Size != 31 {
    t.Fatal("Adding node didn't update size correctly. Size:", n1ac.Dir.Size)
  }
  if n1bc.Dir.Size != 25 {
    t.Fatal("Adding node didn't update size correctly. Size:", n1bc.Dir.Size)
  }
  if n1a1c.Dir.Size != 11 {
    t.Fatal("Adding node didn't update size correctly. Size:", n1a1c.Dir.Size)
  }
  if n1b1c.Dir.Size != 5 {
    t.Fatal("Adding node didn't update size correctly. Size:", n1b1c.Dir.Size)
  }
}

func TestWalk(t *testing.T) {
  p := &Node{Dir: Directory{Basename: "p"}}

  n1 := &Node{Dir: Directory{Basename: "n1", Size: 10}}
  n2 := &Node{Dir: Directory{Basename: "n2"}}
  n3 := &Node{Dir: Directory{Basename: "n3"}}

  n1a := &Node{Dir: Directory{Basename: "n1a", Size: 20}}
  n1b := &Node{Dir: Directory{Basename: "n1b", Size: 20}}

  n1a1 := &Node{Dir: Directory{Basename: "n1a1", Size: 5}}
  n1b1 := &Node{Dir: Directory{Basename: "n1b1", Size: 5}}

  p.Add(n1)
  p.Add(n2)
  p.Add(n3)

  n1.Add(n1a)
  n1.Add(n1b)

  n1a.Add(n1a1)
  n1b.Add(n1b1)

  expected := map[string]bool{
    "p":true,
    "n1":true,
    "n2":true,
    "n3":true,
    "n1a":true,
    "n1b":true,
    "n1a1":true,
    "n1b1":true,
  }

  visited := map[string]bool{}

  p.Walk(func(n *Node){
    visited[n.Dir.Basename] = true
  })

  for k,_ := range expected {
    if _,ok := visited[k]; !ok {
      t.Fatal("Node",k,"wasn't visited")
    }
  }

}
