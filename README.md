# taskd

A production-quality Taskwarrior companion CLI built with Go and the [Charm](https://charm.sh) ecosystem — featuring Bubble Tea event loops, Huh interactive forms, and Lipgloss styled output.

---

## Requirements

| Dependency                                       | Version |
| ------------------------------------------------ | ------- |
| Go                                               | ≥ 1.21  |
| [Taskwarrior](https://taskwarrior.org/download/) | ≥ 2.6   |

Taskwarrior must be installed and `task` must be available in your `$PATH`.

---

## Installation

### From source (recommended)

```bash
git clone https://github.com/saddatahmad19/taskd.git
cd taskd
make install          # installs to $GOPATH/bin/taskd
```

Or without `make`:

```bash
go install -ldflags "-X github.com/saddatahmad19/taskd/internal/commands.Version=dev" \
    ./cmd/taskd
```

### Build locally (without installing)

```bash
make build            # produces ./bin/taskd
./bin/taskd --help
```

---

## Usage

```
taskd --help                        # help overview
taskd version                       # print taskd + Taskwarrior versions

taskd add                           # interactive task creation wizard
taskd add --dry-run                 # preview without saving

taskd list                          # list pending tasks (simple output)
taskd list --project myapp          # filter by project
taskd list --tag urgent --tag work  # filter by tags (AND)
taskd list --status completed -n 10 # last 10 completed tasks

taskd complete                      # (stub) mark task as done
taskd modify                        # (stub) edit an existing task
```

### Global flag

```
--binary string   Path to the task binary (default: auto-detected via PATH)
```

---

## Architecture Decisions

### 1. Taskwarrior Integration: CLI over data files

`taskd` interacts with Taskwarrior exclusively through its CLI (`task export`, `task add`, etc.) rather than reading `~/.task/*.data` files directly.

**Reasons:**

- **Correctness** — writes go through Taskwarrior's own validation, hook pipeline, and UDA handling. Direct file writes would bypass all of these.
- **Stability** — `task export` emits a stable JSON contract regardless of internal data format changes between Taskwarrior versions.
- **Configuration respect** — Taskwarrior applies the user's `~/.taskrc` settings (urgency coefficients, reports, aliases) automatically.
- **Future-proof** — if TaskServer sync is enabled, all changes automatically propagate.

### 2. `Client` interface for the data layer

All Taskwarrior access goes through the `taskwarrior.Client` interface. The `CLIClient` struct is the only current implementation, but `MockClient` is provided for tests. A future implementation could target the TaskServer API or read JSON data files directly for offline/embedded use.

The `deps` struct in `commands/root.go` is the dependency injection container; commands receive `*deps` and interact with Taskwarrior only through `d.tw`. This makes every command unit-testable without spawning a real `task` process.

### 3. Huh for forms, Bubble Tea for everything else

[Huh](https://github.com/charmbracelet/huh) is itself built on Bubble Tea and handles all form state (cursor, filtering, validation). For the `add` wizard, calling `form.Run()` is idiomatic — it's equivalent to running a `tea.Program`.

Commands that need a _persistent_ interactive TUI (the future `list` and `complete` pickers) will use raw Bubble Tea models in their own `ui/<command>/` packages, following the same pattern the `add` wizard establishes.

### 4. Multi-step wizard composition

Rather than one giant `huh.NewForm(...)` call, `RunWizard` breaks the flow into **distinct form runs**:

1. Description (required, validated immediately)
2. Tag select-or-create (two-phase: select → conditional free-text input)
3. Project select-or-create (same pattern)
4. Priority + due date (grouped on one screen)

This keeps each step focused, makes the "(enter new value)" flow natural — an option triggers a follow-up `huh.NewInput` form — and allows future steps to be inserted, reordered, or made conditional without rewriting the whole form.

### 5. Cobra with PersistentPreRunE for dependency injection

`PersistentPreRunE` in `root.go` initialises the `taskwarrior.Client` once before any sub-command runs. This means:

- Commands never construct their own clients (no scattered `NewCLIClient` calls).
- The `--binary` flag is available globally and is respected everywhere.
- Commands that don't need Taskwarrior (e.g., `version`) opt out via an annotation.

---

## Extending taskd

### Add a new command

1. Create `internal/commands/mycommand.go`:

```go
package commands

import (
    "github.com/spf13/cobra"
)

func newMyCmd(d *deps) *cobra.Command {
    return &cobra.Command{
        Use:   "mycommand",
        Short: "Does something useful",
        RunE: func(cmd *cobra.Command, args []string) error {
            // d.tw is fully initialised here.
            tasks, err := d.tw.Export(cmd.Context(), taskwarrior.Filter{})
            // ...
            return nil
        },
    }
}
```

2. Register it in `root.go`:

```go
rootCmd.AddCommand(
    // existing commands …
    newMyCmd(d),
)
```

### Add a new interactive TUI view

Create a package `internal/ui/myview/model.go` that implements the standard Bubble Tea model:

```go
package myview

import tea "github.com/charmbracelet/bubbletea"

type Model struct { /* your state */ }

func (m Model) Init() tea.Cmd                           { return nil }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { /* handle keys */ return m, nil }
func (m Model) View() string                            { return "rendered view" }

func Run(/* deps */) error {
    m := Model{}
    p := tea.NewProgram(m, tea.WithAltScreen())
    _, err := p.Run()
    return err
}
```

Then call `myview.Run(...)` from your command's `RunE`.

### Mock Taskwarrior in tests

```go
func TestMyCommand(t *testing.T) {
    tw := &taskwarrior.MockClient{
        ExportFn: func(_ context.Context, _ taskwarrior.Filter) ([]taskwarrior.Task, error) {
            return []taskwarrior.Task{
                {ID: 1, Description: "Test task", Status: "pending"},
            }, nil
        },
    }
    d := &deps{tw: tw}
    // invoke runMyCommand(context.Background(), d, ...) directly
}
```

---

## Running Tests

```bash
make test
# or
go test ./... -v
```

---

## Contributing

PRs welcome. Priority targets for first contributions:

- [ ] `taskd complete` — interactive Bubble Tea fuzzy picker with multi-select
- [ ] `taskd modify` — pre-populated Huh form reusing the add wizard
- [ ] `taskd list` — full Bubble Tea list model (viewport, sort, column widths)
- [ ] Shell completion (`taskd completion bash|zsh|fish`)
- [ ] Config file support (`~/.config/taskd/config.yaml`)
