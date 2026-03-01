package taskwarrior_test

import (
	"context"
	"testing"
	"time"

	"github.com/saddatahmad19/taskd/internal/taskwarrior"
)

func TestParsePriority(t *testing.T) {
	cases := []struct {
		input   string
		want    taskwarrior.Priority
		wantErr bool
	}{
		{"H", taskwarrior.PriorityHigh, false},
		{"M", taskwarrior.PriorityMedium, false},
		{"L", taskwarrior.PriorityLow, false},
		{"", taskwarrior.PriorityNone, false},
		{"h", taskwarrior.PriorityHigh, false}, // case-insensitive
		{"X", taskwarrior.PriorityNone, true},
	}
	for _, c := range cases {
		got, err := taskwarrior.ParsePriority(c.input)
		if (err != nil) != c.wantErr {
			t.Errorf("ParsePriority(%q) error = %v, wantErr %v", c.input, err, c.wantErr)
		}
		if !c.wantErr && got != c.want {
			t.Errorf("ParsePriority(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestAddRequestValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		req := taskwarrior.AddRequest{Description: "Write tests"}
		if err := req.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty description", func(t *testing.T) {
		req := taskwarrior.AddRequest{Description: "   "}
		if err := req.Validate(); err == nil {
			t.Error("expected error for empty description")
		}
	})
}

func TestPriorityString(t *testing.T) {
	cases := map[taskwarrior.Priority]string{
		taskwarrior.PriorityHigh:   "High",
		taskwarrior.PriorityMedium: "Medium",
		taskwarrior.PriorityLow:    "Low",
		taskwarrior.PriorityNone:   "None",
	}
	for p, want := range cases {
		if got := p.String(); got != want {
			t.Errorf("Priority(%q).String() = %q, want %q", string(p), got, want)
		}
	}
}

func TestMockClientAdd(t *testing.T) {
	called := false
	tw := &taskwarrior.MockClient{
		AddFn: func(_ context.Context, req taskwarrior.AddRequest) (string, error) {
			called = true
			if req.Description != "Test task" {
				t.Errorf("unexpected description: %q", req.Description)
			}
			return "Created task 1.", nil
		},
	}

	due := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := taskwarrior.AddRequest{
		Description: "Test task",
		Tags:        []string{"work"},
		Project:     "myproject",
		Priority:    taskwarrior.PriorityHigh,
		Due:         &due,
	}

	var client taskwarrior.Client = tw
	ctx := context.Background()
	out, err := client.Add(ctx, req)
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if out != "Created task 1." {
		t.Errorf("unexpected output: %q", out)
	}
	if !called {
		t.Error("AddFn was not called")
	}
}
