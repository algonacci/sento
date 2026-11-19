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

	"github.com/algonacci/sento/internal/workspace"
	"github.com/egoist/mygo/ui"
)

// SENTO_SNAPSHOT=dir writes PNGs of the main screens, light and dark, at a
// large and a compact size, for looking at the layout without a window.
func TestSnapshots(t *testing.T) {
	dir := os.Getenv("SENTO_SNAPSHOT")
	if dir == "" {
		t.Skip("set SENTO_SNAPSHOT=dir")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "b7c1e2a4-5d8f-4f1e-9a0b-3c2d1e0f9a8b")
		w.Header().Set("Cache-Control", "no-store")
		if strings.HasSuffix(r.URL.Path, "/big") {
			w.Write([]byte(`"` + strings.Repeat("x", previewLimit+4096) + `"`))
			return
		}
		if r.Method == "POST" {
			w.WriteHeader(201)
		}
		w.Write([]byte(`{"args":{"tag":["go","mygo"]},"headers":{"Accept":"application/json","Host":"api.local"},"url":"` + "http://api.local/get?tag=go&tag=mygo" + `","ok":true}`))
	}))
	defer srv.Close()
	h := &queuedHost{updates: make(chan func(), 8)}
	a := newApp(h)
	defer a.shutdown()
	data, err := os.ReadFile("examples/requests.http")
	if err != nil {
		t.Fatal(err)
	}
	a.source, a.format, a.path = strings.ReplaceAll(string(data), "https://httpbin.org", srv.URL), ".http", "/home/dev/api/requests.http"
	if !a.parse() {
		t.Fatal(a.message)
	}
	p := a.doc.ActiveProject()
	p.Environments = []workspace.Environment{{ID: "dev", Name: "Development"}, {ID: "staging", Name: "Staging"}}
	p.ActiveEnv = "dev"
	a.markSaved()

	a.prefs.Theme = themeSystem // SetDark picks the appearance
	tt := ui.NewTester(a.view, 1280, 800)
	shot := func(name string) {
		t.Helper()
		tt.Frame()
		tt.Advance(time.Second) // let hover and selection fades finish
		tt.Frame()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
	both := func(name string) {
		t.Helper()
		for _, mode := range []struct {
			suffix string
			dark   bool
		}{{"-dark", true}, {"-light", false}} {
			tt.SetDark(mode.dark)
			for _, size := range []struct {
				suffix string
				w, h   int
			}{{"", 1280, 800}, {"-compact", 760, 520}} {
				tt.SetSize(size.w, size.h)
				shot(name + size.suffix + mode.suffix)
			}
		}
		tt.SetSize(1280, 800)
	}

	a.message = "Opened requests.http · 3 requests"
	both("1-not-sent")

	a.send()
	h.next(t)()
	a.message = "Ready"
	both("2-response")
	a.responseTab = "Headers"
	both("3-response-headers")
	a.responseTab = "Body"

	_, r := a.doc.FindRequest("create-item")
	a.selectRequest(r.ID)
	a.requestTab = "Body"
	a.send()
	h.next(t)()
	both("4-post-body")

	_, r = a.doc.FindRequest("get-query")
	a.selectRequest(r.ID)
	a.requestTab = "Headers"
	r.URL = srv.URL + "/big"
	a.send()
	h.next(t)()
	both("5-truncated")
	r.URL = srv.URL + "/get?tag=go&tag=mygo"

	a.selectRequest(r.ID)
	a.requestTab = "Params"
	a.filter = "post"
	both("6-filter")
	a.filter = ""

	a.toggleSource()
	both("7-source")
	a.toggleSource()

	a.ai.open = true
	a.ai.message = "Valid .http · ready to open in editor"
	a.ai.prompt = "Generate GET and POST requests for a users API at localhost:3000"
	both("8-ai")
	a.ai.open = false

	a.selected = ""
	both("9-no-request")

	a.shortcutsOpen = true
	tt.SetDark(true)
	shot("10-shortcuts-dark")
	a.shortcutsOpen = false
}
