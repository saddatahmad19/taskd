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
	tasksByDay  map[string][]taskwarrior.Task // "2006-01-02" -> tasks
	pastDue     []taskwarrior.Task
	pastDueList *tasklist.Model
}

func New(ctx context.Context, tw taskwarrior.Client) *Model {
	now := time.Now()
	m := &Model{
		ctx:  ctx,
		tw:   tw,
		view: viewCalendar,
		month: now.Month(),
		year: now.Year(),
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

	case tasklist.ListViewDoneMsg:
		return m, tea.Quit
	case tasklist.ListDeleteMsg:
		return m.handleDelete(msg)
	}

	if m.view == viewPastDue && m.pastDueList != nil {
		result, cmd := m.pastDueList.Update(msg)
		if lm, ok := result.(tasklist.Model); ok {
			m.pastDueList = &lm
		}
		return m, cmd
	}

	// Calendar view: month navigation
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "h", "left":
			m.month--
			if m.month < 1 {
				m.month = 12
				m.year--
			}
			return m, nil
		case "l", "right":
			m.month++
			if m.month > 12 {
				m.month = 1
				m.year++
			}
			return m, nil
		case "t":
			now := time.Now()
			m.month = now.Month()
			m.year = now.Year()
			return m, nil
		}
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

	// Calendar view
	lines := []string{}

	title := fmt.Sprintf("%s %d", m.month.String(), m.year)
	titleStyle := lipgloss.NewStyle().Foreground(styles.Violet).Bold(true).Align(lipgloss.Center)
	lines = append(lines, titleStyle.Width(m.width).Render(title))
	lines = append(lines, "")

	// Weekday headers
	weekdays := []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	headerStyle := lipgloss.NewStyle().Foreground(styles.Muted).Width(3).Align(lipgloss.Center)
	var headerParts []string
	for _, d := range weekdays {
		headerParts = append(headerParts, headerStyle.Render(d))
	}
	lines = append(lines, strings.Join(headerParts, " "))

	// Month grid
	first := time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local)
	last := first.AddDate(0, 1, -1)
	start := first.Weekday()
	daysInMonth := last.Day()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	var row []string
	// Leading blanks
	for i := 0; i < int(start); i++ {
		row = append(row, "   ")
	}
	for d := 1; d <= daysInMonth; d++ {
		dayDate := time.Date(m.year, m.month, d, 0, 0, 0, 0, time.Local)
		key := dayDate.Format("2006-01-02")
		tasks := m.tasksByDay[key]
		cell := fmt.Sprintf("%2d", d)
		if len(tasks) > 0 {
			cell = lipgloss.NewStyle().Foreground(styles.Mint).Render(fmt.Sprintf("%2d•", d))
		}
		if dayDate.Equal(today) {
			cell = lipgloss.NewStyle().Background(styles.Surface).Render(" " + cell + " ")
		}
		row = append(row, lipgloss.NewStyle().Width(3).Align(lipgloss.Center).Render(cell))
		if len(row) == 7 {
			lines = append(lines, strings.Join(row, " "))
			row = nil
		}
	}
	if len(row) > 0 {
		for len(row) < 7 {
			row = append(row, "   ")
		}
		lines = append(lines, strings.Join(row, " "))
	}

	lines = append(lines, "")
	lines = append(lines, styles.MutedText.Render("  s: overdue tasks  ←/→: change month  t: today  q: quit"))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// Run runs the calendar TUI as a standalone program.
func Run(ctx context.Context, tw taskwarrior.Client) error {
	m := New(ctx, tw)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
