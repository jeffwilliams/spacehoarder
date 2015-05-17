package main

import (
	"flag"
	"fmt"
	"github.com/jeffwilliams/spacehoarder/dirtree"
	"github.com/jeffwilliams/spacehoarder/ui"
	"github.com/jeffwilliams/squarify"
	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/glib"
	"github.com/mattn/go-gtk/gtk"
	"io"
	"os"
	"runtime/pprof"
	"strconv"
	"time"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func makeUi() (*gtk.Window, *gtk.DrawingArea, *gtk.Label) {
	gtk.Init(nil)

	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetTitle("Spacehoarder")
	window.Connect("destroy", func(ctx *glib.CallbackContext) {
		println("got destroy!", ctx.Data().(string))
		gtk.MainQuit()
	}, "foo")

	// Drawing Area
	area := gtk.NewDrawingArea()
	area.SetSizeRequest(640, 480)

	// Label
	progressLabel := gtk.NewLabel("Calculating...")

	// Main UI Layout
	vbox := gtk.NewVBox(false, 0)
	vbox.PackStart(area, true, true, 0)
	vbox.PackStart(progressLabel, false, false, 0)

	window.Add(vbox)
	window.ShowAll()

	return window, area, progressLabel
}

var filesNum int

func writeSvg(w io.Writer, blocks []squarify.Block) {
	fmt.Fprintf(w, "\n\n<svg width=\"%d\" height=\"%d\">\n", 100, 100)
	for _, b := range blocks {
		fmt.Fprintf(w, "  <rect x=\"%f\" y=\"%f\" width=\"%f\" height=\"%f\" style=\"fill:rgb(0,0,255);stroke-width:1;stroke:rgb(0,0,0)\"/>\n", b.X, b.Y, b.W, b.H)
		name := "[files " + strconv.Itoa(filesNum) + "]"
		if b.TreeSizer != nil {
			name = b.TreeSizer.(*dirtree.Node).Dir.Basename
		} else {
			filesNum += 1
		}
		fmt.Fprintf(w, "  <text x=\"%f\" y=\"%f\" style=\"font-size:8\">%s</text>\n", b.X, b.Y, name)
		fmt.Fprintln(w, "")

	}
	fmt.Fprintf(w, "</svg>\n")
}

type RendererContext struct {
	maxDepth  int
	margins   squarify.Margins
	ops       chan dirtree.OpData
	prog      chan string
	resize    chan struct{}
	setPixmap func(p *gdk.Pixmap)
	processed func(file string)
	complete  func(t *dirtree.Dirtree)
	style     *ui.Style
	area      *gtk.DrawingArea
}

func outputSvg(ctx *RendererContext, tree *dirtree.Dirtree, filename string) {
	gdk.ThreadsEnter()
	areaW := ctx.area.GetAllocation().Width
	areaH := ctx.area.GetAllocation().Height
	gdk.ThreadsLeave()

	blocks, _ := squarify.Squarify(tree.Root(), squarify.Rect{X: 0, Y: 0, W: float64(areaW), H: float64(areaH)},
		squarify.Options{MaxDepth: ctx.maxDepth, Margins: &ctx.margins, Sort: squarify.DoSort})

	// Output an SVG
	f, err := os.Create(filename)
	if err == nil {
		writeSvg(f, blocks)
		f.Close()
		fmt.Println("Output SVG to ", filename)
	} else {
		fmt.Println("Error opening file", filename, ":", err)
	}
}

// PixmapRenderer builds a local tree from the operations passed in ops,
// and repeatedly renders it into a pixmap that is passed to setPixmap
func PixmapRenderer(ctx *RendererContext) {
	tree := dirtree.New()

	render := func() {
		gdk.ThreadsEnter()
		areaW := ctx.area.GetAllocation().Width
		areaH := ctx.area.GetAllocation().Height
		gdk.ThreadsLeave()
		blocks, meta := squarify.Squarify(tree.Root(), squarify.Rect{X: 0, Y: 0, W: float64(areaW), H: float64(areaH)},
			squarify.Options{MaxDepth: ctx.maxDepth, Margins: &ctx.margins, Sort: squarify.DoSort, MinW: 7, MinH: 10})
		gdk.ThreadsEnter()
		pixmap := ui.Render(ctx.area.GetWindow().GetDrawable(), areaW, areaH, blocks, meta, ctx.style)
		ctx.setPixmap(pixmap)
		gdk.ThreadsLeave()
	}

	var lastRender time.Time

loop:
	for {
		select {
		case _ = <-ctx.resize:
			render()

		case op, ok := <-ctx.ops:
			if !ok {
				// We're done!
				ctx.ops = nil

				// Uncomment the below to output an SVG of the blocks
				//outputSvg(ctx, tree, "/home/shared/Jeff/sph_test.svg")

				render()
				continue loop
			}

			dirtree.Apply(tree, op)

			if lastRender.IsZero() || time.Now().Sub(lastRender) > 80*time.Millisecond {
				render()
			}

		case p, ok := <-ctx.prog:
			if !ok {
				ctx.prog = nil
				ctx.complete(tree)
				continue loop
			}
			fmt.Println(p)
			gdk.ThreadsEnter()
			ctx.processed(p)
			gdk.ThreadsLeave()
		}
	}

	fmt.Println("Pixmap renderer done")
}

type ExposeReason int

const (
	NoReason ExposeReason = iota
	UpdatePixmap
	UpdateProcessedFile
)

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Println("Opening profile file failed: ", err)
			os.Exit(1)
		}

		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if flag.NArg() < 1 {
		fmt.Println("Usage: sph <directory>")
		os.Exit(1)
	}

	gdk.ThreadsInit()

	// The pointer to the pixmap that the UI thread will draw on expose events.
	var pixmap *gdk.Pixmap
	// File just processed.
	var lastFile string
	// Reason we generated expose_event
	exposeReason := NoReason

	// Start goroutine that explores the directories
	ops, prog := dirtree.Build(flag.Arg(0))

	_, area, progressLabel := makeUi()

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
		margins:  squarify.Margins{3, 3, 20, 3},
		ops:      ops,
		prog:     prog,
		style:    style,
		area:     area,
		resize:   make(chan struct{}),
	}

	ctx.setPixmap = func(p *gdk.Pixmap) {
		if pixmap != nil {
			pixmap.Unref()
		}
		pixmap = p
		exposeReason = UpdatePixmap
		area.Widget.Emit("expose_event")
	}

	ctx.processed = func(file string) {
		lastFile = file
		exposeReason = UpdateProcessedFile
		area.Widget.Emit("expose_event")
	}

	ctx.complete = func(t *dirtree.Dirtree) {
		lastFile = "Completed. Size: " + ui.FancySize(t.Root().Dir.Size)
		exposeReason = UpdateProcessedFile
		area.Widget.Emit("expose_event")
	}

	// Start goroutine that renders directory information to a pixmap
	go PixmapRenderer(ctx)

	area.Connect("expose_event", func(ctx *glib.CallbackContext) {
		// This should be wrapped in beginPaint and EndPaint, but those are not exposed in Golang
		exposeReason = NoReason
		if pixmap != nil && (exposeReason == NoReason || exposeReason == UpdatePixmap) {
			gc := gdk.NewGC(pixmap.GetDrawable())
			area.GetWindow().GetDrawable().DrawDrawable(gc, pixmap.GetDrawable(), 0, 0, 0, 0, -1, -1)
		}
		if exposeReason == NoReason || exposeReason == UpdateProcessedFile {
			progressLabel.SetText(lastFile)
		}
		exposeReason = NoReason
	})

	area.Connect("configure_event", func(_ *glib.CallbackContext) {
		// Notify the rendering goroutine of a window size change.
		// We write to the channel nonblockingly to prevent a deadlock:
		// render thread could be waiting on the GTK ThreadsEnter lock
		// while we have it in this handler, meanwhile we are waiting on
		// the render thread to read from this channel.
		select {
		case ctx.resize <- struct{}{}:
		default:
		}
	})

	gdk.ThreadsEnter()
	gtk.Main()
	gdk.ThreadsLeave()

}
