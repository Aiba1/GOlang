package service

import (
	"fmt"
	"time"

	"textanalyzer-cp1/task"
)

type TaskService struct {
	store     *task.Store
	processor task.Processor
}

func NewTaskService(store *task.Store, processor task.Processor) *TaskService {
	return &TaskService{store: store, processor: processor}
}

func (s *TaskService) CreateTask(input string) (*task.Task, error) {
	return s.store.Create(input)
}

func (s *TaskService) GetTask(id string) (*task.Task, error) {
	return s.store.Get(id)
}

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
