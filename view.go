package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/algonacci/sento/internal/workspace"
	"github.com/egoist/mygo/ui"
)

// Layout: a slim title bar, the request list on the left, and the request
// on the right — its line (method, URL, Send) on top, then its editor and
// the response side by side in a wide window, one above the other in a
// narrow one. Rare actions live in the title bar's More menu, the macOS
// menu bar and the keyboard.
func (a *app) view(c *ui.Context) {
	// The system's appearance, before SetTheme replaces it.
	c.SetTheme(appTheme(isDarkMode(a.prefs.Theme, c.Theme().Dark)))
	defer a.navigation(c)
	width, _ := c.Size()
	compact := width < 900
	ui.Column(c).Fill().Background(colBase).Children(func() {
		a.titleBar(c, compact)
		ui.Row(c).FillWidth().Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.sidebar(c, compact)
			ui.Column(c).Grow(1).MinWidth(0).FillHeight().Padding(12).Gap(10).Children(func() {
				switch {
				case a.ai.open:
					a.aiView(c, compact)
				case a.sourceMode:
					a.sourceView(c, width >= 1200)
				default:
					a.requestView(c, width >= 1200)
				}
			})
		})
		a.statusBar(c)
	})
}

// ---------------------------------------------------------------- chrome

func (a *app) titleBar(c *ui.Context, compact bool) {
	bar := c.TitleBar()
	ui.Row(c).Height(max(bar.Height, 44)).FillWidth().Padding(0, bar.Right+8, 0, bar.Left+12).Gap(8).
		Background(colBar).BorderWidth(0, 0, 1, 0).BorderColor(colLineSoft).DragWindow().Children(func() {
		ui.Row(c).Gap(8).Shrink(0).Children(func() {
			ui.Box(c).Size(22, 22).Radius(6).Background(colAccent).Center().Children(func() {
				ui.Icon(c, icSend).FontSize(12).TextColor(white)
			})
			ui.Text(c, "Sento").FontWeight(700).FontSize(14)
		})
		ui.Box(c).Width(1).Height(20).Background(colLine).Margin(0, 4).Shrink(0)
		a.breadcrumb(c, compact)
		ui.Spacer(c)
		if a.sourceMode {
			barButton(c, icCode, "Source", true).Tooltip("Back to the builder (" + primaryKey() + "Shift+E)").OnClick(a.cmdSource)
		}
		if a.ai.open {
			barButton(c, icSparkles, "AI", true).Tooltip("Close the AI generator (" + primaryKey() + "Shift+G)").OnClick(a.cmdAI)
		}
		themeIcon, themeLabel := icMonitor, "System"
		switch a.prefs.Theme {
		case themeLight:
			themeIcon, themeLabel = icSun, "Light"
		case themeDark:
			themeIcon, themeLabel = icMoon, "Dark"
		}
		barButton(c, themeIcon, themeLabel, false).Label("Appearance").Shrink(0).Tooltip("Appearance: System, Light or Dark").OnClick(a.cycleTheme)
		barButton(c, icMore, "More", false).Shrink(0).Menu(a.moreMenu)
		iconButton(c, icKeyboard, "Shortcuts", true, a.cmdShortcuts).Tooltip("Keyboard shortcuts (" + primaryKey() + "/)")
	})
}

// breadcrumb is where the user is: the workspace (its file), and the
// request, whose name is edited here.
func (a *app) breadcrumb(c *ui.Context, compact bool) {
	ui.Row(c).Gap(6).Grow(1).Shrink(1).MinWidth(0).Children(func() {
		name := "Untitled workspace"
		if a.path != "" {
			name = filepath.Base(a.path)
		}
		if !compact || a.sourceMode || a.ai.open {
			ui.Text(c, name).FontSize(12.5).TextColor(colMuted).SingleLine().Shrink(1).MinWidth(0).Tooltip(a.path)
		}
		if _, r := a.doc.FindRequest(a.selected); r != nil && !a.sourceMode && !a.ai.open {
			if !compact {
				ui.Text(c, "/").TextColor(colFaint).Shrink(0)
			}
			c.Key("name-" + r.ID)
			ui.TextInput(c, &r.Name).Label("Request name").Placeholder("Request name").Height(28).
				FontWeight(600).Basis(0).Grow(1).MinWidth(80).MaxWidth(360).Disabled(a.fileBusy)
		}
		if a.unsaved() {
			ui.Text(c, "Edited").FontSize(11).TextColor(colFaint).SingleLine().Shrink(0).Tooltip("Unsaved changes · Save as (" + primaryKey() + "Shift+S)")
		}
	})
}

