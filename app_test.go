package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/algonacci/sento/internal/workspace"

	"github.com/egoist/mygo/ui"
)

type queuedHost struct {
	updates chan func()
	closed  bool   // Close was called: the window closes and the app quits
	open    string // what OpenFile returns
	save    string // what SaveFile returns
}

func (h *queuedHost) Update(fn func())                { h.updates <- fn }
func (h *queuedHost) OpenFile() (string, error)       { return h.open, nil }
func (h *queuedHost) SaveFile(string) (string, error) { return h.save, nil }
func (h *queuedHost) Close()                          { h.closed = true }

func (h *queuedHost) next(t *testing.T) func() {
	t.Helper()
	select {
	case fn := <-h.updates:
		return fn
	case <-time.After(5 * time.Second):
		t.Fatal("UI update never queued")
		return nil
	}
}

func TestSendSnapshotAndCanceledQueuedResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"ok":true}`)) }))
	defer srv.Close()
	h := &queuedHost{updates: make(chan func(), 4)}
	a := newApp(h)
	defer a.shutdown()
	a.source = "# @name local\nGET " + srv.URL
	a.sourceMode, a.format = true, ".http"
	a.send()
	callback := h.next(t)
	if !a.busy || a.response != "" {
		t.Fatal("worker mutated UI state before update ran")
	}
	a.stop()
	a.resultLabel = "Canceled"
	callback()
	if a.response != "" || a.resultLabel != "Canceled" {
		t.Fatal("late result overwrote cancellation")
	}
	a.send()
	h.next(t)()
	if a.busy || !strings.Contains(a.response, `"ok": true`) {
		t.Fatalf("response not applied: %q", a.response)
	}
}

func TestCanceledAICompletionCannotReplaceNewState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"GET https://example.com"}}]}`))
	}))
	defer srv.Close()
	h := &queuedHost{updates: make(chan func(), 4)}
	a := newApp(h)
	defer a.shutdown()
	a.ai.baseURL, a.ai.model, a.ai.prompt = srv.URL, "model", "generate requests"
	a.generate()
	callback := h.next(t)
	a.stopAI()
	callback()
	if a.ai.answer != "" || a.ai.busy {
		t.Fatal("late AI result applied")
	}
}

