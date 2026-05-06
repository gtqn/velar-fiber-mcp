// Package errors defines all typed domain errors for VELAR-Fiber.
// Every error has a machine-readable Code that AI agents can match against.
package errors

import "fmt"

// Code is a string identifier for a domain error — readable by both humans and AI.
type Code string

const (
	// CodeInvalidInput is returned when tool arguments fail validation.
	CodeInvalidInput Code = "INVALID_INPUT"
	// CodeNotFound is returned when a requested resource does not exist.
	CodeNotFound Code = "NOT_FOUND"
	// CodeUnauthorized is returned when the API key is missing or invalid.
	CodeUnauthorized Code = "UNAUTHORIZED"
	// CodeForbidden is returned when the API key lacks permission for the tool.
	CodeForbidden Code = "FORBIDDEN"
	// CodeTimeout is returned when an external call exceeds its deadline.
	CodeTimeout Code = "TIMEOUT"
	// CodeExternalService is returned when an upstream service returns an error.
	CodeExternalService Code = "EXTERNAL_SERVICE_ERROR"
	// CodeInternal is returned for unexpected internal failures.
	CodeInternal Code = "INTERNAL_ERROR"
	// CodeRateLimited is returned when the client exceeds the rate limit.
	CodeRateLimited Code = "RATE_LIMITED"
)

// DomainError is a structured error that carries context for AI agents.
type DomainError struct {
	// Code is the machine-readable error identifier.
	Code Code `json:"code"`
	// Message is a human-readable description of what went wrong.
	Message string `json:"message"`
	// Field is the input field that caused the error (if applicable).
	Field string `json:"field,omitempty"`
	// Suggestion is a hint for AI agents on how to fix the error.
	Suggestion string `json:"suggestion,omitempty"`
	// Cause is the underlying error (not serialized to JSON).
	Cause error `json:"-"`
}

// Error implements the error interface.
func (e *DomainError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("[%s] %s (field: %s)", e.Code, e.Message, e.Field)
	}

	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap allows errors.Is / errors.As to inspect the cause.
func (e *DomainError) Unwrap() error {
	return e.Cause
}

// New creates a new DomainError with the given code and message.
func New(code Code, message string) *DomainError {
	return &DomainError{Code: code, Message: message}
}

// NewWithField creates a DomainError pointing to a specific input field.
func NewWithField(code Code, message, field, suggestion string) *DomainError {
	return &DomainError{
		Code:       code,
		Message:    message,
		Field:      field,
		Suggestion: suggestion,
	}
}

// Wrap wraps an external error with a domain code and message.
func Wrap(code Code, message string, cause error) *DomainError {
	return &DomainError{Code: code, Message: message, Cause: cause}
}

// IsCode checks whether an error carries a specific domain code.
func IsCode(err error, code Code) bool {
	var de *DomainError
	if asErr, ok := err.(*DomainError); ok { //nolint:errorlint
		de = asErr
	}

	return de != nil && de.Code == code
}
