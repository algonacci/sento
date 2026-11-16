package main

import "github.com/egoist/mygo/ui"

func (a *app) view(c *ui.Context) {
	c.SetTheme(appTheme(c.Theme()))
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).FillWidth().Height(58).Padding(0, 20).Gap(10).
			BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, "Sento").FontSize(20).FontWeight(700)
			ui.Text(c, "API client").TextColor(t.TextMuted)
			ui.Spacer(c)
			ui.Button(c, "New request").Disabled(a.fileBusy).OnClick(a.newRequest)
			ui.Button(c, "Import").Disabled(a.fileBusy).OnClick(a.openFile)
			ui.Button(c, "Save as").Disabled(a.fileBusy).OnClick(func() { a.saveFile(false) })
			label := "Source"
			if a.sourceMode {
				label = "Builder"
			}
			ui.Button(c, label).Disabled(a.fileBusy).OnClick(a.toggleSource)
			ui.Button(c, "AI").OnClick(func() { a.ai.open = !a.ai.open })
		})
		if a.ai.open {
			a.aiView(c)
		} else {
			ui.Row(c).FillWidth().Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
				a.sidebar(c)
				ui.Column(c).Grow(1).MinWidth(0).Padding(16).Gap(10).Children(func() {
					if a.sourceMode {
						ui.Row(c).FillWidth().Gap(10).Children(func() {
							ui.Text(c, "Source "+a.format).FontWeight(600)
							ui.Spacer(c)
							ui.Button(c, "Validate / refresh").Disabled(a.fileBusy).OnClick(func() { a.parse() })
							a.sendButton(c)
						})
						c.Key("source")
						ui.TextArea(c, &a.source).Label("Request source").ReadOnly(a.fileBusy).FillWidth().Grow(1).MinHeight(140)
					} else {
						a.builderView(c)
					}
					label := a.resultLabel
					if label == "" {
						label = "Response · select a request and press Send"
					}
					ui.Text(c, label).FontWeight(600).SingleLine()
					ui.Row(c).FillWidth().Grow(1).MinHeight(100).Gap(10).Children(func() {
						c.Key("response-body")
						ui.TextArea(c, &a.response).Label("Response body").Placeholder("Response body").ReadOnly(true).Grow(2).MinWidth(0).FillHeight()
						c.Key("response-headers")
						ui.TextArea(c, &a.responseHeaders).Label("Response headers").Placeholder("Response headers").ReadOnly(true).Grow(1).MinWidth(0).FillHeight()
					})
				})
			})
		}
		ui.Row(c).FillWidth().Height(32).Padding(0, 16).
			BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, a.message).FontSize(11).TextColor(t.TextMuted).SingleLine().Tooltip(a.message)
		})
	})
}

func (a *app) sidebar(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Width(210).FillHeight().Background(t.Surface).Children(func() {
		ui.Column(c).FillWidth().Padding(14).Gap(8).Children(func() {
			ui.Text(c, "REQUESTS").FontSize(11).FontWeight(600).TextColor(t.TextMuted)
			if a.doc == nil {
				return
			}
			for _, p := range a.doc.Projects {
				ui.Text(c, p.Name).FontWeight(600)
				for _, f := range p.Folders {
					ui.Text(c, f.Name).TextColor(t.TextMuted).FontSize(11)
					for _, r := range f.Requests {
						c.Key(r.ID)
						b := ui.Button(c, r.Method+" "+r.Name).FillWidth().Tooltip(r.URL).
							OnClick(func() { a.selectRequest(r.ID) })
						if a.selected == r.ID {
							b.Background(t.Accent).TextColor(ui.Hex("#ffffff"))
						}
					}
				}
			}
		})
	})
}

func (a *app) aiView(c *ui.Context) {
	s := &a.ai
	ui.Column(c).FillWidth().Grow(1).MinHeight(0).Padding(20).Gap(10).Children(func() {
		ui.Text(c, "Generate .http with AI").FontSize(18).FontWeight(600)
		ui.Text(c, "Describe an API or paste its documentation. The description below is sent to your configured provider.").TextColor(c.Theme().TextMuted)
		ui.Row(c).FillWidth().Gap(10).Children(func() {
			ui.TextInput(c, &s.baseURL).Label("AI base URL").Placeholder("https://api.openai.com/v1").Grow(2)
			ui.TextInput(c, &s.model).Label("AI model").Placeholder("Model").Grow(1)
			ui.TextInput(c, &s.key).Label("AI API key").Placeholder("API key (session only)").Password().Grow(1)
		})
		c.Key("ai-prompt")
		ui.TextArea(c, &s.prompt).Label("API description").Placeholder("Create requests for…").FillWidth().Height(140)
		ui.Row(c).FillWidth().Gap(10).Children(func() {
			if s.busy {
				ui.Button(c, "Stop").OnClick(func() { a.stopAI(); s.message = "Canceled" })
			} else {
				ui.Button(c, "Generate").OnClick(a.generate)
			}
			ui.Button(c, "Open generated requests").Disabled(s.answer == "" || a.fileBusy).OnClick(a.useGenerated)
			ui.Text(c, s.message).SingleLine().Tooltip(s.message)
		})
		c.Key("ai-result")
		ui.TextArea(c, &s.answer).Label("Generated HTTP source").ReadOnly(true).FillWidth().Grow(1).MinHeight(100)
	})
}
