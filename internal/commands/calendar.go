package commands

import (
	"context"

	"github.com/saddatahmad19/taskd/internal/ui/calendar"
	"github.com/spf13/cobra"
)

func newCalendarCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "calendar",
		Short: "Calendar view of tasks with due dates",
		Long: `Shows a calendar of tasks that have due dates.

Features:
  - Only tasks with a due date are included
  - Past-due tasks appear in a separate section (press s to view)
  - Press s to switch between calendar view and overdue list
  - In overdue list: d to delete a task

Navigation:
  ↑↓←→    navigate between dates
  n/p     next / previous month (or l/h)
  t       jump to today
  s       switch to overdue list / back to calendar
  q       quit`,

		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCalendar(cmd.Context(), d)
		},
	}
}

func runCalendar(ctx context.Context, d *deps) error {
	return calendar.Run(ctx, d.tw)
}
