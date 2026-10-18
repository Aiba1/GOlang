// Package service orchestrates the task domain (package task) and a
// processing strategy (any task.Processor, e.g. analyzer.TextAnalyzer)
// into the application-level operations the program actually exposes:
// create, get, and process a task.
//
// Keeping this orchestration separate from both the domain model and
// the processing logic is what lets TaskService depend only on the
// task.Processor interface, not on package analyzer directly — a
// different processor could be substituted (or a fake one used in
// tests) without changing this file.
package service

import (
	"fmt"
	"time"

	"textanalyzer-cp1/task"
)

// TaskService ties a task.Store together with a task.Processor and
// exposes the lifecycle operations the application needs.
type TaskService struct {
	store     *task.Store
	processor task.Processor
}

// NewTaskService returns a TaskService backed by the given store and
// processor. Both must be non-nil.
func NewTaskService(store *task.Store, processor task.Processor) *TaskService {
	return &TaskService{store: store, processor: processor}
}

// CreateTask validates input and stores a new task in StatusPending.
// It returns task.ErrEmptyInput if input is empty or whitespace-only.
func (s *TaskService) CreateTask(input string) (*task.Task, error) {
	return s.store.Create(input)
}

// GetTask returns the task with the given ID, or task.ErrTaskNotFound
// if it does not exist.
func (s *TaskService) GetTask(id string) (*task.Task, error) {
	return s.store.Get(id)
}

// ProcessTask runs the configured Processor over the task's input and
// records the outcome.
//
// Allowed transition: StatusPending -> StatusProcessing -> StatusCompleted
// (on success) or StatusFailed (if the processor returns an error).
//
// ProcessTask rejects, with task.ErrInvalidTransition, any task that is
// not currently StatusPending — in particular, a task that has already
// completed or failed cannot be processed again. This also rejects a
// task that is (in principle) already StatusProcessing, which can only
// happen if ProcessTask were re-entered concurrently; since this
// application is synchronous that case cannot occur in practice, but
// the check keeps the state machine honest regardless.
func (s *TaskService) ProcessTask(id string) (*task.Task, error) {
	t, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}

	if t.Status != task.StatusPending {
		return nil, fmt.Errorf("%w: task %s is %s, not %s", task.ErrInvalidTransition, t.ID, t.Status, task.StatusPending)
	}

	t.Status = task.StatusProcessing
	t.UpdatedAt = time.Now()

	result, procErr := s.processor.Process(t.Input)
	if procErr != nil {
		t.Status = task.StatusFailed
		t.UpdatedAt = time.Now()
		return t, fmt.Errorf("process task %s: %w", t.ID, procErr)
	}

	t.Result = result
	t.Status = task.StatusCompleted
	t.UpdatedAt = time.Now()
	return t, nil
}
