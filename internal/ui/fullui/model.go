package fullui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saddatahmad19/taskd/internal/taskwarrior"
	"github.com/saddatahmad19/taskd/internal/ui/add"
	"github.com/saddatahmad19/taskd/internal/ui/calendar"
	"github.com/saddatahmad19/taskd/internal/ui/styles"
	"github.com/saddatahmad19/taskd/internal/ui/tasklist"
)

const (
	tabAdd = iota
	tabList
	tabComplete
	tabModify
	tabCalendar
	tabCount
)

var tabNames = []string{"Add", "List", "Complete", "Modify", "Calendar"}

var (
	tabStyleActive = lipgloss.NewStyle().
			Foreground(styles.Text).
			Background(styles.Violet).
			Padding(0, 2).
			Bold(true)
	tabStyleInactive = lipgloss.NewStyle().
				Foreground(styles.Muted).
				Padding(0, 2)
)

type Model struct {
	ctx    context.Context
	tw     taskwarrior.Client
	active int
	width  int
	height int

	// Add tab
	addForm *add.FormModel

	// List tab
	listTasks  []taskwarrior.Task
	listModel  *tasklist.Model
	listFilter taskwarrior.Filter

	// Complete tab
	completeTasks []taskwarrior.Task
	completeModel *tasklist.Model

	// Modify tab
	modifyTasks  []taskwarrior.Task
	modifyModel  *tasklist.Model
	modifyEditing *taskwarrior.Task
	modifyForm   *add.FormModel

	// Calendar tab
	calendarModel *calendar.Model

	// Status message
	statusMsg string
	statusErr bool
}

func New(ctx context.Context, tw taskwarrior.Client) *Model {
	m := &Model{
		ctx:        ctx,
		tw:         tw,
		active:     tabAdd,
		listFilter: taskwarrior.Filter{Status: "pending"},
	}
	m.addForm = add.NewFormModel(ctx, tw, add.Defaults{})
	return m
}

func (m *Model) Init() tea.Cmd {
	return m.addForm.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeActiveView()
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "tab" {
			m.active = (m.active + 1) % tabCount
			m.statusMsg = ""
			m.resizeActiveView()
			return m, m.ensureActiveViewInit()
		}
		if msg.String() == "shift+tab" {
			m.active = (m.active + tabCount - 1) % tabCount
			m.statusMsg = ""
			m.resizeActiveView()
			return m, m.ensureActiveViewInit()
		}
		// Alt+1 through Alt+4 (works in Kitty and some terminals)
		switch msg.String() {
		case "alt+1":
			m.active = 0
			m.statusMsg = ""
			m.resizeActiveView()
			return m, m.ensureActiveViewInit()
		case "alt+2":
			m.active = 1
			m.statusMsg = ""
			m.resizeActiveView()
			return m, m.ensureActiveViewInit()
		case "alt+3":
			m.active = 2
			m.statusMsg = ""
			m.resizeActiveView()
			return m, m.ensureActiveViewInit()
		case "alt+4":
			m.active = 3
			m.statusMsg = ""
			m.resizeActiveView()
			return m, m.ensureActiveViewInit()
		case "alt+5":
			m.active = 4
			m.statusMsg = ""
			m.resizeActiveView()
			return m, m.ensureActiveViewInit()
		}
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case add.FormDoneMsg:
		return m.handleFormDone(msg)
	case add.FormCancelMsg:
		return m.handleFormCancel()
	case tasklist.ListViewDoneMsg:
		return m.handleListDone(msg)
	case tasklist.ListDeleteMsg:
		return m.handleListDelete(msg)
	}

	return m.updateActiveView(msg)
}

