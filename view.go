package main

import "github.com/egoist/mygo/ui"

var requestMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

func (a *app) view(c *ui.Context) {
	c.SetTheme(appTheme(c.Theme()))
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).FillWidth().Height(58).Padding(0, 20).Gap(12).
			BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, "Sento").FontSize(20).FontWeight(700)
			ui.Text(c, "API client ala-ala").TextColor(t.TextMuted)
			ui.Spacer(c)
			ui.Button(c, "New request").OnClick(a.newRequest)
		})
		ui.Row(c).FillWidth().Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.sidebar(c)
			ui.Column(c).Grow(1).MinWidth(0).Padding(24).Gap(16).Children(func() {
				ui.Text(c, "Untitled request").FontSize(18).FontWeight(600)
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					c.Key("request-method")
					ui.Select(c, &a.draft.method, requestMethods).Label("HTTP method").Width(112)
					c.Key("request-url")
					ui.TextInput(c, &a.draft.url).Label("Request URL").
						Placeholder("https://api.example.com").Grow(1).MinWidth(0)
					ui.Button(c, "Send").Disabled(true).Tooltip("HTTP execution is coming next")
				})
				ui.Text(c, "Body").FontWeight(600)
				c.Key("request-body")
				ui.TextArea(c, &a.draft.body).Label("Request body").
					Placeholder("Optional request body").FillWidth().Height(160)
				ui.Column(c).FillWidth().Grow(1).MinHeight(0).Padding(20).Gap(12).
					Background(t.Surface).BorderWidth(1).BorderColor(t.Border).Radius(8).Children(func() {
					ui.Text(c, "Response").FontWeight(600)
					ui.Column(c).FillWidth().Grow(1).Center().Gap(8).Children(func() {
						ui.Text(c, "No response yet").FontSize(16)
						ui.Text(c, "Request execution will be added in the next iteration.").TextColor(t.TextMuted)
					})
				})
			})
		})
		ui.Row(c).FillWidth().Height(30).Padding(0, 16).
			BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, "Scaffold · Drafts are kept in memory").FontSize(11).TextColor(t.TextMuted)
		})
	})
}

func (a *app) sidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(200).FillHeight().Padding(20).Gap(12).
		Background(t.Surface).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Text(c, "WORKSPACE").FontSize(11).FontWeight(600).TextColor(t.TextMuted)
		ui.Text(c, "Scratch request").FontWeight(600)
		ui.Text(c, "Start with a method, URL, and optional body.").TextColor(t.TextMuted)
		ui.Spacer(c)
		ui.Text(c, "Collections and history\nComing next").FontSize(12).TextColor(t.TextMuted)
	})
}
