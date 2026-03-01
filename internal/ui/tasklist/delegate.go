package tasklist

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saddatahmad19/taskd/internal/taskwarrior"
	"github.com/saddatahmad19/taskd/internal/ui/styles"
)

const (
	delegateHeight  = 2 // lines per row
	delegateSpacing = 1 // blank lines between rows
)

type taskDelegate struct {
	mode     Mode
	selected map[string]bool // uuid → checked; only relevant in ModeComplete
}

func newTaskDelegate(mode Mode) *taskDelegate {
	return &taskDelegate{
		mode:     mode,
		selected: make(map[string]bool),
	}
}

var _ list.ItemDelegate = (*taskDelegate)(nil)

func (d *taskDelegate) Height() int  { return delegateHeight }
func (d *taskDelegate) Spacing() int { return delegateSpacing }

func (d *taskDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d *taskDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(Item)
	if !ok {
		return
	}
	task := it.Task

	isCurrent := index == m.Index()
	isFiltering := m.SettingFilter()

	// ── Priority accent bar (left edge) ────────────────────────────────────────
	bar := priorityBarStyle(task.Priority).Render("▌")

	// ── Checkbox (complete mode only) ──────────────────────────────────────────
	checkStr := ""
	if d.mode == ModeComplete {
		if d.selected[task.UUID] {
			checkStr = styles.Success.Render("✓ ")
		} else {
			checkStr = styles.MutedText.Render("○ ")
		}
	}

	// ── Description ────────────────────────────────────────────────────────────
	availableWidth := m.Width() - 6 // bar(1) + space(1) + check(2) + padding(2)
	if availableWidth < 10 {
		availableWidth = 10
	}
	descText := task.Description
	if len([]rune(descText)) > availableWidth {
		runes := []rune(descText)
		descText = string(runes[:availableWidth-1]) + "…"
	}

	var descStyle lipgloss.Style
	if isCurrent && !isFiltering {
		descStyle = lipgloss.NewStyle().Foreground(styles.Violet).Bold(true)
	} else {
		descStyle = lipgloss.NewStyle().Foreground(styles.Text)
	}
	descRendered := descStyle.Render(descText)

	titleLine := bar + " " + checkStr + descRendered

	// ── Subtitle / metadata line ───────────────────────────────────────────────
	var meta []string

	// Numeric ID.
	meta = append(meta, styles.MutedText.Render(fmt.Sprintf("#%d", task.ID)))

	// Tag badges.
	for _, t := range task.Tags {
		meta = append(meta, styles.Tag.Render("+"+t))
	}

	// Project.
	if task.Project != "" {
		meta = append(meta,
			lipgloss.NewStyle().Foreground(styles.Indigo).Render("⌂ "+task.Project))
	}

	// Priority label (only when not None, bar already gives a hint).
	if task.Priority != taskwarrior.PriorityNone {
		meta = append(meta, priorityLabel(task.Priority))
	}

	// Due date with urgency colour.
	if due := task.DueTime(); due != nil {
		meta = append(meta, dueBadge(due))
	}

	subtitleLine := "  " + strings.Join(meta, "  ")

	// ── Selected-item background highlight ─────────────────────────────────────
	row := lipgloss.JoinVertical(lipgloss.Left, titleLine, subtitleLine)
	if isCurrent && !isFiltering {
		row = lipgloss.NewStyle().
			Background(styles.Surface).
			PaddingRight(2).
			Render(row)
	}

	fmt.Fprint(w, row)
}

func priorityBarStyle(p taskwarrior.Priority) lipgloss.Style {
	switch p {
	case taskwarrior.PriorityHigh:
		return lipgloss.NewStyle().Foreground(styles.Rose)
	case taskwarrior.PriorityMedium:
		return lipgloss.NewStyle().Foreground(styles.Amber)
	case taskwarrior.PriorityLow:
		return lipgloss.NewStyle().Foreground(styles.Mint)
	default:
		return lipgloss.NewStyle().Foreground(styles.Muted)
	}
}

func priorityLabel(p taskwarrior.Priority) string {
	switch p {
	case taskwarrior.PriorityHigh:
		return styles.PriorityHigh.Render("● H")
	case taskwarrior.PriorityMedium:
		return styles.PriorityMedium.Render("● M")
	case taskwarrior.PriorityLow:
		return styles.PriorityLow.Render("● L")
	default:
		return ""
	}
}

func dueBadge(due *time.Time) string {
	now := time.Now()
	today := truncDay(now)
	dueDay := truncDay(*due)
	diff := int(dueDay.Sub(today).Hours() / 24)

	var label string
	var style lipgloss.Style

	switch {
	case diff < 0:
		label = fmt.Sprintf("⏰ overdue (%s)", due.Format("Jan 02"))
		style = styles.PriorityHigh
	case diff == 0:
		label = "⏰ due today"
		style = styles.PriorityMedium
	case diff == 1:
		label = "⏰ due tomorrow"
		style = styles.PriorityLow
	case diff <= 7:
		label = fmt.Sprintf("⏰ %s (%dd)", due.Format("Jan 02"), diff)
		style = styles.PriorityLow
	default:
		label = "⏰ " + due.Format("2006-01-02")
		style = lipgloss.NewStyle().Foreground(styles.Muted)
	}

	return style.Render(label)
}
