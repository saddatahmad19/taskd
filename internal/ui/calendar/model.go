package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saddatahmad19/taskd/internal/taskwarrior"
	"github.com/saddatahmad19/taskd/internal/ui/styles"
	"github.com/saddatahmad19/taskd/internal/ui/tasklist"
)

type calendarView int

const (
	viewCalendar calendarView = iota
	viewPastDue
)

type Model struct {
	ctx         context.Context
	tw          taskwarrior.Client
	view        calendarView
	width       int
	height      int
	month       time.Month
	year        int
	selectedDay time.Time // currently selected date for arrow nav
	tasksByDay  map[string][]taskwarrior.Task // "2006-01-02" -> tasks
	pastDue     []taskwarrior.Task
	pastDueList *tasklist.Model
}

func New(ctx context.Context, tw taskwarrior.Client) *Model {
	now := time.Now()
	m := &Model{
		ctx:         ctx,
		tw:          tw,
		view:        viewCalendar,
		month:       now.Month(),
		year:        now.Year(),
		selectedDay: truncDay(now),
	}
	m.loadTasks()
	return m
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) loadTasks() {
	tasks, _ := m.tw.Export(m.ctx, taskwarrior.Filter{Status: "pending"})
	if tasks == nil {
		tasks = []taskwarrior.Task{}
	}

	now := time.Now()
	today := truncDay(now)
	m.tasksByDay = make(map[string][]taskwarrior.Task)
	m.pastDue = nil

	for _, t := range tasks {
		due := t.DueTime()
		if due == nil {
			continue
		}
		dueDay := truncDay(*due)
		key := dueDay.Format("2006-01-02")
		if dueDay.Before(today) {
			m.pastDue = append(m.pastDue, t)
		} else {
			m.tasksByDay[key] = append(m.tasksByDay[key], t)
		}
	}
}

func truncDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// SetSize updates dimensions (used when embedded in full-ui).
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	if m.pastDueList != nil {
		m.pastDueList.SetSize(width, height-2)
	}
}

// clampSelected keeps selectedDay within the current month
func (m *Model) clampSelected() {
	first := time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local)
	last := first.AddDate(0, 1, -1)
	if m.selectedDay.Before(first) {
		m.selectedDay = first
	}
	if m.selectedDay.After(last) {
		m.selectedDay = last
	}
	// Keep selected in displayed month
	if m.selectedDay.Month() != m.month || m.selectedDay.Year() != m.year {
		m.selectedDay = time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local)
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.pastDueList != nil {
			m.pastDueList.SetSize(msg.Width, msg.Height-2)
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "s" {
			m.view = 1 - m.view
			if m.view == viewPastDue && m.pastDueList == nil && len(m.pastDue) > 0 {
				m.pastDueList = m.newPastDueModel()
				m.pastDueList.SetSize(m.width, m.height-2)
				return m, m.pastDueList.Init()
			}
			return m, nil
		}
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	if m.view == viewPastDue && m.pastDueList != nil {
		result, cmd := m.pastDueList.Update(msg)
		if lm, ok := result.(tasklist.Model); ok {
			m.pastDueList = &lm
		}
		return m, cmd
	}

	// Calendar view: month nav (n/p, h/l, arrows) and date nav (arrows)
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "n", "l":
			m.month++
			if m.month > 12 {
				m.month = 1
				m.year++
			}
			m.clampSelected()
			return m, nil
		case "p", "h":
			m.month--
			if m.month < 1 {
				m.month = 12
				m.year--
			}
			m.clampSelected()
			return m, nil
		case "t":
			now := time.Now()
			m.month = now.Month()
			m.year = now.Year()
			m.selectedDay = truncDay(now)
			return m, nil
		case "up":
			m.selectedDay = m.selectedDay.AddDate(0, 0, -7)
			if m.selectedDay.Month() != m.month || m.selectedDay.Year() != m.year {
				m.month = m.selectedDay.Month()
				m.year = m.selectedDay.Year()
			}
			return m, nil
		case "down":
			m.selectedDay = m.selectedDay.AddDate(0, 0, 7)
			if m.selectedDay.Month() != m.month || m.selectedDay.Year() != m.year {
				m.month = m.selectedDay.Month()
				m.year = m.selectedDay.Year()
			}
			return m, nil
		case "left":
			m.selectedDay = m.selectedDay.AddDate(0, 0, -1)
			if m.selectedDay.Month() != m.month || m.selectedDay.Year() != m.year {
				m.month = m.selectedDay.Month()
				m.year = m.selectedDay.Year()
			}
			return m, nil
		case "right":
			m.selectedDay = m.selectedDay.AddDate(0, 0, 1)
			if m.selectedDay.Month() != m.month || m.selectedDay.Year() != m.year {
				m.month = m.selectedDay.Month()
				m.year = m.selectedDay.Year()
			}
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case tasklist.ListViewDoneMsg:
		return m, tea.Quit
	case tasklist.ListDeleteMsg:
		return m.handleDelete(msg)
	}

	return m, nil
}

