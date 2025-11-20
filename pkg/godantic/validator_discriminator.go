package godantic

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// validateDiscriminatedUnion handles validation for discriminated union types (interfaces)
// It peeks at the discriminator field, determines the concrete type, and validates accordingly.
func (v *Validator[T]) validateDiscriminatedUnion(data []byte, cfg *discriminatorConfig) (*T, ValidationErrors) {
	// First, peek at the discriminator field to determine which concrete type to use
	var peek map[string]any
	if err := json.Unmarshal(data, &peek); err != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("json unmarshal failed: %v", err),
			Type:    "json_decode",
		}}
	}

	// Get the discriminator value
	discriminatorValue, ok := peek[cfg.field]
	if !ok {
		return nil, ValidationErrors{{
			Loc:     []string{cfg.field},
			Message: fmt.Sprintf("discriminator field '%s' not found", cfg.field),
			Type:    "discriminator_missing",
		}}
	}

	// Convert discriminator value to string
	discriminatorStr := fmt.Sprintf("%v", discriminatorValue)

	// Look up the concrete type
	concreteType, ok := cfg.variants[discriminatorStr]
	if !ok {
		validValues := make([]string, 0, len(cfg.variants))
		for k := range cfg.variants {
			validValues = append(validValues, k)
		}
		return nil, ValidationErrors{{
			Loc:     []string{cfg.field},
			Message: fmt.Sprintf("invalid discriminator value '%s', expected one of: %v", discriminatorStr, validValues),
			Type:    "discriminator_invalid",
		}}
	}

	// Create a new instance of the concrete type
	concretePtr := reflect.New(concreteType)
	concreteInstance := concretePtr.Interface()

	// Unmarshal into the concrete type
	if err := json.Unmarshal(data, concreteInstance); err != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("json unmarshal failed: %v", err),
			Type:    "json_decode",
		}}
	}

	// Scan field options for the concrete type
	concreteFieldOptions := scanner.scanFieldOptionsFromType(concreteType)

	// Apply defaults
	if err := scanner.applyDefaultsToStruct(concretePtr, concreteFieldOptions); err != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("apply defaults failed: %v", err),
			Type:    "internal",
		}}
	}

	// Validate the concrete instance (no union validation for discriminated unions)
	errs := validateFieldsWithReflection(concretePtr, concreteFieldOptions, []string{}, nil)

	// Convert the concrete instance back to interface type T
	concreteValue := concretePtr.Elem().Interface()
	result := concreteValue.(T)

	if len(errs) > 0 {
		return &result, errs
	}

	return &result, nil
}
