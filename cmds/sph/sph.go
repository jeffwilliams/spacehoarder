package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime/debug"
	"time"

	"flag"
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
	sh "github.com/jeffwilliams/spacehoarder"
	dt "github.com/jeffwilliams/spacehoarder/dirtree"
)

var optDebugFileName = flag.String("dbgfile", "", "File to print debug info into")

var app views.Application
var status *views.Text
var keysHelpMsg = "<enter>: expand/collapse  f: show/hide files  r: refresh"

func setStatus(s string, args ...interface{}) {
	if status != nil {
		msg := fmt.Sprintf(s, args...)
		status.SetText(msg)
	}
}
func getStatus() string {
	return status.Text()
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

func main() {

	flag.Parse()

	if *optDebugFileName != "" {
		f, err := os.Create(*optDebugFileName)
		if err != nil {
			fmt.Printf("opening debug file failed: %v\n", err)
			return

		}
		log.SetOutput(f)
	} else {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}

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
	dtw.ShowRoot = true

	app.SetScreen(screen)

	panel := views.NewPanel()
	panel.SetContent(dtw)
	status = views.NewText()
	status.SetText("Welcome to spacehoarder")
	panel.SetStatus(status)
	help := views.NewText()
	help.SetText(keysHelpMsg)
	help.SetStyle(tcell.StyleDefault.Background(tcell.ColorBrown))
	panel.SetMenu(help)

	app.SetRootWidget(panel)

	/*** Build dirtree ***/
	build(screen, dtw, nil, rootPath, nil, nil)
	//ops, prog := dt.Build(rootPath, dt.DefaultBuildOpts)
	//go ApplyAll(screen, dtw.dt, &dtw.Mutex, ops)
	//go drop(prog)
	/*** End build dirtree ***/

	if e := app.Run(); e != nil {
		fmt.Fprintln(os.Stderr, e.Error())
		return
	}
}
