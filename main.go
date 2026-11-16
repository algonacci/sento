package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type desktop struct {
	win *mygo.Window
}

func (d *desktop) Update(fn func()) {
	if d.win != nil {
		d.win.Update(fn)
	}
}

func main() {
	d := &desktop{}
	a := newApp(d)
	mygo.App.WhenReady(func() {
		d.win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "Sento",
			Width:     1120,
			Height:    760,
			MinWidth:  800,
			MinHeight: 560,
			StateKey:  "main",
			Content:   ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
