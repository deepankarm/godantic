package walk

import (
	"fmt"
	"reflect"

	"github.com/deepankarm/godantic/pkg/internal/errors"
	"github.com/deepankarm/godantic/pkg/internal/reflectutil"
)

// UnionValidateProcessor validates union constraints (anyOf/oneOf/discriminator).
type UnionValidateProcessor struct {
	Errors []ValidationError
}

// GetErrors returns collected validation errors.
func (p *UnionValidateProcessor) GetErrors() []ValidationError {
	return p.Errors
}

// NewUnionValidateProcessor creates a new union validation processor.
func NewUnionValidateProcessor() *UnionValidateProcessor {
	return &UnionValidateProcessor{
		Errors: make([]ValidationError, 0),
	}
}

// ProcessField validates union constraints on a field.
func (p *UnionValidateProcessor) ProcessField(ctx *FieldContext) error {
	// Skip root
	if ctx.IsRoot {
		return nil
	}

	// No field options = no constraints
	if ctx.FieldOptions == nil {
		return nil
	}

	// Skip zero values (already handled by ValidateProcessor)
	val := unwrapValue(ctx.Value)
	if isZero(val) {
		return nil
	}

	// Check for discriminated union constraint (oneOf)
	if discConstraint, ok := ctx.FieldOptions.Constraints["discriminator"].(map[string]any); ok {
		if err := p.validateDiscriminator(ctx, discConstraint); err != nil {
			p.Errors = append(p.Errors, *err)
		}
		return nil
	}

	// Check for simple union constraint (anyOf)
	if err := p.validateAnyOf(ctx); err != nil {
		p.Errors = append(p.Errors, *err)
	}

	return nil
}

// validateDiscriminator validates discriminated union (oneOf) constraints.
func (p *UnionValidateProcessor) validateDiscriminator(ctx *FieldContext, discConstraint map[string]any) *ValidationError {
	discriminatorField, _ := discConstraint["propertyName"].(string)
	mapping, _ := discConstraint["mapping"].(map[string]any)

	if discriminatorField == "" || mapping == nil {
		return nil
	}

	val := unwrapValue(ctx.Value)

	// Handle slices of discriminated unions
	if val.Kind() == reflect.Slice {
		for i := 0; i < val.Len(); i++ {
			elemPath := append(append([]string{}, ctx.Path...), fmt.Sprintf("[%d]", i))
			if err := p.validateSingleDiscriminator(val.Index(i), discriminatorField, mapping, elemPath); err != nil {
				return err
			}
		}
		return nil
	}

	// Single discriminated union value
	return p.validateSingleDiscriminator(val, discriminatorField, mapping, ctx.Path)
}

// validateSingleDiscriminator validates a single discriminated union value.
func (p *UnionValidateProcessor) validateSingleDiscriminator(val reflect.Value, discriminatorField string, mapping map[string]any, path []string) *ValidationError {
	// Unwrap pointers and interfaces
	for val.Kind() == reflect.Pointer || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return &ValidationError{
				Loc:     path,
				Message: "discriminated union value cannot be nil",
				Type:    errors.ErrorTypeConstraint,
			}
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return &ValidationError{
			Loc:     path,
			Message: "discriminated union requires a struct type",
			Type:    errors.ErrorTypeConstraint,
		}
	}

	// Find discriminator field by JSON name
	discField := reflectutil.FieldByJSONName(val, val.Type(), discriminatorField)
	if !discField.IsValid() {
		return &ValidationError{
			Loc:     path,
			Message: fmt.Sprintf("discriminator field '%s' not found", discriminatorField),
			Type:    errors.ErrorTypeConstraint,
		}
	}

	discValue := fmt.Sprintf("%v", discField.Interface())

	if _, ok := mapping[discValue]; !ok {
		validValues := make([]string, 0, len(mapping))
		for k := range mapping {
			validValues = append(validValues, k)
		}
		return &ValidationError{
			Loc:     path,
			Message: fmt.Sprintf("invalid discriminator value '%s', expected one of: %v", discValue, validValues),
			Type:    errors.ErrorTypeConstraint,
		}
	}

	return nil
}

// validateAnyOf validates simple union (anyOf) constraints.
func (p *UnionValidateProcessor) validateAnyOf(ctx *FieldContext) *ValidationError {
	constraints := ctx.FieldOptions.Constraints

	// Collect primitive type constraints
	var allowedTypes []string
	if anyOf, ok := constraints["anyOf"]; ok {
		if anyOfSlice, ok := anyOf.([]map[string]string); ok {
			for _, typeMap := range anyOfSlice {
				if typeName, ok := typeMap["type"]; ok {
					allowedTypes = append(allowedTypes, typeName)
				}
			}
		}
	}

	// Collect complex type constraints
	var complexTypes []any
	if anyOfTypes, ok := constraints["anyOfTypes"]; ok {
		if types, ok := anyOfTypes.([]any); ok {
			complexTypes = types
		}
	}

	// If no union constraints, validation passes
	if len(allowedTypes) == 0 && len(complexTypes) == 0 {
		return nil
	}

	val := unwrapValue(ctx.Value)
	valType := val.Type()

	// Check against complex types first
	for _, complexType := range complexTypes {
		expectedType := reflect.TypeOf(complexType)
		if valType == expectedType {
			return nil // Match found
		}
		// Also check if value is a slice/array and the element types match
		if valType.Kind() == reflect.Slice && expectedType.Kind() == reflect.Slice {
			if valType.Elem() == expectedType.Elem() {
				return nil // Match found
			}
		}
	}

	// Check against primitive types
	for _, allowedType := range allowedTypes {
		if reflectutil.MatchesJSONSchemaType(val, allowedType) {
			return nil // Match found
		}
	}

	// No match found
	allTypes := append([]string{}, allowedTypes...)
	for _, ct := range complexTypes {
		allTypes = append(allTypes, reflect.TypeOf(ct).String())
	}

	return &ValidationError{
		Loc:     ctx.Path,
		Message: fmt.Sprintf("value does not match any allowed type: %v", allTypes),
		Type:    errors.ErrorTypeConstraint,
	}
}

// ShouldDescend - union validation doesn't need to descend, ValidateProcessor handles that.
func (p *UnionValidateProcessor) ShouldDescend(ctx *FieldContext) bool {
	return false
}