func (m *Model) handleDelete(msg tasklist.ListDeleteMsg) (tea.Model, tea.Cmd) {
	if msg.Task == nil {
		return m, nil
	}
	if err := m.tw.Delete(m.ctx, msg.Task.UUID); err != nil {
		return m, nil
	}
	m.loadTasks()
	if len(m.pastDue) == 0 {
		m.pastDueList = nil
		m.view = viewCalendar
		return m, nil
	}
	m.pastDueList = m.newPastDueModel()
	m.pastDueList.SetSize(m.width, m.height-2)
	return m, m.pastDueList.Init()
}

func (m *Model) newPastDueModel() *tasklist.Model {
	lm := tasklist.New(m.pastDue, tasklist.ModePastDue)
	lm.OnQuit = func(r tasklist.Result) tea.Cmd {
		return func() tea.Msg { return tasklist.ListViewDoneMsg{Result: r} }
	}
	lm.OnDelete = func(t *taskwarrior.Task) tea.Cmd {
		return func() tea.Msg { return tasklist.ListDeleteMsg{Task: t} }
	}
	return &lm
}

func (m *Model) View() string {
	if m.view == viewPastDue {
		if m.pastDueList != nil {
			return m.pastDueList.View()
		}
		if len(m.pastDue) == 0 {
			return lipgloss.JoinVertical(lipgloss.Left,
				styles.MutedText.Render("  No overdue tasks."),
				"",
				styles.MutedText.Render("  s: back to calendar  q: quit"),
			)
		}
		return styles.MutedText.Render("  Loading…")
	}

	// Calendar view — use full terminal
	effectiveWidth := m.width
	if effectiveWidth < 20 {
		effectiveWidth = 80
	}
	effectiveHeight := m.height
	if effectiveHeight < 10 {
		effectiveHeight = 24
	}

	// Cell width: 7 columns, leave 2 for padding
	cellW := (effectiveWidth - 2) / 7
	if cellW < 3 {
		cellW = 3
	}
	colWidth := cellW + 1 // +1 for spacing

	lines := []string{}

	// Title — full width
	title := fmt.Sprintf("%s %d", m.month.String(), m.year)
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Violet).
		Bold(true).
		Width(effectiveWidth).
		Align(lipgloss.Center).
		PaddingBottom(1)
	lines = append(lines, titleStyle.Render(title))

	// Weekday headers
	weekdays := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	headerStyle := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Bold(true).
		Width(colWidth).
		Align(lipgloss.Center)
	var headerParts []string
	for _, d := range weekdays {
		headerParts = append(headerParts, headerStyle.Render(d))
	}
	lines = append(lines, strings.Join(headerParts, ""))
	lines = append(lines, "")

	// Month grid
	first := time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local)
	last := first.AddDate(0, 1, -1)
	start := first.Weekday()
	daysInMonth := last.Day()

	now := time.Now()
	today := truncDay(now)

	cellStyle := lipgloss.NewStyle().Width(colWidth).Align(lipgloss.Center)
	selectedStyle := lipgloss.NewStyle().
		Width(colWidth).
		Align(lipgloss.Center).
		Background(styles.Surface).
		Foreground(styles.Text)
	todayStyle := lipgloss.NewStyle().
		Width(colWidth).
		Align(lipgloss.Center).
		Foreground(styles.Mint).
		Bold(true)
	taskStyle := lipgloss.NewStyle().Foreground(styles.Mint)

	var row []string
	for i := 0; i < int(start); i++ {
		row = append(row, cellStyle.Render(""))
	}
	for d := 1; d <= daysInMonth; d++ {
		dayDate := time.Date(m.year, m.month, d, 0, 0, 0, 0, time.Local)
		key := dayDate.Format("2006-01-02")
		tasks := m.tasksByDay[key]

		cell := fmt.Sprintf("%2d", d)
		if len(tasks) > 0 {
			cell = fmt.Sprintf("%2d•", d)
		}

		var styled string
		switch {
		case dayDate.Equal(m.selectedDay):
			styled = selectedStyle.Render(" " + cell + " ")
		case dayDate.Equal(today):
			styled = todayStyle.Render(cell)
		case len(tasks) > 0:
			styled = taskStyle.Render(cell)
		default:
			styled = cellStyle.Render(cell)
		}
		row = append(row, styled)
		if len(row) == 7 {
			lines = append(lines, strings.Join(row, ""))
			row = nil
		}
	}
	if len(row) > 0 {
		for len(row) < 7 {
			row = append(row, cellStyle.Render(""))
		}
		lines = append(lines, strings.Join(row, ""))
	}

	// Task details for selected day — use remaining height
	key := m.selectedDay.Format("2006-01-02")
	selectedTasks := m.tasksByDay[key]
	remainingH := effectiveHeight - len(lines) - 5 // title, grid, spacing, hint
	if remainingH > 2 && len(selectedTasks) > 0 {
		lines = append(lines, "")
		sectionTitle := lipgloss.NewStyle().
			Foreground(styles.Violet).
			Bold(true).
			Render(fmt.Sprintf("  %s — %d task(s)", m.selectedDay.Format("Monday, Jan 2"), len(selectedTasks)))
		lines = append(lines, sectionTitle)
		lines = append(lines, "")

		for i, t := range selectedTasks {
			if i >= remainingH-1 {
				lines = append(lines, styles.MutedText.Render("  …"))
				break
			}
			desc := t.Description
			if len(desc) > effectiveWidth-10 {
				desc = desc[:effectiveWidth-13] + "…"
			}
			taskLine := "    " + styles.TaskDesc.Render(desc)
			if t.Project != "" {
				taskLine += "  " + lipgloss.NewStyle().Foreground(styles.Indigo).Render("⌂ "+t.Project)
			}
			for _, tag := range t.Tags {
				taskLine += "  " + styles.Tag.Render("+"+tag)
			}
			if t.Priority != taskwarrior.PriorityNone {
				taskLine += "  " + priorityLabel(t.Priority)
			}
			lines = append(lines, taskLine)
		}
	} else if remainingH > 2 {
		lines = append(lines, "")
		lines = append(lines, styles.MutedText.Render(fmt.Sprintf("  %s — no tasks", m.selectedDay.Format("Monday, Jan 2"))))
	}

	// Keybindings
	lines = append(lines, "")
	hint := styles.MutedText.Render("  ↑↓←→: navigate dates  n/p: next/prev month  s: overdue  t: today  q: quit")
	lines = append(lines, hint)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func priorityLabel(p taskwarrior.Priority) string {
	switch p {
	case taskwarrior.PriorityHigh:
		return styles.PriorityHigh.Render("●H")
	case taskwarrior.PriorityMedium:
		return styles.PriorityMedium.Render("●M")
	case taskwarrior.PriorityLow:
		return styles.PriorityLow.Render("●L")
	default:
		return ""
	}
}

// Run runs the calendar TUI as a standalone program.
func Run(ctx context.Context, tw taskwarrior.Client) error {
	m := New(ctx, tw)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
