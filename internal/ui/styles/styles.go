package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/saddatahmad19/taskd/internal/config"
)

// ── Palette ───────────────────────────────────────────────────────────────────
// These variables hold the active colors. They start with built-in defaults
// and are reassigned by Init when a user config is loaded.

var (
	Indigo  = lipgloss.Color("#7C3AED")
	Violet  = lipgloss.Color("#A78BFA")
	Mint    = lipgloss.Color("#10B981")
	Rose    = lipgloss.Color("#F43F5E")
	Amber   = lipgloss.Color("#F59E0B")
	Muted   = lipgloss.Color("#6B7280")
	Surface = lipgloss.Color("#1F2937")
	Text    = lipgloss.Color("#F9FAFB")
	NeonRed = lipgloss.Color("#FF0000")
)

// ── Component styles ──────────────────────────────────────────────────────────
// Declared as vars so they can be rebuilt after Init updates the palette.

var App = lipgloss.NewStyle().
	Margin(1, 2)

var Header = lipgloss.NewStyle().
	Bold(true).
	Foreground(Violet).
	MarginBottom(1)

var SubHeader = lipgloss.NewStyle().
	Foreground(Muted).
	Italic(true)

var Success = lipgloss.NewStyle().
	Foreground(Mint).
	Bold(true)

var Error = lipgloss.NewStyle().
	Foreground(Rose).
	Bold(true)

var Warning = lipgloss.NewStyle().
	Foreground(Amber)

var MutedText = lipgloss.NewStyle().
	Foreground(Muted).
	Italic(true)

var Tag = lipgloss.NewStyle().
	Foreground(Violet).
	Background(Surface).
	Padding(0, 1).
	Bold(true)

var (
	PriorityHigh   = lipgloss.NewStyle().Foreground(Rose).Bold(true)
	PriorityMedium = lipgloss.NewStyle().Foreground(Amber).Bold(true)
	PriorityLow    = lipgloss.NewStyle().Foreground(Mint)
	PriorityNone   = lipgloss.NewStyle().Foreground(Muted)
)

var Divider = lipgloss.NewStyle().
	Foreground(Muted).
	SetString("─────────────────────────────────────────")

var FormHint = lipgloss.NewStyle().
	Foreground(Muted).
	Italic(true).
	PaddingLeft(2)

var TaskRow = lipgloss.NewStyle().
	PaddingLeft(1)

var TaskID = lipgloss.NewStyle().
	Foreground(Muted).
	Width(4)

var TaskDesc = lipgloss.NewStyle().
	Foreground(Text)

// ── Init ──────────────────────────────────────────────────────────────────────

// Init applies the theme colors from cfg and rebuilds every component style.
// Call this once at program startup, after loading the user config, before any
// UI code renders.
func Init(cfg config.Config) {
	t := cfg.Theme

	// Update the palette vars.
	if t.Accent != "" {
		Violet = lipgloss.Color(t.Accent)
	}
	if t.AccentSecondary != "" {
		Indigo = lipgloss.Color(t.AccentSecondary)
	}
	if t.Success != "" {
		Mint = lipgloss.Color(t.Success)
	}
	if t.Error != "" {
		Rose = lipgloss.Color(t.Error)
	}
	if t.Warning != "" {
		Amber = lipgloss.Color(t.Warning)
	}
	if t.Muted != "" {
		Muted = lipgloss.Color(t.Muted)
	}
	if t.Surface != "" {
		Surface = lipgloss.Color(t.Surface)
	}
	if t.Text != "" {
		Text = lipgloss.Color(t.Text)
	}

	// Rebuild all component styles using the updated palette.
	rebuild()
}

// rebuild reconstructs every component-style variable from the current palette.
// This is necessary because lipgloss.Style captures color values by copy at
// creation time, so simply reassigning palette vars is not enough.
func rebuild() {
	App = lipgloss.NewStyle().
		Margin(1, 2)

	Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(Violet).
		MarginBottom(1)

	SubHeader = lipgloss.NewStyle().
		Foreground(Muted).
		Italic(true)

	Success = lipgloss.NewStyle().
		Foreground(Mint).
		Bold(true)

	Error = lipgloss.NewStyle().
		Foreground(Rose).
		Bold(true)

	Warning = lipgloss.NewStyle().
		Foreground(Amber)

	MutedText = lipgloss.NewStyle().
		Foreground(Muted).
		Italic(true)

	Tag = lipgloss.NewStyle().
		Foreground(Violet).
		Background(Surface).
		Padding(0, 1).
		Bold(true)

	PriorityHigh = lipgloss.NewStyle().Foreground(Rose).Bold(true)
	PriorityMedium = lipgloss.NewStyle().Foreground(Amber).Bold(true)
	PriorityLow = lipgloss.NewStyle().Foreground(Mint)
	PriorityNone = lipgloss.NewStyle().Foreground(Muted)

	Divider = lipgloss.NewStyle().
		Foreground(Muted).
		SetString("─────────────────────────────────────────")

	FormHint = lipgloss.NewStyle().
		Foreground(Muted).
		Italic(true).
		PaddingLeft(2)

	TaskRow = lipgloss.NewStyle().
		PaddingLeft(1)

	TaskID = lipgloss.NewStyle().
		Foreground(Muted).
		Width(4)

	TaskDesc = lipgloss.NewStyle().
		Foreground(Text)
}
