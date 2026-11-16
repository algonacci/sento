package main

// host keeps the native window outside application state. Future background
// jobs must apply their results through Update, which queues work on the UI thread.
type host interface {
	Update(func())
}

type requestDraft struct {
	method string
	url    string
	body   string
}

type app struct {
	host  host
	draft requestDraft
}

func newApp(h host) *app {
	a := &app{host: h}
	a.newRequest()
	return a
}

func (a *app) newRequest() {
	a.draft = requestDraft{method: "GET"}
}
