package styles

import "github.com/charmbracelet/lipgloss"

var (
	Indigo  = lipgloss.Color("#7C3AED")
	Violet  = lipgloss.Color("#A78BFA")
	Mint    = lipgloss.Color("#10B981")
	Rose    = lipgloss.Color("#F43F5E")
	Amber   = lipgloss.Color("#F59E0B")
	Muted   = lipgloss.Color("#6B7280")
	Surface = lipgloss.Color("#1F2937")
	Text    = lipgloss.Color("#F9FAFB")
)

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