func TestNativeViewsAndResize(t *testing.T) {
	a := newApp(&queuedHost{updates: make(chan func(), 4)})
	defer a.shutdown()
	tt := ui.NewTester(a.view, 1280, 800)
	for _, size := range [][2]int{{1280, 800}, {1120, 760}, {760, 520}, {720, 480}} {
		tt.SetSize(size[0], size[1])
		for _, label := range []string{"Request name", "Request URL", "HTTP method", "Send", "More", "Shortcuts", "New request", "Filter requests", "Environment", "Params", "Body"} {
			box, ok := tt.Find(label)
			if !ok || box.W <= 0 || box.H <= 0 || box.X < 0 || box.X+box.W > float32(size[0]) || box.Y+box.H > float32(size[1]) {
				t.Fatalf("missing %q at %v: %+v", label, size, box)
			}
		}
		if !tt.HasText("Not sent yet") {
			t.Fatalf("no empty response state at %v", size)
		}
	}
	// The rare actions are in the More menu.
	tt.SetSize(1280, 800)
	r, _ := tt.Find("More")
	tt.Press(r.X+r.W/2, r.Y+r.H/2)
	tt.Release(r.X+r.W/2, r.Y+r.H/2)
	want := []string{"New request", "Import .http or JSON…", "Save as…", "-", "Edit workspace source", "Generate with AI", "-", "Keyboard shortcuts"}
	if got := tt.Menu(); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("More menu %q, want %q", got, want)
	}
	if err := tt.ChooseMenuItem("Generate with AI"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if _, ok := tt.Find("API description"); !ok || !a.ai.open {
		t.Fatal("AI view did not open")
	}
	if err := tt.Click("Close AI"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.ai.open {
		t.Fatal("AI view did not close")
	}
}

// TestResponseViewAndEnterSends: Enter in the URL sends; the response shows
// its status, body and headers in tabs.
func TestResponseViewAndEnterSends(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "yes")
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"missing"}`))
	}))
	defer srv.Close()
	h := &queuedHost{updates: make(chan func(), 4)}
	a := newApp(h)
	defer a.shutdown()
	tt := ui.NewTester(a.view, 1280, 800)
	tt.Frame()
	if err := tt.Click("Request URL"); err != nil {
		t.Fatal(err)
	}
	tt.Type(srv.URL)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !a.busy {
		t.Fatal("Enter in the URL did not send")
	}
	h.next(t)()
	tt.Frame()
	if a.respCode != 404 || !tt.HasText("404") {
		t.Fatalf("status %d; texts %v", a.respCode, tt.Texts())
	}
	if _, ok := tt.Find("Response body"); !ok {
		t.Fatal("no response body")
	}
	if err := tt.Click("Response headers tab"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if _, ok := tt.Find("Response headers"); !ok || !tt.HasText("X-Test") {
		t.Fatalf("no response headers: %v", tt.Texts())
	}
}

// TestThemeCyclesAndPersists: the title bar button goes System → Light →
// Dark → System, the view follows, and the choice is kept for next time.
func TestThemeCyclesAndPersists(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SENTO_CONFIG_DIR", dir)
	var applied []string
	defer func(old func(string)) { applyThemeSource = old }(applyThemeSource)
	applyThemeSource = func(mode string) { applied = append(applied, mode) }
	a := newApp(&queuedHost{updates: make(chan func(), 4)})
	defer a.shutdown()
	if a.prefs.Theme != themeSystem {
		t.Fatalf("default theme %q", a.prefs.Theme)
	}
	tt := ui.NewTester(a.view, 1280, 800)
	tt.SetDark(true)
	for _, want := range []struct {
		mode, label string
		dark        bool
	}{{themeLight, "Light", false}, {themeDark, "Dark", true}, {themeSystem, "System", true}} {
		tt.Frame()
		if err := tt.Click("Appearance"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		if a.prefs.Theme != want.mode || !tt.HasText(want.label) || isDark != want.dark {
			t.Fatalf("after click: theme %q, dark %v; want %+v", a.prefs.Theme, isDark, want)
		}
		if got := loadSettings().Theme; got != want.mode {
			t.Fatalf("saved %q, want %q", got, want.mode)
		}
	}
	if strings.Join(applied, ",") != "light,dark,system" {
		t.Fatalf("platform appearance %v", applied)
	}
	// System follows the system.
	tt.SetDark(false)
	tt.Frame()
	if isDark {
		t.Fatal("system mode ignored a light system")
	}
	a.cycleTheme()
	if b := newApp(&queuedHost{updates: make(chan func(), 1)}); b.prefs.Theme != themeLight {
		t.Fatalf("restart theme %q", b.prefs.Theme)
	}
}

func TestPreviewBodyLimit(t *testing.T) {
	if body, cut := previewBody("small"); cut || body != "small" {
		t.Fatal("short body cut")
	}
	long := strings.Repeat("a", previewLimit-1) + "é" + "tail"
	body, cut := previewBody(long)
	if !cut || len(body) > previewLimit || !utf8.ValidString(body) {
		t.Fatalf("cut %v, len %d, valid %v", cut, len(body), utf8.ValidString(body))
	}
}

func TestFilterAndEnvironment(t *testing.T) {
	a := newApp(&queuedHost{updates: make(chan func(), 4)})
	defer a.shutdown()
	a.source, a.format = "### one\n# @name list\nGET https://example.test/users\n\n### two\n# @name make\nPOST https://example.test/items\n", ".http"
	if !a.parse() {
		t.Fatal(a.message)
	}
	p := a.doc.ActiveProject()
	p.Environments = []workspace.Environment{{ID: "dev", Name: "Development"}, {ID: "prod", Name: "Production"}}
	tt := ui.NewTester(a.view, 1280, 800)
	a.filter = "post"
	tt.Frame()
	if tt.HasText("one") || !tt.HasText("two") {
		t.Fatalf("filter: %v", tt.Texts())
	}
	a.filter = "nothing"
	tt.Frame()
	if !tt.HasText("No requests match “nothing”.") {
		t.Fatalf("no empty filter state: %v", tt.Texts())
	}
	a.filter = ""
	a.envPick = "Production"
	// Choosing in the select is the platform's; drive the same change.
	for _, env := range p.Environments {
		if env.Name == a.envPick {
			p.ActiveEnv = env.ID
		}
	}
	tt.Frame()
	if a.envPick != "Production" || p.ActiveEnv != "prod" {
		t.Fatalf("environment %q / %q", a.envPick, p.ActiveEnv)
	}
}
