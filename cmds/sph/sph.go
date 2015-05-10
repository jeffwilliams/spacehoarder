package main

import (
  "github.com/mattn/go-gtk/glib"
  "github.com/mattn/go-gtk/gtk"
  "github.com/mattn/go-gtk/gdk"
  "github.com/jeffwilliams/spacehoarder/dirtree"
  "github.com/jeffwilliams/spacehoarder/squarify"
  "github.com/jeffwilliams/spacehoarder/ui"
  "fmt"
  "flag"
  "os"
  "io"
  "strconv"
)

func makeUi() (*gtk.Window, *gtk.DrawingArea) {
  gtk.Init(nil)

  window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
  window.SetTitle("Spacehoarder")
  window.Connect("destroy", func(ctx *glib.CallbackContext) {
        println("got destroy!", ctx.Data().(string))
        gtk.MainQuit()
    }, "foo")

  // Drawing Area
  area := gtk.NewDrawingArea()
  area.SetSizeRequest(640,480)

  // Label
  progressLabel := gtk.NewLabel("Calculating...")

  // Main UI Layout
  vbox := gtk.NewVBox(false, 0)
  vbox.PackStart(area, true, true, 0)
  vbox.PackStart(progressLabel, false, false, 0)

  window.Add(vbox)
  //window.Add(button)
  //window.Add(fontbutton)
  window.ShowAll()

  return window, area
}

// Protyping function. Just print out the directories processed
// TODO: Remove.
func doProg() {
  ops, prog := dirtree.Build(".")
  loop: for {
    select {
      case _,_ = <-ops:

      case p, ok := <-prog:
        if ! ok {
          break loop
        }
        fmt.Println(p)
    }
  }
}

// Protyping function. Calculate dirtree once and display.
// TODO: Remove.
func easyMode() {
  tree := dirtree.BuildSync(".")

  tree.Root().Walk(func(n *dirtree.Node) {
    fmt.Println(n.Dir.Path)
  })

  margins := &squarify.Margins{3, 3, 20, 3}

  blocks, meta := squarify.Squarify(tree.Root(), squarify.Block{X: 0, Y: 0, W: 640, H: 480}, 2, margins, squarify.DoSort)

  for i, b := range blocks {
    fmt.Println(b)
    fmt.Println("  meta: ",meta[i])
  }

  // Output an SVG
  fmt.Printf("\n\n<svg width=\"%d\" height=\"%d\">\n", 100, 100)
  for _, b := range blocks {
    fmt.Printf("  <rect x=\"%f\" y=\"%f\" width=\"%f\" height=\"%f\" style=\"fill:rgb(0,0,255);stroke-width:1;stroke:rgb(0,0,0)\"/>\n", b.X, b.Y, b.W, b.H)
  }
  fmt.Printf("</svg>\n")

  _, area := makeUi()

  ui.InitGC(&area.Widget)

  style := ui.NewStyle(
    gdk.NewColor("#000000"),
    gdk.NewColor("#BBBBBB"),
    []*gdk.Color{
      gdk.NewColor("#75A3D1"),
      gdk.NewColor("#C28547"),
      gdk.NewColor("#669933"),
      gdk.NewColor("#996633"),
      gdk.NewColor("#4785C2"),
      gdk.NewColor("#993333"),
    },
    8,
  )

  pixmap := ui.Render(area.GetWindow().GetDrawable(), area.GetAllocation().Width, area.GetAllocation().Height, blocks, meta, style)

  area.Connect("expose_event", func(ctx *glib.CallbackContext) {
    println("got expose event")
    // This should be wrapped in beginPaint and EndPaint, but those are not exposed in Golang
    gc := gdk.NewGC(pixmap.GetDrawable())
    //style.BgColor = gc.GetColormap().AllocColorRGB(style.BgColor.Red(), style.BgColor.Green(), style.BgColor.Blue())
    gc.SetForeground(style.BgColor)
    area.GetWindow().GetDrawable().DrawDrawable(gc, pixmap.GetDrawable(), 0, 0, 0, 0, -1, -1)
  })

  // Test if Emit works
  area.Widget.Emit("expose_event")

  gtk.Main()
}

var filesNum int
func writeSvg(w io.Writer, blocks []squarify.Block) {
  fmt.Fprintf(w, "\n\n<svg width=\"%d\" height=\"%d\">\n", 100, 100)
  for _, b := range blocks {
    fmt.Fprintf(w, "  <rect x=\"%f\" y=\"%f\" width=\"%f\" height=\"%f\" style=\"fill:rgb(0,0,255);stroke-width:1;stroke:rgb(0,0,0)\"/>\n", b.X, b.Y, b.W, b.H)
    name := "[files "+strconv.Itoa(filesNum)+"]"
    if b.Data != nil {
      name = b.Data.(*dirtree.Node).Dir.Basename
    } else {
      filesNum += 1
    }
    fmt.Fprintf(w, "  <text x=\"%f\" y=\"%f\" style=\"font-size:8\">%s</text>\n", b.X, b.Y, name)
    fmt.Fprintln(w, "")

  }
  fmt.Fprintf(w, "</svg>\n")
}