func (a *app) moreMenu(m *ui.Menu) {
	if m.Item("New request").Shortcut(ui.Cmd, ui.KeyN).Disabled(a.fileBusy).Chosen() {
		a.cmdNewRequest()
	}
	if m.Item("Import .http or JSON…").Shortcut(ui.Cmd, ui.KeyO).Disabled(a.fileBusy).Chosen() {
		a.cmdImport()
	}
	if m.Item("Save as…").Shortcut(ui.Cmd|ui.Shift, ui.KeyS).Disabled(a.fileBusy).Chosen() {
		a.cmdSaveAs()
	}
	m.Separator()
	if m.Item("Edit workspace source").Checked(a.sourceMode).Shortcut(ui.Cmd|ui.Shift, ui.KeyE).Disabled(a.fileBusy).Chosen() {
		a.cmdSource()
	}
	if m.Item("Generate with AI").Checked(a.ai.open).Shortcut(ui.Cmd|ui.Shift, ui.KeyG).Chosen() {
		a.cmdAI()
	}
	m.Separator()
	if m.Item("Keyboard shortcuts").Shortcut(ui.Cmd, ui.KeySlash).Chosen() {
		a.cmdShortcuts()
	}
}

func (a *app) statusBar(c *ui.Context) {
	ui.Row(c).Height(28).FillWidth().Padding(0, 12).Gap(12).Background(colBar).
		BorderWidth(1, 0, 0, 0).BorderColor(colLineSoft).Children(func() {
		msg := a.message
		if msg == "" {
			msg = "Ready"
		}
		ui.Text(c, msg).FontSize(11.5).TextColor(colMuted).SingleLine().Grow(1).Shrink(1).MinWidth(0).Tooltip(a.message)
		if a.path != "" {
			ui.Text(c, a.path).FontSize(11.5).TextColor(colFaint).SingleLine().Shrink(1).MinWidth(0).MaxWidth(360).Tooltip(a.path)
		}
	})
}

// ---------------------------------------------------------------- sidebar

// envProject is the project whose environment applies: the selected
// request's, or the active one.
func (a *app) envProject() *workspace.Project {
	if p, _ := a.doc.FindRequest(a.selected); p != nil {
		return p
	}
	return a.doc.ActiveProject()
}

func (a *app) sidebar(c *ui.Context, compact bool) {
	w := float32(248)
	if compact {
		w = 208
	}
	ui.Column(c).Width(w).Shrink(0).FillHeight().Background(colBar).BorderWidth(0, 1, 0, 0).BorderColor(colLineSoft).Children(func() {
		ui.Column(c).FillWidth().Padding(12, 12, 6, 12).Gap(6).Children(func() {
			sectionLabel(c, "ENVIRONMENT")
			a.envSelect(c)
		})
		ui.Row(c).FillWidth().Padding(8, 8, 4, 12).Gap(6).Children(func() {
			sectionLabel(c, "REQUESTS")
			if n := a.requestCount(); n > 0 {
				ui.Text(c, fmt.Sprint(n)).FontSize(10.5).TextColor(colFaint)
			}
			ui.Spacer(c)
			iconButton(c, icPlus, "New request", !a.fileBusy, a.cmdNewRequest).Tooltip("New request (" + primaryKey() + "N)")
		})
		ui.Box(c).FillWidth().Padding(0, 12, 6, 12).Children(func() {
			ui.SearchField(c, &a.filter).Label("Filter requests").Placeholder("Filter").Bind(&a.filterField).FillWidth()
		})
		ui.Scroll(c).FillWidth().Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().Padding(0, 6, 8, 6).Gap(1).Children(func() { a.requestList(c) })
		})
	})
}

func (a *app) envSelect(c *ui.Context) {
	p := a.envProject()
	if p == nil || len(p.Environments) == 0 {
		none := "No environment"
		ui.Select(c, &none, []string{none}).Label("Environment").Disabled(true).FillWidth().
			Tooltip("Environments are defined in workspace JSON (More › Edit workspace source)")
		return
	}
	names := []string{"No environment"}
	a.envPick = names[0]
	for _, env := range p.Environments {
		names = append(names, env.Name)
		if env.ID == p.ActiveEnv {
			a.envPick = env.Name
		}
	}
	if ui.Select(c, &a.envPick, names).Label("Environment").FillWidth().Disabled(a.fileBusy).Changed() {
		p.ActiveEnv = ""
		for _, env := range p.Environments {
			if env.Name == a.envPick {
				p.ActiveEnv = env.ID
			}
		}
	}
}

