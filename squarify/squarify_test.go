package squarify

import (
  "testing"
  "fmt"
)

func TestRowPushTemporarily(t *testing.T) {
  r := NewRow(Vertical, 20, 30, 30)

  r.Push(&Area{40, nil})

  if len(r.areas) != 1 {
    t.Fatal("Push failed")
  }

  r.PushTemporarily(&Area{50,nil}, func(){
    if len(r.areas) != 2 {
      t.Fatal("Temporary Push failed")
    }

    if r.areas[0].Area != 40 {
      t.Fatal("Temporary Push was wrong")
    }

    if r.areas[1].Area != 50 {
      t.Fatal("Temporary Push was wrong")
    }
  })

  if len(r.areas) != 1 {
    t.Fatal("Temporary Push didn't clean up")
  }

  if r.areas[0].Area != 40 {
    t.Fatal("Temporary Push was wrong")
  }

  if r.min != 40 {
    t.Fatal("Temporary Push was wrong")
  }

  if r.max != 40 {
    t.Fatal("Temporary Push was wrong")
  }

  if r.sum != 40 {
    t.Fatal("Temporary Push was wrong")
  }
}

type TestNode struct {
  name string
  children []*TestNode
  size float64
}

func (t TestNode) Size() float64 {
  return t.size
}

func (t TestNode) NumChildren() int {
  return len(t.children)
}

func (t TestNode) Child(i int) TreeSizer {
  return t.children[i]
}
/*
type TreeSizer interface {
  Size() float64
  NumChildren() int
  Child(i int) TreeSizer
}
*/

func TestSquarifyAreas(t *testing.T) {
  // func squarify(root TreeSizer, block Block, maxDepth int, margins *Margins, sort bool, depth int) (blocks []Block, meta []Meta) {

  // root -> size 30 + 20 local files
  //   b  -> size 10
  //   c  -> size 20

  nodes := map[string]*TestNode{}
  addToMap := func(t *TestNode) *TestNode {
    nodes[t.name] = t
    return t
  }

  b := addToMap(&TestNode{name: "b", size: 10})
  c := addToMap(&TestNode{name: "c", size: 20})
  root := TestNode{name:"root", children: []*TestNode{b,c}, size: 80 }

  canvas := Block{X:0,Y:0,W:100,H:100}

  validateSize := func(tn TestNode, b Block) {
    expectedArea := tn.size / root.size
    actualArea := (b.W * b.H) / (canvas.W * canvas.H)

    if expectedArea != actualArea {
      t.Fatal("Bad area for",tn.name,": expected", expectedArea, "got", actualArea)
    }
  }

  blocks, _ := Squarify(root, canvas, 20, nil, DoSort)

  for _, blk := range blocks {
    if blk.Data != nil {
      fmt.Println(blk.Data.(*TestNode).name)
      fmt.Println(blk)

      n, ok := nodes[blk.Data.(*TestNode).name]
      if ! ok {
        t.Fatal("Squarify produced a block with no matching TestNode")
      }

      validateSize(*n, blk)
    }
  }

}
