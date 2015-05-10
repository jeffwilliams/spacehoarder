package squarify

import (
  "sort"
)

type TreeSizer interface {
  Size() float64
  NumChildren() int
  Child(i int) TreeSizer
}

type Block struct {
  X, Y, W, H float64
  Data TreeSizer
}

type Area struct {
  Area float64
  Data TreeSizer
}

type Direction int

const (
  Vertical Direction = iota
  Horizontal
)

type Row struct {
  areas []*Area
  X,Y float64
  min, max float64 // Min and max areas in the row
  sum float64 // Sum of areas
  Width float64
  Dir Direction
}

func NewRow(dir Direction, width,x,y float64) *Row {
  return &Row{
    areas: make([]*Area,0),
    Width: width,
    X: x,
    Y: y,
    Dir: dir,
  }
}

func (r *Row) Push(a *Area) {
  if a.Area <= 0 {
    // We use 0 area as a sentinel in min and max.
    panic("Area must be >= 0")
  }

  r.areas = append(r.areas, a)
  r.updateCached(a)
}

func (r *Row) Pop() *Area {
  r.min = 0
  r.max = 0
  r.sum = 0

  if len(r.areas) > 0 {
    last := len(r.areas)-1
    a := r.areas[last]
    r.areas[last] = nil
    r.areas = r.areas[0:last]
    return a
  } else {
    return nil
  }
}

func (r *Row) PushTemporarily(a *Area, f func()) {
  min := r.min
  max := r.max
  sum := r.sum
  r.Push(a)
  f()
  r.Pop()
  r.min = min
  r.max = max
  r.sum = sum
}

func (r *Row) calcCached() {
  r.min = 0
  r.max = 0
  r.sum = 0
  for _,a := range r.areas {
    r.updateCached(a)
  }
}

// Number of elements
func (r Row) Size() int {
  return len(r.areas)
}

func (r *Row) updateCached(a *Area) {
  if r.min <= 0 || a.Area < r.min {
    r.min = a.Area
  }
  if r.max <= 0 || a.Area > r.max {
    r.max = a.Area
  }
  r.sum += a.Area
}


// Calculate the worst aspect ratio of all rectangles in the row
func (r *Row) Worst() float64 {
  if r.min == 0 {
    // We need to calculate min, max, and sum
    r.calcCached()
  }

  w2 := r.Width*r.Width
  sum2 := r.sum*r.sum
  worst1 := w2*r.max / sum2
  worst2 := sum2 / (r.min * w2)
  if worst1 > worst2 {
    return worst1
  } else {
    return worst2
  }
}

func (r *Row) MakeBlocks() (height float64, blocks []Block) {
  if r.min == 0 {
    // We need to calculate min, max, and sum
    r.calcCached()
  }

  blocks = make([]Block,0)
  x := r.X
  y := r.Y

  for _, a := range r.areas {
    // Item width relative to the row
    relativeWidth := a.Area/r.sum
    itemWidth := relativeWidth * r.Width
    itemHeight := a.Area/itemWidth

    if height == 0 {
      height = itemHeight
    } else if itemHeight != height {
      itemHeight = height
    }

    if r.Dir == Vertical {
      // swap
      itemWidth, itemHeight = itemHeight, itemWidth
    }

    blocks = append(blocks, Block{X: x, Y: y, W: itemWidth, H: itemHeight, Data: a.Data})

    if r.Dir == Vertical {
      y += itemHeight
    } else {
      x += itemWidth
    }
  }

  return
}

type Margins struct {
  L,R,T,B float64
}

const (
  DoSort = true
  DontSort = false
)

type Meta struct {
  Depth int
}

// Squarify lays out the children of `root` inside the area represented by block. 
func Squarify(root TreeSizer, block Block, maxDepth int, margins *Margins, sort bool) (blocks []Block, meta []Meta) {
  return squarify(root, block, maxDepth, margins, sort, 0)
}


func squarify(root TreeSizer, block Block, maxDepth int, margins *Margins, sort bool, depth int) (blocks []Block, meta []Meta) {
  blocks = make([]Block, 0)
  meta = make([]Meta, 0)
  if block.W <= 0 || block.H <= 0 || maxDepth == 0 {
    return
  }

  output := func(newBlocks []Block) {
    blocks = append(blocks, newBlocks...)
    for i := 0; i < len(newBlocks); i++ {
      meta = append(meta, Meta{Depth: depth})
    }
  }

  areas := areas(root, block, sort)

  rowX := block.X
  rowY := block.Y
  freeWidth := block.W
  freeHeight := block.H

  makeRow := func() (row *Row) {
    if block.W > block.H {
      row = NewRow(Vertical, freeHeight, rowX, rowY)
    } else {
      row = NewRow(Horizontal, freeWidth, rowX, rowY)
    }
    return row
  }

  // Decide which direction to create the new row
  row := makeRow()

  for _, area := range areas {
    if row.Size() > 0 {
      worstBefore := row.Worst()
      worstAfter := float64(0)
      row.PushTemporarily(&area, func() {
        worstAfter = row.Worst()
      })

      if worstBefore < worstAfter {
        // It's better to make a new row now.
        // Output the current blocks and make a new row
        offset, newBlocks := row.MakeBlocks()
        //blocks = append(blocks, newBlocks...)
        output(newBlocks)

        if row.Dir == Vertical {
          rowX += offset
          freeWidth -= offset
        } else {
          rowY += offset
          freeHeight -= offset
        }

        row = makeRow()
      }
    }

    cp := &Area{}
    *cp = area
    row.Push(cp)
  }

  if row.Size() > 0 {
    _, newBlocks := row.MakeBlocks()
    output(newBlocks)
  }

  // Now, for each of the items we just processed, if they have children then
  // lay them out inside their parent box. The available area may be reduced by
  // certain size.
  for _, block := range(blocks) {
    if block.Data != nil {
      if margins != nil {
        block.X += margins.L
        block.Y += margins.T
        block.W -= margins.L + margins.R
        block.H -= margins.T + margins.B
      }

      newBlocks, newMeta := squarify(block.Data, block, maxDepth-1, margins, sort, depth+1)
      blocks = append(blocks, newBlocks...)
      meta = append(meta, newMeta...)
    }
  }

  return
}

// Sort areas by area.
type byArea []Area

func (a byArea) Len() int {
  return len(a)
}

func (a byArea) Less(i, j int) bool {
  //if a[i].Data == a[j].Data {
    return a[i].Area > a[j].Area
  /*} else {
    if a[i].Data != nil {
      return true
    } else {
      return false
    }
  }*/
}

func (a byArea) Swap(i, j int) {
  a[i], a[j] = a[j], a[i]
}

func areas(root TreeSizer, block Block, dosort bool) (areas []Area) {
  blockArea := block.W * block.H

  areas = make([]Area,0)
  itemsTotalSize := float64(0)

  for i := 0; i < root.NumChildren(); i++ {
    item := root.Child(i)

    // Ignore 0-size items
    if item.Size() <= 0 {
      continue
    }

    areas = append(areas, Area{Area: item.Size()/root.Size()*blockArea, Data: item})
    itemsTotalSize += item.Size()
  }

  // Add a placeholder area for extra space
  if itemsTotalSize < root.Size() {
    area := (root.Size()-itemsTotalSize)/root.Size()*blockArea
    areas = append(areas, Area{Area: area, Data: nil})
  }

  if dosort {
    sort.Sort(byArea(areas))
  }

  return
}

