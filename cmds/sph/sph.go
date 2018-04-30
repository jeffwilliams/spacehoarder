package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	dt "github.com/jeffwilliams/spacehoarder/dirtree"
)

var app views.Application

// MainText is simply a views.Text, but that overrides the
// HandleEvent method so that it can quit the application.
type MainText struct {
	vp *views.ViewPort
	views.Text
}

func (m *MainText) HandleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'A', 'a':
				m.Text.SetText("a")
			case 'Q', 'q':
				app.Quit()
				return true
			}
		case tcell.KeyLeft:
			m.Text.SetText("<-")
		case tcell.KeyRight:
			m.Text.SetText("->")
		case tcell.KeyDown:
			m.Text.SetText("v")
		}
	}

	return m.Text.HandleEvent(ev)
}

func BuildSampleDirtree() (tree *dt.Dirtree) {
	tree = &dt.Dirtree{Root: &dt.Node{Dir: dt.Directory{"/tmp", "tmp", 100}, SortChildren: true}}

	n := tree.Root.Add(&dt.Node{Dir: dt.Directory{"/tmp/stuff", "stuff", 60}})
	n.Add(&dt.Node{Dir: dt.Directory{"/tmp/stuff/dir2", "dir2", 20}})
	n.Add(&dt.Node{Dir: dt.Directory{"/tmp/stuff/dir1", "dir1", 1040}})

	n = tree.Root.Add(&dt.Node{Dir: dt.Directory{"/tmp/things", "things", 40}})
	n.Add(&dt.Node{Dir: dt.Directory{"/tmp/stuff/pics", "pics", 20}})
	n = n.Add(&dt.Node{Dir: dt.Directory{"/tmp/stuff/music", "music", 20}})
	n.Add(&dt.Node{Dir: dt.Directory{"/tmp/stuff/music/old", "old", 10}})
	return
}

type DirtreeOpEvent struct {
	dt.OpData
	Time time.Time
}

func (e DirtreeOpEvent) When() time.Time {
	return e.Time
}

type DirtreeProgEvent struct {
	Path string
	Time time.Time
}

func (e DirtreeProgEvent) When() time.Time {
	return e.Time
}

type DirtreeDrawEvent time.Time

func (e DirtreeDrawEvent) When() time.Time {
	return time.Time(e)
}

func toEvents(screen tcell.Screen, ops chan dt.OpData, prog chan string) {
	for {
		var e tcell.Event

		select {
		case op, ok := <-ops:
			if !ok {
				// We're done!
				continue
			}

			e = &DirtreeOpEvent{op, time.Now()}

		case p, ok := <-prog:
			if !ok {
				// We're done!
				t := DirtreeDrawEvent(time.Now())
				e = &t
			} else {
				e = &DirtreeProgEvent{p, time.Now()}
			}
		}

		for err := screen.PostEvent(e); err != nil; err = screen.PostEvent(e) {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func main() {

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Printf("terminal initialization failed: %v\n", err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			screen.Fini()
			fmt.Fprintf(os.Stderr, "panic: %v\n", r)
			debug.PrintStack()
		}
	}()

	dtw := NewDirtreeWidget(screen)

	app.SetScreen(screen)
	app.SetRootWidget(dtw)

	/*** Build dirtree ***/
	ops, prog := dt.Build(".")
	go toEvents(screen, ops, prog)
	/*** End build dirtree ***/

	if e := app.Run(); e != nil {
		fmt.Fprintln(os.Stderr, e.Error())
		return
	}
}
