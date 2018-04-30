/*
  Package ui is used to render squarified directory information into a pixmap.
*/
package ui

import (
	sh "github.com/jeffwilliams/spacehoarder"
	"github.com/jeffwilliams/spacehoarder/dirtree"
	"github.com/jeffwilliams/squarify"
	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
	"github.com/mattn/go-gtk/pango"
	"math"
	"strconv"
)

var (
	// The GC used when drawing.
	gc *gdk.GC
	// The pango context used to create text
	pc *pango.Context
)

func InitGC(template *gtk.Widget) {
	gc = gdk.NewGC(template.GetWindow().GetDrawable())
	pc = template.GetPangoContext()
}

type Style struct {
	BorderColor, BgColor *gdk.Color
	ColorByDepth         []*gdk.Color
	TitleLayout          *pango.Layout
}

// NewStyle returns a new Style with the specified colors, with the colors re-allocated
// from this package's GC so that they are usable when drawing blocks.
func NewStyle(border, bg *gdk.Color, colorByDepth []*gdk.Color, fontSize int) *Style {
	if gc == nil {
		panic("InitGC must be called before calling NewStyle")
	}

	alloc := func(c *gdk.Color) *gdk.Color {
		return gc.GetColormap().AllocColorRGB(c.Red(), c.Green(), c.Blue())
	}

	for i, v := range colorByDepth {
		colorByDepth[i] = alloc(v)
	}

	fontDesc := pango.NewFontDescription()
	fontDesc.SetSize(fontSize * pango.SCALE)

	layout := pango.NewLayout(pc)
	layout.SetFontDescription(fontDesc)

	return &Style{
		BgColor:      alloc(bg),
		BorderColor:  alloc(border),
		ColorByDepth: colorByDepth,
		TitleLayout:  layout,
	}
}

// Render creates a new GDK Pixbuf which is a rendering of the blocks. The template drawable
// is used to determine default values for the new pixmap. This would usually be the window we
// will end up drawing to, but this is threadsafe since we are only reading.
func Render(template *gdk.Drawable, width, height int, blocks []squarify.Block, meta []squarify.Meta, style *Style) *gdk.Pixmap {

	round := func(block squarify.Block, i int) (X, Y, W, H int) {
		// Convert floating point coords to int.
		X = int(math.Floor(block.X))
		Y = int(math.Floor(block.Y))
		W = int(math.Floor(block.W))
		H = int(math.Floor(block.H))
		return
	}

	pixmap := gdk.NewPixmap(template, width, height, -1)

	// Fill whole space with bg color
	gc.SetClipRectangle(0, 0, width, height)
	gc.SetForeground(style.BgColor)
	gc.SetBackground(style.BgColor)
	pixmap.GetDrawable().DrawRectangle(gc, true, 0, 0, width, height)

	filesNum := 0
	// Draw all blocks
	for i, block := range blocks {
		// Don't draw the placeholder (non-directory) blocks.
		/*
		   if block.TreeSizer == nil {
		     continue
		   }*/

		meta := meta[i]
		color := style.ColorByDepth[meta.Depth%len(style.ColorByDepth)]

		x, y, w, h := round(block, i)

		gc.SetClipRectangle(x, y, w+1, h+1)

		// Fill block
		gc.SetForeground(color)
		pixmap.GetDrawable().DrawRectangle(gc, true, x, y, w, h)

		// Draw border
		gc.SetForeground(style.BorderColor)
		pixmap.GetDrawable().DrawRectangle(gc, false, x, y, w, h)

		// Draw title
		if block.TreeSizer != nil {
			dir := &block.TreeSizer.(*dirtree.Node).Dir
			style.TitleLayout.SetText(dir.Basename + " (" + sh.FancySize(dir.Size) + ")")
			style.TitleLayout.SetWidth(w * pango.SCALE)
			pixmap.GetDrawable().DrawLayout(gc, x+1, y+1, style.TitleLayout)
		} else {
			style.TitleLayout.SetText("<files " + strconv.Itoa(filesNum) + ">")
			style.TitleLayout.SetWidth(w * pango.SCALE)
			pixmap.GetDrawable().DrawLayout(gc, x+1, y+1, style.TitleLayout)
			filesNum += 1
		}
	}

	return pixmap
}
