package walk

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/deepankarm/godantic/pkg/internal/errors"
	"github.com/deepankarm/godantic/pkg/internal/reflectutil"
)

// UnmarshalProcessor unmarshals JSON data into struct fields.
// It handles regular fields and discriminated unions.
type UnmarshalProcessor struct {
	Errors []ValidationError
}

// GetErrors returns collected validation errors.
func (p *UnmarshalProcessor) GetErrors() []ValidationError {
	return p.Errors
}

// NewUnmarshalProcessor creates a new unmarshal processor.
func NewUnmarshalProcessor() *UnmarshalProcessor {
	return &UnmarshalProcessor{
		Errors: make([]ValidationError, 0),
	}
}

// ProcessField unmarshals JSON into a field.
func (p *UnmarshalProcessor) ProcessField(ctx *FieldContext) error {
	// Skip root - walker handles root separately
	if ctx.IsRoot {
		return nil
	}

	// No JSON data for this field
	if len(ctx.RawJSON) == 0 {
		return nil
	}

	// Field must be settable
	if !ctx.Value.CanSet() {
		return nil
	}

	// Check for discriminated union constraint
	if ctx.FieldOptions != nil {
		if discConstraint, ok := ctx.FieldOptions.Constraints["discriminator"].(map[string]any); ok {
			return p.unmarshalDiscriminated(ctx, discConstraint)
		}
	}

	// Regular field - unmarshal directly
	return p.unmarshalRegular(ctx)
}

// unmarshalRegular unmarshals a regular (non-discriminated) field.
func (p *UnmarshalProcessor) unmarshalRegular(ctx *FieldContext) error {
	fieldPtr := ctx.Value.Addr()
	if err := json.Unmarshal(ctx.RawJSON, fieldPtr.Interface()); err != nil {
		p.Errors = append(p.Errors, ValidationError{
			Loc:     ctx.Path,
			Message: fmt.Sprintf("JSON unmarshal failed: %v", err),
			Type:    errors.ErrorTypeJSONDecode,
		})
	}
	return nil
}

// unmarshalDiscriminated handles discriminated union unmarshaling.
func (p *UnmarshalProcessor) unmarshalDiscriminated(ctx *FieldContext, discConstraint map[string]any) error {
	discriminatorField, _ := discConstraint["propertyName"].(string)
	mapping, _ := discConstraint["mapping"].(map[string]any)

	if discriminatorField == "" || mapping == nil {
		return p.unmarshalRegular(ctx)
	}

	fieldType := ctx.Value.Type()

	// Check if field is a slice of discriminated unions
	if fieldType.Kind() == reflect.Slice {
		return p.unmarshalDiscriminatedSlice(ctx, discriminatorField, mapping)
	}

	// Single discriminated union value
	return p.unmarshalDiscriminatedSingle(ctx, discriminatorField, mapping)
}

// unmarshalDiscriminatedSingle unmarshals a single discriminated union value.
func (p *UnmarshalProcessor) unmarshalDiscriminatedSingle(ctx *FieldContext, discriminatorField string, mapping map[string]any) error {
	// Peek at JSON to find discriminator value
	var fieldMap map[string]any
	if err := json.Unmarshal(ctx.RawJSON, &fieldMap); err != nil {
		p.Errors = append(p.Errors, ValidationError{
			Loc:     ctx.Path,
			Message: fmt.Sprintf("failed to parse discriminated union field: %v", err),
			Type:    errors.ErrorTypeJSONDecode,
		})
		return nil
	}

	discriminatorValue, ok := fieldMap[discriminatorField]
	if !ok {
		p.Errors = append(p.Errors, ValidationError{
			Loc:     append(ctx.Path, discriminatorField),
			Message: fmt.Sprintf("discriminator field '%s' not found", discriminatorField),
			Type:    errors.ErrorTypeDiscriminatorMissing,
		})
		return nil
	}

	// Look up concrete type
	concreteTypeExample, ok := mapping[fmt.Sprintf("%v", discriminatorValue)]
	if !ok {
		validValues := make([]string, 0, len(mapping))
		for k := range mapping {
			validValues = append(validValues, k)
		}
		p.Errors = append(p.Errors, ValidationError{
			Loc:     append(ctx.Path, discriminatorField),
			Message: fmt.Sprintf("invalid discriminator value '%v', expected one of: %v", discriminatorValue, validValues),
			Type:    errors.ErrorTypeDiscriminatorInvalid,
		})
		return nil
	}

	// Create and unmarshal into concrete type
	concreteType := reflect.TypeOf(concreteTypeExample)
	elemType := concreteType
	if concreteType.Kind() == reflect.Pointer {
		elemType = concreteType.Elem()
	}

	concretePtr := reflect.New(elemType)
	if err := json.Unmarshal(ctx.RawJSON, concretePtr.Interface()); err != nil {
		p.Errors = append(p.Errors, ValidationError{
			Loc:     ctx.Path,
			Message: fmt.Sprintf("failed to unmarshal discriminated union: %v", err),
			Type:    errors.ErrorTypeJSONDecode,
		})
		return nil
	}

	// Set the field value
	if concreteType.Kind() == reflect.Pointer {
		ctx.Value.Set(concretePtr)
	} else {
		ctx.Value.Set(concretePtr.Elem())
	}

	return nil
}

