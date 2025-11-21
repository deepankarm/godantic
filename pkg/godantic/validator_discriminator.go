package godantic

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// validateDiscriminatedUnion handles validation for discriminated union types (interfaces)
func (v *Validator[T]) validateDiscriminatedUnion(data []byte, cfg *discriminatorConfig) (*T, ValidationErrors) {
	instance, errs := v.newUnionInstanceFromJSON(data, cfg)
	if errs != nil {
		return nil, errs
	}

	if decodeErr := processRecursive(instance.ptr, data); decodeErr != nil {
		return nil, decodeErr
	}

	if validationErrs := validateAndApplyDefaults(instance.ptr, instance.elemType); len(validationErrs) > 0 {
		result := convertToInterfaceType[T](instance.ptr, instance.concreteType)
		return &result, validationErrs
	}

	result := convertToInterfaceType[T](instance.ptr, instance.concreteType)
	return &result, nil
}

// unmarshalDiscriminatedUnion handles unmarshaling (struct → JSON) for discriminated unions
func (v *Validator[T]) unmarshalDiscriminatedUnion(obj *T, cfg *discriminatorConfig) ([]byte, ValidationErrors) {
	instance, err := v.newUnionInstanceFromStruct(obj, cfg)
	if err != nil {
		return nil, err
	}

	if syncErr := instance.syncNestedFromStruct(); syncErr != nil {
		return nil, syncErr
	}

	if errs := validateAndApplyDefaults(instance.ptr, instance.elemType); len(errs) > 0 {
		return nil, errs
	}

	data, marshalErr := json.Marshal(instance.ptr.Interface())
	if marshalErr != nil {
		return nil, ValidationErrors{{Loc: []string{}, Message: fmt.Sprintf("json marshal failed: %v", marshalErr), Type: "json_encode"}}
	}
	return data, nil
}

// lookupConcreteType looks up the concrete type for a discriminator value
func lookupConcreteType(discriminatorValue string, cfg *discriminatorConfig) (reflect.Type, *ValidationError) {
	if concreteType, ok := cfg.variants[discriminatorValue]; ok {
		return concreteType, nil
	}
	validValues := make([]string, 0, len(cfg.variants))
	for k := range cfg.variants {
		validValues = append(validValues, k)
	}
	return nil, &ValidationError{Loc: []string{cfg.field}, Message: fmt.Sprintf("invalid discriminator value '%s', expected one of: %v", discriminatorValue, validValues), Type: "discriminator_invalid"}
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

// processRecursive unmarshals JSON into a value, handling discriminated unions recursively
func processRecursive(val reflect.Value, data []byte) ValidationErrors {
	// Handle pointer types - recurse into the element
	if val.Type().Elem().Kind() == reflect.Pointer {
		if val.Elem().IsNil() {
			val.Elem().Set(reflect.New(val.Type().Elem().Elem()))
		}
		return processRecursive(val.Elem(), data)
	}

	t := val.Type().Elem() // val is pointer to struct or simple type

	// Base case: Not a struct, just unmarshal
	if t.Kind() != reflect.Struct {
		if err := json.Unmarshal(data, val.Interface()); err != nil {
			return ValidationErrors{{Loc: []string{}, Message: fmt.Sprintf("json unmarshal failed: %v", err), Type: "json_decode"}}
		}
		return nil
	}

	// It is a struct. Parse JSON to map for field access.
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		return ValidationErrors{{Loc: []string{}, Message: fmt.Sprintf("failed to parse JSON: %v", err), Type: "json_decode"}}
	}

	fieldOptions := scanner.scanFieldOptionsFromType(t)
	structVal := val.Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := structVal.Field(i)
		if !fieldVal.CanSet() {
			continue
		}

		// Handle embedded/anonymous fields - they use the full JSON data
		if field.Anonymous {
			fieldPtr := reflect.New(field.Type)
			if err := processRecursive(fieldPtr, data); err != nil {
				return err
			}
			fieldVal.Set(fieldPtr.Elem())
			continue
		}

		jsonName := getJSONFieldName(field)
		if jsonName == "-" {
			continue // Skip ignored fields
		}

		rawFieldData, hasFieldData := rawData[jsonName]
		if !hasFieldData {
			continue
		}

		// Check for discriminated union constraint
		var discriminatorConstraint map[string]any
		hasDiscriminator := false
		if opts, ok := fieldOptions[field.Name]; ok {
			discriminatorConstraint, hasDiscriminator = opts.Constraints()[ConstraintDiscriminator].(map[string]any)
		}

		if hasDiscriminator {
			// Handle Union
			discriminatorField, _ := discriminatorConstraint["propertyName"].(string)
			mapping, _ := discriminatorConstraint["mapping"].(map[string]any)

			// Peek at discriminator value
			var fieldMap map[string]any
			if err := json.Unmarshal(rawFieldData, &fieldMap); err != nil {
				return ValidationErrors{{Loc: []string{field.Name}, Message: fmt.Sprintf("failed to parse discriminated union field: %v", err), Type: "json_decode"}}
			}

			discriminatorValue, ok := fieldMap[discriminatorField]
			if !ok {
				return ValidationErrors{{Loc: []string{field.Name, discriminatorField}, Message: fmt.Sprintf("discriminator field '%s' not found", discriminatorField), Type: "discriminator_missing"}}
			}

			// Look up concrete type
			concreteTypeExample, ok := mapping[fmt.Sprintf("%v", discriminatorValue)]
			if !ok {
				validValues := make([]string, 0, len(mapping))
				for k := range mapping {
					validValues = append(validValues, k)
				}
				return ValidationErrors{{Loc: []string{field.Name, discriminatorField}, Message: fmt.Sprintf("invalid discriminator value '%v', expected one of: %v", discriminatorValue, validValues), Type: "discriminator_invalid"}}
			}

			concreteType := reflect.TypeOf(concreteTypeExample)
			elemType := unwrapPointerType(concreteType)
			concretePtr := reflect.New(elemType)

			// Recurse into the concrete type
			if err := processRecursive(concretePtr, rawFieldData); err != nil {
				return err
			}

			if concreteType.Kind() == reflect.Pointer {
				fieldVal.Set(concretePtr)
			} else {
				fieldVal.Set(concretePtr.Elem())
			}
		} else {
			// Regular field - recurse
			fieldPtr := reflect.New(field.Type)
			if err := processRecursive(fieldPtr, rawFieldData); err != nil {
				return err
			}
			fieldVal.Set(fieldPtr.Elem())
		}
	}
	return nil
}

