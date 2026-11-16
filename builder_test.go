package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/algonacci/sento/internal/workspace"
)

func TestBuilderSendsEditedRequestAndPreservesAcrossSource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != "POST" || string(body) != `{"name":"Ada"}` || r.Header.Get("X-Test") != "builder" {
			t.Errorf("unexpected request: %s %s %v", r.Method, body, r.Header)
		}
		w.WriteHeader(201)
	}))
	defer srv.Close()
	h := &queuedHost{updates: make(chan func(), 4)}
	a := newApp(h)
	defer a.shutdown()
	if a.sourceMode {
		t.Fatal("source should not be the default")
	}
	_, r := a.doc.FindRequest(a.selected)
	r.URL, r.Method, r.Body, r.JSON = srv.URL, "POST", workspace.BodyJSON, `{"name":"Ada"}`
	r.Headers = []workspace.Pair{{Key: "X-Test", Value: "builder"}}
	a.toggleSource()
	a.toggleSource()
	if a.sourceMode {
		t.Fatalf("failed to return to builder: %s", a.message)
	}
	a.send()
	h.next(t)()
	if a.resultLabel[:3] != "201" {
		t.Fatal(a.resultLabel)
	}
	previous := a.selected
	a.newRequest()
	if a.selected == previous {
		t.Fatal("new request reused ID")
	}
	_, old := a.doc.FindRequest(previous)
	if old.JSON != `{"name":"Ada"}` {
		t.Fatal("new request lost old body")
	}
}
