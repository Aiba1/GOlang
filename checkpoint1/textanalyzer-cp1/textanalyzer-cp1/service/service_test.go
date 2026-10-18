package service

import (
	"errors"
	"testing"

	"textanalyzer-cp1/task"
)

// fakeResult and fakeProcessor are small test doubles that satisfy
// task.Result and task.Processor. Using a fake here (instead of the
// real analyzer.TextAnalyzer) keeps these tests focused purely on
// TaskService's orchestration and lifecycle logic — exactly what
// defining Processor as an interface in package task is meant to
// enable. analyzer.TextAnalyzer's own output is verified separately in
// analyzer/analyzer_test.go.
type fakeResult struct{ text string }

func (f fakeResult) Summary() string { return f.text }

type fakeProcessor struct {
	result task.Result
	err    error
}

func (f fakeProcessor) Process(input string) (task.Result, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestTaskService_ProcessTask_Success(t *testing.T) {
	svc := NewTaskService(task.NewStore(), fakeProcessor{result: fakeResult{text: "ok"}})

	created, err := svc.CreateTask("some input")
	if err != nil {
		t.Fatalf("CreateTask() returned unexpected error: %v", err)
	}

	got, err := svc.ProcessTask(created.ID)
	if err != nil {
		t.Fatalf("ProcessTask() returned unexpected error: %v", err)
	}

	if got.Status != task.StatusCompleted {
		t.Errorf("Status = %v, want %v", got.Status, task.StatusCompleted)
	}
	if got.Result == nil {
		t.Fatal("Result = nil, want the processor's result to be attached")
	}
	if got.Result.Summary() != "ok" {
		t.Errorf("Result.Summary() = %q, want %q", got.Result.Summary(), "ok")
	}
}

func TestTaskService_ProcessTask_NotFound(t *testing.T) {
	svc := NewTaskService(task.NewStore(), fakeProcessor{result: fakeResult{text: "ok"}})

	_, err := svc.ProcessTask("no-such-id")
	if !errors.Is(err, task.ErrTaskNotFound) {
		t.Fatalf("ProcessTask() error = %v, want ErrTaskNotFound", err)
	}
}

func TestTaskService_ProcessTask_AlreadyCompleted(t *testing.T) {
	// Invalid-operation case: processing an already-completed task
	// again must be rejected, not silently re-run.
	svc := NewTaskService(task.NewStore(), fakeProcessor{result: fakeResult{text: "ok"}})

	created, err := svc.CreateTask("some input")
	if err != nil {
		t.Fatalf("CreateTask() returned unexpected error: %v", err)
	}
	if _, err := svc.ProcessTask(created.ID); err != nil {
		t.Fatalf("first ProcessTask() returned unexpected error: %v", err)
	}

	_, err = svc.ProcessTask(created.ID)
	if !errors.Is(err, task.ErrInvalidTransition) {
		t.Fatalf("second ProcessTask() error = %v, want ErrInvalidTransition", err)
	}
}

func TestTaskService_ProcessTask_ProcessorError(t *testing.T) {
	wantErr := errors.New("boom")
	svc := NewTaskService(task.NewStore(), fakeProcessor{err: wantErr})

	created, err := svc.CreateTask("some input")
	if err != nil {
		t.Fatalf("CreateTask() returned unexpected error: %v", err)
	}

	got, err := svc.ProcessTask(created.ID)
	if err == nil {
		t.Fatal("ProcessTask() returned nil error, want the wrapped processor error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("ProcessTask() error = %v, want it to wrap %v", err, wantErr)
	}
	if got.Status != task.StatusFailed {
		t.Errorf("Status = %v, want %v", got.Status, task.StatusFailed)
	}
	if got.Result != nil {
		t.Errorf("Result = %v, want nil after a failed processing run", got.Result)
	}
}

func TestTaskService_CreateTask_InvalidInput(t *testing.T) {
	svc := NewTaskService(task.NewStore(), fakeProcessor{result: fakeResult{text: "ok"}})

	_, err := svc.CreateTask("   ")
	if !errors.Is(err, task.ErrEmptyInput) {
		t.Fatalf("CreateTask() error = %v, want ErrEmptyInput", err)
	}
}

func TestTaskService_MultipleIndependentTasks(t *testing.T) {
	svc := NewTaskService(task.NewStore(), fakeProcessor{result: fakeResult{text: "ok"}})

	t1, err := svc.CreateTask("first")
	if err != nil {
		t.Fatalf("CreateTask(first) returned unexpected error: %v", err)
	}
	t2, err := svc.CreateTask("second")
	if err != nil {
		t.Fatalf("CreateTask(second) returned unexpected error: %v", err)
	}

	if _, err := svc.ProcessTask(t1.ID); err != nil {
		t.Fatalf("ProcessTask(t1) returned unexpected error: %v", err)
	}

	got2, err := svc.GetTask(t2.ID)
	if err != nil {
		t.Fatalf("GetTask(t2) returned unexpected error: %v", err)
	}
	if got2.Status != task.StatusPending {
		t.Errorf("t2.Status = %v after processing t1, want %v (tasks must be independent)", got2.Status, task.StatusPending)
	}
}
