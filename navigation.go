package main

import "github.com/egoist/mygo/ui"

func (a *app) navigation(c *ui.Context) {
	ui.DialogBase(c, &a.shortcutsOpen, func(_, panel ui.Element) {
		panel.Width(480).Padding(20).Gap(12).Background(c.Theme().Surface)
		ui.Text(c, "Keyboard shortcuts").FontWeight(700)
		ui.Text(c, "Cmd/Ctrl+N — New request\nCmd/Ctrl+O — Import\nCmd/Ctrl+Shift+S — Save as\nCmd/Ctrl+Enter — Send request\nCmd/Ctrl+. — Cancel request\nCmd/Ctrl+/ — Keyboard shortcuts\nEsc — Close this dialog").Wrap()
		ui.Button(c, "Close shortcuts").AutoFocus().OnClick(func() { a.shortcutsOpen = false })
	})
	if a.shortcutsOpen || a.closed {
		return
	}
	c.OnShortcut(ui.Cmd, ui.KeySlash, func() { a.shortcutsOpen = true })
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
