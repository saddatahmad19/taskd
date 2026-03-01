package taskwarrior

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Priority string

const (
	PriorityHigh   Priority = "H"
	PriorityMedium Priority = "M"
	PriorityLow    Priority = "L"
	PriorityNone   Priority = ""
)

func ParsePriority(s string) (Priority, error) {
	switch strings.ToUpper(s) {
	case "H":
		return PriorityHigh, nil
	case "M":
		return PriorityMedium, nil
	case "L":
		return PriorityLow, nil
	case "":
		return PriorityNone, nil
	default:
		return PriorityNone, fmt.Errorf("unknown priority: %q (valid: H, M, L, or empty)", s)
	}
}

func (p Priority) String() string {
	switch p {
	case PriorityHigh:
		return "High"
	case PriorityMedium:
		return "Medium"
	case PriorityLow:
		return "Low"
	default:
		return "None"
	}
}

type taskDate struct{ time.Time }

func (t *taskDate) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		return nil
	}
	// Taskwarrior uses ISO 8601 compact format
	formats := []string{
		"20060102T150405Z",
		"20060102T150405",
		time.RFC3339,
	}
	for _, f := range formats {
		if parsed, err := time.Parse(f, s); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("cannot parse task date: %q", s)
}

func (t taskDate) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.UTC().Format("20060102T150405Z"))
}

type Task struct {
	ID          int       `json:"id"`
	UUID        string    `json:"uuid"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Tags        []string  `json:"tags,omitempty"`
	Project     string    `json:"project,omitempty"`
	Priority    Priority  `json:"priority,omitempty"`
	Due         *taskDate `json:"due,omitempty"`
	Entry       taskDate  `json:"entry"`
	Modified    taskDate  `json:"modified"`
	Urgency     float64   `json:"urgency"`
}

func (t *Task) DueTime() *time.Time {
	if t.Due == nil {
		return nil
	}
	v := t.Due.Time
	return &v
}

type Filter struct {
	// Status filters by task status (pending, completed, deleted, etc.)
	// Defaults to "pending" if empty.
	Status string
	// Project filters to tasks belonging to a specific project.
	Project string
	// Tags filters to tasks that have all the specified tags.
	Tags []string
	// Limit caps the number of tasks returned (0 = no limit).
	Limit int
	// RawArgs allows passing arbitrary raw filter args to the task CLI.
	RawArgs []string
}

type AddRequest struct {
	Description string
	Tags        []string
	Project     string
	Priority    Priority
	Due         *time.Time
}

func (r AddRequest) Validate() error {
	if strings.TrimSpace(r.Description) == "" {
		return fmt.Errorf("description is required")
	}
	return nil
}
