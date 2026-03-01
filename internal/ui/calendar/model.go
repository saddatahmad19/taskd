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
	viewDayList
)

type Model struct {
	ctx         context.Context
	tw          taskwarrior.Client
	view        calendarView
	width       int
	height      int
	month       time.Month
	year        int
	selectedDay int // 1-31, 0 = none
	tasksByDay  map[string][]taskwarrior.Task // "2006-01-02" -> tasks
	pastDue     []taskwarrior.Task
	pastDueList *tasklist.Model
	dayListDate time.Time       // when in viewDayList
	dayList     *tasklist.Model // tasks for selected date
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

func (m *Model) daysInMonth() int {
	return time.Date(m.year, m.month+1, 0, 0, 0, 0, 0, time.Local).Day()
}

func (m *Model) firstWeekday() time.Weekday {
	return time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local).Weekday()
}

// SetSize updates dimensions (used when embedded in full-ui).
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	if m.pastDueList != nil {
		m.pastDueList.SetSize(width, height-2)
	}
	if m.dayList != nil {
		m.dayList.SetSize(width, height-2)
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
		if m.dayList != nil {
			m.dayList.SetSize(msg.Width, msg.Height-2)
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "s" {
			m.view = 1 - m.view
			if m.view == viewPastDue && m.pastDueList == nil && len(m.pastDue) > 0 {
				m.pastDueList = m.newPastDueModel()
				m.pastDueList.SetSize(m.width, m.height-2)
				return m, m.pastDueList.Init()
			}
			return m, nil
		}
		// Backspace: return from day list to calendar
		if msg.String() == "backspace" && m.view == viewDayList {
			m.view = viewCalendar
			m.dayList = nil
			return m, nil
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

	if m.view == viewDayList && m.dayList != nil {
		result, cmd := m.dayList.Update(msg)
		if lm, ok := result.(tasklist.Model); ok {
			m.dayList = &lm
		}
		return m, cmd
	}

	// Calendar view: navigation
	if k, ok := msg.(tea.KeyMsg); ok {
		daysInMonth := m.daysInMonth()

		switch k.String() {
		case "n":
			m.month++
			if m.month > 12 {
				m.month = 1
				m.year++
			}
			m.selectedDay = min(m.selectedDay, m.daysInMonth())
			return m, nil
		case "p":
			m.month--
			if m.month < 1 {
				m.month = 12
				m.year--
			}
			m.selectedDay = min(m.selectedDay, m.daysInMonth())
			return m, nil
		case "t":
			now := time.Now()
			m.month = now.Month()
			m.year = now.Year()
			m.selectedDay = now.Day()
			return m, nil
		case "up", "k":
			if m.selectedDay <= 0 {
				m.selectedDay = 1
			} else {
				m.selectedDay -= 7
				if m.selectedDay < 1 {
					m.month--
					if m.month < 1 {
						m.month = 12
						m.year--
					}
					m.selectedDay = m.daysInMonth() + m.selectedDay
					if m.selectedDay < 1 {
						m.selectedDay = 1
					}
				}
			}
			return m, nil
		case "down", "j":
			if m.selectedDay <= 0 {
				m.selectedDay = 1
			} else {
				m.selectedDay += 7
				if m.selectedDay > daysInMonth {
					m.selectedDay -= daysInMonth
					m.month++
					if m.month > 12 {
						m.month = 1
						m.year++
					}
					m.selectedDay = min(m.selectedDay, m.daysInMonth())
				}
			}
			return m, nil
		case "left", "h":
			if m.selectedDay <= 1 {
				m.month--
				if m.month < 1 {
					m.month = 12
					m.year--
				}
				m.selectedDay = m.daysInMonth()
			} else {
				m.selectedDay--
			}
			return m, nil
		case "right", "l":
			if m.selectedDay >= daysInMonth {
				m.month++
				if m.month > 12 {
					m.month = 1
					m.year++
				}
				m.selectedDay = 1
			} else {
				m.selectedDay++
			}
			return m, nil
		case "enter":
			if m.selectedDay >= 1 && m.selectedDay <= daysInMonth {
				dayDate := time.Date(m.year, m.month, m.selectedDay, 0, 0, 0, 0, time.Local)
				key := dayDate.Format("2006-01-02")
				tasks := m.tasksByDay[key]
				if len(tasks) > 0 {
					m.view = viewDayList
					m.dayListDate = dayDate
					m.dayList = m.newDayListModel(tasks, dayDate)
					m.dayList.SetSize(m.width, m.height-2)
					return m, m.dayList.Init()
				}
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
	if m.view == viewPastDue {
		if len(m.pastDue) == 0 {
			m.pastDueList = nil
			m.view = viewCalendar
			return m, nil
		}
		m.pastDueList = m.newPastDueModel()
		m.pastDueList.SetSize(m.width, m.height-2)
		return m, m.pastDueList.Init()
	}
	if m.view == viewDayList {
		key := m.dayListDate.Format("2006-01-02")
		tasks := m.tasksByDay[key]
		if len(tasks) == 0 {
			m.view = viewCalendar
			m.dayList = nil
			return m, nil
		}
		m.dayList = m.newDayListModel(tasks, m.dayListDate)
		m.dayList.SetSize(m.width, m.height-2)
		return m, m.dayList.Init()
	}
	return m, nil
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

func (m *Model) newDayListModel(tasks []taskwarrior.Task, date time.Time) *tasklist.Model {
	title := fmt.Sprintf("✦  Tasks for %s", date.Format("Mon Jan 2, 2006"))
	lm := tasklist.NewWithTitle(tasks, tasklist.ModeList, title)
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

	if m.view == viewDayList && m.dayList != nil {
		return lipgloss.JoinVertical(lipgloss.Left,
			m.dayList.View(),
			"",
			styles.MutedText.Render("  backspace: back to calendar"),
		)
	}

	// Calendar view - full width and height
	return m.renderCalendar()
}

func (m *Model) renderCalendar() string {
	if m.width < 10 || m.height < 8 {
		return styles.MutedText.Render("  Terminal too small")
	}

	daysInMonth := m.daysInMonth()
	if m.selectedDay < 1 {
		m.selectedDay = 1
	}
	if m.selectedDay > daysInMonth {
		m.selectedDay = daysInMonth
	}
	first := time.Date(m.year, m.month, 1, 0, 0, 0, 0, time.Local)
	start := int(first.Weekday())
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Layout: header, weekday row, 6 rows of days, optional task preview
	availH := m.height - 2
	gridRows := 6
	headerH := 3
	taskPreviewH := availH - headerH - gridRows
	if taskPreviewH < 1 {
		taskPreviewH = 0
	}

	cellWidth := (m.width - 2) / 7
	if cellWidth < 3 {
		cellWidth = 3
	}
	cellHeight := 1

	cellStyle := lipgloss.NewStyle().Width(cellWidth).Height(cellHeight).Align(lipgloss.Center)
	headerCellStyle := lipgloss.NewStyle().Width(cellWidth).Align(lipgloss.Center).Foreground(styles.Muted).Bold(true)
	selectedStyle := lipgloss.NewStyle().Width(cellWidth).Height(cellHeight).Align(lipgloss.Center).
		Background(styles.Violet).Foreground(styles.Text)
	todayStyle := lipgloss.NewStyle().Width(cellWidth).Height(cellHeight).Align(lipgloss.Center).
		Background(styles.Mint).Foreground(styles.Surface)
	hasTasksStyle := lipgloss.NewStyle().Width(cellWidth).Height(cellHeight).Align(lipgloss.Center).
		Foreground(styles.Mint).Bold(true)

	// Title
	title := fmt.Sprintf("%s %d", m.month.String(), m.year)
	titleStyle := lipgloss.NewStyle().Foreground(styles.Violet).Bold(true).Align(lipgloss.Center)
	titleBlock := titleStyle.Width(m.width).Render(title)

	// Weekday headers
	weekdays := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	var headerParts []string
	for _, d := range weekdays {
		headerParts = append(headerParts, headerCellStyle.Render(d))
	}
	headerRow := strings.Join(headerParts, " ")

	// Build grid
	var rows []string
	rows = append(rows, titleBlock, "", headerRow)

	day := 1
	for row := 0; row < gridRows; row++ {
		var rowParts []string
		for col := 0; col < 7; col++ {
			idx := row*7 + col
			if idx < start || day > daysInMonth {
				rowParts = append(rowParts, cellStyle.Render(""))
				continue
			}
			dayDate := time.Date(m.year, m.month, day, 0, 0, 0, 0, time.Local)
			key := dayDate.Format("2006-01-02")
			tasks := m.tasksByDay[key]
			cellContent := fmt.Sprintf("%d", day)
			if len(tasks) > 0 {
				cellContent = fmt.Sprintf("%d•%d", day, len(tasks))
			}

			var styled string
			switch {
			case m.selectedDay == day:
				styled = selectedStyle.Render(cellContent)
			case dayDate.Equal(today):
				styled = todayStyle.Render(cellContent)
			case len(tasks) > 0:
				styled = hasTasksStyle.Render(cellContent)
			default:
				styled = cellStyle.Render(cellContent)
			}
			rowParts = append(rowParts, styled)
			day++
		}
		rows = append(rows, strings.Join(rowParts, " "))
	}

	// Task preview panel (when we have space and a selected day with tasks)
	if taskPreviewH > 0 && m.selectedDay >= 1 && m.selectedDay <= daysInMonth {
		dayDate := time.Date(m.year, m.month, m.selectedDay, 0, 0, 0, 0, time.Local)
		key := dayDate.Format("2006-01-02")
		tasks := m.tasksByDay[key]
		if len(tasks) > 0 {
			previewTitle := lipgloss.NewStyle().Foreground(styles.Violet).Bold(true).Render(
				fmt.Sprintf("  %s — %d task(s) (Enter to open)", dayDate.Format("Mon Jan 2"), len(tasks)))
			rows = append(rows, "")
			rows = append(rows, previewTitle)
			maxTasks := taskPreviewH - 2
			if maxTasks < 1 {
				maxTasks = 1
			}
			for i := 0; i < len(tasks) && i < maxTasks; i++ {
				t := tasks[i]
				desc := t.Description
				if len(desc) > m.width-6 {
					desc = desc[:m.width-9] + "…"
				}
				line := fmt.Sprintf("    #%d  %s", t.ID, desc)
				if t.Project != "" {
					line += " " + styles.MutedText.Render("⌂ "+t.Project)
				}
				rows = append(rows, lipgloss.NewStyle().Foreground(styles.Text).Render(line))
			}
			if len(tasks) > maxTasks {
				rows = append(rows, styles.MutedText.Render(fmt.Sprintf("    … and %d more", len(tasks)-maxTasks)))
			}
		}
	}

	// Help line
	help := styles.MutedText.Render("  ↑↓←→ move  n/p month  Enter open  s overdue  t today  ⌫ back  q quit")
	rows = append(rows, "")
	rows = append(rows, help)

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	// Use full available space
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, content)
}

// Run runs the calendar TUI as a standalone program.
func Run(ctx context.Context, tw taskwarrior.Client) error {
	m := New(ctx, tw)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
