package main

import (
	"fmt"
	"strings"

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
	a.clearResponse()
}

var methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

// requestLine is the one line a request is: method, URL and Send. Enter in
// the URL sends, as Cmd/Ctrl+Enter does anywhere.
func (a *app) requestLine(c *ui.Context, r *workspace.Request) {
	ui.Row(c).FillWidth().Gap(8).Disabled(a.fileBusy).Children(func() {
		ui.Select(c, &r.Method, methods).Label("HTTP method").Width(112).Height(34).Shrink(0).
			TextColor(methodTone(r.Method)).FontWeight(700)
		url := ui.TextInput(c, &r.URL).Label("Request URL").Placeholder("https://api.example.com/users  ·  {{baseUrl}}/users").
			Bind(&a.urlField).Height(34).Grow(1).MinWidth(0)
		if url.Submitted() && !a.busy {
			a.send()
		}
		a.sendButton(c)
	})
}

// requestEditor holds the parts of a request besides its line: params,
// headers, body and auth, one tab at a time.
func (a *app) requestEditor(c *ui.Context, r *workspace.Request) {
	card(c).Fill().Disabled(a.fileBusy).Children(func() {
		ui.Row(c).FillWidth().Padding(0, 8).BorderWidth(0, 0, 1, 0).BorderColor(colLineSoft).Children(func() {
			for _, t := range []struct{ name, note string }{
				{"Params", count(len(r.Params))},
				{"Headers", count(len(r.Headers))},
				{"Body", bodyNote(r.Body)},
				{"Auth", authNote(r.Auth.Type)},
			} {
				c.Key("tab-" + t.name)
				tab(c, t.name, t.name, t.note, a.requestTab == t.name, func() { a.requestTab = t.name })
			}
		})
		c.Key("editor-" + r.ID + "-" + a.requestTab)
		ui.Column(c).FillWidth().Grow(1).MinHeight(0).Padding(12).Gap(10).Children(func() {
			switch a.requestTab {
			case "Headers":
				a.pairsView(c, &r.Headers, "header", "No headers. Auth and a JSON body add their own when sent.")
			case "Body":
				a.bodyView(c, r)
			case "Auth":
				a.authView(c, r)
			default:
				a.pairsView(c, &r.Params, "parameter", "No query parameters. They are added to the URL as ?key=value.")
			}
		})
	})
}

func count(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprint(n)
}

func bodyNote(b workspace.BodyType) string {
	if b == "" || b == workspace.BodyNone {
		return ""
	}
	return strings.ToUpper(string(b))
}

func authNote(t workspace.AuthType) string {
	switch t {
	case "", workspace.AuthInherit, workspace.AuthNone:
		return ""
	case workspace.AuthKey:
		return "API key"
	}
	return strings.ToUpper(string(t[:1])) + string(t[1:])
}

func (a *app) bodyView(c *ui.Context, r *workspace.Request) {
	kind := r.Body
	if kind == "" {
		kind = workspace.BodyNone
	}
	ui.Row(c).Gap(4).Label("Body type").Children(func() {
		for _, k := range []struct {
			t     workspace.BodyType
			label string
		}{{workspace.BodyNone, "None"}, {workspace.BodyJSON, "JSON"}, {workspace.BodyRaw, "Raw"}, {workspace.BodyForm, "Form"}, {workspace.BodyFile, "File"}} {
			chip(c, k.label, kind == k.t, func() { r.Body = k.t })
		}
	})
	switch kind {
	case workspace.BodyJSON:
		ui.TextArea(c, &r.JSON).Label("Request body").Placeholder("{\"name\": \"Ada\"}").Font("monospace").FillWidth().Grow(1).MinHeight(80)
	case workspace.BodyRaw:
		ui.TextArea(c, &r.Raw).Label("Request body").Placeholder("Request body").Font("monospace").FillWidth().Grow(1).MinHeight(80)
	case workspace.BodyForm:
		a.pairsView(c, &r.Form, "form field", "No form fields. They are sent as multipart/form-data.")
	case workspace.BodyFile:
		ui.TextInput(c, &r.File).Label("Body file path").Placeholder("Path relative to the workspace file").FillWidth()
	default:
		ui.Text(c, "This request sends no body. Choose JSON, Raw, Form or File to add one.").FontSize(12).TextColor(colMuted).Wrap()
	}
}