func (m *Model) handleFormDone(msg add.FormDoneMsg) (tea.Model, tea.Cmd) {
	if m.modifyEditing != nil {
		target := m.modifyEditing
		m.modifyEditing = nil
		m.modifyForm = nil

		req := msg.Result.ToAddRequest()
		if err := m.tw.Modify(m.ctx, target.UUID, req); err != nil {
			m.statusMsg = fmt.Sprintf("Failed to modify: %v", err)
			m.statusErr = true
		} else {
			m.statusMsg = "Task modified!"
			m.statusErr = false
		}
		m.refreshModifyTasks()
		m.modifyModel = m.newModifyModel()
		m.resizeActiveView()
		return m, m.modifyModel.Init()
	}

	if _, err := m.tw.Add(m.ctx, msg.Result.ToAddRequest()); err != nil {
		m.statusMsg = fmt.Sprintf("Failed to add: %v", err)
		m.statusErr = true
	} else {
		m.statusMsg = "Task added!"
		m.statusErr = false
	}
	m.addForm = add.NewFormModel(m.ctx, m.tw, add.Defaults{})
	return m, m.addForm.Init()
}

func (m *Model) handleFormCancel() (tea.Model, tea.Cmd) {
	if m.modifyEditing != nil {
		m.modifyEditing = nil
		m.modifyForm = nil
		m.modifyModel = m.newModifyModel()
		m.resizeActiveView()
		return m, m.modifyModel.Init()
	}
	m.addForm = add.NewFormModel(m.ctx, m.tw, add.Defaults{})
	return m, m.addForm.Init()
}

func (m *Model) handleListDone(msg tasklist.ListViewDoneMsg) (tea.Model, tea.Cmd) {
	if msg.Result.Aborted {
		return m, tea.Quit
	}

	if len(msg.Result.CompletedUUIDs) > 0 {
		var errCount int
		for _, uuid := range msg.Result.CompletedUUIDs {
			if err := m.tw.Complete(m.ctx, uuid); err != nil {
				m.statusMsg = fmt.Sprintf("Failed to complete %s: %v", uuid[:8], err)
				m.statusErr = true
				errCount++
			}
		}
		if errCount == 0 {
			m.statusMsg = fmt.Sprintf("%d task(s) completed", len(msg.Result.CompletedUUIDs))
			m.statusErr = false
		}
		m.refreshCompleteTasks()
		m.completeModel = m.newCompleteModel()
		m.resizeActiveView()
		return m, m.completeModel.Init()
	}

	if msg.Result.SelectedTask != nil {
		t := *msg.Result.SelectedTask
		m.modifyEditing = &t
		m.modifyForm = add.NewFormModel(m.ctx, m.tw, add.DefaultsFromTask(t))
		return m, m.modifyForm.Init()
	}

	return m, nil
}

func (m *Model) handleListDelete(msg tasklist.ListDeleteMsg) (tea.Model, tea.Cmd) {
	if msg.Task == nil {
		return m, nil
	}
	if err := m.tw.Delete(m.ctx, msg.Task.UUID); err != nil {
		m.statusMsg = fmt.Sprintf("Failed to delete: %v", err)
		m.statusErr = true
		return m, nil
	}
	m.statusMsg = "Task deleted"
	m.statusErr = false

	switch m.active {
	case tabList:
		m.loadListTasks()
		m.listModel = m.newListModel()
		m.resizeActiveView()
		return m, m.listModel.Init()
	case tabComplete:
		m.refreshCompleteTasks()
		m.completeModel = m.newCompleteModel()
		m.resizeActiveView()
		return m, m.completeModel.Init()
	case tabModify:
		m.refreshModifyTasks()
		m.modifyModel = m.newModifyModel()
		m.resizeActiveView()
		return m, m.modifyModel.Init()
	case tabCalendar:
		m.calendarModel = calendar.New(m.ctx, m.tw)
		m.resizeActiveView()
		return m, nil
	}
	return m, nil
}

