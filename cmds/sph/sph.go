package main

import (
  "github.com/mattn/go-gtk/glib"
  "github.com/mattn/go-gtk/gtk"
  "github.com/jeffwilliams/spacehoarder/dirtree"
  "fmt"
)

func doUi() {
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
  gtk.Main()
}


func main() {
  //doUi()

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
