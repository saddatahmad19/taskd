package add

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/saddatahmad19/taskd/internal/taskwarrior"
	"github.com/saddatahmad19/taskd/internal/ui/styles"
)

const (
	sentinelNone = "__none__"
	sentinelNew  = "__new__"
)

// ── Result & defaults ─────────────────────────────────────────────────────────

// FormResult holds the validated values collected by the form wizard.
type FormResult struct {
	Description string
	Tag         string
	Project     string
	Priority    taskwarrior.Priority
	Due         *time.Time
}

func (r FormResult) ToAddRequest() taskwarrior.AddRequest {
	req := taskwarrior.AddRequest{
		Description: r.Description,
		Priority:    r.Priority,
		Project:     r.Project,
		Due:         r.Due,
	}
	if r.Tag != "" {
		req.Tags = []string{r.Tag}
	}
	return req
}

// Defaults pre-populates the form when editing an existing task.
type Defaults struct {
	Description string
	Tag         string
	Project     string
	Priority    taskwarrior.Priority
	Due         *time.Time
}

// DefaultsFromTask extracts Defaults from an existing task for the edit form.
func DefaultsFromTask(t taskwarrior.Task) Defaults {
	d := Defaults{
		Description: t.Description,
		Project:     t.Project,
		Priority:    t.Priority,
	}
	if len(t.Tags) > 0 {
		d.Tag = t.Tags[0]
	}
	if due := t.DueTime(); due != nil {
		v := *due
		d.Due = &v
	}
	return d
}

// ── Bubble Tea wrapper model ──────────────────────────────────────────────────

// formModel wraps a huh.Form and renders a persistent header above it.
// It is run with tea.WithAltScreen() for a full-screen experience.
type formModel struct {
	form   *huh.Form
	title  string
	width  int
	height int
}

// headerLines is the number of lines consumed by the rendered header.
// Matches the lines produced by formModel.header().
const headerLines = 4 // title + hint + divider + blank gap

func (m formModel) Init() tea.Cmd { return m.form.Init() }

func (m formModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
		m.form = m.form.
			WithWidth(m.width).
			WithHeight(m.height - headerLines)
		return m, nil
	}
	f, cmd := m.form.Update(msg)
	m.form = f.(*huh.Form)
	return m, cmd
}

func (m formModel) View() string {
	if m.width == 0 {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.header(), m.form.View())
}

func (m formModel) header() string {
	titleBar := lipgloss.NewStyle().
		Background(styles.Surface).
		Width(m.width).
		Padding(0, 2).
		Render(
			lipgloss.NewStyle().
				Foreground(styles.Violet).
				Bold(true).
				Render(m.title),
		)

	hint := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Italic(true).
		PaddingLeft(2).
		Render("Tab / ↑↓ navigate  ·  Enter confirm  ·  Ctrl+C abort")

	divider := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Render(strings.Repeat("─", m.width))

	return lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		hint,
		divider,
		"",
	)
}

// ── Public API ────────────────────────────────────────────────────────────────

// RunWizard launches the full-screen add-task wizard and returns the result.
func RunWizard(ctx context.Context, tw taskwarrior.Client) (*FormResult, error) {
	return RunWizardWithDefaults(ctx, tw, Defaults{})
}

