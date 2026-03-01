package commands

import (
	"context"

	"github.com/saddatahmad19/taskd/internal/ui/fullui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func newFullUICmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "full-ui",
		Short: "Full TUI with tabs for add, list, complete, modify, and calendar",
		Long: `Launches a tabbed full-screen TUI combining all taskd features.

Tabs:
  Add      — Add new tasks (stays open after each add for another)
  List     — Browse and filter tasks
  Complete — Mark tasks as done (toggle with Space, confirm with Enter)
  Modify   — Edit tasks (select one, then edit in the form)
  Calendar — Calendar view of due dates (s for overdue list, d to delete)

Navigation:
  Tab / Shift+Tab    cycle through tabs
  Alt+1 .. Alt+5     jump to tab (may not work in all terminals)
  q / Ctrl+C         quit`,

		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFullUI(cmd.Context(), d)
		},
	}
}

func runFullUI(ctx context.Context, d *deps) error {
	m := fullui.New(ctx, d.tw)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
