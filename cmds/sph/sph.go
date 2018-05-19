package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"flag"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	sh "github.com/jeffwilliams/spacehoarder"
	dt "github.com/jeffwilliams/spacehoarder/dirtree"
)

var app views.Application
var status *views.Text

func setStatus(s string, args ...interface{}) {
	if status != nil {
		msg := fmt.Sprintf(s, args...)
		status.SetText(msg)
	}
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

func ApplyAll(screen tcell.Screen, t *dt.Dirtree, m *sync.Mutex, ops chan dt.OpData) {

	ch := make(chan struct{})

	go func() {
		for _ = range ch {
			de := DirtreeDrawEvent(time.Now())
			screen.PostEvent(&de)
			time.Sleep(1000 * time.Millisecond)
		}
	}()

	for op := range ops {
		m.Lock()
		added := t.Apply(op)
		if added != nil {
			updateHiddenFlag(added)
			setStatus("Processing %s", added.Dir.Path)
		}
		if added == t.Root {
			// Root node is always expanded
			setTreeNodeFlags(t.Root, treeNodeFlags(t.Root)|TreeNodeFlagExpanded)
		}
		m.Unlock()

		select {
		case ch <- struct{}{}:
		default:
		}
	}

	if t.Root != nil {
		setStatus("Completed. Total %s", sh.FancySize(t.Root.Dir.Size))
	} else {
		setStatus("Completed. ")
	}
	de := DirtreeDrawEvent(time.Now())
	screen.PostEvent(&de)
}

func drop(prog chan string) {
	for _ = range prog {

	}
}

func main() {

	flag.Parse()

	rootPath := "."

	// Test if getting device id is supported
	_, err := sh.GetFsDevId(rootPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Printf("terminal initialization failed: %v\n", err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			//screen.Fini()
			fmt.Fprintf(os.Stderr, "panic: %v\n", r)
			debug.PrintStack()
		}
	}()

	dtw := NewDirtreeWidget(screen)

	app.SetScreen(screen)

	panel := views.NewPanel()
	panel.SetContent(dtw)
	status = views.NewText()
	status.SetText("Welcome to spacehoarder")
	panel.SetStatus(status)

	app.SetRootWidget(panel)

	/*** Build dirtree ***/
	ops, prog := dt.Build(rootPath, true)
	go ApplyAll(screen, dtw.dt, &dtw.Mutex, ops)
	go drop(prog)
	/*** End build dirtree ***/

	if e := app.Run(); e != nil {
		fmt.Fprintln(os.Stderr, e.Error())
		return
	}
}
