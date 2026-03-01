package commands

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/saddatahmad19/taskd/internal/config"
	"github.com/saddatahmad19/taskd/internal/taskwarrior"
	"github.com/saddatahmad19/taskd/internal/ui/styles"
	"github.com/spf13/cobra"
)

type deps struct {
	tw taskwarrior.Client
}

var rootCmd *cobra.Command

func Execute() {
	// Load user config from ~/.config/taskd/config.json (created with defaults
	// on first run) and apply theme colors before any UI renders.
	cfg, cfgErr := config.Load()
	styles.Init(cfg)
	// cfgErr is non-fatal — we surface it as a soft warning after cobra runs.

	// Shared dependency container — initialised lazily so --help never requires
	// Taskwarrior to be installed.
	d := &deps{}
	_ = cfgErr // used below

	// PersistentPreRunE wires up the taskwarrior client before any sub-command runs.
	// Commands that don't need the client (e.g., help) bypass this via cobra's
	// annotation mechanism or by not calling the parent's pre-run.
	rootCmd = &cobra.Command{
		Use:   "taskd",
		Short: "A Charm-powered Taskwarrior CLI companion",
		Long:  rootLong(),

		// Disable the default "completion" command added by Cobra.
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},

		// SilenceUsage prevents Cobra from printing usage on every error —
		// we handle errors explicitly in main().
		SilenceUsage: true,

		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Commands that declare themselves as not needing the client
			// (e.g. version) can set this annotation to skip initialization.
			if cmd.Annotations["skip_tw_init"] == "true" {
				return nil
			}
			binary, _ := cmd.Flags().GetString("binary")
			client, err := taskwarrior.NewCLIClient(binary)
			if err != nil {
				return fmt.Errorf("%s\n\nMake sure Taskwarrior is installed and accessible via PATH.\n"+
					"Installation: https://taskwarrior.org/download/", err)
			}
			d.tw = client
			return nil
		},
	}

	// Global flags available to every sub-command.
	rootCmd.PersistentFlags().String(
		"binary", "",
		"Path to the task binary (default: auto-detected via PATH)",
	)

	// Register all sub-commands.
	rootCmd.AddCommand(
		newAddCmd(d),
		newListCmd(d),
		newCompleteCmd(d),
		newModifyCmd(d),
		newFullUICmd(d),
		newVersionCmd(d),
	)

	if cfgErr != nil {
		fmt.Fprintln(os.Stderr, styles.Warning.Render("Warning: ")+cfgErr.Error())
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, styles.Error.Render("Error: ")+err.Error())
		os.Exit(1)
	}
}

func rootLong() string {
	header := styles.Header.Render("taskd — Taskwarrior companion")
	body := lipgloss.NewStyle().Foreground(styles.Text).Render(
		"A Charm-powered CLI that wraps Taskwarrior with an interactive TUI.\n\n" +
			"Taskwarrior must be installed and accessible via your system PATH.\n" +
			"Configuration is read from ~/.taskrc as usual.",
	)
	return lipgloss.JoinVertical(lipgloss.Left, "", header, body, "")
}
