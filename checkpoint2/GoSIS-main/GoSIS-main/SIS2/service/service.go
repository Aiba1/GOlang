package service

import "textanalyzer-cp1/task"

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

// ProcessTask processes a Pending task synchronously in the caller's
// goroutine (the Checkpoint 1 path). It shares Store.Process with the
// worker pool, so a task can only ever be processed once.
func (s *TaskService) ProcessTask(id string) (*task.Task, error) {
	return s.store.Process(id, task.StatusPending, s.processor)
}
