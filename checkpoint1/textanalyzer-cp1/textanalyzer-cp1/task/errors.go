package task

import "errors"

// Sentinel errors returned by this package. Callers should compare
// against these with errors.Is, since Store and the service layer wrap
// them with additional context (e.g. the offending task ID).
var (
	// ErrEmptyInput is returned when task input is empty or contains
	// only whitespace.
	ErrEmptyInput = errors.New("task: input must not be empty or whitespace-only")

	// ErrTaskNotFound is returned when a lookup is made for a task ID
	// that does not exist in the store.
	ErrTaskNotFound = errors.New("task: not found")

	// ErrInvalidTransition is returned when an operation is attempted
	// that does not correspond to a valid lifecycle transition for the
	// task's current status (e.g. processing a task that has already
	// completed or failed).
	ErrInvalidTransition = errors.New("task: invalid state transition")
)
