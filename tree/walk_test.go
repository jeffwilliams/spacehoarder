package tree

import (
	"testing"
)

type Node struct {
	name     string
	parent   *Node
	children []*Node
}

func (n Node) GetParent() Tree {
	if n.parent == nil {
		return nil
	}
	return n.parent
}

func (n Node) GetChild(i int) Tree {
	return n.children[i]
}

func (n Node) NumChildren() int {
	return len(n.children)
}

func (n *Node) String() string {
	return n.name
}

/*

  a
		b
			d
			e
				f
					m
		c
			g
				i
				j
			h
				k
					n
				l
					o

*/

type TestData struct {
	root  *Node
	nodes map[string]*Node
	depth map[string]int
}

func makeTestData() (d *TestData) {
	d = &TestData{nodes: map[string]*Node{}, depth: map[string]int{}}

	mkNode := func(nm string, parent string) *Node {
		n := &Node{name: nm}
		d.nodes[nm] = n

		if parent != "" {
			n.parent = d.nodes[parent]
			d.nodes[parent].children = append(d.nodes[parent].children, n)
		}

		return n
	}

	d.root = mkNode("a", "")
	mkNode("b", "a")
	mkNode("c", "a")
	mkNode("d", "b")
	mkNode("e", "b")
	mkNode("f", "e")
	mkNode("m", "f")
	mkNode("g", "c")
	mkNode("h", "c")
	mkNode("i", "g")
	mkNode("j", "g")
	mkNode("k", "h")
	mkNode("l", "h")
	mkNode("n", "k")
	mkNode("o", "l")

	d.depth = map[string]int{"a": 0, "b": 1, "d": 2, "e": 2, "f": 3, "m": 4, "c": 1, "g": 2, "i": 3, "j": 3, "h": 2, "k": 3, "n": 4, "l": 3, "o": 4}

	return d
}

func makeSimpleVisitor(t *testing.T, data *TestData, expectedOrder []string, ndx *int) Visitor {
	return func(tree Tree, depth int) (continu, skipChildren bool) {
		n := tree.(*Node)

		if *ndx >= len(expectedOrder) {
			t.Fatalf("Visitor was called for node %s after the end of the expected nodes (called too many times)", n.name)
		}

		if expectedOrder[*ndx] != n.name {
			t.Fatalf("Expected %s but got node %s", expectedOrder[*ndx], n.name)
		}
		if data.depth[n.name] != depth {
			t.Fatalf("Expected depth %d but got depth %d at node %s", data.depth[n.name], depth, n.name)
		}
		(*ndx)++
		return true, false
	}

}

func testWalk(t *testing.T, expectedOrder []string, treeNode string, dir WalkDirection, order WalkOrder, depth int, skip bool) {
	data := makeTestData()

	ndx := 0

	tree := data.nodes[treeNode]

	visitor := makeSimpleVisitor(t, data, expectedOrder, &ndx)

	Walk(tree, visitor, dir, order, depth, skip)

	if ndx < len(expectedOrder) {
		t.Fatalf("Not enough nodes visited. Walk stopped at %s", expectedOrder[ndx-1])
	}

}

func TestWalk(t *testing.T) {

	tests := []struct {
		name          string
		expectedOrder []string
		tree          string
		dir           WalkDirection
		order         WalkOrder
		depth         int
		skip          bool
	}{
		{
			"PreOrderForwardWalkFromRoot",
			[]string{"a", "b", "d", "e", "f", "m", "c", "g", "i", "j", "h", "k", "n", "l", "o"},
			"a",
			Forward, PreOrder, 0, false,
		},
		{
			"PostOrderForwardWalkFromRoot",
			[]string{"d", "m", "f", "e", "b", "i", "j", "g", "n", "k", "o", "l", "h", "c", "a"},
			"a",
			Forward, PostOrder, 0, false,
		},
		{
			"PreOrderReverseWalkFromRoot",
			[]string{"a", "c", "h", "l", "o", "k", "n", "g", "j", "i", "b", "e", "f", "m", "d"},
			"a",
			Reverse, PreOrder, 0, false,
		},
		{
			"PostOrderReverseWalkFromRoot",
			[]string{"o", "l", "n", "k", "h", "j", "i", "g", "c", "m", "f", "e", "d", "b", "a"},
			"a",
			Reverse, PostOrder, 0, false,
		},
		{
			"PreOrderForwardWalkFromDepth1Node",
			[]string{"b", "d", "e", "f", "m", "c", "g", "i", "j", "h", "k", "n", "l", "o"},
			"b",
			Forward, PreOrder, 1, false,
		},
		{
			"PreOrderForwardWalkFromLeafNode",
			[]string{"m", "c", "g", "i", "j", "h", "k", "n", "l", "o"},
			"m",
			Forward, PreOrder, 4, false,
		},
		{
			"PreOrderForwardWalkFromDepth1NodeSkip",
			[]string{"c", "g", "i", "j", "h", "k", "n", "l", "o"},
			"b",
			Forward, PreOrder, 1, true,
		},
		{
			"PostOrderForwardWalkFromDepth1Node",
			[]string{"d", "m", "f", "e", "b", "i", "j", "g", "n", "k", "o", "l", "h", "c", "a"},
			"b",
			Forward, PostOrder, 1, false,
		},
		{
			"PreOrderReverseWalkFromDepth1Node",
			[]string{"b", "e", "f", "m", "d"},
			"b",
			Reverse, PreOrder, 1, false,
		},
		{
			"PreOrderReverseWalkFromLeafNode",
			[]string{"m", "d"},
			"m",
			Reverse, PreOrder, 4, false,
		},
		{
			"PostOrderReverseWalkFromDepth1Node",
			[]string{"m", "f", "e", "d", "b", "a"},
			"b",
			Reverse, PostOrder, 1, false,
		},
		{
			"PostOrderReverseWalkFromDepth1NodeSkip",
			[]string{"m", "f", "e", "d", "b", "a"},
			"c",
			Reverse, PostOrder, 1, true,
		},
		{
			"PostOrderReverseWalkFromDepth2NodeSkip",
			[]string{"c", "m", "f", "e", "d", "b", "a"},
			"g",
			Reverse, PostOrder, 2, true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testWalk(t, tc.expectedOrder, tc.tree, tc.dir, tc.order, tc.depth, tc.skip)
		})
	}

}
