package commands

import (
	"context"
	"fmt"
	"strconv"

	"github.com/saddatahmad19/taskd/internal/taskwarrior"
	"github.com/saddatahmad19/taskd/internal/ui/add"
	"github.com/saddatahmad19/taskd/internal/ui/styles"
	"github.com/saddatahmad19/taskd/internal/ui/tasklist"
	"github.com/spf13/cobra"
)

func newModifyCmd(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modify [id|uuid]",
		Short: "Edit a task — pick from the list or pass an ID directly",
		Long: `Interactively modify an existing task.

Without arguments:
  Opens the full-screen task list. Navigate to the task you want and press Enter.
  The add wizard opens, pre-populated with the task's current values.
  Modify any field (or leave it unchanged) and confirm. Press d to delete a task.

With an argument (ID or UUID):
  Skips the list picker and opens the edit form immediately for that task.

Fields that can be edited:
  Description, Tag, Project, Priority, Due date`,

		Example: `  taskd modify           # list picker then edit form
  taskd modify 5         # open edit form for task 5
  taskd modify abc-1234  # open edit form by UUID prefix`,

		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModify(cmd.Context(), d, args)
		},
	}
	return cmd
}

func runModify(ctx context.Context, d *deps, args []string) error {
	// ── Resolve the target task ───────────────────────────────────────────────
	var target *taskwarrior.Task

	if len(args) == 1 {
		// Direct: resolve numeric ID or UUID from arguments.
		t, err := resolveTask(ctx, d, args[0])
		if err != nil {
			return err
		}
		target = t
	} else {
		// Interactive: open the list picker.
		tasks, err := d.tw.Export(ctx, taskwarrior.Filter{Status: "pending"})
		if err != nil {
			return fmt.Errorf("fetch tasks: %w", err)
		}
		if len(tasks) == 0 {
			fmt.Println()
			fmt.Println(styles.MutedText.Render("  No pending tasks found."))
			fmt.Println()
			return nil
		}

		result, err := tasklist.RunWithDelete(ctx, d.tw, taskwarrior.Filter{Status: "pending"}, tasklist.ModeModify)
		if err != nil {
			return err
		}
		if result.Aborted || result.SelectedTask == nil {
			fmt.Println(styles.MutedText.Render("  Aborted — no task modified."))
			return nil
		}
		target = result.SelectedTask
	}

	// ── Open the edit form pre-populated with the task's current values ───────
	defaults := add.DefaultsFromTask(*target)
	result, err := add.RunWizardWithDefaults(ctx, d.tw, defaults)
	if err != nil {
		if isAbortError(err) {
			fmt.Println(styles.MutedText.Render("  Aborted — task unchanged."))
			return nil
		}
		return fmt.Errorf("modify form: %w", err)
	}

	req := result.ToAddRequest()
	if err := d.tw.Modify(ctx, target.UUID, req); err != nil {
		return fmt.Errorf("modify task: %w", err)
	}

	fmt.Println()
	fmt.Println(styles.Success.Render("  ✓ Task modified!"))
	fmt.Printf("  %s  %s\n\n",
		styles.MutedText.Render("Description:"),
		styles.TaskDesc.Render(result.Description),
	)
	return nil
}

func resolveTask(ctx context.Context, d *deps, arg string) (*taskwarrior.Task, error) {
	tasks, err := d.tw.Export(ctx, taskwarrior.Filter{Status: "pending"})
	if err != nil {
		return nil, fmt.Errorf("fetch tasks: %w", err)
	}

	// Try numeric ID first.
	if id, err := strconv.Atoi(arg); err == nil {
		for _, t := range tasks {
			if t.ID == id {
				found := t
				return &found, nil
			}
		}
		return nil, fmt.Errorf("no pending task with ID %d", id)
	}

	// Try UUID (full or prefix).
	for _, t := range tasks {
		if t.UUID == arg || len(arg) >= 8 && t.UUID[:8] == arg[:8] {
			found := t
			return &found, nil
		}
	}
	return nil, fmt.Errorf("no pending task matching %q", arg)
}
