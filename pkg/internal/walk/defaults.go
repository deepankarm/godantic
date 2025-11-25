package walk

import (
	"reflect"
)

// DefaultsProcessor applies default values to zero-valued fields.
type DefaultsProcessor struct{}

// GetErrors returns collected errors (defaults processor doesn't generate errors).
func (p *DefaultsProcessor) GetErrors() []ValidationError {
	return nil
}

// NewDefaultsProcessor creates a new defaults processor.
func NewDefaultsProcessor() *DefaultsProcessor {
	return &DefaultsProcessor{}
}

// ProcessField applies defaults to a field if it's zero-valued.
func (p *DefaultsProcessor) ProcessField(ctx *FieldContext) error {
	// Skip root
	if ctx.IsRoot {
		return nil
	}

	// No field options means no defaults
	if ctx.FieldOptions == nil {
		return nil
	}

	// Check if field has a default
	defaultVal, hasDefault := ctx.FieldOptions.Constraints["default"]
	if !hasDefault {
		return nil
	}

	// Only apply to settable fields
	if !ctx.Value.CanSet() {
		return nil
	}

	// Only apply to zero values
	if !ctx.Value.IsZero() {
		return nil
	}

	// Set the default
	defaultReflect := reflect.ValueOf(defaultVal)
	if defaultReflect.Type().AssignableTo(ctx.Value.Type()) {
		ctx.Value.Set(defaultReflect)
	}

	return nil
}
