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

// Commands are what the toolbar menu, the macOS menu bar and the keyboard
// share, so each guard lives in one place.

// blocked reports whether a dialog holds the window, or it is going.
func (a *app) blocked() bool {
	return a.shortcutsOpen || a.closed || a.confirmAction != nil
}

func (a *app) cmdNewRequest() {
	if a.blocked() || a.fileBusy {
		return
	}
	a.ai.open = false
	a.newRequest()
	a.urlField.Focus()
}

func (a *app) cmdImport() {
	if a.blocked() || a.fileBusy {
		return
	}
	a.ai.open = false
	a.openFile()
}

func (a *app) cmdSaveAs() {
	if a.blocked() || a.fileBusy {
		return
	}
	a.saveFile(false)
}

func (a *app) cmdSource() {
	if a.blocked() || a.fileBusy {
		return
	}
	a.ai.open = false
	a.toggleSource()
}

func (a *app) cmdAI() {
	if a.blocked() {
		return
	}
	a.ai.open = !a.ai.open
}

func (a *app) cmdSend() {
	if a.blocked() || a.fileBusy || a.ai.open || a.busy {
		return
	}
	a.send()
}

func (a *app) cmdCancel() {
	if !a.blocked() {
		a.cancelSend()
	}
}

func (a *app) cmdFocusURL() {
	if !a.blocked() && !a.ai.open && !a.sourceMode {
		a.urlField.Focus()
	}
}

func (a *app) cmdFilter() {
	if !a.blocked() {
		a.filterField.Focus()
	}
}

func (a *app) cmdShortcuts() {
	if !a.closed && a.confirmAction == nil {
		a.shortcutsOpen = true
	}
}

// shortcutRows is the keyboard help, in the order a request is made.
func shortcutRows() [][2]string {
	k := primaryKey()
	return [][2]string{
		{k + "Enter", "Send the request (Enter in the URL too)"},
		{k + ".", "Cancel the running request"},
		{k + "L", "Go to the URL"},
		{k + "N", "New request"},
		{k + "F", "Filter requests"},
		{k + "O", "Import .http or workspace JSON"},
		{k + "Shift+S", "Save as workspace JSON"},
		{k + "Shift+E", "Edit the workspace source"},
		{k + "Shift+G", "Generate requests with AI"},
		{k + "/", "Keyboard shortcuts"},
		{closeKey(), "Close Sento (asks first if a request runs or changes are unsaved)"},
		{"Esc", "Close this dialog"},
	}
}

func (a *app) navigation(c *ui.Context) {
	a.confirmDialog(c)
	ui.DialogBase(c, &a.shortcutsOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(black.Alpha(0.4))
		panel.Width(500).Padding(20).Gap(14).Radius(12).Border(1, colLine).Background(colPane)
		ui.Text(c, "Keyboard shortcuts").FontWeight(700).FontSize(15)
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			for _, row := range shortcutRows() {
				ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Start).Children(func() {
					ui.Text(c, row[0]).FontSize(12).Font("monospace").TextColor(colMuted).Width(120).Shrink(0).SingleLine()
					ui.Text(c, row[1]).FontSize(12.5).Wrap().Grow(1).Shrink(1).MinWidth(0)
				})
			}
		})
		ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
			ui.Button(c, "Close shortcuts").AutoFocus().OnClick(func() { a.shortcutsOpen = false })
		})
	})
	if a.blocked() {
		return
	}
	c.OnShortcut(ui.Cmd, ui.KeySlash, a.cmdShortcuts)
	c.OnShortcut(ui.Cmd, ui.KeyW, a.closeInner)
	c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyG, a.cmdAI)
	c.OnShortcut(ui.Cmd, ui.KeyPeriod, a.cmdCancel)
	c.OnShortcut(ui.Cmd, ui.KeyN, a.cmdNewRequest)
	c.OnShortcut(ui.Cmd, ui.KeyO, a.cmdImport)
	c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyS, a.cmdSaveAs)
	c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyE, a.cmdSource)
	c.OnShortcut(ui.Cmd, ui.KeyF, a.cmdFilter)
	if !a.ai.open {
		c.OnShortcut(ui.Cmd, ui.KeyEnter, a.cmdSend)
		c.OnShortcut(ui.Cmd, ui.KeyL, a.cmdFocusURL)
	}
}
