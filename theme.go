package main

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Palette, after Nimbo's: a near-black charcoal (or soft greys in light
// mode) with one blue accent. Verbs and status classes get their own quiet
// tones; red is kept for DELETE, 5xx and failures. The col* variables are
// swapped by usePalette at the start of each frame, on the UI thread.
var (
	colBase, colBar, colPane, colHover, colPress ui.Color
	colField, colLine, colLineSoft, colText      ui.Color
	colMuted, colFaint, colAccent, colAccentHov  ui.Color
	colAccentPrs, colDanger, colWarn, colOK      ui.Color
	colInfo, colPurple, colTeal                  ui.Color
	isDark                                       bool

	white = ui.Hex("#ffffff")
	black = ui.Hex("#000000")
)

type palette struct {
	dark                                 bool
	base, bar, pane, hover, press, field string
	line, lineSoft, text, muted, faint   string
	accent, accentHov, accentPrs         string
	danger, warn, ok, info, purple, teal string
}

var darkPalette = palette{
	dark: true,
	base: "#0c0e14", bar: "#10131b", pane: "#13161f", hover: "#1d2230", press: "#242a3a", field: "#0f121a",
	line: "#262b3b", lineSoft: "#1c2030", text: "#e8eaf2", muted: "#8b92a8", faint: "#5c6378",
	accent: "#4f9cf5", accentHov: "#6aadf7", accentPrs: "#3d88e0",
	danger: "#f0475e", warn: "#f5b454", ok: "#4ade9b", info: "#60a5fa", purple: "#b18cf5", teal: "#5cc8c8",
}

var lightPalette = palette{
	base: "#f3f4f7", bar: "#eaecf1", pane: "#ffffff", hover: "#e9ecf2", press: "#dde1e9", field: "#ffffff",
	line: "#d5d9e1", lineSoft: "#e3e6ec", text: "#1b1f2a", muted: "#566074", faint: "#8a92a4",
	accent: "#2f7cf0", accentHov: "#4a8ff3", accentPrs: "#1f68d6",
	danger: "#d42f46", warn: "#b86e0a", ok: "#17905a", info: "#2f6fd6", purple: "#7a4fd6", teal: "#178a8a",
}

func usePalette(p palette) {
	h := ui.Hex
	isDark = p.dark
	colBase, colBar, colPane, colHover, colPress = h(p.base), h(p.bar), h(p.pane), h(p.hover), h(p.press)
	colField, colLine, colLineSoft, colText = h(p.field), h(p.line), h(p.lineSoft), h(p.text)
	colMuted, colFaint, colAccent, colAccentHov, colAccentPrs = h(p.muted), h(p.faint), h(p.accent), h(p.accentHov), h(p.accentPrs)
	colDanger, colWarn, colOK, colInfo, colPurple, colTeal = h(p.danger), h(p.warn), h(p.ok), h(p.info), h(p.purple), h(p.teal)
}

func init() { usePalette(darkPalette) }

// Theme modes, in the order the title bar button cycles through them.
const (
	themeSystem = "system"
	themeLight  = "light"
	themeDark   = "dark"
)

func nextTheme(mode string) string {
	switch mode {
	case themeLight:
		return themeDark
	case themeDark:
		return themeSystem
	}
	return themeLight
}

// isDarkMode resolves a mode against the system's appearance.
func isDarkMode(mode string, systemDark bool) bool {
	switch mode {
	case themeLight:
		return false
	case themeDark:
		return true
	}
	return systemDark
}

// appTheme is Sento's palette, light or dark, as a MyGo theme. It builds a
// new theme, so the shared defaults are never mutated.
func appTheme(dark bool) *ui.Theme {
	if dark {
		usePalette(darkPalette)
	} else {
		usePalette(lightPalette)
	}
	t := *ui.LightTheme()
	if isDark {
		t = *ui.DarkTheme()
	}
	t.Background = colBase
	t.Surface = colField
	t.SurfaceHover = colHover
	t.SurfacePressed = colPress
	t.Border = colLine
	t.Text = colText
	t.TextMuted = colMuted
	t.Accent = colAccent
	t.AccentHover = colAccentHov
	t.AccentPressed = colAccentPrs
	t.AccentText = white
	t.Danger = colDanger
	t.Warning = colWarn
	t.Success = colOK
	t.Selection = colAccent.Alpha(0.3)
	t.Focus = colAccent.Alpha(0.75)
	t.Scrollbar = ui.RGBA(255, 255, 255, 0.14)
	if !isDark {
		t.Scrollbar = ui.RGBA(0, 0, 0, 0.18)
	}
	t.Radius = 7
	t.FontSize = 13
	return &t
}

// methodTone colours a verb as most API clients do: reads green, creates
// amber, updates blue or purple, deletes red.
func methodTone(method string) ui.Color {
	switch strings.ToUpper(method) {
	case "GET":
		return colOK
	case "POST":
		return colWarn
	case "PUT":
		return colInfo
	case "PATCH":
		return colPurple
	case "DELETE":
		return colDanger
	case "HEAD", "OPTIONS":
		return colTeal
	}
	return colMuted
}

