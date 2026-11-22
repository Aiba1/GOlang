package task

import (
	"fmt"
	"strings"
	"time"
)

type Store struct {
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

func (s *Store) Get(id string) (*Task, error) {
	t, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("%w: id=%q", ErrTaskNotFound, id)
	}
	return t, nil
}

func (s *Store) All() []*Task {
	result := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		result = append(result, t)
	}
	return result
}

func (s *Store) Len() int {
	return len(s.tasks)
}
