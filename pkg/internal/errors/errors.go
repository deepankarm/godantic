// Package errors defines shared error types for godantic.
package errors

import (
	"fmt"
	"strings"
)

// ErrorType is an enum for validation error categories.
type ErrorType string

// Error type constants.
const (
	ErrorTypeRequired             ErrorType = "required"              // Field is required but missing/zero
	ErrorTypeConstraint           ErrorType = "constraint"            // Field constraint violation (min, max, etc.)
	ErrorTypeInternal             ErrorType = "internal"              // Internal error (nil pointer, reflection issue)
	ErrorTypeJSONDecode           ErrorType = "json_decode"           // JSON unmarshaling failed
	ErrorTypeJSONEncode           ErrorType = "json_encode"           // JSON marshaling failed
	ErrorTypeHookError            ErrorType = "hook_error"            // Lifecycle hook returned error
	ErrorTypeDiscriminatorMissing ErrorType = "discriminator_missing" // Discriminator field not found
	ErrorTypeDiscriminatorInvalid ErrorType = "discriminator_invalid" // Discriminator value not in mapping
	ErrorTypeMismatch             ErrorType = "type_error"            // Type mismatch during validation
	ErrorTypeMarshalError         ErrorType = "marshal_error"         // Marshal error (map validation)
)

// ValidationError represents a validation error with location information.
type ValidationError struct {
	Loc     []string  // Path to the field, e.g., ["Address", "ZipCode"]
	Message string    // Human-readable error message
	Type    ErrorType // Error category
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	if len(e.Loc) == 0 {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", strings.Join(e.Loc, "."), e.Message)
}

// ValidationErrors is a slice of ValidationError that implements error.
type ValidationErrors []ValidationError

// Error implements the error interface.
func (es ValidationErrors) Error() string {
	if len(es) == 0 {
		return "validation errors: (none)"
	}
	if len(es) == 1 {
		return es[0].Error()
	}
	var msgs []string
	for _, e := range es {
		msgs = append(msgs, e.Error())
	}
	return fmt.Sprintf("validation errors (%d): %s", len(es), strings.Join(msgs, "; "))
}

// Unwrap returns the errors as a slice for errors.As/errors.Is compatibility.
func (es ValidationErrors) Unwrap() []error {
	errs := make([]error, len(es))
	for i, e := range es {
		errs[i] = e
	}
	return errs
}

func (es ValidationErrors) HasJSONDecodeError() bool {
	for _, e := range es {
		if e.Type == ErrorTypeJSONDecode {
			return true
		}
	}
	return false
}
