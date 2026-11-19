package main

import (
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	tt := ui.NewTester(a.view, 1120, 760)
	for _, size := range [][2]int{{1120, 760}, {960, 680}, {720, 600}} {
		tt.SetSize(size[0], size[1])
		for _, label := range []string{"Request URL", "HTTP method", "Send", "Save as", "Response body", "Response headers"} {
			box, ok := tt.Find(label)
			if !ok || box.W <= 0 || box.H <= 0 || box.X < 0 || box.X+box.W > float32(size[0]) {
				t.Fatalf("missing %q at %v: %+v", label, size, box)
			}
		}
		if size[0] == 720 {
			snapshot(t, tt, "compact")
		}
	}
	tt.SetSize(1120, 760)
	snapshot(t, tt, "workspace")
	if err := tt.Click("AI"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("API description"); !ok {
		t.Fatal("AI view did not open")
	}
	snapshot(t, tt, "ai")
	if err := tt.Click("AI"); err != nil {
		t.Fatal(err)
	}
	if a.ai.open {
		t.Fatal("AI view did not close")
	}
}

func snapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SENTO_SNAPSHOT")
	if dir == "" {
		return
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
