package dirtree

import (
	"fmt"
	"testing"
)

func sameElems(a, b []*Node) bool {
	for _, i := range a {
		found := false
		for _, j := range b {
			if i == j {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func hasBasenames(a []*Node, names []string) bool {
	for _, j := range names {
		found := false
		for _, i := range a {
			if i.Dir.Basename == j {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func dumpOrigMap(t *Dirtree) {
	for k, v := range t.origToCopy {
		fmt.Printf("%p --> %p\n", k, v)
	}
}

func childWithBasename(n *Node, name string) *Node {
	for _, v := range n.Children {
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

	if !sameElems(p.Children, []*Node{n1, n2}) {
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

	if !sameElems(p.Children, []*Node{n2}) {
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

	if !sameElems(p.Children, []*Node{n1}) {
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

	if !sameElems(p.Children, []*Node{n1, n3}) {
		t.Fatal("Deleting child from parent with two children failed. Wrong child deleted.")
	}

	if p.Dir.Size != 40 {
		t.Fatal("Deleting child didn't update root size correctly. Root size is ", p.Dir.Size)
	}
}
