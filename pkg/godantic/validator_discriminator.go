package godantic

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
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

	discriminatorStr := fmt.Sprintf("%v", discriminatorValue)
	concreteType, err := lookupConcreteType(discriminatorStr, cfg)
	if err != nil {
		return nil, ValidationErrors{*err}
	}

	elemType := unwrapPointerType(concreteType)
	concretePtr := reflect.New(elemType)
	concreteInstance := concretePtr.Interface()

	if err := json.Unmarshal(data, concreteInstance); err != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("json unmarshal failed: %v", err),
			Type:    "json_decode",
		}}
	}

	errs := validateAndApplyDefaults(concretePtr, elemType)
	if len(errs) > 0 {
		result := convertToInterfaceType[T](concretePtr, concreteType)
		return &result, errs
	}

	result := convertToInterfaceType[T](concretePtr, concreteType)
	return &result, nil
}

// unmarshalDiscriminatedUnion handles unmarshaling (struct → JSON) for discriminated unions.
// It extracts the concrete type from the interface, applies defaults, validates, and marshals.
func (v *Validator[T]) unmarshalDiscriminatedUnion(obj *T, cfg *discriminatorConfig) ([]byte, ValidationErrors) {
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() != reflect.Pointer {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: "obj must be a pointer",
			Type:    "internal",
		}}
	}

	concreteValue := objValue.Elem()
	if !concreteValue.IsValid() || concreteValue.IsZero() {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: "obj is nil or zero value",
			Type:    "internal",
		}}
	}

	if concreteValue.Kind() == reflect.Interface {
		concreteValue = concreteValue.Elem()
		if !concreteValue.IsValid() {
			return nil, ValidationErrors{{
				Loc:     []string{},
				Message: "interface value is nil",
				Type:    "internal",
			}}
		}
	}

	concreteType := concreteValue.Type()
	discriminatorField := findFieldByJSONTag(concreteValue, concreteType, cfg.field)
	if !discriminatorField.IsValid() {
		return nil, ValidationErrors{{
			Loc:     []string{cfg.field},
			Message: fmt.Sprintf("discriminator field '%s' not found in type %s", cfg.field, concreteType.Name()),
			Type:    "discriminator_missing",
		}}
	}

	discriminatorValue := fmt.Sprintf("%v", discriminatorField.Interface())
	expectedType, err := lookupConcreteType(discriminatorValue, cfg)
	if err != nil {
		return nil, ValidationErrors{*err}
	}

	expectedElemType := unwrapPointerType(expectedType)
	if concreteType != expectedElemType && concreteType != expectedType {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("type mismatch: expected %s for discriminator '%s', got %s", expectedType.Name(), discriminatorValue, concreteType.Name()),
			Type:    "type_error",
		}}
	}

	// Get the element type (unwrap pointer if needed) for validation
	elemType := unwrapPointerType(concreteType)

	// If concreteValue is a pointer, dereference it to get the struct value
	structValue := concreteValue
	if concreteValue.Kind() == reflect.Pointer {
		structValue = concreteValue.Elem()
	}

	concretePtr := makeAddressable(structValue, elemType)
	errs := validateAndApplyDefaults(concretePtr, elemType)
	if len(errs) > 0 {
		return nil, errs
	}

	data, jsonErr := json.Marshal(concretePtr.Interface())
	if jsonErr != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("json marshal failed: %v", jsonErr),
			Type:    "json_encode",
		}}
	}
	return data, nil
}

// lookupConcreteType looks up the concrete type for a discriminator value
func lookupConcreteType(discriminatorValue string, cfg *discriminatorConfig) (reflect.Type, *ValidationError) {
	concreteType, ok := cfg.variants[discriminatorValue]
	if !ok {
		validValues := make([]string, 0, len(cfg.variants))
		for k := range cfg.variants {
			validValues = append(validValues, k)
		}
		return nil, &ValidationError{
			Loc:     []string{cfg.field},
			Message: fmt.Sprintf("invalid discriminator value '%s', expected one of: %v", discriminatorValue, validValues),
			Type:    "discriminator_invalid",
		}
	}
	return concreteType, nil
}

// unwrapPointerType returns the element type if the type is a pointer, otherwise returns the type itself
func unwrapPointerType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Pointer {
		return t.Elem()
	}
	return t
}

// validateAndApplyDefaults scans field options, applies defaults, and validates
func validateAndApplyDefaults(concretePtr reflect.Value, elemType reflect.Type) ValidationErrors {
	concreteFieldOptions := scanner.scanFieldOptionsFromType(elemType)
	if err := scanner.applyDefaultsToStruct(concretePtr, concreteFieldOptions); err != nil {
		return ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("apply defaults failed: %v", err),
			Type:    "internal",
		}}
	}

	return validateFieldsWithReflection(concretePtr, concreteFieldOptions, []string{}, nil)
}

// convertToInterfaceType converts a concrete value to interface type T
func convertToInterfaceType[T any](concretePtr reflect.Value, originalType reflect.Type) T {
	var result T
	if originalType.Kind() == reflect.Pointer {
		result = concretePtr.Interface().(T)
	} else {
		result = concretePtr.Elem().Interface().(T)
	}
	return result
}

// makeAddressable creates an addressable value from a possibly non-addressable value
func makeAddressable(value reflect.Value, typ reflect.Type) reflect.Value {
	if value.CanAddr() {
		return value.Addr()
	}
	newValue := reflect.New(typ)
	newValue.Elem().Set(value)
	return newValue
}

// findFieldByJSONTag finds a struct field by its JSON tag name or Go field name
func findFieldByJSONTag(value reflect.Value, typ reflect.Type, jsonName string) reflect.Value {
	if value.Kind() == reflect.Pointer {
		value = value.Elem()
		typ = typ.Elem()
	}

	if field := value.FieldByName(jsonName); field.IsValid() {
		return field
	}

	// Try capitalized version (common case: "event" -> "Event")
	if len(jsonName) > 0 {
		capitalized := strings.ToUpper(jsonName[:1]) + jsonName[1:]
		if field := value.FieldByName(capitalized); field.IsValid() {
			return field
		}
	}

	// Search by JSON tag
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			continue
		}
		tagName := strings.Split(jsonTag, ",")[0]
		if tagName == jsonName {
			return value.Field(i)
		}
	}

	return reflect.Value{}
}