type RendererContext struct {
  maxDepth int
  margins squarify.Margins
  ops chan dirtree.OpData
  prog chan string
  resize chan struct{}
  setPixmap func(p *gdk.Pixmap)
  style *ui.Style
  area *gtk.DrawingArea
}

// PixmapRenderer builds a local tree from the operations passed in ops, 
// and repeatedly renders it into a pixmap that is passed to setPixmap
func PixmapRenderer(ctx *RendererContext) {
  tree := dirtree.New()

  render := func(){
    gdk.ThreadsEnter()
    areaW := ctx.area.GetAllocation().Width
    areaH := ctx.area.GetAllocation().Height
    gdk.ThreadsLeave()
    blocks, meta := squarify.Squarify(tree.Root(), squarify.Block{X: 0, Y: 0, W: float64(areaW), H: float64(areaH)}, ctx.maxDepth, &ctx.margins, squarify.DoSort)
    gdk.ThreadsEnter()
    pixmap := ui.Render(ctx.area.GetWindow().GetDrawable(), areaW, areaH, blocks, meta, ctx.style)
    ctx.setPixmap(pixmap)
    gdk.ThreadsLeave()
  }

  loop: for {
    select {
      case _ = <-ctx.resize:
        render()

      case op, ok := <-ctx.ops:
        if !ok {
          // We're done!
          ctx.ops = nil

    gdk.ThreadsEnter()
    areaW := ctx.area.GetAllocation().Width
    areaH := ctx.area.GetAllocation().Height
    gdk.ThreadsLeave()
blocks, _ := squarify.Squarify(tree.Root(), squarify.Block{X: 0, Y: 0, W: float64(areaW), H: float64(areaH)}, ctx.maxDepth, &ctx.margins, squarify.DoSort)
// Output an SVG
f, err := os.Create("/home/shared/Jeff/sph_test.svg")
if err == nil {
  writeSvg(f, blocks)
  f.Close()
  fmt.Println("Output SVG to /home/shared/Jeff/sph_test.svg")
} else {
  fmt.Println("Error opening file:", err)
}
/*
fmt.Printf("\n\n<svg width=\"%d\" height=\"%d\">\n", 100, 100)
for _, b := range blocks {
  fmt.Printf("  <rect x=\"%f\" y=\"%f\" width=\"%f\" height=\"%f\" style=\"fill:rgb(0,0,255);stroke-width:1;stroke:rgb(0,0,0)\"/>\n", b.X, b.Y, b.W, b.H)
  fmt.Printf("  <text x=\"%f\" y=\"%f\" font-size=\"8\">%s</text>\n", b.X, b.Y, b.Data.)
  fmt.Println("")

}
fmt.Printf("</svg>\n")
*/
          continue loop
        }

        dirtree.Apply(tree, op)
        render()

      // TODO: Move this to a different goroutine
      case p, ok := <-ctx.prog:
        if ! ok {
          ctx.prog = nil
          continue loop
        }
        fmt.Println(p)
    }
  }

  fmt.Println("Pixmap renderer done")
}

func main() {
  //easyMode()

  flag.Parse()

  if flag.NArg() < 1 {
    fmt.Println("Usage: sph <directory>")
    os.Exit(1)
  }

  gdk.ThreadsInit()

  // The pointer to the pixmap that the UI thread will draw on expose events.
  var pixmap *gdk.Pixmap

  // Start goroutine that explores the directories
  ops, prog := dirtree.Build(flag.Arg(0))

  _, area := makeUi()

  ui.InitGC(&area.Widget)

  style := ui.NewStyle(
    gdk.NewColor("#000000"),
    gdk.NewColor("#BBBBBB"),
    []*gdk.Color{
      gdk.NewColor("#75A3D1"),
      gdk.NewColor("#C28547"),
      gdk.NewColor("#669933"),
      gdk.NewColor("#996633"),
      gdk.NewColor("#4785C2"),
      gdk.NewColor("#993333"),
    },
    8,
  )

  ctx := &RendererContext{
    maxDepth: 6,
    margins: squarify.Margins{3, 3, 20, 20},
    ops: ops,
    prog: prog,
    style: style,
    area: area,
    resize: make(chan struct{}),
  }

  setPixmap := func(p *gdk.Pixmap) {
    if pixmap != nil {
      pixmap.Unref()
    }
    pixmap = p
    area.Widget.Emit("expose_event")
  }

  ctx.setPixmap = setPixmap

  // Start goroutine that renders directory information to a pixmap
  go PixmapRenderer(ctx)

  area.Connect("expose_event", func(ctx *glib.CallbackContext) {
    println("got expose event")
    // This should be wrapped in beginPaint and EndPaint, but those are not exposed in Golang
    //gc.SetForeground(style.BgColor)
    if pixmap != nil {
      gc := gdk.NewGC(pixmap.GetDrawable())
      area.GetWindow().GetDrawable().DrawDrawable(gc, pixmap.GetDrawable(), 0, 0, 0, 0, -1, -1)
    }
  })

  area.Connect("configure_event", func(_ *glib.CallbackContext) {
    println("got configure_event")
    ctx.resize <- struct{}{}
  })

  gdk.ThreadsEnter()
  gtk.Main()
  gdk.ThreadsLeave()

}
