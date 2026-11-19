package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// newCloseFixture is an app in a window, with HOME and the config dir in a
// temporary directory.
func newCloseFixture(t *testing.T) (*app, *queuedHost, *ui.Tester) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("AppData", dir)
	h := &queuedHost{updates: make(chan func(), 8)}
	a := newApp(h)
	t.Cleanup(a.shutdown)
	tt := ui.NewTester(a.view, 1120, 760)
	tt.Frame()
	return a, h, tt
}

func pressClose(tt *ui.Tester) {
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyW)
	tt.Frame()
}

func clickOK(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	tt.Frame()
	if err := tt.Click(label); err != nil {
		t.Fatalf("click %q: %v; have %v", label, err, tt.Texts())
	}
	tt.Frame()
}

// TestCloseKeyQuitsIdle: with nothing running and nothing changed, Cmd/Ctrl+W
// closes the window at once, also after a look at the source.
func TestCloseKeyQuitsIdle(t *testing.T) {
	a, h, tt := newCloseFixture(t)
	// Source refuses to go back to the builder while a request lacks a URL.
	_, r := a.doc.FindRequest(a.selected)
	r.URL = "https://example.test"
	a.markSaved()
	a.toggleSource()
	a.toggleSource()
	if a.sourceMode || a.unsaved() {
		t.Fatalf("source round trip: source mode %v, unsaved %v (%s)", a.sourceMode, a.unsaved(), a.message)
	}
	pressClose(tt)
	if a.confirmAction != nil || !h.closed || !a.allowClose {
		t.Fatalf("confirm %q, closed %v", a.confirmTitle, h.closed)
	}
}

// TestCloseKeyAsksForUnsavedEdits: edits in the builder ask first; Cancel
// (also Enter, the default) keeps them and the keyboard where it was; after
// Save as there is nothing to ask about.
func TestCloseKeyAsksForUnsavedEdits(t *testing.T) {
	a, h, tt := newCloseFixture(t)
	clickOK(t, tt, "Request URL")
	tt.Type("https://example.test/users")
	_, r := a.doc.FindRequest(a.selected)
	if r.URL != "https://example.test/users" {
		t.Fatalf("typing did not reach the URL: %q", r.URL)
	}

	pressClose(tt)
	if h.closed || a.confirmTitle != "Discard unsaved changes to this workspace?" || a.confirmLabel != "Discard and quit" {
		t.Fatalf("closed %v, confirm %q %q", h.closed, a.confirmTitle, a.confirmLabel)
	}
	// Behind the question, W does nothing more.
	pressClose(tt)
	if h.closed || a.confirmAction == nil {
		t.Fatal("W acted behind the confirmation")
	}
	clickOK(t, tt, "Cancel")
	if h.closed || a.allowClose || a.confirmAction != nil {
		t.Fatalf("Cancel quit: closed %v", h.closed)
	}
	tt.Type("/1")
	tt.Frame()
	if r.URL != "https://example.test/users/1" {
		t.Fatalf("keyboard not back in the URL field: %q", r.URL)
	}

	pressClose(tt)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if h.closed || a.confirmAction != nil || r.URL != "https://example.test/users/1" {
		t.Fatalf("Enter did not cancel: closed %v, confirm %q", h.closed, a.confirmTitle)
	}

	h.save = filepath.Join(t.TempDir(), "workspace.sento.json")
	a.saveFile(false)
	h.next(t)()
	if _, err := os.Stat(h.save); err != nil || a.unsaved() {
		t.Fatalf("save: %v, unsaved %v (%s)", err, a.unsaved(), a.message)
	}
	pressClose(tt)
	if a.confirmAction != nil || !h.closed {
		t.Fatalf("after save: confirm %q, closed %v", a.confirmTitle, h.closed)
	}
}

// TestDiscardAndQuit: confirming the question quits.
func TestDiscardAndQuit(t *testing.T) {
	a, h, tt := newCloseFixture(t)
	a.newRequest()
	a.requestQuit()
	clickOK(t, tt, "Discard and quit")
	if !h.closed || !a.allowClose {
		t.Fatalf("closed %v, allowClose %v", h.closed, a.allowClose)
	}
}

