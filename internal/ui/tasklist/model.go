package tasklist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/saddatahmad19/taskd/internal/taskwarrior"
	"github.com/saddatahmad19/taskd/internal/ui/styles"
)

type Mode int

const (
	// ModeList is a read-only browse mode.
	ModeList Mode = iota
	// ModeComplete lets the user mark tasks done (Space to toggle, Enter to confirm).
	ModeComplete
	// ModeModify lets the user pick a task to edit (Enter to select).
	ModeModify
)

type Result struct {
	// SelectedTask is the task the user chose in ModeModify.
	SelectedTask *taskwarrior.Task
	// CompletedUUIDs holds UUIDs to mark done in ModeComplete.
	CompletedUUIDs []string
	// Aborted is true when the user pressed q/Esc/Ctrl+C without confirming.
	Aborted bool
}

// ListViewDoneMsg is sent when the list exits in embedded mode (e.g. full-ui).
// The parent receives this and can process the result without quitting.
type ListViewDoneMsg struct {
	Result Result
}

var (
	keyQuit = key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	)
	keyForceQuit = key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	)
	keyToggleSelect = key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	)
	keySelectAll = key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "select all"),
	)
	keyConfirm = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "confirm"),
	)
	keyEdit = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "edit task"),
	)
)


type Model struct {
	list     list.Model
	delegate *taskDelegate // pointer — shared with list's internal delegate
	tasks    []taskwarrior.Task
	mode     Mode
	result   Result
	width    int
	height   int
	quitting bool

	// OnQuit, when non-nil, is called instead of tea.Quit when the list exits.
	// Used when embedding the list in a parent (e.g. full-ui).
	OnQuit func(Result) tea.Cmd
}

func New(tasks []taskwarrior.Task, mode Mode) Model {
	delegate := newTaskDelegate(mode)

	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = Item{Task: t}
	}

	l := list.New(items, delegate, 0, 0)
	l.Title = modeTitle(mode)
	l.SetFilteringEnabled(true)
	l.Filter = taskFilter
	l.SetShowStatusBar(true)
	l.SetShowPagination(true)
	l.SetStatusBarItemName("task", "tasks")

	// ── Styles ────────────────────────────────────────────────────────────────
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(styles.Violet).
		Bold(true).
		Background(styles.Surface).
		Padding(0, 1)

	l.Styles.TitleBar = lipgloss.NewStyle().
		Background(styles.Surface).
		Padding(0, 0, 1, 0)

	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(styles.Mint)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(styles.Violet)

	l.Styles.StatusBar = lipgloss.NewStyle().
		Foreground(styles.Muted).
		PaddingLeft(1)

	l.Styles.StatusEmpty = lipgloss.NewStyle().
		Foreground(styles.Muted).
		Italic(true).
		PaddingLeft(1)

	l.Styles.NoItems = lipgloss.NewStyle().
		Foreground(styles.Muted).
		Italic(true).
		PaddingLeft(4).
		PaddingTop(2)

	l.FilterInput.Placeholder = "filter by description, tag:foo, project:bar, priority:H, due:today…"
	l.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(styles.Mint)
	l.FilterInput.TextStyle = lipgloss.NewStyle().Foreground(styles.Text)

	// Disable the list's built-in quit so we control it ourselves.
	l.KeyMap.Quit.SetEnabled(false)
	l.KeyMap.ForceQuit.SetEnabled(false)

	// Help keys — override defaults to show our bindings.
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return shortHelpKeys(mode)
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return fullHelpKeys(mode)
	}

	return Model{
		list:     l,
		delegate: delegate,
		tasks:    tasks,
		mode:     mode,
	}
}

func (m Model) Init() tea.Cmd { return nil }

// SetSize updates the list dimensions (used when embedded in full-ui).
func (m *Model) SetSize(width, height int) {
	m.list.SetSize(width, height-2)
}

