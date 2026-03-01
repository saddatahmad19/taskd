package taskwarrior

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

type Client interface {
	// Add creates a new task and returns Taskwarrior's output message.
	Add(ctx context.Context, req AddRequest) (string, error)

	// Export returns tasks matching the given filter.
	Export(ctx context.Context, f Filter) ([]Task, error)

	// Tags returns a sorted list of unique tags across all pending tasks.
	Tags(ctx context.Context) ([]string, error)

	// Projects returns a sorted list of unique project names across all pending tasks.
	Projects(ctx context.Context) ([]string, error)

	// Complete marks the task identified by uuid as done.
	Complete(ctx context.Context, uuid string) error

	// Modify applies updates to the task identified by uuid.
	Modify(ctx context.Context, uuid string, req AddRequest) error

	// Delete marks the task identified by uuid as deleted.
	Delete(ctx context.Context, uuid string) error

	// Version returns the Taskwarrior version string.
	Version(ctx context.Context) (string, error)
}

type CLIClient struct {
	binary string // absolute path to `task` binary
}

var _ Client = (*CLIClient)(nil)

func NewCLIClient(binary string) (*CLIClient, error) {
	if binary == "" {
		binary = "task"
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return nil, fmt.Errorf("taskwarrior: binary %q not found in PATH — is Taskwarrior installed? (%w)", binary, err)
	}
	return &CLIClient{binary: path}, nil
}

func (c *CLIClient) run(ctx context.Context, args ...string) ([]byte, error) {
	// rc:0 disables Taskwarrior's interactive confirmation prompts.
	args = append([]string{"rc.confirmation=off", "rc.verbose=new-id,affected,project,special,edit,sync"}, args...)
	cmd := exec.CommandContext(ctx, c.binary, args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("task: %s\n%s", strings.TrimSpace(string(ee.Stderr)), strings.TrimSpace(string(out)))
		}
		return nil, fmt.Errorf("task: %w", err)
	}
	return out, nil
}

func (c *CLIClient) Add(ctx context.Context, req AddRequest) (string, error) {
	if err := req.Validate(); err != nil {
		return "", err
	}

	// Build args: `task add <description> [key:value ...]`
	// Description must come first; Taskwarrior treats anything without a colon
	// and before the first attribute as description text.
	args := []string{"add", req.Description}

	if len(req.Tags) > 0 {
		// Tags can be passed individually with + prefix or as tags:a,b,c
		for _, tag := range req.Tags {
			if tag = strings.TrimSpace(tag); tag != "" {
				args = append(args, "+"+tag)
			}
		}
	}
	if req.Project != "" {
		args = append(args, "project:"+req.Project)
	}
	if req.Priority != PriorityNone {
		args = append(args, "priority:"+string(req.Priority))
	}
	if req.Due != nil {
		args = append(args, "due:"+req.Due.Format("2006-01-02"))
	}

	out, err := c.run(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("add task: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *CLIClient) Export(ctx context.Context, f Filter) ([]Task, error) {
	// Taskwarrior requires filters to come BEFORE the subcommand:
	//   task [filter...] export
	// Placing filters after "export" causes Taskwarrior 2.6+ to treat them as
	// report names and fail with "Unable to find report that matches '...'".
	var args []string

	// Apply status filter; default to pending.
	status := f.Status
	if status == "" {
		status = "pending"
	}
	args = append(args, "status:"+status)

	if f.Project != "" {
		args = append(args, "project:"+f.Project)
	}
	for _, tag := range f.Tags {
		args = append(args, "+"+tag)
	}
	args = append(args, f.RawArgs...)

	// Subcommand comes last, after all filters.
	args = append(args, "export")

	// `task export` returns JSON even on "no tasks" (empty array).
	out, err := c.run(ctx, args...)
	if err != nil {
		// Taskwarrior exits 1 when there are no matching tasks in some versions.
		// Detect empty-array response to avoid propagating a spurious error.
		trimmed := strings.TrimSpace(string(out))
		if trimmed == "[]" || trimmed == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("export tasks: %w", err)
	}

	var tasks []Task
	if err := json.Unmarshal(out, &tasks); err != nil {
		return nil, fmt.Errorf("parse task export JSON: %w", err)
	}

	if f.Limit > 0 && len(tasks) > f.Limit {
		tasks = tasks[:f.Limit]
	}
	return tasks, nil
}

func (c *CLIClient) Tags(ctx context.Context) ([]string, error) {
	out, err := c.run(ctx, "tags")
	if err != nil {
		// `task tags` exits non-zero when there are no tags at all — treat as empty.
		return nil, nil
	}
	return parseFirstColumn(string(out), []string{"Tag", "---"}), nil
}

func (c *CLIClient) Projects(ctx context.Context) ([]string, error) {
	out, err := c.run(ctx, "projects")
	if err != nil {
		// `task projects` exits non-zero when there are no projects — treat as empty.
		return nil, nil
	}
	return parseFirstColumn(string(out), []string{"Project", "---"}), nil
}

func parseFirstColumn(output string, skipPrefixes []string) []string {
	var results []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// First token on the line.
		token := strings.Fields(line)[0]

		// Skip header and separator rows.
		skip := false
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(token, prefix) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		results = append(results, token)
	}
	sort.Strings(results)
	return results
}

func (c *CLIClient) Complete(ctx context.Context, uuid string) error {
	if _, err := c.run(ctx, uuid, "done"); err != nil {
		return fmt.Errorf("complete task %s: %w", uuid, err)
	}
	return nil
}

func (c *CLIClient) Modify(ctx context.Context, uuid string, req AddRequest) error {
	args := []string{uuid, "modify"}
	if req.Description != "" {
		args = append(args, "description:"+req.Description)
	}
	for _, tag := range req.Tags {
		if tag = strings.TrimSpace(tag); tag != "" {
			args = append(args, "+"+tag)
		}
	}
	if req.Project != "" {
		args = append(args, "project:"+req.Project)
	}
	if req.Priority != "" {
		args = append(args, "priority:"+string(req.Priority))
	}
	if req.Due != nil {
		args = append(args, "due:"+req.Due.Format("2006-01-02"))
	}
	if _, err := c.run(ctx, args...); err != nil {
		return fmt.Errorf("modify task %s: %w", uuid, err)
	}
	return nil
}

func (c *CLIClient) Delete(ctx context.Context, uuid string) error {
	if _, err := c.run(ctx, uuid, "delete"); err != nil {
		return fmt.Errorf("delete task %s: %w", uuid, err)
	}
	return nil
}

func (c *CLIClient) Version(ctx context.Context) (string, error) {
	out, err := c.run(ctx, "--version")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
