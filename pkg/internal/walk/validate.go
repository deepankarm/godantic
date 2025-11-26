package walk

import (
	"reflect"

	"github.com/deepankarm/godantic/pkg/internal/errors"
	"github.com/deepankarm/godantic/pkg/internal/reflectutil"
)

// ValidationError is an alias for the shared error type.
type ValidationError = errors.ValidationError

// ValidateProcessor runs validators and checks required fields.
// It collects all errors rather than stopping at the first one.
type ValidateProcessor struct {
	Errors []ValidationError
}

// GetErrors returns collected validation errors.
func (p *ValidateProcessor) GetErrors() []ValidationError {
	return p.Errors
}

// NewValidateProcessor creates a new validation processor.
func NewValidateProcessor() *ValidateProcessor {
	return &ValidateProcessor{
		Errors: make([]ValidationError, 0),
	}
}

// ProcessField validates a single field.
func (p *ValidateProcessor) ProcessField(ctx *FieldContext) error {
	// Skip root - we only validate fields
	if ctx.IsRoot {
		return nil
	}

	// No field options means no validation rules
	if ctx.FieldOptions == nil {
		return nil
	}

	val := reflectutil.UnwrapValue(ctx.Value)
	_, hasDefault := ctx.FieldOptions.Constraints["default"]
	isStruct := val.Kind() == reflect.Struct && !reflectutil.IsBasicType(val.Type())

	// Check required fields (but don't skip nested struct validation)
	if ctx.FieldOptions.Required && isZero(val) {
		if !hasDefault {
			// For structs, still validate nested fields to give more specific errors
			// (walker will descend into them, so don't add error here for structs)
			if !isStruct {
				p.Errors = append(p.Errors, ValidationError{
					Loc:     ctx.Path,
					Message: "required field",
					Type:    errors.ErrorTypeRequired,
				})
				return nil // Skip validators for required+zero+no-default
			}
		}
	}

	// Skip validation for zero values if:
	// 1. Field has a default (will be applied later), OR
	// 2. Field is not required (zero value means "not provided" for optional fields)
	// Otherwise: validate zero values (they may have been explicitly provided)
	if isZero(val) && !isStruct {
		if hasDefault || !ctx.FieldOptions.Required {
			return nil
		}
		// Field is required with no default: fall through to validate the zero value
	}

	// Run validators
	for _, validator := range ctx.FieldOptions.Validators {
		if err := validator(val.Interface()); err != nil {
			p.Errors = append(p.Errors, ValidationError{
				Loc:     ctx.Path,
				Message: err.Error(),
				Type:    errors.ErrorTypeConstraint,
			})
		}
	}

	return nil
}

// ShouldDescend returns true for nested structs that have validation.
func (p *ValidateProcessor) ShouldDescend(ctx *FieldContext) bool {
	val := reflectutil.UnwrapValue(ctx.Value)

	// Always descend into slices (let walker handle elements)
	if val.Kind() == reflect.Slice {
		return true
	}

	// Only descend into struct types
	if val.Kind() != reflect.Struct {
		return false
	}

	// Skip basic types like time.Time
	return !reflectutil.IsBasicType(val.Type())
}

// isZero checks if a value is the zero value for its type.
func isZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	return v.IsZero()
}