func TestQuitWaitsForSaveAs(t *testing.T) {
	a, h, tt := newCloseFixture(t)
	_, r := a.doc.FindRequest(a.selected)
	r.URL = "https://example.test/users"
	h.save = filepath.Join(t.TempDir(), "workspace.sento.json")
	a.saveFile(false)
	if !a.fileBusy {
		t.Fatal("Save as did not start")
	}
	pressClose(tt)
	a.requestQuit() // window close / Cmd+Q uses the same guard
	if h.closed || a.allowClose || a.confirmAction != nil || a.message != "Finish or cancel the file dialog before quitting Sento." {
		t.Fatalf("quit during Save as: closed %v, confirm %q, message %q", h.closed, a.confirmTitle, a.message)
	}
	h.next(t)()
	if a.fileBusy || a.unsaved() {
		t.Fatalf("Save as did not finish: busy %v, unsaved %v", a.fileBusy, a.unsaved())
	}
	pressClose(tt)
	if !h.closed || a.confirmAction != nil {
		t.Fatalf("quit after Save as: closed %v, confirm %q", h.closed, a.confirmTitle)
	}
}

// TestUnsavedSourceAndOpenedFile: source typed but not yet parsed counts;
// a file just imported does not.
func TestUnsavedSourceAndOpenedFile(t *testing.T) {
	a, h, _ := newCloseFixture(t)
	a.toggleSource()
	a.source += "\n"
	if !a.unsaved() {
		t.Fatal("unparsed source edit not counted")
	}
	data, err := os.ReadFile("examples/requests.http")
	if err != nil {
		t.Fatal(err)
	}
	h.open = filepath.Join(t.TempDir(), "requests.http")
	if err := os.WriteFile(h.open, data, 0o600); err != nil {
		t.Fatal(err)
	}
	a.openFile()
	h.next(t)()
	if a.path != h.open || a.unsaved() {
		t.Fatalf("imported file: path %q, unsaved %v (%s)", a.path, a.unsaved(), a.message)
	}
	a.toggleSource()
	a.toggleSource()
	if a.sourceMode || a.unsaved() {
		t.Fatalf("source round trip of an import: source mode %v, unsaved %v (%s)", a.sourceMode, a.unsaved(), a.message)
	}
	_, r := a.doc.FindRequest(a.selected)
	r.Name = "renamed"
	a.requestQuit()
	if a.confirmTitle != "Discard unsaved changes to requests.http?" || h.closed {
		t.Fatalf("confirm %q, closed %v", a.confirmTitle, h.closed)
	}
}

// TestQuitAsksWhileRequestRuns: closing the window (its button, Cmd+Q, the
// window manager all call requestQuit) asks while Send runs.
func TestQuitAsksWhileRequestRuns(t *testing.T) {
	started := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		<-r.Context().Done()
	}))
	defer srv.Close()
	a, h, tt := newCloseFixture(t)
	defer a.shutdown() // before srv.Close: canceling ends the handler
	_, r := a.doc.FindRequest(a.selected)
	r.URL = srv.URL
	a.markSaved() // as if just saved: only the request is at stake
	a.send()
	<-started

	a.requestQuit()
	tt.Frame()
	if h.closed || a.confirmTitle != "The request GET Untitled request is still running. Quitting Sento stops it." || a.confirmLabel != "Quit Sento" {
		t.Fatalf("closed %v, confirm %q %q", h.closed, a.confirmTitle, a.confirmLabel)
	}
	clickOK(t, tt, "Cancel")
	if h.closed || !a.busy || a.cancel == nil {
		t.Fatalf("Cancel stopped the request: closed %v, busy %v", h.closed, a.busy)
	}

	a.requestQuit()
	clickOK(t, tt, "Quit Sento")
	if !h.closed || !a.allowClose {
		t.Fatalf("closed %v, allowClose %v", h.closed, a.allowClose)
	}
	// The window is gone: OnClosed runs shutdown, which cancels the
	// request; its late result is dropped.
	a.shutdown()
	h.next(t)()
	if a.busy || a.response != "" {
		t.Fatalf("busy %v, response %q", a.busy, a.response)
	}
}

// TestCloseIgnoredBehindDialog: with the shortcuts dialog open, neither the
// key nor the menu item (which calls closeInner) closes anything.
func TestCloseIgnoredBehindDialog(t *testing.T) {
	a, h, tt := newCloseFixture(t)
	clickOK(t, tt, "Shortcuts")
	if !a.shortcutsOpen || !tt.HasText(closeKey()) || !tt.HasText("Close Sento (asks first if a request runs or changes are unsaved)") {
		t.Fatalf("shortcuts %v; have %v", a.shortcutsOpen, tt.Texts())
	}
	pressClose(tt)
	a.closeInner()
	if h.closed || a.confirmAction != nil {
		t.Fatalf("closed %v, confirm %q", h.closed, a.confirmTitle)
	}
}
