package main

import (
	"encoding/json"
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo/ui"
)

// closeKey names the close shortcut as the platform shows it.
func closeKey() string {
	if runtime.GOOS == "darwin" {
		return "⌘W"
	}
	return "Ctrl+W"
}

// closeInner is Cmd/Ctrl+W. The request list and builder are the whole
// workspace (Source and AI are modes of it, not views opened from it), so
// it closes the window. Behind a dialog it does nothing (Esc closes it).
func (a *app) closeInner() {
	if a.shortcutsOpen || a.confirmAction != nil {
		return
	}
	a.requestQuit()
}

// requestQuit is every way of closing the window: Cmd/Ctrl+W, the window's
// close button, Cmd+Q, the window manager. It asks first while a request or
// an AI generation runs, or the workspace has unsaved changes.
func (a *app) requestQuit() {
	if a.closed || a.allowClose || a.confirmAction != nil {
		return
	}
	if a.fileBusy {
		a.message = "Finish or cancel the file dialog before quitting Sento."
		return
	}
	running := a.running()
	unsaved := a.unsaved()
	switch {
	case running != "" && unsaved:
		a.ask(running+", and "+a.workspaceName()+" has unsaved changes. Quitting Sento stops it and discards the changes.", "Discard and quit")
	case running != "":
		a.ask(running+". Quitting Sento stops it.", "Quit Sento")
	case unsaved:
		a.ask("Discard unsaved changes to "+a.workspaceName()+"?", "Discard and quit")
	default:
		a.quit()
	}
}

// running says what background work is in flight, or "" when none is.
func (a *app) running() string {
	switch {
	case a.busy:
		if _, r := a.doc.FindRequest(a.selected); r != nil {
			return "The request " + r.Method + " " + r.Name + " is still running"
		}
		return "A request is still running"
	case a.ai.busy:
		return "AI is still generating requests"
	}
	return ""
}

// workspaceName says where unsaved changes are: the file they came from.
func (a *app) workspaceName() string {
	if a.path == "" {
		return "this workspace"
	}
	return filepath.Base(a.path)
}

// docState is the workspace as it would be saved.
func (a *app) docState() string {
	b, err := json.Marshal(a.doc)
	if err != nil {
		return ""
	}
	return string(b)
}

// markSaved records the workspace as matching what is on disk (or, for a
// new workspace, as having nothing worth saving yet).
func (a *app) markSaved() {
	a.savedDoc, a.sourceBase = a.docState(), a.source
}

// unsaved reports whether quitting would lose edits: to the workspace
// since it was last opened or saved, or to the source not yet parsed.
func (a *app) unsaved() bool {
	return a.docState() != a.savedDoc || (a.sourceMode && a.source != a.sourceBase)
}

// quit closes the window, now that nothing needs asking. Requests are
// stopped as the window goes (shutdown, from OnClosed).
func (a *app) quit() {
	a.allowClose = true
	a.host.Close()
}

// ask opens a confirmation whose action quits; Cancel is the default and
// does nothing.
func (a *app) ask(title, label string) {
	a.shortcutsOpen = false // the window's close button can ask over it
	a.confirmTitle, a.confirmLabel, a.confirmAction = title, label, a.quit
	a.confirmCancel.Focus()
}

func (a *app) confirmDialog(c *ui.Context) {
	open := a.confirmAction != nil
	ui.DialogBase(c, &open, func(_, panel ui.Element) {
		t := c.Theme()
		panel.Width(420).Padding(20).Gap(16).Radius(t.Radius).Border(1, t.Border).Background(t.Surface)
		ui.Text(c, a.confirmTitle).FontWeight(600).Wrap()
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			c.Key("confirm-cancel")
			ui.Button(c, "Cancel").Bind(&a.confirmCancel).OnClick(func() { a.confirmAction = nil })
			c.Key("confirm-ok")
			ui.Button(c, a.confirmLabel).TextColor(t.Danger).OnClick(func() {
				fn := a.confirmAction
				a.confirmAction = nil
				if fn != nil {
					fn()
				}
			})
		})
	})
	// Esc or a click on the backdrop dismisses it. However it closes, the
	// focus goes back to where it was when it opened (ask focuses Cancel by
	// its handle, not AutoFocus, which would hide that element from MyGo).
	if !open {
		a.confirmAction = nil
	}
}