// getJSONFieldName returns the JSON field name for a struct field
func getJSONFieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return field.Name
	}
	tagName := strings.Split(jsonTag, ",")[0]
	return tagName // Return "-" as-is so caller can skip it
}

// unionInstance encapsulates the reusable discriminated union processing state
type unionInstance struct {
	ptr          reflect.Value
	elemType     reflect.Type
	concreteType reflect.Type
	fieldOptions map[string]*fieldOptionHolder
}

func (v *Validator[T]) newUnionInstanceFromJSON(data []byte, cfg *discriminatorConfig) (*unionInstance, ValidationErrors) {
	var peek map[string]any
	if err := json.Unmarshal(data, &peek); err != nil {
		return nil, ValidationErrors{{Loc: []string{}, Message: fmt.Sprintf("json unmarshal failed: %v", err), Type: "json_decode"}}
	}

	discriminatorValue, ok := peek[cfg.field]
	if !ok {
		return nil, ValidationErrors{{Loc: []string{cfg.field}, Message: fmt.Sprintf("discriminator field '%s' not found", cfg.field), Type: "discriminator_missing"}}
	}

	concreteType, validationErr := lookupConcreteType(fmt.Sprintf("%v", discriminatorValue), cfg)
	if validationErr != nil {
		return nil, ValidationErrors{*validationErr}
	}

	elemType := unwrapPointerType(concreteType)
	return &unionInstance{
		ptr:          reflect.New(elemType),
		elemType:     elemType,
		concreteType: concreteType,
		fieldOptions: scanner.scanFieldOptionsFromType(elemType),
	}, nil
}

func (v *Validator[T]) newUnionInstanceFromStruct(obj *T, cfg *discriminatorConfig) (*unionInstance, ValidationErrors) {
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() != reflect.Pointer {
		return nil, ValidationErrors{{Loc: []string{}, Message: "obj must be a pointer", Type: "internal"}}
	}

	concreteValue := objValue.Elem()
	if !concreteValue.IsValid() || concreteValue.IsZero() {
		return nil, ValidationErrors{{Loc: []string{}, Message: "obj is nil or zero value", Type: "internal"}}
	}

	if concreteValue.Kind() == reflect.Interface {
		concreteValue = concreteValue.Elem()
		if !concreteValue.IsValid() {
			return nil, ValidationErrors{{Loc: []string{}, Message: "interface value is nil", Type: "internal"}}
		}
	}

	concreteType := concreteValue.Type()
	discriminatorField := findFieldByJSONTag(concreteValue, concreteType, cfg.field)
	if !discriminatorField.IsValid() {
		return nil, ValidationErrors{{Loc: []string{cfg.field}, Message: fmt.Sprintf("discriminator field '%s' not found in type %s", cfg.field, concreteType.Name()), Type: "discriminator_missing"}}
	}

	expectedType, validationErr := lookupConcreteType(fmt.Sprintf("%v", discriminatorField.Interface()), cfg)
	if validationErr != nil {
		return nil, ValidationErrors{*validationErr}
	}

	expectedElemType := unwrapPointerType(expectedType)
	if concreteType != expectedElemType && concreteType != expectedType {
		return nil, ValidationErrors{{Loc: []string{}, Message: fmt.Sprintf("type mismatch: expected %s, got %s", expectedType.Name(), concreteType.Name()), Type: "type_error"}}
	}

	elemType := unwrapPointerType(concreteType)
	structValue := concreteValue
	if concreteValue.Kind() == reflect.Pointer {
		structValue = concreteValue.Elem()
	}

	return &unionInstance{
		ptr:          makeAddressable(structValue, elemType),
		elemType:     elemType,
		concreteType: concreteType,
		fieldOptions: scanner.scanFieldOptionsFromType(elemType),
	}, nil
}

func (i *unionInstance) syncNestedFromStruct() ValidationErrors {
	if !hasNestedDiscriminatedUnions(i.fieldOptions) {
		return nil
	}

	tempJSON, err := json.Marshal(i.ptr.Interface())
	if err != nil {
		return ValidationErrors{{Loc: []string{}, Message: fmt.Sprintf("json marshal failed: %v", err), Type: "json_encode"}}
	}
	return processRecursive(i.ptr, tempJSON)
}

func hasNestedDiscriminatedUnions(fieldOptions map[string]*fieldOptionHolder) bool {
	for _, opts := range fieldOptions {
		if _, ok := opts.Constraints()[ConstraintDiscriminator]; ok {
			return true
		}
	}
	return false
}
