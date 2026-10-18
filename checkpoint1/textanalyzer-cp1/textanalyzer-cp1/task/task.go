// Package task contains the domain model for this application: a Task,
// its lifecycle, and the small interfaces that a concrete processing
// implementation (see package analyzer) must satisfy.
//
// This package intentionally knows nothing about *how* a task is
// processed. It only knows the shape of a result (Result) and the shape
// of something that can turn input into a result (Processor). That keeps
// the domain model independent from any particular processing strategy
// and is what package service relies on to wire things together.
package task

import "time"

// Status represents where a Task currently is in its processing
// lifecycle. See the package-level comment in store.go and the README
// for the full set of allowed transitions.
type Status string

const (
	// StatusPending is the state of a task that has been created and
	// validated but has not started processing yet.
	StatusPending Status = "pending"

	// StatusProcessing is the state of a task while it is actively being
	// processed. A task only stays in this state for the duration of a
	// single ProcessTask call.
	StatusProcessing Status = "processing"

	// StatusCompleted is the state of a task that finished processing
	// successfully and has a Result attached.
	StatusCompleted Status = "completed"

	// StatusFailed is the state of a task whose processing was attempted
	// but returned an error. It is a terminal state, just like
	// StatusCompleted.
	StatusFailed Status = "failed"
)

// String implements fmt.Stringer with a value receiver, so a Status
// prints as its plain text form instead of a quoted Go string.
func (s Status) String() string {
	return string(s)
}

// Result is whatever a Processor produces from a task's input. It is
// deliberately a minimal interface: package task only ever needs to
// store a result and ask it to describe itself, never to know its
// concrete shape.
type Result interface {
	// Summary returns a short, human-readable description of the
	// result. Used for printing/logging; not meant to be parsed.
	Summary() string
}

// Processor turns a task's raw input into a Result, or returns an error
// if the input cannot be processed. Implementations live outside this
// package (see analyzer.TextAnalyzer) so that package task never depends
// on a specific processing strategy.
type Processor interface {
	Process(input string) (Result, error)
}

// Task is a single unit of work tracked by the application: its
// domain-specific input, its current lifecycle state, and — once
// processing has completed — its result.
type Task struct {
	ID        string
	Input     string
	Status    Status
	Result    Result
	CreatedAt time.Time
	UpdatedAt time.Time
}
