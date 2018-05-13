package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	sh "github.com/jeffwilliams/spacehoarder"
	dt "github.com/jeffwilliams/spacehoarder/dirtree"
	"github.com/jeffwilliams/spacehoarder/tree"
)

type TcellPrintContext struct {
	View  views.View
	Style tcell.Style
	X, Y  int
}

func ViewPrint(ctx *TcellPrintContext, frmt string, args ...interface{}) (updatedCtx TcellPrintContext) {
	updatedCtx = *ctx

	str := fmt.Sprintf(frmt, args...)

	maxX, maxY := ctx.View.Size()

	// No wrapping supported, but \n starts a new line.

	x, y := ctx.X, ctx.Y

	defer func() {
		updatedCtx.X = x
		updatedCtx.Y = y
	}()

	if x < 0 {
		x = 0
	}
	if x >= maxX {
		return
	}
	if y < 0 {
		y = 0
	}
	if y >= maxY {
		return
	}

	for _, rn := range []rune(str) {
		if rn == '\n' {
			x = 0
			y++
		}
		if y >= maxY {
			return
		}
		if x >= maxX {
			continue // maybe a \n will occur
		}
		ctx.View.SetContent(x, y, rn, nil, ctx.Style)
		x++
	}

	return
}

type DirtreeWidget struct {
	dt        *dt.Dirtree
	view      views.View
	listeners map[tcell.EventHandler]interface{}
	//damper    *DrawDampener
	// mutex protects dt.
	Mutex        sync.Mutex
	selectedNode *dt.Node
	selectedRow  int
}

func NewDirtreeWidget(screen tcell.Screen) *DirtreeWidget {
	dt := dt.New()
	dt.SortChildren = true
	return &DirtreeWidget{
		dt:        dt,
		listeners: make(map[tcell.EventHandler]interface{}),
		//damper: NewDrawDampener(screen),
	}
}

func (w DirtreeWidget) Draw() {
	if *useOldDrawFlag {
		w.drawOld()
	} else {
		w.draw()
	}
}

func (w DirtreeWidget) clampSelectedRow() {
	_, maxY := w.view.Size()
	if w.selectedRow < 0 {
		w.selectedRow = 0
	}
	if w.selectedRow >= maxY {
		w.selectedRow = maxY - 1
	}
}

func (w DirtreeWidget) draw() {
	if w.dt == nil || w.view == nil {
		return
	}

	if w.selectedNode == nil && w.dt.Root != nil {
		w.selectedNode = w.dt.Root.Child(0).(*dt.Node)
	}

	w.view.Clear()

	ctx := TcellPrintContext{
		View:  w.view,
		Style: tcell.StyleDefault,
		X:     0,
		Y:     0,
	}

	printNode := func(n *dt.Node, depth, y int) {
		ctx.X = 0
		ctx.Y = y
		ctx = ViewPrint(&ctx, "%s+ ", strings.Repeat(" ", depth*2))
		origStyle := ctx.Style
		ctx.Style = ctx.Style.Foreground(tcell.Color(172))
		ctx = ViewPrint(&ctx, "[%s]", sh.FancySize(n.Dir.Size))
		ctx.Style = origStyle
		ViewPrint(&ctx, " %s", n.Dir.Basename)
	}

	/*
		New algorithm to implement:

		1. Make sure the selected row is within the screen. If not, clamp it to the screen.
		2. Print the lines above the selected row: do a reverse, post-order tree walk of the dirtree,
			 starting at the selected node, until there are no more rows above the selected row to draw.
		4. Print the lines from the selected row and after: do a forward, pre-order tree walk of the dirtree,
			 starting at the selected node, until there are no more rows below the selected row to draw.
	*/

	w.clampSelectedRow()

	y := w.selectedRow - 1
	delta := -1

	visitor := func(t tree.Tree, depth int) (cont, skipChildren bool) {
		n := t.(*dt.Node)

		cont = true
		if n == w.dt.Root {
			return
		}
		if y < 0 || y >= maxY {
			cont = false
			return
		}

		if y == w.selectedRow {
			ctx.Style = ctx.Style.Background(tcell.ColorBlue)
		} else {
			ctx.Style = tcell.StyleDefault
		}

		printNode(n, depth, y)

		y += delta

		return
	}

	tree.Walk(w.selectedNode, visitor, tree.Reverse, tree.PostOrder, w.selectedNode.Depth()-1, true)

	y = w.selectedRow
	delta = 1

	//tree.Walk(w.selectedNode, visitor, tree.Forward, tree.PreOrder, w.selectedNode.Depth(), true)
	tree.Walk(w.selectedNode, visitor, tree.Forward, tree.PreOrder, w.selectedNode.Depth()-1, false)

}

func (w DirtreeWidget) drawOld() {
	if w.dt == nil || w.view == nil {
		return
	}

	w.view.Clear()

	ctx := TcellPrintContext{
		View:  w.view,
		Style: tcell.StyleDefault,
		X:     0,
		Y:     0,
	}

	_, maxY := w.view.Size()

	visitor := func(n *dt.Node, depth int) (cont, skipChildren bool) {
		cont = true
		if n == w.dt.Root {
			return
		}

		ctx.X = 0
		ctx = ViewPrint(&ctx, "%s+ ", strings.Repeat(" ", depth*2))
		ctx.Style = ctx.Style.Foreground(tcell.Color(172))
		ctx = ViewPrint(&ctx, "[%s]", sh.FancySize(n.Dir.Size))
		ctx.Style = tcell.StyleDefault
		ViewPrint(&ctx, " %s", n.Dir.Basename)
		ctx.Y += 1

		if ctx.Y >= maxY {
			cont = false
		}
		return
	}

	w.dt.Root.Walk(visitor, -1)
}

func (w DirtreeWidget) Resize() {
}

func (w DirtreeWidget) HandleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'Q', 'q':
				app.Quit()
				return true
			}
		}
	case *DirtreeDrawEvent:
		return true
	case *DirtreeProgEvent:
		return true
	case tcell.KeyDown:
		w.clampSelectedRow()
		w.selectedRow += 1
		w.clampSelectedRow()
	}

	return false
}

func (w *DirtreeWidget) SetView(view views.View) {
	w.view = view
}

func (w DirtreeWidget) Size() (int, int) {
	return w.view.Size()
}

func (w *DirtreeWidget) Watch(handler tcell.EventHandler) {
	w.listeners[handler] = nil
}

func (w DirtreeWidget) Unwatch(handler tcell.EventHandler) {
	delete(w.listeners, handler)
}
