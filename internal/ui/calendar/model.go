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
	selectedDay int // 1-31, or 0 when day not in current month
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
		selectedDay: now.Day(),
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

	// Calendar view: month and date navigation
	if k, ok := msg.(tea.KeyMsg); ok {
		first := time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local)
		last := first.AddDate(0, 1, -1)
		daysInMonth := last.Day()

		switch k.String() {
		case "p", "h", "left":
			m.month--
			if m.month < 1 {
				m.month = 12
				m.year--
			}
			m.selectedDay = min(m.selectedDay, time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, -1).Day())
			return m, nil
		case "n", "l", "right":
			m.month++
			if m.month > 12 {
				m.month = 1
				m.year++
			}
			m.selectedDay = min(m.selectedDay, time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, -1).Day())
			return m, nil
		case "t":
			now := time.Now()
			m.month = now.Month()
			m.year = now.Year()
			m.selectedDay = now.Day()
			return m, nil
		case "up", "k":
			if m.selectedDay > 1 {
				m.selectedDay--
			} else {
				m.month--
				if m.month < 1 {
					m.month = 12
					m.year--
				}
				m.selectedDay = time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local).AddDate(0, 1, -1).Day()
			}
			return m, nil
		case "down", "j":
			if m.selectedDay < daysInMonth {
				m.selectedDay++
			} else {
				m.month++
				if m.month > 12 {
					m.month = 1
					m.year++
				}
				m.selectedDay = 1
			}
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

	// Calendar view — use full terminal
	if m.width < 10 || m.height < 8 {
		return ""
	}

	// Cell width: make calendar grid fill width (7 columns)
	cellW := max(4, m.width/7)
	calWidth := cellW * 7

	// Month/year header
	title := fmt.Sprintf("  %s %d", m.month.String(), m.year)
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Violet).
		Bold(true).
		Width(calWidth).
		Padding(0, 1)
	header := titleStyle.Render(title)

	// Weekday headers
	weekdays := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	wdStyle := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Width(cellW).
		Align(lipgloss.Center).
		Bold(true)
	var wdParts []string
	for _, d := range weekdays {
		wdParts = append(wdParts, wdStyle.Render(d))
	}
	wdRow := strings.Join(wdParts, "")

	// Month grid
	first := time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local)
	last := first.AddDate(0, 1, -1)
	start := first.Weekday()
	daysInMonth := last.Day()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	cellStyle := lipgloss.NewStyle().Width(cellW).Align(lipgloss.Center).Padding(0, 1)
	selectedStyle := lipgloss.NewStyle().Width(cellW).Align(lipgloss.Center).Padding(0, 1).
		Background(styles.Violet).Foreground(styles.Text).Bold(true)
	todayStyle := lipgloss.NewStyle().Width(cellW).Align(lipgloss.Center).Padding(0, 1).
		Background(styles.Surface).Foreground(styles.Mint)
	var gridLines []string
	var row string
	for i := 0; i < int(start); i++ {
		row += cellStyle.Render("")
	}
	for d := 1; d <= daysInMonth; d++ {
		dayDate := time.Date(m.year, m.month, d, 0, 0, 0, 0, time.Local)
		key := dayDate.Format("2006-01-02")
		tasks := m.tasksByDay[key]
		cellText := fmt.Sprintf("%d", d)
		if len(tasks) > 0 {
			cellText = fmt.Sprintf("%d •%d", d, len(tasks))
		}

		var style lipgloss.Style
		switch {
		case d == m.selectedDay:
			style = selectedStyle
		case dayDate.Equal(today):
			style = todayStyle
		default:
			style = cellStyle
		}
		if len(tasks) > 0 && d != m.selectedDay {
			style = style.Foreground(styles.Mint)
		}
		row += style.Render(cellText)
		if (int(start)+d)%7 == 0 {
			gridLines = append(gridLines, row)
			row = ""
		}
	}
	if row != "" {
		cellsInLastRow := (int(start) + daysInMonth) % 7
		if cellsInLastRow != 0 {
			for i := 0; i < 7-cellsInLastRow; i++ {
				row += cellStyle.Render("")
			}
		}
		gridLines = append(gridLines, row)
	}

	// Build calendar block
	calBlock := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		wdRow,
		"",
	)
	for _, line := range gridLines {
		calBlock += "\n" + line
	}

	// Task detail panel for selected day (when we have height)
	selectedKey := ""
	if m.selectedDay >= 1 && m.selectedDay <= daysInMonth {
		selectedKey = time.Date(m.year, m.month, m.selectedDay, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	}
	selectedTasks := m.tasksByDay[selectedKey]

	var detailBlock string
	taskPanelHeight := m.height - len(strings.Split(calBlock, "\n")) - 6
	if taskPanelHeight >= 2 && len(selectedTasks) > 0 {
		detailBlock = "\n\n"
		dateLabel := time.Date(m.year, m.month, m.selectedDay, 0, 0, 0, 0, time.Local).Format("Monday, January 2, 2006")
		detailBlock += lipgloss.NewStyle().Foreground(styles.Violet).Bold(true).Render("  Tasks for "+dateLabel+":") + "\n"
		maxTasks := min(len(selectedTasks), taskPanelHeight-1)
		descStyle := lipgloss.NewStyle().Foreground(styles.Text).PaddingLeft(2)
		for i := 0; i < maxTasks; i++ {
			t := selectedTasks[i]
			desc := t.Description
			if len([]rune(desc)) > m.width-6 {
				desc = string([]rune(desc)[:m.width-9]) + "…"
			}
			detailBlock += descStyle.Render(fmt.Sprintf("• %s", desc)) + "\n"
		}
		if len(selectedTasks) > maxTasks {
			detailBlock += styles.MutedText.Render(fmt.Sprintf("  … and %d more", len(selectedTasks)-maxTasks)) + "\n"
		}
	}

	// Help line
	help := styles.MutedText.Render("  ↑↓/jk: dates  n/p: months  s: overdue  t: today  q: quit")

	return lipgloss.JoinVertical(lipgloss.Left, calBlock, detailBlock, "", help)
}

// Run runs the calendar TUI as a standalone program.
func Run(ctx context.Context, tw taskwarrior.Client) error {
	m := New(ctx, tw)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
