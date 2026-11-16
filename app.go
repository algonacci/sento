package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/algonacci/sento/internal/httpfile"
	"github.com/algonacci/sento/internal/request"
	"github.com/algonacci/sento/internal/workspace"
)

type host interface {
	Update(func())
	OpenFile() (string, error)
	SaveFile(string) (string, error)
}

type app struct {
	host                                            host
	source, path, format, selected                  string
	doc                                             *workspace.Document
	message, response, responseHeaders, resultLabel string
	busy, fileBusy, closed                          bool
	cancel                                          context.CancelFunc
	generation                                      int
	ai                                              aiState
	sourceMode                                      bool
	requestTab                                      string
}

func newApp(h host) *app {
	a := &app{host: h, format: ".json", requestTab: "Body"}
	a.ai.baseURL = os.Getenv("SENTO_AI_BASE_URL")
	a.ai.model = os.Getenv("SENTO_AI_MODEL")
	a.ai.key = os.Getenv("SENTO_AI_API_KEY")
	a.doc = &workspace.Document{Version: workspace.Version, Active: "workspace", Projects: []workspace.Project{{
		ID: "workspace", Name: "My workspace", HTTP: workspace.HTTPOptions{TimeoutSeconds: 30},
		Folders: []workspace.Folder{{ID: "requests", Name: "Requests"}},
	}}}
	a.newRequest()
	return a
}

func (a *app) parse() bool {
	var d *workspace.Document
	var err error
	if a.format == ".json" {
		d, err = workspace.Decode([]byte(a.source))
	} else {
		d, err = httpfile.Parse(a.source)
	}
	if err != nil {
		a.message = err.Error()
		return false
	}
	a.doc = d
	if _, r := d.FindRequest(a.selected); r == nil {
		a.selected = ""
		for _, p := range d.Projects {
			for _, f := range p.Folders {
				if len(f.Requests) > 0 && a.selected == "" {
					a.selected = f.Requests[0].ID
				}
			}
		}
	}
	a.message = "Valid " + a.format + " · changes stay in the editor until saved"
	return true
}

func (a *app) toggleSource() {
	if a.sourceMode {
		if a.parse() {
			a.sourceMode = false
		}
		return
	}
	// JSON preserves all builder settings, including auth, forms and environments.
	data, err := json.MarshalIndent(a.doc, "", "  ")
	if err != nil {
		a.message = err.Error()
		return
	}
	a.source, a.format, a.sourceMode = string(data), ".json", true
	a.message = "Advanced workspace source · .http files can be opened with Import"
}

func (a *app) stop() {
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.generation++
	a.busy = false
}

func (a *app) shutdown() {
	a.closed = true
	a.stop()
	a.stopAI()
}

func (a *app) send() {
	if a.busy || (a.sourceMode && !a.parse()) {
		return
	}
	p, r := a.doc.FindRequest(a.selected)
	if r == nil {
		a.message = "Select a request first"
		return
	}
	req, opts, err := workspace.Prepare(*p, *r, "", filepath.Dir(a.path), os.LookupEnv)
	if err != nil {
		a.message = err.Error()
		return
	}
	client, err := request.Build(opts, nil)
	if err != nil {
		a.message = err.Error()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel, a.busy = cancel, true
	a.generation++
	gen := a.generation
	a.response, a.responseHeaders, a.resultLabel = "", "", "Sending "+r.Name+"…"
	go func() {
		defer cancel()
		defer client.CloseIdleConnections()
		result := request.Send(ctx, req, workspace.Auth{}, nil, client)
		// Formatting is also work: keep it off the UI thread.
		body := request.BodyPreview(result.Body)
		if len(body) > 256<<10 {
			body = body[:256<<10] + "\n[Preview limited to 256 KiB; use the CLI for the full response]"
		}
		var headers strings.Builder
		for _, h := range result.Headers {
			fmt.Fprintf(&headers, "%s: %s\n", h.Key, h.Value)
		}
		a.host.Update(func() {
			if a.closed || gen != a.generation {
				return
			}
			a.busy, a.cancel = false, nil
			a.response, a.responseHeaders = body, headers.String()
			a.resultLabel = request.StatusLabel(result.Code, result.Status) + " · " + request.FormatDuration(result.Duration) + " · " + request.FormatSize(len(result.Body))
			if result.Err != "" {
				a.resultLabel = result.Err
			}
		})
	}()
}
