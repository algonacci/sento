package main

import (
	"log"
	"path/filepath"
	"strings"

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

func (d *desktop) OpenFile() (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: d.win, Title: "Open Sento requests",
		Filters: []mygo.FileFilter{{Name: "Request files", Extensions: []string{"http", "json"}}},
	})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return paths[0], nil
}

func (d *desktop) SaveFile(name string) (string, error) {
	ext := strings.TrimPrefix(filepath.Ext(name), ".")
	if ext == "" {
		ext = "http"
	}
	return mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent: d.win, Title: "Save requests", DefaultPath: name,
		Filters: []mygo.FileFilter{{Name: "Request file", Extensions: []string{ext}}},
	})
}

func main() {
	d := &desktop{}
	a := newApp(d)
	mygo.App.WhenReady(func() {
		d.win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "Sento",
			Width:     1120,
			Height:    760,
			MinWidth:  960,
			MinHeight: 680,
			StateKey:  "main",
			Content:   ui.View(a.view),
		})
		d.win.OnClose(func(_ *mygo.CloseEvent) { a.shutdown() })
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
