package task

import (
	"errors"
	"testing"
)

func TestStore_Create_Success(t *testing.T) {
	s := NewStore()

	got, err := s.Create("Hello Go world")
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	if got.ID == "" {
		t.Error("Create() returned a task with an empty ID")
	}
	if got.Input != "Hello Go world" {
		t.Errorf("Input = %q, want %q", got.Input, "Hello Go world")
	}
	if got.Status != StatusPending {
		t.Errorf("Status = %v, want %v", got.Status, StatusPending)
	}
	if got.Result != nil {
		t.Errorf("Result = %v, want nil for a freshly created task", got.Result)
	}
}

func TestStore_Create_InvalidInput(t *testing.T) {
	// Table-driven test covering the input-validation rule (requirement:
	// "invalid task input").
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "spaces only", input: "    "},
		{name: "tabs and newlines only", input: "\t\n\t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStore()
			got, err := s.Create(tt.input)

			if !errors.Is(err, ErrEmptyInput) {
				t.Fatalf("Create(%q) error = %v, want ErrEmptyInput", tt.input, err)
			}
			if got != nil {
				t.Errorf("Create(%q) task = %v, want nil on error", tt.input, got)
			}
			if s.Len() != 0 {
				t.Errorf("store has %d tasks after a rejected Create, want 0", s.Len())
			}
		})
	}
}

func TestStore_Get_Success(t *testing.T) {
	s := NewStore()
	created, err := s.Create("some text")
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}

	got, err := s.Get(created.ID)
	if err != nil {
		t.Fatalf("Get(%q) returned unexpected error: %v", created.ID, err)
	}
	if got != created {
		t.Errorf("Get(%q) returned a different *Task than Create produced", created.ID)
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	s := NewStore()

	got, err := s.Get("does-not-exist")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("Get() error = %v, want ErrTaskNotFound", err)
	}
	if got != nil {
		t.Errorf("Get() task = %v, want nil on error", got)
	}
}

func TestStore_MultipleIndependentTasks(t *testing.T) {
	s := NewStore()

	a, err := s.Create("first task text")
	if err != nil {
		t.Fatalf("Create(first) returned unexpected error: %v", err)
	}
	b, err := s.Create("second task text")
	if err != nil {
		t.Fatalf("Create(second) returned unexpected error: %v", err)
	}
	c, err := s.Create("third task text")
	if err != nil {
		t.Fatalf("Create(third) returned unexpected error: %v", err)
	}

	if a.ID == b.ID || b.ID == c.ID || a.ID == c.ID {
		t.Fatalf("expected distinct IDs, got %q, %q, %q", a.ID, b.ID, c.ID)
	}
	if s.Len() != 3 {
		t.Fatalf("store.Len() = %d, want 3", s.Len())
	}

	// Mutating one task's state must never bleed into another.
	a.Status = StatusCompleted

	gotB, err := s.Get(b.ID)
	if err != nil {
		t.Fatalf("Get(%q) returned unexpected error: %v", b.ID, err)
	}
	if gotB.Status != StatusPending {
		t.Errorf("task b.Status = %v after mutating task a, want %v (tasks must be independent)", gotB.Status, StatusPending)
	}
	if gotB.Input != "second task text" {
		t.Errorf("task b.Input = %q, want %q (tasks must not mix data)", gotB.Input, "second task text")
	}

	gotC, err := s.Get(c.ID)
	if err != nil {
		t.Fatalf("Get(%q) returned unexpected error: %v", c.ID, err)
	}
	if gotC.Input != "third task text" {
		t.Errorf("task c.Input = %q, want %q", gotC.Input, "third task text")
	}
}
