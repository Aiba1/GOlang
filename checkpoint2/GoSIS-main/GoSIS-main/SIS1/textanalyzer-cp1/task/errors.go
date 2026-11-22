package task

import "errors"

var (
	ErrEmptyInput        = errors.New("task: input must not be empty or whitespace-only")
	ErrTaskNotFound      = errors.New("task: not found")
	ErrInvalidTransition = errors.New("task: invalid state transition")
)
