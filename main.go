package main

import (
	"log"
	"path/filepath"
	"runtime"
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

func (d *desktop) Close() {
	if d.win != nil {
		d.win.Close()
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
			MinWidth:  960,
			MinHeight: 680,
			StateKey:  "main",
			Content:   ui.View(a.view),
		})
		if runtime.GOOS == "darwin" {
			installMenu(a)
		}
		// Every way of closing the window (its button, Cmd+Q, the window
		// manager) is held here and decided on the UI thread, which asks
		// while a request runs or changes are unsaved. A session ending
		// (logout, shutdown) cannot be held, so requests are stopped once
		// the window is gone.
		d.win.OnClose(func(e *mygo.CloseEvent) {
			if !a.allowClose {
				e.PreventDefault()
				d.Update(a.requestQuit)
			}
		})
		d.win.OnClosed(func() { a.shutdown() })
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// installMenu replaces macOS's default menu bar so that ⌘W reaches
// closeInner, which leaves an open dialog alone, as Ctrl+W does elsewhere.
func installMenu(a *app) {
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "File", Submenu: []*mygo.MenuItem{
			{ID: "close", Label: "Close", Accelerator: "Cmd+W", Click: func(*mygo.MenuItem, *mygo.Window) {
				a.host.Update(a.closeInner)
			}},
		}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	}))
}