func (m *Model) ensureActiveViewInit() tea.Cmd {
	contentH := m.height - 4
	if contentH < 2 {
		contentH = 2
	}

	switch m.active {
	case tabList:
		if m.listModel == nil {
			m.loadListTasks()
			m.listModel = m.newListModel()
			m.listModel.SetSize(m.width, contentH)
			return m.listModel.Init()
		}
	case tabComplete:
		if m.completeModel == nil {
			m.loadCompleteTasks()
			m.completeModel = m.newCompleteModel()
			m.completeModel.SetSize(m.width, contentH)
			return m.completeModel.Init()
		}
	case tabModify:
		if m.modifyForm == nil && m.modifyModel == nil {
			m.loadModifyTasks()
			m.modifyModel = m.newModifyModel()
			m.modifyModel.SetSize(m.width, contentH)
			return m.modifyModel.Init()
		}
	case tabCalendar:
		if m.calendarModel == nil {
			m.calendarModel = calendar.New(m.ctx, m.tw)
			m.resizeActiveView()
		}
	}
	return nil
}

func (m *Model) resizeActiveView() {
	contentH := m.height - 4
	if contentH < 2 {
		contentH = 2
	}

	if m.addForm != nil {
		m.addForm.SetSize(m.width, contentH)
	}
	if m.listModel != nil {
		m.listModel.SetSize(m.width, contentH)
	}
	if m.completeModel != nil {
		m.completeModel.SetSize(m.width, contentH)
	}
	if m.modifyForm != nil {
		m.modifyForm.SetSize(m.width, contentH)
	}
	if m.modifyModel != nil {
		m.modifyModel.SetSize(m.width, contentH)
	}
	if m.calendarModel != nil {
		m.calendarModel.SetSize(m.width, contentH)
	}
}