func (a *app) authView(c *ui.Context, r *workspace.Request) {
	kind := r.Auth.Type
	if kind == "" {
		kind = workspace.AuthInherit
	}
	ui.Row(c).Gap(4).Label("Auth type").Children(func() {
		for _, k := range []struct {
			t     workspace.AuthType
			label string
		}{{workspace.AuthInherit, "Inherit"}, {workspace.AuthNone, "None"}, {workspace.AuthBearer, "Bearer"}, {workspace.AuthBasic, "Basic"}, {workspace.AuthKey, "API key"}} {
			chip(c, k.label, kind == k.t, func() { r.Auth.Type = k.t })
		}
	})
	field := func(value *string, label, placeholder string, secret bool) {
		ui.Column(c).FillWidth().Gap(4).Children(func() {
			ui.Text(c, label).FontSize(11.5).TextColor(colMuted)
			in := ui.TextInput(c, value).Label(label).Placeholder(placeholder).FillWidth()
			if secret {
				in.Password()
			}
		})
	}
	switch kind {
	case workspace.AuthBearer:
		field(&r.Auth.Token, "Bearer token", "Token or {{env.API_TOKEN}}", true)
	case workspace.AuthBasic:
		field(&r.Auth.Username, "Username", "Username", false)
		field(&r.Auth.Password, "Password", "Password or {{env.API_PASSWORD}}", true)
	case workspace.AuthKey:
		field(&r.Auth.Header, "API key header", "X-API-Key", false)
		field(&r.Auth.Value, "API key value", "Value or {{env.API_KEY}}", true)
	case workspace.AuthNone:
		ui.Text(c, "No credentials are sent, even if the workspace sets some.").FontSize(12).TextColor(colMuted).Wrap()
	default:
		ui.Text(c, "Uses the workspace's auth, if it sets any.").FontSize(12).TextColor(colMuted).Wrap()
	}
	ui.Text(c, "Write {{env.NAME}} to read a secret from the environment instead of saving it.").FontSize(11.5).TextColor(colFaint).Wrap()
}

func (a *app) pairsView(c *ui.Context, pairs *[]workspace.Pair, label, empty string) {
	add := func() {
		ui.Row(c).Children(func() {
			ghostButton(c, "Add "+label, false, func() { *pairs = append(*pairs, workspace.Pair{}) })
		})
	}
	if len(*pairs) == 0 {
		ui.Text(c, empty).FontSize(12).TextColor(colMuted).Wrap()
		add()
		return
	}
	ui.Scroll(c).FillWidth().Grow(1).MinHeight(40).Children(func() {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			for i := range *pairs {
				c.Key(fmt.Sprintf("%s-%d", label, i))
				ui.Row(c).FillWidth().Gap(6).Children(func() {
					ui.TextInput(c, &(*pairs)[i].Key).Label(fmt.Sprintf("%s key %d", label, i+1)).Placeholder("Key").Basis(0).Grow(2).MinWidth(0)
					ui.TextInput(c, &(*pairs)[i].Value).Label(fmt.Sprintf("%s value %d", label, i+1)).Placeholder("Value").Basis(0).Grow(3).MinWidth(0)
					iconButton(c, icX, fmt.Sprintf("Remove %s %d", label, i+1), true, func() {
						*pairs = append((*pairs)[:i], (*pairs)[i+1:]...)
					}).Shrink(0)
				})
			}
			// Right under the rows, where the eye already is.
			ui.Box(c).Margin(2, 0, 0, 0).Children(add)
		})
	})
}

func (a *app) sendButton(c *ui.Context) {
	if a.busy {
		b := ui.ButtonBase(c).Height(34).Padding(0, 16).Radius(7).Justify(ui.Center).Label("Cancel").
			Border(1, colDanger.Alpha(0.5)).TextColor(colDanger).Transition(hoverFade).Shrink(0)
		if b.Hovered() {
			b.Background(colDanger.Alpha(0.1))
		}
		b.OnClick(a.cancelSend)
		b.Children(func() { ui.Text(c, "Cancel").FontSize(12.5).FontWeight(600).SingleLine() })
		return
	}
	primaryButton(c, icSend, "Send", a.fileBusy, a.send).Tooltip("Send (Enter in the URL, or " + primaryKey() + "Enter)")
}
