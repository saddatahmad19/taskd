package commands

import (
	"context"
	"fmt"
	"strconv"

	"github.com/saddatahmad19/taskd/internal/taskwarrior"
	"github.com/saddatahmad19/taskd/internal/ui/styles"
	"github.com/saddatahmad19/taskd/internal/ui/tasklist"
	"github.com/spf13/cobra"
)

func newCompleteCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "complete [id|uuid ...]",
		Short: "Mark tasks as done — interactively or by ID",
		Long: `Mark one or more tasks as completed.

Without arguments:
  Opens the full-screen task list. Use Space to toggle tasks, Enter to confirm.
  You can also filter first (/), then select visible items with 'a' (select all).

With arguments (IDs or UUIDs):
  Marks those tasks done immediately without opening the TUI.

Navigation in TUI:
  ↑/↓  scroll      Space  toggle      a  select all
  /    filter       Enter  confirm     d  delete     q  quit`,

		Example: `  taskd complete           # interactive picker
  taskd complete 3          # mark task 3 done directly
  taskd complete 3 5 7      # mark tasks 3, 5, 7 done directly`,

		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runComplete(cmd.Context(), d, args)
		},
	}
}

func runComplete(ctx context.Context, d *deps, args []string) error {
	// ── Direct mode: IDs/UUIDs provided on the command line ──────────────────
	if len(args) > 0 {
		return completeByArgs(ctx, d, args)
	}

	// ── Interactive mode: open TUI picker ────────────────────────────────────
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

	result, err := tasklist.RunWithDelete(ctx, d.tw, taskwarrior.Filter{Status: "pending"}, tasklist.ModeComplete)
	if err != nil {
		return err
	}
	if result.Aborted || len(result.CompletedUUIDs) == 0 {
		fmt.Println(styles.MutedText.Render("  Aborted — no tasks marked done."))
		return nil
	}

	return completeUUIDs(ctx, d, result.CompletedUUIDs)
}

func completeByArgs(ctx context.Context, d *deps, args []string) error {
	// Load all pending tasks so we can resolve numeric IDs to UUIDs.
	tasks, err := d.tw.Export(ctx, taskwarrior.Filter{Status: "pending"})
	if err != nil {
		return fmt.Errorf("fetch tasks: %w", err)
	}

	// Build lookup maps.
	byID := make(map[int]taskwarrior.Task, len(tasks))
	byUUID := make(map[string]taskwarrior.Task, len(tasks))
	for _, t := range tasks {
		byID[t.ID] = t
		byUUID[t.UUID] = t
	}

	var uuids []string
	for _, arg := range args {
		if id, err := strconv.Atoi(arg); err == nil {
			if t, ok := byID[id]; ok {
				uuids = append(uuids, t.UUID)
			} else {
				fmt.Println(styles.Warning.Render(fmt.Sprintf("  ⚠  Task ID %d not found — skipping", id)))
			}
		} else {
			if t, ok := byUUID[arg]; ok {
				uuids = append(uuids, t.UUID)
			} else {
				fmt.Println(styles.Warning.Render(fmt.Sprintf("  ⚠  UUID %q not found — skipping", arg)))
			}
		}
	}

	if len(uuids) == 0 {
		return fmt.Errorf("no valid task identifiers found in arguments")
	}
	return completeUUIDs(ctx, d, uuids)
}

func completeUUIDs(ctx context.Context, d *deps, uuids []string) error {
	fmt.Println()
	var failed int
	for _, uuid := range uuids {
		if err := d.tw.Complete(ctx, uuid); err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("  ✗ %s: %s", uuid[:8], err)))
			failed++
		} else {
			fmt.Println(styles.Success.Render(fmt.Sprintf("  ✓ %s  marked done", uuid[:8])))
		}
	}
	fmt.Println()
	if failed == len(uuids) {
		return fmt.Errorf("all %d task(s) failed to complete", failed)
	}
	fmt.Printf(styles.MutedText.Render("  %d/%d task(s) completed\n"), len(uuids)-failed, len(uuids))
	fmt.Println()
	return nil
}