func (m *Model) updateActiveView(msg tea.Msg) (tea.Model, tea.Cmd) {
	contentH := m.height - 4
	if contentH < 2 {
		contentH = 2
	}

	switch m.active {
	case tabAdd:
		if m.addForm != nil {
			result, cmd := m.addForm.Update(msg)
			if fm, ok := result.(*add.FormModel); ok {
				m.addForm = fm
			}
			return m, cmd
		}
	case tabList:
		if m.listModel == nil {
			m.loadListTasks()
			m.listModel = m.newListModel()
			m.listModel.SetSize(m.width, contentH)
			return m, m.listModel.Init()
		}
		result, cmd := m.listModel.Update(msg)
		if lm, ok := result.(tasklist.Model); ok {
			m.listModel = &lm
		}
		return m, cmd
	case tabComplete:
		if m.completeModel == nil {
			m.loadCompleteTasks()
			m.completeModel = m.newCompleteModel()
			m.completeModel.SetSize(m.width, contentH)
			return m, m.completeModel.Init()
		}
		result, cmd := m.completeModel.Update(msg)
		if lm, ok := result.(tasklist.Model); ok {
			m.completeModel = &lm
		}
		return m, cmd
	case tabModify:
		if m.modifyForm != nil {
			result, cmd := m.modifyForm.Update(msg)
			if fm, ok := result.(*add.FormModel); ok {
				m.modifyForm = fm
			}
			return m, cmd
		}
		if m.modifyModel == nil {
			m.loadModifyTasks()
			m.modifyModel = m.newModifyModel()
			m.modifyModel.SetSize(m.width, contentH)
			return m, m.modifyModel.Init()
		}
		result, cmd := m.modifyModel.Update(msg)
		if lm, ok := result.(tasklist.Model); ok {
			m.modifyModel = &lm
		}
		return m, cmd
	}
	if m.active == tabCalendar && m.calendarModel != nil {
		result, cmd := m.calendarModel.Update(msg)
		if cm, ok := result.(*calendar.Model); ok {
			m.calendarModel = cm
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) loadListTasks() {
	m.listTasks, _ = m.tw.Export(m.ctx, m.listFilter)
	if m.listTasks == nil {
		m.listTasks = []taskwarrior.Task{}
	}
}

func (m *Model) loadCompleteTasks() {
	m.completeTasks, _ = m.tw.Export(m.ctx, taskwarrior.Filter{Status: "pending"})
	if m.completeTasks == nil {
		m.completeTasks = []taskwarrior.Task{}
	}
}

func (m *Model) loadModifyTasks() {
	m.modifyTasks, _ = m.tw.Export(m.ctx, taskwarrior.Filter{Status: "pending"})
	if m.modifyTasks == nil {
		m.modifyTasks = []taskwarrior.Task{}
	}
}

func (m *Model) refreshCompleteTasks() {
	m.loadCompleteTasks()
}

func (m *Model) refreshModifyTasks() {
	m.loadModifyTasks()
}

func (m *Model) newListModel() *tasklist.Model {
	lm := tasklist.New(m.listTasks, tasklist.ModeList)
	lm.OnQuit = func(r tasklist.Result) tea.Cmd {
		return func() tea.Msg { return tasklist.ListViewDoneMsg{Result: r} }
	}
	lm.OnDelete = func(t *taskwarrior.Task) tea.Cmd {
		return func() tea.Msg { return tasklist.ListDeleteMsg{Task: t} }
	}
	return &lm
}

func (m *Model) newCompleteModel() *tasklist.Model {
	lm := tasklist.New(m.completeTasks, tasklist.ModeComplete)
	lm.OnQuit = func(r tasklist.Result) tea.Cmd {
		return func() tea.Msg { return tasklist.ListViewDoneMsg{Result: r} }
	}
	lm.OnDelete = func(t *taskwarrior.Task) tea.Cmd {
		return func() tea.Msg { return tasklist.ListDeleteMsg{Task: t} }
	}
	return &lm
}

func (m *Model) newModifyModel() *tasklist.Model {
	lm := tasklist.New(m.modifyTasks, tasklist.ModeModify)
	lm.OnQuit = func(r tasklist.Result) tea.Cmd {
		return func() tea.Msg { return tasklist.ListViewDoneMsg{Result: r} }
	}
	lm.OnDelete = func(t *taskwarrior.Task) tea.Cmd {
		return func() tea.Msg { return tasklist.ListDeleteMsg{Task: t} }
	}
	return &lm
}

func (m *Model) View() string {
	// Tab bar
	var tabs []string
	for i, name := range tabNames {
		if i == m.active {
			tabs = append(tabs, tabStyleActive.Render(name))
		} else {
			tabs = append(tabs, tabStyleInactive.Render(name))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	divider := lipgloss.NewStyle().Foreground(styles.Muted).
		Render(strings.Repeat("─", max(m.width, 20)))

	// Content area — constrain height so tab bar stays visible (add form can overflow)
	contentH := m.height - 5 // tab bar, divider, blank, status
	if contentH < 2 {
		contentH = 2
	}
	contentStyle := lipgloss.NewStyle().MaxHeight(contentH)

	var content string
	switch m.active {
	case tabAdd:
		if m.addForm != nil {
			content = contentStyle.Render(m.addForm.View())
		}
	case tabList:
		if m.listModel != nil {
			content = contentStyle.Render(m.listModel.View())
		} else {
			content = contentStyle.Render(styles.MutedText.Render("  Loading…"))
		}
	case tabComplete:
		if m.completeModel != nil {
			content = contentStyle.Render(m.completeModel.View())
		} else {
			content = contentStyle.Render(styles.MutedText.Render("  Loading…"))
		}
	case tabModify:
		if m.modifyForm != nil {
			content = contentStyle.Render(m.modifyForm.View())
		} else if m.modifyModel != nil {
			content = contentStyle.Render(m.modifyModel.View())
		} else {
			content = contentStyle.Render(styles.MutedText.Render("  Loading…"))
		}
	case tabCalendar:
		if m.calendarModel != nil {
			content = contentStyle.Render(m.calendarModel.View())
		} else {
			content = contentStyle.Render(styles.MutedText.Render("  Loading…"))
		}
	}

	// Status line
	statusLine := ""
	if m.statusMsg != "" {
		if m.statusErr {
			statusLine = styles.Error.Render("  "+m.statusMsg) + "\n"
		} else {
			statusLine = styles.Success.Render("  "+m.statusMsg) + "\n"
		}
	}
	statusLine += styles.MutedText.Render("  Tab/Shift+Tab: switch tab  ·  Alt+1-5: jump to tab  ·  q: quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		tabBar,
		divider,
		content,
		"",
		statusLine,
	)
}
