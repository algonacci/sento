package main

import (
	"runtime"

	"github.com/egoist/mygo/ui"
)

// primaryKey names Cmd on macOS and Ctrl elsewhere, as shortcuts show it.
func primaryKey() string {
	if runtime.GOOS == "darwin" {
		return "⌘"
	}
	return "Ctrl+"
}

func (a *app) navigation(c *ui.Context) {
	a.confirmDialog(c)
	ui.DialogBase(c, &a.shortcutsOpen, func(_, panel ui.Element) {
		panel.Width(480).Padding(20).Gap(12).Background(c.Theme().Surface)
		ui.Text(c, "Keyboard shortcuts").FontWeight(700)
		k := primaryKey()
		ui.Text(c, k+"N — New request\n"+k+"O — Import\n"+k+"Shift+S — Save as\n"+k+"Enter — Send request\n"+k+". — Cancel request\n"+closeKey()+" — Close Sento (asks first if a request runs or changes are unsaved)\n"+k+"/ — Keyboard shortcuts\nEsc — Close this dialog").Wrap()
		ui.Button(c, "Close shortcuts").AutoFocus().OnClick(func() { a.shortcutsOpen = false })
	})
	if a.shortcutsOpen || a.closed || a.confirmAction != nil {
		return
	}
	c.OnShortcut(ui.Cmd, ui.KeySlash, func() { a.shortcutsOpen = true })
	c.OnShortcut(ui.Cmd, ui.KeyW, a.closeInner)
	if !a.fileBusy && !a.ai.open {
		c.OnShortcut(ui.Cmd, ui.KeyN, a.newRequest)
		c.OnShortcut(ui.Cmd, ui.KeyO, a.openFile)
		c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyS, func() { a.saveFile(false) })
		c.OnShortcut(ui.Cmd, ui.KeyEnter, func() {
			if !a.busy {
				a.send()
			}
		})
	}
	c.OnShortcut(ui.Cmd, ui.KeyPeriod, func() {
		if a.busy {
			a.stop()
			a.resultLabel = "Request cancelled"
		}
	})
}