func (a *app) requestCount() int {
	n := 0
	for _, p := range a.doc.Projects {
		n += p.CountRequests()
	}
	return n
}

// matches is the sidebar filter: the name, method or URL holds the text.
func matches(r workspace.Request, filter string) bool {
	f := strings.ToLower(strings.TrimSpace(filter))
	return f == "" || strings.Contains(strings.ToLower(r.Name), f) ||
		strings.Contains(strings.ToLower(r.Method), f) || strings.Contains(strings.ToLower(r.URL), f)
}

func (a *app) requestList(c *ui.Context) {
	folders := 0
	for _, p := range a.doc.Projects {
		folders += len(p.Folders)
	}
	shown := 0
	for _, p := range a.doc.Projects {
		if len(a.doc.Projects) > 1 {
			c.Key("project-" + p.ID)
			ui.Text(c, p.Name).FontSize(12).FontWeight(600).Padding(8, 6, 2, 6).SingleLine()
		}
		for _, f := range p.Folders {
			heading := folders > 1
			for _, r := range f.Requests {
				if !matches(r, a.filter) {
					continue
				}
				if heading {
					c.Key("folder-" + p.ID + "-" + f.ID)
					sectionLabel(c, strings.ToUpper(f.Name)).Padding(8, 8, 2, 8).SingleLine()
					heading = false
				}
				shown++
				a.requestRow(c, r)
			}
		}
	}
	switch {
	case a.requestCount() == 0:
		ui.Text(c, "No requests yet. Press + to create one, or import a .http file from More.").FontSize(12).TextColor(colMuted).Wrap().Padding(6, 8)
	case shown == 0:
		ui.Text(c, "No requests match “"+a.filter+"”.").FontSize(12).TextColor(colMuted).Wrap().Padding(6, 8)
	}
}

func (a *app) requestRow(c *ui.Context, r workspace.Request) {
	c.Key("request-" + r.ID)
	on := a.selected == r.ID
	name := r.Name
	if strings.TrimSpace(name) == "" {
		name = "Untitled request"
	}
	tip := r.URL
	if tip == "" {
		tip = "No URL yet"
	}
	b := ui.ButtonBase(c).FillWidth().Height(30).Padding(0, 8).Gap(8).Radius(7).Justify(ui.Start).
		Label(r.Method + " " + name).Tooltip(tip).Transition(hoverFade)
	switch {
	case on:
		b.Background(colAccent.Alpha(0.18)).TextColor(colText)
	case b.Hovered():
		b.Background(colHover).TextColor(colText)
	default:
		b.TextColor(colMuted)
	}
	id := r.ID
	b.OnClick(func() {
		if a.selected != id {
			a.selectRequest(id)
		}
	})
	b.Children(func() {
		methodBadge(c, r.Method, 34)
		ui.Text(c, name).FontSize(12.5).SingleLine().Grow(1).Shrink(1).MinWidth(0)
		if on && a.busy {
			ui.Spinner(c).Size(11, 11).Label("Sending")
		}
	})
}

// ---------------------------------------------------------------- main

func (a *app) requestView(c *ui.Context, wide bool) {
	_, r := a.doc.FindRequest(a.selected)
	if r == nil {
		card(c).Fill().Children(func() {
			emptyState(c, icInbox, "No request selected", "Pick a request on the left, create a new one, or import a .http file.", func() {
				primaryButton(c, icPlus, "New request", a.fileBusy, a.cmdNewRequest)
				ghostButton(c, "Import…", a.fileBusy, a.cmdImport)
			})
		})
		return
	}
	c.Key("line-" + r.ID)
	a.requestLine(c, r)
	a.panes(c, wide, func() { a.requestEditor(c, r) }, func() { a.responseView(c) })
}

// panes puts the request and the response side by side in a wide window
// and one above the other in a narrow one; the divider between them moves.
func (a *app) panes(c *ui.Context, wide bool, first, second func()) {
	if wide {
		ui.SplitPercent(c, &a.sideSplit,
			func() { ui.Column(c).Fill().Padding(0, 5, 0, 0).Children(first) },
			func() { ui.Column(c).Fill().Padding(0, 0, 0, 5).Children(second) }).FillWidth().Grow(1).MinHeight(0)
		return
	}
	ui.SplitVerticalPercent(c, &a.stackSplit,
		func() { ui.Column(c).Fill().Padding(0, 0, 5, 0).Children(first) },
		func() { ui.Column(c).Fill().Padding(5, 0, 0, 0).Children(second) }).FillWidth().Grow(1).MinHeight(0)
}

