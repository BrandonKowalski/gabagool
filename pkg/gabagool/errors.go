package gabagool

import (
	"errors"
	"fmt"
)

// Sentinel errors for common conditions.
var (
	// ErrCancelled indicates the user cancelled an operation (pressed back, etc.).
	// This is a normal flow control error, not an infrastructure failure.
	ErrCancelled = errors.New("operation cancelled by user")

	// ErrDownloadCancelled indicates a download was cancelled by the user.
	// This is a domain-specific cancellation error for download operations.
	ErrDownloadCancelled = errors.New("download cancelled by user")
)

// InfrastructureError represents a framework-level error that indicates
// something is wrong with gabagool itself (rendering failed, SDL crashed,
// font missing, etc.). These errors are typically fatal or require
// framework-level recovery.
//
// Use this for errors that the consuming application cannot reasonably
// handle or recover from at the domain level.
type InfrastructureError struct {
	Op  string // Operation that failed (e.g., "render", "load_font")
	Err error  // Underlying error
}

func (e *InfrastructureError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("gabagool: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("gabagool: %s", e.Op)
}

func (e *InfrastructureError) Unwrap() error {
	return e.Err
}

// NewInfrastructureError creates a new infrastructure error.
func NewInfrastructureError(op string, err error) *InfrastructureError {
	return &InfrastructureError{Op: op, Err: err}
}

// IsInfrastructureError checks if an error is an infrastructure error.
func IsInfrastructureError(err error) bool {
	var infraErr *InfrastructureError
	return errors.As(err, &infraErr)
}

// IsCancelled checks if an error indicates user cancellation.
func IsCancelled(err error) bool {
	return errors.Is(err, ErrCancelled)
}