// unmarshalDiscriminatedSlice unmarshals a slice of discriminated union values.
func (p *UnmarshalProcessor) unmarshalDiscriminatedSlice(ctx *FieldContext, discriminatorField string, mapping map[string]any) error {
	var arrayData []json.RawMessage
	if err := json.Unmarshal(ctx.RawJSON, &arrayData); err != nil {
		p.Errors = append(p.Errors, ValidationError{
			Loc:     ctx.Path,
			Message: fmt.Sprintf("failed to parse array: %v", err),
			Type:    errors.ErrorTypeJSONDecode,
		})
		return nil
	}

	sliceType := ctx.Value.Type()
	sliceVal := reflect.MakeSlice(sliceType, 0, len(arrayData))

	for idx, elemData := range arrayData {
		elemPath := append(append([]string{}, ctx.Path...), fmt.Sprintf("[%d]", idx))

		// Peek at element JSON to find discriminator
		var elemMap map[string]any
		if err := json.Unmarshal(elemData, &elemMap); err != nil {
			p.Errors = append(p.Errors, ValidationError{
				Loc:     elemPath,
				Message: fmt.Sprintf("failed to parse element: %v", err),
				Type:    errors.ErrorTypeJSONDecode,
			})
			continue
		}

		discriminatorValue, ok := elemMap[discriminatorField]
		if !ok {
			p.Errors = append(p.Errors, ValidationError{
				Loc:     append(elemPath, discriminatorField),
				Message: fmt.Sprintf("discriminator field '%s' not found", discriminatorField),
				Type:    errors.ErrorTypeDiscriminatorMissing,
			})
			continue
		}

		// Look up concrete type
		concreteTypeExample, ok := mapping[fmt.Sprintf("%v", discriminatorValue)]
		if !ok {
			validValues := make([]string, 0, len(mapping))
			for k := range mapping {
				validValues = append(validValues, k)
			}
			p.Errors = append(p.Errors, ValidationError{
				Loc:     append(elemPath, discriminatorField),
				Message: fmt.Sprintf("invalid discriminator value '%v', expected one of: %v", discriminatorValue, validValues),
				Type:    errors.ErrorTypeDiscriminatorInvalid,
			})
			continue
		}

		// Create and unmarshal into concrete type
		concreteType := reflect.TypeOf(concreteTypeExample)
		elemType := concreteType
		if concreteType.Kind() == reflect.Pointer {
			elemType = concreteType.Elem()
		}

		concretePtr := reflect.New(elemType)
		if err := json.Unmarshal(elemData, concretePtr.Interface()); err != nil {
			p.Errors = append(p.Errors, ValidationError{
				Loc:     elemPath,
				Message: fmt.Sprintf("failed to unmarshal element: %v", err),
				Type:    errors.ErrorTypeJSONDecode,
			})
			continue
		}

		// Append to slice
		if concreteType.Kind() == reflect.Pointer {
			sliceVal = reflect.Append(sliceVal, concretePtr)
		} else {
			sliceVal = reflect.Append(sliceVal, concretePtr.Elem())
		}
	}

	ctx.Value.Set(sliceVal)
	return nil
}

// ShouldDescend controls recursion for unmarshaling.
// We allow descent into discriminated unions so the walker can validate elements.
func (p *UnmarshalProcessor) ShouldDescend(ctx *FieldContext) bool {
	// Allow descent even for discriminated unions - we've already unmarshaled them,
	// and now the walker needs to descend to validate individual fields of each element
	val := reflectutil.UnwrapValue(ctx.Value)

	// Descend into slices (let walker handle elements)
	if val.Kind() == reflect.Slice {
		return reflectutil.IsWalkableSliceElem(val.Type())
	}

	// Descend into non-basic struct types
	if val.Kind() != reflect.Struct {
		return false
	}
	return !reflectutil.IsBasicType(val.Type())
}
