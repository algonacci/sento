package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// settings are Sento's own preferences, apart from any workspace: a small
// settings.json in the user's config directory (SENTO_CONFIG_DIR overrides
// it). Workspaces stay in the files the user opens and saves.
type settings struct {
	// Theme is themeSystem, themeLight or themeDark; empty is system.
	Theme string `json:"theme,omitempty"`
}

func settingsDir() string {
	if d := os.Getenv("SENTO_CONFIG_DIR"); d != "" {
		return d
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Sento")
}

// loadSettings reads the settings, or the defaults when there are none or
// they cannot be read.
func loadSettings() settings {
	var s settings
	if d := settingsDir(); d != "" {
		if b, err := os.ReadFile(filepath.Join(d, "settings.json")); err == nil {
			_ = json.Unmarshal(b, &s)
		}
	}
	switch s.Theme {
	case themeLight, themeDark:
	default:
		s.Theme = themeSystem
	}
	return s
}

// saveSettings writes the settings whole, through a temporary file, so a
// crash never leaves half a file.
func saveSettings(s settings) error {
	d := settingsDir()
	if d == "" {
		return nil
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(d, "settings.json.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(d, "settings.json"))
}

// applyThemeSource tells the platform the chosen appearance, so native
// menus, dialogs and the window frame follow it; main sets it.
var applyThemeSource = func(mode string) {}

// cycleTheme moves the appearance on, System → Light → Dark, and keeps it.
func (a *app) cycleTheme() {
	a.prefs.Theme = nextTheme(a.prefs.Theme)
	applyThemeSource(a.prefs.Theme)
	if err := saveSettings(a.prefs); err != nil {
		a.message = "Could not save settings: " + err.Error()
	}
}