func (m Model) quitCmd() tea.Cmd {
	if m.OnQuit != nil {
		return m.OnQuit(m.result)
	}
	return tea.Quit
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve two lines for our status bar at the very bottom.
		m.list.SetSize(msg.Width, msg.Height-2)
		return m, nil

	case tea.KeyMsg:
		// While the filter input is active, pass all keys straight to the list.
		if m.list.SettingFilter() {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		switch {
		// ── Quit ────────────────────────────────────────────────────────────
		case key.Matches(msg, keyForceQuit):
			m.result.Aborted = true
			m.quitting = true
			return m, m.quitCmd()

		case key.Matches(msg, keyQuit):
			// If a filter is applied, q clears the filter first.
			if m.list.FilterState() == list.FilterApplied {
				m.list.ResetFilter()
				return m, nil
			}
			m.result.Aborted = true
			m.quitting = true
			return m, m.quitCmd()

		case msg.String() == "esc":
			if m.list.FilterState() == list.FilterApplied {
				m.list.ResetFilter()
				return m, nil
			}
			m.result.Aborted = true
			m.quitting = true
			return m, m.quitCmd()

		// ── Complete mode: toggle current item ───────────────────────────────
		case key.Matches(msg, keyToggleSelect) && m.mode == ModeComplete:
			if it, ok := m.list.SelectedItem().(Item); ok {
				uuid := it.Task.UUID
				m.delegate.selected[uuid] = !m.delegate.selected[uuid]
			}
			return m, nil

		// ── Complete mode: toggle all items ──────────────────────────────────
		case key.Matches(msg, keySelectAll) && m.mode == ModeComplete:
			m.toggleAll()
			return m, nil

		// ── Enter: mode-specific action ───────────────────────────────────────
		case msg.String() == "enter":
			return m.handleEnter()
		}
	}

	// Default: let the bubbles list handle navigation, filter toggling, etc.
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.mode {

	case ModeModify:
		// Return the selected task and quit so the command layer can open the form.
		if it, ok := m.list.SelectedItem().(Item); ok {
			t := it.Task
			m.result.SelectedTask = &t
			m.quitting = true
			return m, m.quitCmd()
		}

	case ModeComplete:
		// Collect everything that has been checked.
		for uuid, checked := range m.delegate.selected {
			if checked {
				m.result.CompletedUUIDs = append(m.result.CompletedUUIDs, uuid)
			}
		}
		// If nothing was checked, complete the currently highlighted item.
		if len(m.result.CompletedUUIDs) == 0 {
			if it, ok := m.list.SelectedItem().(Item); ok {
				m.result.CompletedUUIDs = []string{it.Task.UUID}
			}
		}
		if len(m.result.CompletedUUIDs) > 0 {
			m.quitting = true
			return m, m.quitCmd()
		}

	case ModeList:
		// Nothing to do in browse mode.
	}

	return m, nil
}

func (m *Model) toggleAll() {
	visible := m.list.VisibleItems()
	allSelected := true
	for _, it := range visible {
		if item, ok := it.(Item); ok {
			if !m.delegate.selected[item.Task.UUID] {
				allSelected = false
				break
			}
		}
	}
	for _, it := range visible {
		if item, ok := it.(Item); ok {
			m.delegate.selected[item.Task.UUID] = !allSelected
		}
	}
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.list.View(),
		m.renderHint(),
	)
}

func (m Model) renderHint() string {
	var line string
	switch m.mode {
	case ModeComplete:
		n := 0
		for _, v := range m.delegate.selected {
			if v {
				n++
			}
		}
		if n > 0 {
			line = styles.Success.Render(fmt.Sprintf("  %d selected  ", n)) +
				styles.MutedText.Render("· space toggle  a select-all  enter confirm  q quit")
		} else {
			line = styles.MutedText.Render("  space: toggle  a: select all  enter: confirm current  q: quit")
		}
	case ModeModify:
		line = styles.MutedText.Render("  enter: open edit form  /: filter  q: quit")
	default:
		line = styles.MutedText.Render("  /: filter  q: quit")
	}

	// Append filter indicator if a filter is active.
	if m.list.FilterState() == list.FilterApplied {
		applied := lipgloss.NewStyle().Foreground(styles.Mint).Render(
			fmt.Sprintf("  [filter: %q]", m.list.FilterValue()),
		)
		line = strings.TrimRight(line, " ") + applied
	}

	return line
}

func (m Model) GetResult() Result { return m.result }


func Run(tasks []taskwarrior.Task, mode Mode) (Result, error) {
	if len(tasks) == 0 {
		return Result{Aborted: true}, nil
	}
	m := New(tasks, mode)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return Result{Aborted: true}, fmt.Errorf("tasklist TUI: %w", err)
	}
	fm, ok := final.(Model)
	if !ok {
		return Result{Aborted: true}, fmt.Errorf("tasklist: unexpected model type from tea.Program")
	}
	return fm.GetResult(), nil
}


func modeTitle(mode Mode) string {
	switch mode {
	case ModeComplete:
		return "✦  Complete Tasks"
	case ModeModify:
		return "✦  Modify Task — choose one"
	default:
		return "✦  Tasks"
	}
}

func shortHelpKeys(mode Mode) []key.Binding {
	switch mode {
	case ModeComplete:
		return []key.Binding{keyToggleSelect, keySelectAll, keyConfirm, keyQuit}
	case ModeModify:
		return []key.Binding{keyEdit, keyQuit}
	default:
		return []key.Binding{keyQuit}
	}
}

func fullHelpKeys(mode Mode) []key.Binding {
	base := []key.Binding{keyQuit, keyForceQuit}
	switch mode {
	case ModeComplete:
		return append([]key.Binding{keyToggleSelect, keySelectAll, keyConfirm}, base...)
	case ModeModify:
		return append([]key.Binding{keyEdit}, base...)
	default:
		return base
	}
}
