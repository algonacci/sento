package main

import (
	"fmt"

	"github.com/algonacci/sento/internal/workspace"
	"github.com/egoist/mygo/ui"
)

func (a *app) newRequest() {
	if a.sourceMode && !a.parse() {
		return
	}
	a.sourceMode = false
	p := a.doc.ActiveProject()
	if p == nil {
		return
	}
	if len(p.Folders) == 0 {
		p.Folders = append(p.Folders, workspace.Folder{ID: "requests-" + p.ID, Name: "Requests"})
	}
	for n := 1; ; n++ {
		id := fmt.Sprintf("request-%d", n)
		used := false
		for _, project := range a.doc.Projects {
			used = used || project.ID == id
			for _, env := range project.Environments {
				used = used || env.ID == id
			}
			for _, folder := range project.Folders {
				used = used || folder.ID == id
				for _, r := range folder.Requests {
					used = used || r.ID == id
				}
			}
		}
		if used {
			continue
		}
		p.Folders[0].Requests = append(p.Folders[0].Requests, workspace.Request{ID: id, Name: "Untitled request", Method: "GET", Body: workspace.BodyNone})
		a.selectRequest(id)
		break
	}
	a.message = "Ready · enter a URL and press Send"
}

func (a *app) selectRequest(id string) {
	a.stop()
	a.selected = id
	a.response, a.responseHeaders, a.resultLabel = "", "", ""
}

func (a *app) builderView(c *ui.Context) {
	_, r := a.doc.FindRequest(a.selected)
	if r == nil {
		ui.Text(c, "Create a new request to get started")
		return
	}
	c.Key("builder-" + r.ID)
	ui.Column(c).FillWidth().Grow(1).MinHeight(180).Gap(10).Disabled(a.fileBusy).Children(func() {
		ui.TextInput(c, &r.Name).Label("Request name").Placeholder("Request name").FillWidth()
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Select(c, &r.Method, []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}).Label("HTTP method").Width(110)
			ui.TextInput(c, &r.URL).Label("Request URL").Placeholder("Enter URL, e.g. https://api.example.com/users").Grow(1).MinWidth(0)
			a.sendButton(c)
		})
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			for _, tab := range []string{"Params", "Headers", "Body", "Auth"} {
				b := ui.Button(c, tab).OnClick(func() { a.requestTab = tab })
				if a.requestTab == tab {
					b.TextColor(c.Theme().Accent)
				}
			}
		})
		switch a.requestTab {
		case "Headers":
			a.pairsView(c, &r.Headers, "header")
		case "Params":
			a.pairsView(c, &r.Params, "parameter")
		case "Auth":
			kind := string(r.Auth.Type)
			if kind == "" {
				kind = "inherit"
			}
			if ui.Select(c, &kind, []string{"inherit", "none", "bearer", "basic", "key"}).Label("Auth type").Width(160).Changed() {
				r.Auth.Type = workspace.AuthType(kind)
			}
			switch kind {
			case "bearer":
				ui.TextInput(c, &r.Auth.Token).Label("Bearer token").Placeholder("Token or {{env.API_TOKEN}}").Password().FillWidth()
			case "basic":
				ui.TextInput(c, &r.Auth.Username).Label("Username").Placeholder("Username").FillWidth()
				ui.TextInput(c, &r.Auth.Password).Label("Password").Placeholder("Password").Password().FillWidth()
			case "key":
				ui.TextInput(c, &r.Auth.Header).Label("API key header").Placeholder("X-API-Key").FillWidth()
				ui.TextInput(c, &r.Auth.Value).Label("API key value").Placeholder("Value").Password().FillWidth()
			}
		case "Body":
			kind := string(r.Body)
			if kind == "" {
				kind = "none"
			}
			if ui.Select(c, &kind, []string{"none", "json", "raw", "form", "file"}).Label("Body type").Width(160).Changed() {
				r.Body = workspace.BodyType(kind)
			}
			switch kind {
			case "json":
				ui.TextArea(c, &r.JSON).Label("Request body").Placeholder("{\"name\": \"Ada\"}").FillWidth().Grow(1).MinHeight(80)
			case "raw":
				ui.TextArea(c, &r.Raw).Label("Request body").Placeholder("Request body").FillWidth().Grow(1).MinHeight(80)
			case "form":
				a.pairsView(c, &r.Form, "form field")
			case "file":
				ui.TextInput(c, &r.File).Label("Body file path").Placeholder("Path relative to the workspace file").FillWidth()
			default:
				ui.Text(c, "This request has no body. Choose JSON or raw to add one.").TextColor(c.Theme().TextMuted)
			}
		}
	})
}

func (a *app) pairsView(c *ui.Context, pairs *[]workspace.Pair, label string) {
	ui.Button(c, "Add "+label).OnClick(func() { *pairs = append(*pairs, workspace.Pair{}) })
	ui.Scroll(c).FillWidth().Grow(1).MinHeight(60).Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			for i := range *pairs {
				c.Key(fmt.Sprintf("%s-%d", label, i))
				ui.Row(c).FillWidth().Gap(6).Children(func() {
					ui.TextInput(c, &(*pairs)[i].Key).Label(fmt.Sprintf("%s key %d", label, i+1)).Placeholder("Key").Grow(1).MinWidth(0)
					ui.TextInput(c, &(*pairs)[i].Value).Label(fmt.Sprintf("%s value %d", label, i+1)).Placeholder("Value").Grow(2).MinWidth(0)
					ui.Button(c, "Remove").OnClick(func() { *pairs = append((*pairs)[:i], (*pairs)[i+1:]...) })
				})
			}
		})
	})
}

func (a *app) sendButton(c *ui.Context) {
	if a.busy {
		ui.Button(c, "Cancel").OnClick(func() { a.stop(); a.resultLabel = "Canceled" })
	} else {
		ui.Button(c, "Send").Disabled(a.fileBusy).OnClick(a.send)
	}
}
