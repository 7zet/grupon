package catalog

import (
	"errors"
	"fmt"
)

// ErrNotFound reports that a requested public resource does not exist or is not visible.
var ErrNotFound = errors.New("catalog: not found")

// InvalidInputError reports a client error with a stable, translatable code.
type InvalidInputError struct {
	Field string
	Code  string
}

func (e *InvalidInputError) Error() string {
	return fmt.Sprintf("catalog: invalid %s: %s", e.Field, e.Code)
}

func invalid(field, code string) error {
	return &InvalidInputError{Field: field, Code: code}
}