// statusTone colours a status code by its class; 0 is a failed call.
func statusTone(code int) ui.Color {
	switch {
	case code >= 200 && code < 300:
		return colOK
	case code >= 300 && code < 400:
		return colInfo
	case code >= 400 && code < 500:
		return colWarn
	}
	return colDanger
}

// methodShort keeps the badge column narrow: OPTIONS and DELETE are long.
func methodShort(method string) string {
	switch m := strings.ToUpper(method); m {
	case "OPTIONS":
		return "OPT"
	case "DELETE":
		return "DEL"
	default:
		return m
	}
}

// onTint is readable text in tone over a faint tone background.
func onTint(tone ui.Color) ui.Color {
	if isDark {
		return tone.Mix(white, 0.35)
	}
	return tone.Mix(black, 0.2)
}

var hoverFade = ui.ElementTransition{Colors: true, Duration: 120 * time.Millisecond}

// Icons, after Lucide (ISC).
func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	icSend     = icon(`<path d="M14.54 21.69a.5.5 0 0 0 .94-.03l6.5-19a.5.5 0 0 0-.64-.64l-19 6.5a.5.5 0 0 0-.03.94l7.93 3.18a2 2 0 0 1 1.11 1.11z"/><path d="m21.85 2.15-10.94 10.93"/>`)
	icPlus     = icon(`<path d="M5 12h14"/><path d="M12 5v14"/>`)
	icX        = icon(`<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`)
	icMore     = icon(`<circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/>`)
	icKeyboard = icon(`<path d="M10 8h.01"/><path d="M12 12h.01"/><path d="M14 8h.01"/><path d="M16 12h.01"/><path d="M18 8h.01"/><path d="M6 8h.01"/><path d="M7 16h10"/><path d="M8 12h.01"/><rect width="20" height="16" x="2" y="4" rx="2"/>`)
	icAlert    = icon(`<circle cx="12" cy="12" r="10"/><path d="M12 8v4"/><path d="M12 16h.01"/>`)
	icCode     = icon(`<path d="m16 18 6-6-6-6"/><path d="m8 6-6 6 6 6"/>`)
	icSparkles = icon(`<path d="M9.94 15.5A2 2 0 0 0 8.5 14.06l-6.14-1.58a.5.5 0 0 1 0-.96L8.5 9.94A2 2 0 0 0 9.94 8.5l1.58-6.14a.5.5 0 0 1 .96 0L14.06 8.5A2 2 0 0 0 15.5 9.94l6.14 1.58a.5.5 0 0 1 0 .96L15.5 14.06a2 2 0 0 0-1.44 1.44l-1.58 6.14a.5.5 0 0 1-.96 0z"/>`)
	icInbox    = icon(`<polyline points="22 12 16 12 14 15 10 15 8 12 2 12"/><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/>`)
	icFile     = icon(`<path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/>`)
	icSun      = icon(`<circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/>`)
	icMoon     = icon(`<path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/>`)
	icMonitor  = icon(`<rect width="20" height="14" x="2" y="3" rx="2"/><path d="M8 21h8"/><path d="M12 17v4"/>`)
	icBan      = icon(`<circle cx="12" cy="12" r="10"/><path d="m4.9 4.9 14.2 14.2"/>`)
)

// ---------------------------------------------------------------- pieces

func sectionLabel(c *ui.Context, text string) ui.Element {
	return ui.Text(c, text).FontSize(10.5).FontWeight(650).TextColor(colFaint).LetterSpacing(0.6)
}

// methodBadge is the verb in its tone, in a fixed-width column so names
// line up.
func methodBadge(c *ui.Context, method string, width float32) {
	ui.Text(c, methodShort(method)).FontSize(10.5).FontWeight(700).TextColor(methodTone(method)).
		Font("monospace").SingleLine().Width(width).Shrink(0)
}

func iconButton(c *ui.Context, ic *ui.SVG, label string, enabled bool, fn func()) ui.Element {
	b := ui.ButtonBase(c).Size(28, 28).Radius(7).Center().Label(label).Tooltip(label).Disabled(!enabled).Transition(hoverFade)
	b.TextColor(colMuted)
	switch {
	case !enabled:
		b.TextColor(colFaint).Opacity(0.5)
	case b.Pressed():
		b.Background(colPress).TextColor(colText)
	case b.Hovered():
		b.Background(colHover).TextColor(colText)
	}
	b.OnClick(fn)
	b.Children(func() { ui.Icon(c, ic).FontSize(15) })
	return b
}

// barButton is a title-bar button: an icon and a short label.
func barButton(c *ui.Context, ic *ui.SVG, label string, on bool) ui.Element {
	b := ui.ButtonBase(c).Height(30).Padding(0, 10).Gap(6).Radius(7).Label(label).Transition(hoverFade)
	b.TextColor(colMuted)
	switch {
	case on:
		b.Background(colAccent.Alpha(0.18)).TextColor(colText)
	case b.Pressed():
		b.Background(colPress).TextColor(colText)
	case b.Hovered():
		b.Background(colHover).TextColor(colText)
	}
	b.Children(func() {
		ui.Icon(c, ic).FontSize(15)
		ui.Text(c, label).FontSize(12).FontWeight(550).SingleLine()
	})
	return b
}

