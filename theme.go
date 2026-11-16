package main

import "github.com/egoist/mygo/ui"

// Copy the system theme so application overrides do not mutate shared defaults.
func appTheme(base *ui.Theme) *ui.Theme {
	t := *base
	t.FontSize = 13
	t.Radius = 6
	return &t
}
