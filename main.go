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
	// Native menus, dialogs and the frame follow the chosen appearance.
	applyThemeSource = func(mode string) { mygo.Theme.SetSource(mygo.ThemeSource(mode)) }
	applyThemeSource(a.prefs.Theme)
	mygo.App.WhenReady(func() {
		d.win = mygo.NewWindow(mygo.WindowOptions{
			Title:          "Sento",
			Width:          1200,
			Height:         780,
			MinWidth:       720,
			MinHeight:      480,
			StateKey:       "main",
			TitleBarStyle:  mygo.TitleBarHidden,
			TitleBarHeight: 44,
			Content:        ui.View(a.view),
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

// installMenu replaces macOS's default menu bar: ⌘W reaches closeInner,
// which leaves an open dialog alone, as Ctrl+W does elsewhere, and the
// title bar's More menu is in the menu bar too, with the same commands.
func installMenu(a *app) {
	do := func(fn func()) func(*mygo.MenuItem, *mygo.Window) {
		return func(*mygo.MenuItem, *mygo.Window) { a.host.Update(fn) }
	}
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "File", Submenu: []*mygo.MenuItem{
			{ID: "new-request", Label: "New Request", Accelerator: "Cmd+N", Click: do(a.cmdNewRequest)},
			{ID: "import", Label: "Import…", Accelerator: "Cmd+O", Click: do(a.cmdImport)},
			{ID: "save-as", Label: "Save As…", Accelerator: "Cmd+Shift+S", Click: do(a.cmdSaveAs)},
			mygo.Separator(),
			{ID: "close", Label: "Close", Accelerator: "Cmd+W", Click: do(a.closeInner)},
		}},
		{Role: mygo.RoleEditMenu},
		{Label: "Request", Submenu: []*mygo.MenuItem{
			{ID: "send", Label: "Send", Accelerator: "Cmd+Enter", Click: do(a.cmdSend)},
			{ID: "cancel", Label: "Cancel", Accelerator: "Cmd+.", Click: do(a.cmdCancel)},
			mygo.Separator(),
			{ID: "focus-url", Label: "Go to URL", Accelerator: "Cmd+L", Click: do(a.cmdFocusURL)},
			{ID: "filter", Label: "Filter Requests", Accelerator: "Cmd+F", Click: do(a.cmdFilter)},
		}},
		{Label: "View", Submenu: []*mygo.MenuItem{
			{ID: "source", Label: "Workspace Source", Accelerator: "Cmd+Shift+E", Click: do(a.cmdSource)},
			{ID: "ai", Label: "Generate with AI", Accelerator: "Cmd+Shift+G", Click: do(a.cmdAI)},
			mygo.Separator(),
			{ID: "shortcuts", Label: "Keyboard Shortcuts", Accelerator: "Cmd+/", Click: do(a.cmdShortcuts)},
		}},
		{Role: mygo.RoleWindowMenu},
	}))
}
