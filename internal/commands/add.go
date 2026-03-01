package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/saddatahmad19/taskd/internal/ui/add"
	"github.com/saddatahmad19/taskd/internal/ui/styles"
	"github.com/spf13/cobra"
)

func newAddCmd(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Interactively add a new task",
		Long: `Launches a step-by-step TUI form to create a new task.

Fields:
  Description  — required; what needs to be done
  Tag          — optional; choose from existing tags or enter a new one
  Project      — optional; choose from existing projects or enter a new one
  Priority     — optional; None / Low / Medium / High
  Due date     — optional; YYYY-MM-DD, 'today', 'tomorrow', 'eow', 'eom'

Press Enter on any optional field to skip it.
Ctrl+C or Esc at any point to abort without saving.`,

		Example: `  taskd add                  # fully interactive
  taskd add --dry-run        # preview without saving`,

		RunE: func(cmd *cobra.Command, _ []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			return runAdd(cmd.Context(), d, dryRun)
		},
	}

	cmd.Flags().Bool("dry-run", false, "Print the task that would be created without saving")

	return cmd
}

func runAdd(ctx context.Context, d *deps, dryRun bool) error {
	// Run the interactive wizard — this is the Huh/Bubble Tea layer.
	result, err := add.RunWizard(ctx, d.tw)
	if err != nil {
		// huh returns a specific error when the user presses Ctrl+C.
		if isAbortError(err) {
			fmt.Println(styles.MutedText.Render("  Aborted — no task created."))
			return nil
		}
		return fmt.Errorf("wizard: %w", err)
	}

	req := result.ToAddRequest()

	if dryRun {
		printDryRun(result)
		return nil
	}

	output, err := d.tw.Add(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to add task: %w", err)
	}

	printSuccess(result, output)
	return nil
}

func isAbortError(err error) bool {
	// huh v0.4 returns huh.ErrUserAborted on Ctrl+C.
	// We check by error string to avoid hard-coding the huh package import here.
	if err == nil {
		return false
	}
	s := err.Error()
	return s == "user aborted" || s == "interrupted"
}

func printDryRun(r *add.FormResult) {
	fmt.Println()
	fmt.Println(styles.Warning.Render("  ── Dry run — task NOT saved ──"))
	printTaskSummary(r)
}

func printSuccess(r *add.FormResult, twOutput string) {
	fmt.Println()
	fmt.Println(styles.Success.Render("  ✓ Task created!"))
	printTaskSummary(r)
	if twOutput != "" {
		fmt.Println(styles.MutedText.Render("  " + twOutput))
	}
}

func printTaskSummary(r *add.FormResult) {
	fmt.Printf("  %s  %s\n",
		styles.MutedText.Render("Description:"),
		styles.TaskDesc.Render(r.Description),
	)
	if r.Tag != "" {
		fmt.Printf("  %s  %s\n",
			styles.MutedText.Render("Tag:        "),
			styles.Tag.Render(r.Tag),
		)
	}
	if r.Project != "" {
		fmt.Printf("  %s  %s\n",
			styles.MutedText.Render("Project:    "),
			styles.TaskDesc.Render(r.Project),
		)
	}
	if r.Priority != "" {
		var p string
		switch r.Priority {
		case "H":
			p = styles.PriorityHigh.Render("High")
		case "M":
			p = styles.PriorityMedium.Render("Medium")
		case "L":
			p = styles.PriorityLow.Render("Low")
		}
		fmt.Printf("  %s  %s\n", styles.MutedText.Render("Priority:   "), p)
	}
	if r.Due != nil {
		fmt.Printf("  %s  %s\n",
			styles.MutedText.Render("Due:        "),
			styles.TaskDesc.Render(r.Due.Format(time.DateOnly)),
		)
	}
	fmt.Println()
}
