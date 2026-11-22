package task

import "time"

type Status string

const (
	StatusPending    Status = "pending"
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

func (s Status) String() string {
	return string(s)
}

type Result interface {
	Summary() string
}

type Processor interface {
	Process(input string) (Result, error)
}

type Task struct {
	ID        string
	Input     string
	Status    Status
	Result    Result
	CreatedAt time.Time
	UpdatedAt time.Time
}
