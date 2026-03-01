package commands

import (
	"context"
	"fmt"

	"github.com/saddatahmad19/taskd/internal/taskwarrior"
	"github.com/saddatahmad19/taskd/internal/ui/styles"
	"github.com/saddatahmad19/taskd/internal/ui/tasklist"
	"github.com/spf13/cobra"
)

func newListCmd(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Browse tasks in a full-screen interactive list",
		Long: `Opens a full-screen, scrollable, filterable task list.

Navigation:
  ↑/↓ or j/k   scroll
  /             enter filter mode
  d             delete selected task
  Esc or q      quit (clears filter first if one is active)

Filter syntax (space-separated terms are ANDed):
  write tests         — description contains "write tests"
  tag:backend         — has tag "backend"
  project:myapp       — belongs to project "myapp"
  priority:H          — priority is High (H/M/L)
  due:today           — due today
  due:tomorrow        — due tomorrow
  due:overdue         — past due
  tag:work priority:H — combined filter`,

		Example: `  taskd list
  taskd list --project myapp
  taskd list --tag urgent --status completed`,

		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := buildFilter(cmd)
			if err != nil {
				return err
			}
			return runList(cmd.Context(), d, f)
		},
	}

	cmd.Flags().StringP("project", "p", "", "Pre-filter by project")
	cmd.Flags().StringArrayP("tag", "t", nil, "Pre-filter by tag (repeatable)")
	cmd.Flags().StringP("status", "s", "pending", "Task status (pending|completed|deleted)")
	cmd.Flags().IntP("limit", "n", 0, "Max tasks to load (0 = no limit)")

	return cmd
}

func buildFilter(cmd *cobra.Command) (taskwarrior.Filter, error) {
	project, _ := cmd.Flags().GetString("project")
	tags, _ := cmd.Flags().GetStringArray("tag")
	status, _ := cmd.Flags().GetString("status")
	limit, _ := cmd.Flags().GetInt("limit")

	valid := map[string]bool{"pending": true, "completed": true, "deleted": true, "all": true}
	if !valid[status] {
		return taskwarrior.Filter{}, fmt.Errorf(
			"invalid status %q — choose one of: pending, completed, deleted, all", status,
		)
	}
	return taskwarrior.Filter{
		Project: project,
		Tags:    tags,
		Status:  status,
		Limit:   limit,
	}, nil
}

func runList(ctx context.Context, d *deps, f taskwarrior.Filter) error {
	tasks, err := d.tw.Export(ctx, f)
	if err != nil {
		return fmt.Errorf("fetch tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println()
		fmt.Println(styles.MutedText.Render("  No tasks match the current filter."))
		fmt.Println()
		return nil
	}

	result, err := tasklist.RunWithDelete(ctx, d.tw, f, tasklist.ModeList)
	if err != nil {
		return err
	}
	_ = result // browse mode has no post-action
	return nil
}