// RunWizardWithDefaults launches the wizard pre-populated with def.
func RunWizardWithDefaults(ctx context.Context, tw taskwarrior.Client, def Defaults) (*FormResult, error) {
	existingTags, _ := tw.Tags(ctx)
	existingProjects, _ := tw.Projects(ctx)

	// Variables captured by pointer into the huh fields.
	description := def.Description
	tagChoice := initialChoice(def.Tag, existingTags)
	newTag := ""
	projectChoice := initialChoice(def.Project, existingProjects)
	newProject := ""
	priority := string(def.Priority)
	dueStr := ""
	if def.Due != nil {
		dueStr = def.Due.Format("2006-01-02")
	}

	descHint := "What needs to be done?"
	if def.Description != "" {
		descHint = "Edit the description — or leave it unchanged and press Enter."
	}

	form := huh.NewForm(
		// Step 1 — description
		huh.NewGroup(
			huh.NewInput().
				Title("Description").
				Description(descHint).
				Placeholder("e.g. Write unit tests for auth module").
				Value(&description).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("description cannot be empty")
					}
					return nil
				}),
		),

		// Step 2 — tag select
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Tag").
				Description("Choose an existing tag or add a new one.").
				Options(buildOptions(existingTags, "tag")...).
				Value(&tagChoice).
				Filtering(true),
		),

		// Step 3 — new tag name (shown only when "Add new tag…" was chosen)
		huh.NewGroup(
			huh.NewInput().
				Title("New tag name").
				Description("Enter a name for the new tag, or leave blank to skip.").
				Placeholder("e.g. frontend").
				Value(&newTag),
		).WithHideFunc(func() bool { return tagChoice != sentinelNew }),

		// Step 4 — project select
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Project").
				Description("Choose an existing project or add a new one.").
				Options(buildOptions(existingProjects, "project")...).
				Value(&projectChoice).
				Filtering(true),
		),

		// Step 5 — new project name (shown only when "Add new project…" was chosen)
		huh.NewGroup(
			huh.NewInput().
				Title("New project name").
				Description("Enter a name for the new project, or leave blank to skip.").
				Placeholder("e.g. myapp").
				Value(&newProject),
		).WithHideFunc(func() bool { return projectChoice != sentinelNew }),

		// Step 6 — priority + due date
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Priority").
				Description("Task urgency level.").
				Options(
					huh.NewOption[string]("None", string(taskwarrior.PriorityNone)),
					huh.NewOption[string]("Low  (L)", string(taskwarrior.PriorityLow)),
					huh.NewOption[string]("Medium (M)", string(taskwarrior.PriorityMedium)),
					huh.NewOption[string]("High (H)", string(taskwarrior.PriorityHigh)),
				).
				Value(&priority),

			huh.NewInput().
				Title("Due date").
				Description("YYYY-MM-DD, 'today', 'tomorrow', 'eow', 'eom' — or leave blank.").
				Placeholder("2026-12-31").
				Value(&dueStr).
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					if _, err := parseDue(s); err != nil {
						return fmt.Errorf("invalid date: %s (use YYYY-MM-DD or keyword)", err)
					}
					return nil
				}),
		),
	).WithTheme(taskdTheme())

	// SubmitCmd and CancelCmd are only wired up inside huh's own Run(); since
	// we embed the form in our own tea.Program we must set them manually so the
	// program actually quits when the form completes or is aborted.
	form.SubmitCmd = tea.Quit
	form.CancelCmd = tea.Quit

	title := "✦  Add Task"
	if def.Description != "" {
		title = "✦  Edit Task"
	}

	m := formModel{form: form, title: title}
	p := tea.NewProgram(m, tea.WithAltScreen())
	fm, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("form: %w", err)
	}

	final := fm.(formModel)
	if final.form.State == huh.StateAborted {
		return nil, huh.ErrUserAborted
	}

	tag := resolveChoice(tagChoice, newTag)
	project := resolveChoice(projectChoice, newProject)

	var due *time.Time
	if dueStr != "" {
		if parsed, parseErr := parseDue(dueStr); parseErr == nil {
			due = &parsed
		}
	}

	return &FormResult{
		Description: strings.TrimSpace(description),
		Tag:         tag,
		Project:     project,
		Priority:    taskwarrior.Priority(priority),
		Due:         due,
	}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// initialChoice returns the sentinel value to pre-select in a tag/project
// dropdown: the current value if it exists in the list, otherwise sentinelNone.
func initialChoice(current string, existing []string) string {
	if current == "" {
		return sentinelNone
	}
	for _, v := range existing {
		if v == current {
			return current
		}
	}
	return sentinelNone
}

// resolveChoice maps a select sentinel value to the final string.
func resolveChoice(choice, newVal string) string {
	switch choice {
	case sentinelNone, "":
		return ""
	case sentinelNew:
		return strings.TrimSpace(newVal)
	default:
		return choice
	}
}

// buildOptions constructs the option list for a tag or project select field.
func buildOptions(existing []string, label string) []huh.Option[string] {
	opts := make([]huh.Option[string], 0, len(existing)+2)
	opts = append(opts, huh.NewOption[string](fmt.Sprintf("(no %s)", label), sentinelNone))
	for _, v := range existing {
		opts = append(opts, huh.NewOption[string](v, v))
	}
	opts = append(opts, huh.NewOption[string](fmt.Sprintf("+ Add new %s…", label), sentinelNew))
	return opts
}

// ── Date parsing ──────────────────────────────────────────────────────────────

func parseDue(s string) (time.Time, error) {
	now := time.Now()
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "today":
		return truncateDay(now), nil
	case "tomorrow":
		return truncateDay(now.AddDate(0, 0, 1)), nil
	case "eow", "end of week":
		d := (7 - int(now.Weekday())) % 7
		if d == 0 {
			d = 7
		}
		return truncateDay(now.AddDate(0, 0, d)), nil
	case "eom", "end of month":
		return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1), nil
	}
	formats := []string{
		"2006-01-02", "01/02/2006", "02-01-2006", "January 2, 2006", "Jan 2, 2006",
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, now.Location()); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised date %q", s)
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// ── Theme ─────────────────────────────────────────────────────────────────────

func taskdTheme() *huh.Theme {
	theme := huh.ThemeCharm()
	theme.Focused.Title = theme.Focused.Title.Foreground(styles.Violet).Bold(true)
	theme.Focused.SelectedOption = theme.Focused.SelectedOption.Foreground(styles.Mint).Bold(true)
	theme.Focused.SelectSelector = theme.Focused.SelectSelector.Foreground(styles.Indigo)
	theme.Focused.Description = theme.Focused.Description.Foreground(styles.Muted).Italic(true)
	theme.Blurred.Title = theme.Blurred.Title.Foreground(styles.Muted)
	return theme
}
