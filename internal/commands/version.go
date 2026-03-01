package commands

import (
	"context"
	"fmt"
	"runtime"

	"github.com/saddatahmad19/taskd/internal/taskwarrior"
	"github.com/saddatahmad19/taskd/internal/ui/styles"
	"github.com/spf13/cobra"
)

var Version = "dev"

func newVersionCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print taskd and Taskwarrior version information",

		// version does not need the TW client to be initialised — override pre-run.
		Annotations: map[string]string{"skip_tw_init": "true"},

		RunE: func(cmd *cobra.Command, _ []string) error {
			return runVersion(cmd.Context(), d)
		},
	}
}

func runVersion(ctx context.Context, d *deps) error {
	fmt.Println()
	fmt.Printf("  %s  %s\n", styles.Header.Render("taskd"), styles.TaskDesc.Render(Version))
	fmt.Printf("  %s  %s/%s\n",
		styles.MutedText.Render("Go:    "),
		runtime.GOOS, runtime.GOARCH,
	)
	fmt.Printf("  %s  %s\n",
		styles.MutedText.Render("Built: "),
		styles.MutedText.Render(runtime.Version()),
	)

	// Attempt to detect Taskwarrior — non-fatal if not found.
	twVersion := detectTWVersion(ctx, d)
	fmt.Printf("  %s  %s\n",
		styles.MutedText.Render("task:  "),
		styles.TaskDesc.Render(twVersion),
	)
	fmt.Println()

	return nil
}

func detectTWVersion(ctx context.Context, d *deps) string {
	if d.tw != nil {
		if v, err := d.tw.Version(ctx); err == nil {
			return v
		}
	}
	// Try creating a fresh client just for version detection.
	client, err := taskwarrior.NewCLIClient("")
	if err != nil {
		return "(not found in PATH)"
	}
	v, err := client.Version(ctx)
	if err != nil {
		return "(error: " + err.Error() + ")"
	}
	return v
}