func ghostButton(c *ui.Context, text string, disabled bool, fn func()) ui.Element {
	b := ui.ButtonBase(c).Height(30).Padding(0, 12).Radius(7).Justify(ui.Center).Border(1, colLine).
		TextColor(colText).Label(text).Disabled(disabled).Transition(hoverFade).Shrink(0)
	switch {
	case disabled:
		b.TextColor(colFaint).BorderColor(colLineSoft)
	case b.Pressed():
		b.Background(colPress)
	case b.Hovered():
		b.Background(colHover)
	}
	b.OnClick(fn)
	b.Children(func() { ui.Text(c, text).FontSize(12).FontWeight(500).SingleLine() })
	return b
}

func primaryButton(c *ui.Context, ic *ui.SVG, text string, disabled bool, fn func()) ui.Element {
	b := ui.ButtonBase(c).Height(34).Padding(0, 16).Gap(7).Radius(7).Justify(ui.Center).Label(text).Disabled(disabled).Transition(hoverFade).Shrink(0)
	switch {
	case disabled:
		b.Background(colHover).TextColor(colFaint)
	case b.Pressed():
		b.Background(colAccentPrs).TextColor(white)
	case b.Hovered():
		b.Background(colAccentHov).TextColor(white)
	default:
		b.Background(colAccent).TextColor(white)
	}
	b.OnClick(fn)
	b.Children(func() {
		if ic != nil {
			ui.Icon(c, ic).FontSize(14)
		}
		ui.Text(c, text).FontSize(12.5).FontWeight(600).SingleLine()
	})
	return b
}

// chip is one choice of a small segmented row, as a body or auth type.
func chip(c *ui.Context, text string, on bool, fn func()) ui.Element {
	b := ui.ButtonBase(c).Height(24).Padding(0, 10).Radius(6).Label(text).Transition(hoverFade).Shrink(0)
	switch {
	case on:
		b.Background(colAccent.Alpha(0.18)).TextColor(colText)
	case b.Hovered():
		b.Background(colHover).TextColor(colText)
	default:
		b.TextColor(colMuted)
	}
	b.OnClick(fn)
	b.Children(func() { ui.Text(c, text).FontSize(12).FontWeight(550).SingleLine() })
	return b
}

// tab is an underlined tab, with an optional count or note after it.
func tab(c *ui.Context, label, text, note string, on bool, fn func()) ui.Element {
	b := ui.ButtonBase(c).Height(34).Padding(0, 10).Gap(6).Label(label).Transition(hoverFade).Shrink(0).
		BorderWidth(0, 0, 2, 0).BorderColor(ui.RGBA(0, 0, 0, 0))
	switch {
	case on:
		b.TextColor(colText).BorderColor(colAccent)
	case b.Hovered():
		b.TextColor(colText)
	default:
		b.TextColor(colMuted)
	}
	b.OnClick(fn)
	b.Children(func() {
		ui.Text(c, text).FontSize(12.5).FontWeight(600).SingleLine()
		if note != "" {
			ui.Text(c, note).FontSize(11).TextColor(colFaint).SingleLine()
		}
	})
	return b
}

// pill is a small tinted label, as a status code.
func pill(c *ui.Context, text string, tone ui.Color) ui.Element {
	return ui.Row(c).Height(22).Padding(0, 8).Radius(6).Background(tone.Alpha(0.16)).Shrink(0).Children(func() {
		ui.Text(c, text).FontSize(12).FontWeight(650).TextColor(onTint(tone)).SingleLine()
	})
}

func emptyState(c *ui.Context, ic *ui.SVG, title, body string, actions func()) {
	ui.Column(c).Grow(1).FillWidth().Center().Gap(6).Padding(24).Children(func() {
		ui.Box(c).Size(44, 44).Radius(12).Background(colAccent.Alpha(0.12)).Center().Margin(0, 0, 6, 0).Children(func() {
			ui.Icon(c, ic).FontSize(20).TextColor(colAccentHov)
		})
		ui.Text(c, title).FontWeight(600)
		ui.Text(c, body).FontSize(12).TextColor(colMuted).Wrap().TextAlign(ui.Center).MaxWidth(360)
		if actions != nil {
			ui.Row(c).Gap(8).Margin(8, 0, 0, 0).Children(actions)
		}
	})
}

func banner(c *ui.Context, msg string, tone ui.Color) {
	ui.Row(c).FillWidth().Padding(8, 12).Gap(10).AlignItems(ui.Start).Radius(8).
		Background(tone.Alpha(0.1)).Border(1, tone.Alpha(0.3)).Children(func() {
		ui.Icon(c, icAlert).FontSize(14).TextColor(tone).Margin(1, 0, 0, 0)
		ui.Text(c, msg).FontSize(12).TextColor(onTint(tone)).Wrap().Selectable().Grow(1).Shrink(1).MinWidth(0)
	})
}

// card is a bordered panel of the main area.
func card(c *ui.Context) ui.Element {
	return ui.Column(c).Radius(10).Background(colPane).Border(1, colLineSoft).Clip()
}
