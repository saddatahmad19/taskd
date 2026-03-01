package tasklist

import (
	"fmt"
	"strings"
	"time"

	"github.com/saddatahmad19/taskd/internal/taskwarrior"
)

type Item struct {
	Task taskwarrior.Task
}

func (i Item) Title() string {
	return i.Task.Description
}

func (i Item) Description() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("#%d", i.Task.ID))
	for _, t := range i.Task.Tags {
		parts = append(parts, "+"+t)
	}
	if i.Task.Project != "" {
		parts = append(parts, "⌂ "+i.Task.Project)
	}
	if i.Task.Priority != taskwarrior.PriorityNone {
		parts = append(parts, "pri:"+string(i.Task.Priority))
	}
	if due := i.Task.DueTime(); due != nil {
		parts = append(parts, "due:"+due.Format("2006-01-02"))
	}
	return strings.Join(parts, "  ")
}

func (i Item) FilterValue() string {
	parts := []string{strings.ToLower(i.Task.Description)}

	for _, t := range i.Task.Tags {
		parts = append(parts, "tag:"+strings.ToLower(t))
	}
	if i.Task.Project != "" {
		parts = append(parts, "project:"+strings.ToLower(i.Task.Project))
	}
	if i.Task.Priority != taskwarrior.PriorityNone {
		parts = append(parts, "priority:"+strings.ToLower(string(i.Task.Priority)))
	}
	if due := i.Task.DueTime(); due != nil {
		parts = append(parts, "due:"+due.Format("2006-01-02"))
		// Add natural keyword aliases so "due:today" / "due:tomorrow" work.
		now := time.Now()
		today := truncDay(now)
		dueDay := truncDay(*due)
		switch {
		case dueDay.Equal(today):
			parts = append(parts, "due:today")
		case dueDay.Equal(today.Add(24*time.Hour)):
			parts = append(parts, "due:tomorrow")
		}
		if due.Before(now) {
			parts = append(parts, "due:overdue")
		}
	}

	return strings.Join(parts, " ")
}

func truncDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
