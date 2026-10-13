package data

import "errors"

// Sentinel application errors returned by repos (mock or real) and
// translated by handlers into HTTP status codes. Repos may wrap these
// with additional context via fmt.Errorf("%w: ...", ...); matching
// happens via errors.Is so wrapped errors are still recognized.
var (
	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrConflict indicates the operation conflicts with existing state.
	ErrConflict = errors.New("resource conflict")

	// ErrForbidden indicates the caller lacks permission for the operation.
	ErrForbidden = errors.New("forbidden: insufficient permissions")

	// ErrInternal indicates an unexpected internal failure.
	ErrInternal = errors.New("internal error")

	// ErrQueueFull indicates a background queue has reached its maximum
	// number of waiting jobs and is not accepting new ones right now.
	ErrQueueFull = errors.New("queue is full")
)
