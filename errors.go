package gomorph

import "fmt"

type ValidationError struct {
	Field   string
	Value   any
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %q: %s", e.Field, e.Message)
}

func NewValidationError(field string, value any, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}
