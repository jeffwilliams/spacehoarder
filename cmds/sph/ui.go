package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	sh "github.com/jeffwilliams/spacehoarder"
	dt "github.com/jeffwilliams/spacehoarder/dirtree"
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

type DrawDampener struct {
	screen tcell.Screen
	ch     chan struct{}
}

func NewDrawDampener(s tcell.Screen) *DrawDampener {
	d := DrawDampener{screen: s,
		ch: make(chan struct{}),
	}

	go func() {
		for _ = range d.ch {
			de := DirtreeDrawEvent(time.Now())
			d.screen.PostEvent(&de)
			time.Sleep(200 * time.Millisecond)
		}
	}()

	return &d
}

// Draw schedules a draw event sometime in the future
func (d DrawDampener) Draw() {
	select {
	case d.ch <- struct{}{}:

	default:
	}
}

type DirtreeWidget struct {
	dt        *dt.Dirtree
	view      views.View
	listeners map[tcell.EventHandler]interface{}
	damper    *DrawDampener
}

func NewDirtreeWidget(screen tcell.Screen) *DirtreeWidget {
	return &DirtreeWidget{
		dt:     dt.New(),
		damper: NewDrawDampener(screen),
	}
}

func (w DirtreeWidget) Draw() {
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
	case *DirtreeOpEvent:
		w.dt.Apply(ev.OpData)
		// schedule a draw
		w.damper.Draw()
		//w.Draw()
		return true
	case *DirtreeDrawEvent:
		w.Draw()
		return true
	case *DirtreeProgEvent:
		return true

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