func (a *app) sourceView(c *ui.Context, wide bool) {
	a.panes(c, wide, func() {
		card(c).Fill().Children(func() {
			ui.Row(c).FillWidth().Height(40).Padding(0, 8, 0, 12).Gap(8).BorderWidth(0, 0, 1, 0).BorderColor(colLineSoft).Children(func() {
				ui.Text(c, "Workspace source").FontWeight(600).FontSize(12.5).SingleLine().Shrink(1).MinWidth(0)
				ui.Text(c, strings.TrimPrefix(a.format, ".")).FontSize(11).TextColor(colFaint).SingleLine()
				ui.Spacer(c)
				ghostButton(c, "Validate", a.fileBusy, func() { a.parse() }).Tooltip("Validate and refresh the request list")
				ghostButton(c, "Back to builder", a.fileBusy, a.cmdSource)
				a.sendButton(c)
			})
			c.Key("source")
			ui.TextArea(c, &a.source).Label("Request source").ReadOnly(a.fileBusy).Font("monospace").
				FillWidth().Grow(1).MinHeight(120).Radius(0).BorderWidth(0, 0, 0, 0).Background(colPane)
		})
	}, func() { a.responseView(c) })
}

// ---------------------------------------------------------------- response

func (a *app) responseView(c *ui.Context) {
	card(c).Fill().Children(func() {
		ui.Row(c).FillWidth().Height(40).Padding(0, 12, 0, 8).Gap(8).BorderWidth(0, 0, 1, 0).BorderColor(colLineSoft).Children(func() {
			done := !a.busy && a.resultLabel != "" && a.respErr == "" && a.respCode != 0
			c.Key("rtab-body")
			tab(c, "Response body tab", "Body", "", a.responseTab != "Headers", func() { a.responseTab = "Body" })
			c.Key("rtab-headers")
			tab(c, "Response headers tab", "Headers", count(len(a.respHeaders)), a.responseTab == "Headers", func() { a.responseTab = "Headers" })
			ui.Spacer(c)
			if done {
				ui.Row(c).Gap(8).Shrink(1).MinWidth(0).Label(a.resultLabel).Children(func() {
					pill(c, fmt.Sprint(a.respCode), statusTone(a.respCode)).Tooltip(a.resultLabel)
					ui.Text(c, a.respTime).FontSize(12).TextColor(colMuted).SingleLine().Shrink(0)
					ui.Text(c, a.respSize).FontSize(12).TextColor(colMuted).SingleLine().Shrink(0)
				})
			}
		})
		body := ui.Column(c).FillWidth().Grow(1).MinHeight(0).Gap(8)
		if a.showsBody() {
			// The body text fills the card, edge to edge, as an editor.
			body.Padding(0)
		} else {
			body.Padding(12)
		}
		body.Children(func() { a.responseBody(c) })
	})
}

// showsBody reports whether the response card shows a body's text.
func (a *app) showsBody() bool {
	return !a.busy && a.resultLabel != "" && a.respErr == "" && a.respCode != 0 && a.responseTab != "Headers" && a.response != ""
}

func (a *app) responseBody(c *ui.Context) {
	switch {
	case a.busy:
		ui.Column(c).Grow(1).FillWidth().Center().Gap(10).Children(func() {
			ui.Spinner(c).Size(20, 20).Label("Sending")
			ui.Text(c, a.resultLabel).FontSize(12.5).TextColor(colMuted).SingleLine()
			ghostButton(c, "Cancel request", false, a.cmdCancel).Tooltip(primaryKey() + ".")
		})
	case a.resultLabel == "":
		body := "Press Send, or Enter in the URL, to see the response here."
		if a.sourceMode {
			body = "Press Send to run the selected request from the source; the response shows here."
		} else if _, r := a.doc.FindRequest(a.selected); r == nil || strings.TrimSpace(r.URL) == "" {
			body = "Enter a URL above, then press Send. The status, time, size, body and headers show here."
		}
		emptyState(c, icSend, "Not sent yet", body, nil)
	case a.respErr != "":
		banner(c, a.respErr, colDanger)
		ui.Text(c, "No response: the call did not reach a server or was refused. Check the URL, the environment and your network.").FontSize(12).TextColor(colMuted).Wrap()
	case a.respCode == 0:
		emptyState(c, icBan, "Request canceled", "Nothing came back. Send it again when you are ready.", nil)
	case a.responseTab == "Headers":
		a.headersView(c)
	default:
		if a.response == "" {
			ui.Text(c, "The response has no body.").FontSize(12).TextColor(colMuted)
			return
		}
		if a.respTruncated {
			ui.Box(c).FillWidth().Padding(10, 12, 0, 12).Children(func() {
				banner(c, fmt.Sprintf("Preview limited to the first 256 KiB of %s. Run it with the CLI (sento run) for the full response.", a.respSize), colWarn)
			})
		}
		c.Key("response-body")
		ui.TextArea(c, &a.response).Label("Response body").ReadOnly(true).Font("monospace").
			FillWidth().Grow(1).MinHeight(60).Radius(0).BorderWidth(0, 0, 0, 0).Background(colPane)
	}
}

