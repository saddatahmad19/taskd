package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	dirName  = "taskd"
	fileName = "config.json"
)

// ThemeConfig holds all configurable colors for the taskd TUI.
// Every field is a CSS hex color string (e.g. "#A78BFA").
// An empty string means "use the built-in default".
type ThemeConfig struct {
	// Accent is the primary accent color used for titles, selected items,
	// tag text, filter cursor, and the huh form focused title.
	// Default: #A78BFA (soft violet)
	Accent string `json:"accent"`

	// AccentSecondary is the secondary accent used for project labels
	// and the huh form select-selector.
	// Default: #7C3AED (indigo)
	AccentSecondary string `json:"accent_secondary"`

	// Success is used for success messages, priority-low badges,
	// the filter prompt, and the huh selected-option highlight.
	// Default: #10B981 (mint green)
	Success string `json:"success"`

	// Error is used for error messages and priority-high badges.
	// Default: #F43F5E (rose)
	Error string `json:"error"`

	// Warning is used for warning messages and priority-medium badges.
	// Default: #F59E0B (amber)
	Warning string `json:"warning"`

	// Muted is used for secondary / dimmed text, the divider line,
	// status bar labels, hints, and the huh blurred-title.
	// Default: #6B7280 (cool gray)
	Muted string `json:"muted"`

	// Surface is used as a background color for tag chips, the list title
	// bar, the selected-row highlight, and the form header bar.
	// Default: #1F2937 (dark gray-blue)
	Surface string `json:"surface"`

	// Text is used for the primary task description text and filter input.
	// Default: #F9FAFB (near white)
	Text string `json:"text"`
}

// Config is the root configuration object stored in
// ~/.config/taskd/config.json.
type Config struct {
	// FullOnLaunch controls what happens when `taskd` is invoked with no
	// subcommand arguments.
	//
	//   false (default) — print the normal help text.
	//   true            — launch the full tabbed TUI (equivalent to `taskd full-ui`).
	FullOnLaunch bool `json:"fullOnLaunch"`

	Theme ThemeConfig `json:"theme"`
}

// Default returns a Config populated with every built-in default value.
func Default() Config {
	return Config{
		Theme: ThemeConfig{
			Accent:          "#A78BFA",
			AccentSecondary: "#7C3AED",
			Success:         "#10B981",
			Error:           "#F43F5E",
			Warning:         "#F59E0B",
			Muted:           "#6B7280",
			Surface:         "#1F2937",
			Text:            "#F9FAFB",
		},
	}
}

// Load reads the taskd config file, creating the directory and file with
// defaults if they do not yet exist.
//
// The function always returns a usable Config; loading errors fall back to
// defaults and are returned as a non-fatal second value so callers can warn
// the user if they choose.
func Load() (Config, error) {
	dir, err := configDir()
	if err != nil {
		return Default(), fmt.Errorf("resolve config directory: %w", err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Default(), fmt.Errorf("create config directory %q: %w", dir, err)
	}

	path := filepath.Join(dir, fileName)

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		cfg := Default()
		// Write a commented default file so the user can discover and edit it.
		if writeErr := writeDefault(path, cfg); writeErr != nil {
			// Non-fatal: we simply won't persist the file this run.
			return cfg, fmt.Errorf("write default config to %q: %w", path, writeErr)
		}
		return cfg, nil
	}
	if err != nil {
		return Default(), fmt.Errorf("read config file %q: %w", path, err)
	}

	// Unmarshal on top of defaults so any key absent from the file keeps its
	// default value without extra merging logic.
	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), fmt.Errorf("parse config file %q: %w", path, err)
	}

	// Explicit merge: zero-value (empty string) fields fall back to defaults.
	cfg.Theme = mergeTheme(cfg.Theme, Default().Theme)

	return cfg, nil
}

// ConfigPath returns the absolute path to the config file without reading it.
func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, dirName), nil
}

// mergeTheme fills in any empty field in t with the corresponding value from def.
func mergeTheme(t, def ThemeConfig) ThemeConfig {
	if t.Accent == "" {
		t.Accent = def.Accent
	}
	if t.AccentSecondary == "" {
		t.AccentSecondary = def.AccentSecondary
	}
	if t.Success == "" {
		t.Success = def.Success
	}
	if t.Error == "" {
		t.Error = def.Error
	}
	if t.Warning == "" {
		t.Warning = def.Warning
	}
	if t.Muted == "" {
		t.Muted = def.Muted
	}
	if t.Surface == "" {
		t.Surface = def.Surface
	}
	if t.Text == "" {
		t.Text = def.Text
	}
	return t
}

// writeDefault serialises cfg as indented JSON and writes it to path.
func writeDefault(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
