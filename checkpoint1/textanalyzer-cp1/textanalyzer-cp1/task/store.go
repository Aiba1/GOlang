package task

import (
	"fmt"
	"strings"
	"time"
)

// Store is a synchronous, in-memory collection of Tasks keyed by ID.
// It is intentionally the only place in the application that creates
// Task values and assigns identifiers, so task creation and ID
// assignment can never drift out of sync.
//
// Store is not safe for concurrent use. That is fine for this
// checkpoint, which is explicitly synchronous; concurrency safety is a
// later-checkpoint concern and is deliberately not implemented here.
type Store struct {
	tasks  map[string]*Task
	nextID int
}

// NewStore returns an empty, ready-to-use Store.
func NewStore() *Store {
	return &Store{
		tasks: make(map[string]*Task),
	}
}

// Create validates input, assigns it a new unique ID, stores a new
// Task in StatusPending, and returns it. It returns ErrEmptyInput if
// input is empty or contains only whitespace.
func (s *Store) Create(input string) (*Task, error) {
	if strings.TrimSpace(input) == "" {
		return nil, ErrEmptyInput
	}

	s.nextID++
	now := time.Now()
	t := &Task{
		ID:        fmt.Sprintf("task-%d", s.nextID),
		Input:     input,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.tasks[t.ID] = t
	return t, nil
}

// Get returns the task with the given ID. It returns ErrTaskNotFound
// (wrapped with the requested ID) if no such task exists.
//
// The returned pointer refers to the Task actually held by the store,
// not a copy, so callers (notably package service) can update its
// fields in place to drive lifecycle transitions.
func (s *Store) Get(id string) (*Task, error) {
	t, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("%w: id=%q", ErrTaskNotFound, id)
	}
	return t, nil
}

// All returns every task currently in the store, in no particular
// order. Useful for diagnostics and for tests that need to check that
// tasks don't interfere with one another.
func (s *Store) All() []*Task {
	result := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		result = append(result, t)
	}
	return result
}

// Len reports how many tasks are currently stored.
func (s *Store) Len() int {
	return len(s.tasks)
}