func (a *app) headersView(c *ui.Context) {
	if len(a.respHeaders) == 0 {
		ui.Text(c, "The response has no headers.").FontSize(12).TextColor(colMuted)
		return
	}
	ui.Scroll(c).FillWidth().Grow(1).MinHeight(0).Label("Response headers").Children(func() {
		ui.Column(c).FillWidth().Gap(0).Children(func() {
			for i, h := range a.respHeaders {
				c.Key(fmt.Sprint("header-", i))
				ui.Row(c).FillWidth().Gap(12).Padding(5, 4).AlignItems(ui.Start).BorderWidth(0, 0, 1, 0).BorderColor(colLineSoft).Children(func() {
					ui.Text(c, h.Key).FontSize(12).FontWeight(600).TextColor(colMuted).Selectable().Wrap().Basis(0).Grow(2).MinWidth(0)
					ui.Text(c, h.Value).FontSize(12).Font("monospace").Selectable().Wrap().Basis(0).Grow(5).MinWidth(0)
				})
			}
		})
	})
}

// ---------------------------------------------------------------- AI

func (a *app) aiView(c *ui.Context, compact bool) {
	s := &a.ai
	card(c).Fill().Children(func() {
		ui.Row(c).FillWidth().Height(44).Padding(0, 8, 0, 14).Gap(8).BorderWidth(0, 0, 1, 0).BorderColor(colLineSoft).Children(func() {
			ui.Icon(c, icSparkles).FontSize(15).TextColor(colAccent)
			ui.Text(c, "Generate requests with AI").FontWeight(600).SingleLine().Grow(1).Shrink(1).MinWidth(0)
			iconButton(c, icX, "Close AI", true, a.cmdAI)
		})
		ui.Column(c).FillWidth().Grow(1).MinHeight(0).Padding(14).Gap(10).Children(func() {
			ui.Text(c, "Describe an API or paste its documentation. Sento sends this text to your provider and shows the .http it returns for review; nothing is sent to the API until you press Send.").
				FontSize(12).TextColor(colMuted).Wrap()
			provider := func(grow bool) {
				size := func(e ui.Element, n float32) {
					if grow {
						e.Basis(0).Grow(n).MinWidth(0)
					} else {
						e.FillWidth()
					}
				}
				size(ui.TextInput(c, &s.baseURL).Label("AI base URL").Placeholder("Base URL, e.g. https://api.openai.com/v1"), 2)
				size(ui.TextInput(c, &s.model).Label("AI model").Placeholder("Model"), 1)
				size(ui.TextInput(c, &s.key).Label("AI API key").Placeholder("API key (this session only)").Password(), 1)
			}
			if compact {
				ui.Column(c).FillWidth().Gap(8).Shrink(0).Children(func() { provider(false) })
			} else {
				ui.Row(c).FillWidth().Gap(8).Children(func() { provider(true) })
			}
			c.Key("ai-prompt")
			prompt := ui.TextArea(c, &s.prompt).Label("API description").Placeholder("Create requests for…").FillWidth().Height(120).Shrink(0)
			if compact {
				prompt.Height(72)
			}
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				if s.busy {
					ghostButton(c, "Stop", false, func() { a.stopAI(); s.message = "Canceled" })
				} else {
					primaryButton(c, icSparkles, "Generate", false, a.generate)
				}
				ghostButton(c, "Open generated requests", s.answer == "" || a.fileBusy, a.useGenerated)
				if s.busy {
					ui.Spinner(c).Size(14, 14)
				}
				ui.Text(c, s.message).FontSize(12).TextColor(colMuted).SingleLine().Shrink(1).MinWidth(0).Tooltip(s.message)
			})
			c.Key("ai-result")
			ui.TextArea(c, &s.answer).Label("Generated HTTP source").Placeholder("The generated .http shows here").ReadOnly(true).Font("monospace").FillWidth().Grow(1).MinHeight(80)
		})
	})
}
