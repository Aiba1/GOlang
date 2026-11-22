package task

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Store is the single owner of all task data. Every read and write of the
// map, of nextID and of the fields of stored tasks happens under mu, and
// callers only ever receive copies, so a snapshot they hold can never be
// modified by a worker while they read it.
type Store struct {
	mu     sync.RWMutex
	tasks  map[string]*Task
	nextID int
}

func NewStore() *Store {
	return &Store{
		tasks: make(map[string]*Task),
	}
}

func (s *Store) Create(input string) (*Task, error) {
	if strings.TrimSpace(input) == "" {
		return nil, ErrEmptyInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()
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
	return snapshot(t), nil
}

func (s *Store) Get(id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("%w: id=%q", ErrTaskNotFound, id)
	}
	return snapshot(t), nil
}

func (s *Store) All() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		result = append(result, snapshot(t))
	}
	return result
}

func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tasks)
}

// Transition atomically moves a task from one state to another. The check
// and the write happen under the same lock, so two goroutines can never
// both see "Pending" and both claim the task.
func (s *Store) Transition(id string, from, to Status) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("%w: id=%q", ErrTaskNotFound, id)
	}
	if t.Status != from {
		return nil, fmt.Errorf("%w: task %s is %s, not %s", ErrInvalidTransition, id, t.Status, from)
	}
	t.Status = to
	t.UpdatedAt = time.Now()
	return snapshot(t), nil
}

// Process runs one task through from -> Processing -> Completed/Failed.
// The lock is held only for the two state changes, never while p.Process
// runs, so slow analysis does not block readers or other workers.
func (s *Store) Process(id string, from Status, p Processor) (*Task, error) {
	t, err := s.Transition(id, from, StatusProcessing)
	if err != nil {
		return nil, err
	}

	result, procErr := p.Process(t.Input)

	s.mu.Lock()
	defer s.mu.Unlock()
	stored := s.tasks[id] // tasks are never deleted, so the entry exists
	stored.UpdatedAt = time.Now()
	if procErr != nil {
		stored.Status = StatusFailed
		return snapshot(stored), fmt.Errorf("process task %s: %w", id, procErr)
	}
	stored.Status = StatusCompleted
	stored.Result = result
	return snapshot(stored), nil
}

func snapshot(t *Task) *Task {
	c := *t
	return &c
}
