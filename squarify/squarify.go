// Package squarify implements the Squarified Treemap algorithm of Bruls, Huizing, and Van Wijk:
//
//    http://www.win.tue.nl/~vanwijk/stm.pdf
//
// The basic idea is to generate a tiling of items of various sizes, each of which may have children which 
// are tiled nested inside their parent.
//
// Tiling is performed by calling the Squarify function.
package squarify

import (
  "sort"
)

type TreeSizer interface {
  Size() float64
  NumChildren() int
  Child(i int) TreeSizer
}

type Rect struct {
  X, Y, W, H float64
}

type Block struct {
  Rect
  TreeSizer TreeSizer
}

type area struct {
  Area float64
  TreeSizer TreeSizer
}

type direction int

const (
  Vertical direction = iota
  Horizontal
)

const (
  DoSort = true
  DontSort = false
)

type Margins struct {
  L,R,T,B float64
}

type Meta struct {
  Depth int
}

type Options struct {
  MaxDepth int
  Margins *Margins
  Sort bool
  MinW, MinH float64
}

// Squarify lays out the children of `root` inside the area represented by rect. 
func Squarify(root TreeSizer, rect Rect, options Options) (blocks []Block, meta []Meta) {
  if options.MaxDepth <= 0 {
    options.MaxDepth = 20
  }

  return squarify(root, Block{Rect: rect}, options, 0)
}
/*
func Squarify(root TreeSizer, rect Rect, maxDepth int, margins *Margins, sort bool) (blocks []Block, meta []Meta) {
  return squarify(root, Block{Rect: rect}, maxDepth, margins, sort, 0)
}
*/

type row struct {
  areas []*area
  X,Y float64
  min, max float64 // Min and max areas in the row
  sum float64 // Sum of areas
  Width float64
  Dir direction
}

func newRow(dir direction, width,x,y float64) *row {
  return &row{
    areas: make([]*area,0),
    Width: width,
    X: x,
    Y: y,
    Dir: dir,
  }
}

func (r *row) push(a *area) {
  if a.Area <= 0 {
    // We use 0 area as a sentinel in min and max.
    panic("Area must be >= 0")
  }

  r.areas = append(r.areas, a)
  r.updateCached(a)
}

func (r *row) pop() *area {
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

func (r *row) pushTemporarily(a *area, f func()) {
  min := r.min
  max := r.max
  sum := r.sum
  r.push(a)
  f()
  r.pop()
  r.min = min
  r.max = max
  r.sum = sum
}

func (r *row) calcCached() {
  r.min = 0
  r.max = 0
  r.sum = 0
  for _,a := range r.areas {
    r.updateCached(a)
  }
}

// Number of elements
func (r row) size() int {
  return len(r.areas)
}

func (r *row) updateCached(a *area) {
  if r.min <= 0 || a.Area < r.min {
    r.min = a.Area
  }
  if r.max <= 0 || a.Area > r.max {
    r.max = a.Area
  }
  r.sum += a.Area
}


// Calculate the worst aspect ratio of all rectangles in the row
func (r *row) worst() float64 {
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

func (r *row) makeBlocks() (height float64, blocks []Block) {
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

    blocks = append(blocks, Block{Rect: Rect{X: x, Y: y, W: itemWidth, H: itemHeight}, TreeSizer: a.TreeSizer})

    if r.Dir == Vertical {
      y += itemHeight
    } else {
      x += itemWidth
    }
  }

  return
}

func squarify(root TreeSizer, block Block, options Options, depth int) (blocks []Block, meta []Meta) {
  blocks = make([]Block, 0)
  meta = make([]Meta, 0)

  if block.W <= options.MinW || block.H <= options.MinH || depth >= options.MaxDepth {
    return
  }

  output := func(newBlocks []Block) {
    for i := 0; i < len(newBlocks); i++ {
      // Filter out any blocks that are just placeholders for extra space
      if newBlocks[i].TreeSizer != nil {
        // Filter out any blocks that are too small
        if newBlocks[i].W > options.MinW || newBlocks[i].H > options.MinH {
          blocks = append(blocks, newBlocks[i])
          meta = append(meta, Meta{Depth: depth})
        }
      }
    }
/*
    blocks = append(blocks, newBlocks...)
    for i := 0; i < len(newBlocks); i++ {
      meta = append(meta, Meta{Depth: depth})
    }
*/
  }

  areas := areas(root, block, options.Sort)

  rowX := block.X
  rowY := block.Y
  freeWidth := block.W
  freeHeight := block.H

  makeRow := func() (row *row) {
    if block.W > block.H {
      row = newRow(Vertical, freeHeight, rowX, rowY)
    } else {
      row = newRow(Horizontal, freeWidth, rowX, rowY)
    }
    return row
  }

  // Decide which direction to create the new row
  row := makeRow()

  for _, a := range areas {
    if row.size() > 0 {
      worstBefore := row.worst()
      worstAfter := float64(0)
      row.pushTemporarily(&a, func() {
        worstAfter = row.worst()
      })

      if worstBefore < worstAfter {
        // It's better to make a new row now.
        // Output the current blocks and make a new row
        offset, newBlocks := row.makeBlocks()
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

    cp := &area{}
    *cp = a
    row.push(cp)
  }

  if row.size() > 0 {
    _, newBlocks := row.makeBlocks()
    output(newBlocks)
  }

  // Now, for each of the items we just processed, if they have children then
  // lay them out inside their parent box. The available area may be reduced by
  // certain size.
  for _, block := range(blocks) {
    if block.TreeSizer != nil {
      if options.Margins != nil {
        block.X += options.Margins.L
        block.Y += options.Margins.T
        block.W -= options.Margins.L + options.Margins.R
        block.H -= options.Margins.T + options.Margins.B
      }

      newBlocks, newMeta := squarify(block.TreeSizer, block, options, depth+1)
      blocks = append(blocks, newBlocks...)
      meta = append(meta, newMeta...)
    }
  }

  return
}

// Sort areas by area.
type byAreaAndPlaceholder []area

func (a byAreaAndPlaceholder) Len() int {
  return len(a)
}

func (a byAreaAndPlaceholder) Less(i, j int) bool {
  //return a[i].Area > a[j].Area

  if a[i].TreeSizer != nil && a[j].TreeSizer != nil || a[i].TreeSizer == nil && a[j].TreeSizer == nil {
    return a[i].Area > a[j].Area
  } else {
    return a[i].TreeSizer != nil
  }
}

func (a byAreaAndPlaceholder) Swap(i, j int) {
  a[i], a[j] = a[j], a[i]
}

func areas(root TreeSizer, block Block, dosort bool) (areas []area) {
  blockArea := block.W * block.H

  areas = make([]area,0)
  itemsTotalSize := float64(0)

  for i := 0; i < root.NumChildren(); i++ {
    item := root.Child(i)

    // Ignore 0-size items
    if item.Size() <= 0 {
      continue
    }

    areas = append(areas, area{Area: item.Size()/root.Size()*blockArea, TreeSizer: item})
    itemsTotalSize += item.Size()
  }

  // Add a placeholder area for extra space
  if itemsTotalSize < root.Size() {
    a := (root.Size()-itemsTotalSize)/root.Size()*blockArea
    areas = append(areas, area{Area: a, TreeSizer: nil})
  }

  if dosort {
    sort.Sort(byAreaAndPlaceholder(areas))
  }

  return
}

