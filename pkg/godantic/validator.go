package godantic

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// ValidationError represents a validation error with location information
type ValidationError struct {
	Loc     []string // Path to the field, e.g., ["Address", "ZipCode"]
	Message string   // Human-readable error message
	Type    string   // Error type, e.g., "required", "constraint", "format"
}

// Error implements the error interface
func (e ValidationError) Error() string {
	if len(e.Loc) == 0 {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", strings.Join(e.Loc, "."), e.Message)
}

// Ordered is a constraint for types that support comparison
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 | ~string
}

// FieldOptions defines validation rules and metadata
type FieldOptions[T any] struct {
	Required_    bool
	Validators_  []func(T) error
	Constraints_ map[string]any // For schema generation (description, example, min, max, minLength, etc.)
}

func (fo FieldOptions[T]) validateWith(fn func(T) error) FieldOptions[T] {
	fo.Validators_ = append(fo.Validators_, fn)
	return fo
}

// Field creates a FieldOptions with multiple constraints applied
func Field[T any](fns ...func(FieldOptions[T]) FieldOptions[T]) FieldOptions[T] {
	fo := FieldOptions[T]{}
	for _, fn := range fns {
		fo = fn(fo)
	}
	return fo
}

// Required marks a field as required (can be used with Field)
func Required[T any]() func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo.Required_ = true
		return fo
	}
}

// Validate adds a custom validator function (can be used with Field)
func Validate[T any](fn func(T) error) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo.Validators_ = append(fo.Validators_, fn)
		return fo
	}
}

// fieldOptionHolder holds field options with type erasure
type fieldOptionHolder struct {
	required    bool
	validators  []func(any) error
	constraints map[string]any // Includes description, example, and all schema metadata
}

// Required returns whether the field is required
func (foh *fieldOptionHolder) Required() bool {
	return foh.required
}

// Constraints returns the constraints map
func (foh *fieldOptionHolder) Constraints() map[string]any {
	return foh.constraints
}

// Validator validates structs
type Validator[T any] struct {
	fieldOptions map[string]*fieldOptionHolder
}

func NewValidator[T any]() *Validator[T] {
	v := &Validator[T]{
		fieldOptions: make(map[string]*fieldOptionHolder),
	}
	v.scanFieldOptions()
	return v
}

func (v *Validator[T]) scanFieldOptions() {
	var zero T
	val := reflect.ValueOf(&zero).Elem()
	typ := val.Type()

	// First, look for Field{Name}() methods on the parent struct
	ptrType := reflect.PointerTo(typ)
	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		if len(method.Name) > 5 && method.Name[:5] == "Field" {
			fieldName := method.Name[5:]
			// Call method to get options
			result := method.Func.Call([]reflect.Value{reflect.New(typ)})
			if len(result) > 0 {
				holder := v.extractFieldOptions(result[0])
				v.fieldOptions[fieldName] = holder
			}
		}
	}

	// Second, check each struct field for type-level validation
	// Only if parent struct didn't define Field{Name}() method
	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		fieldName := structField.Name

		// Skip if parent struct already defined validation for this field
		if _, exists := v.fieldOptions[fieldName]; exists {
			continue
		}

		// Check if the field's type has Field{TypeName}() method
		fieldType := structField.Type
		typeName := fieldType.Name()
		if typeName == "" {
			continue // Skip anonymous types
		}

		methodName := "Field" + typeName

		// Try pointer receiver first
		ptrFieldType := reflect.PointerTo(fieldType)
		method, found := ptrFieldType.MethodByName(methodName)
		if !found {
			// Try value receiver
			method, found = fieldType.MethodByName(methodName)
		}

		if found {
			// Create a zero value instance of the field type
			var fieldInstance reflect.Value
			if method.Type.In(0).Kind() == reflect.Pointer {
				fieldInstance = reflect.New(fieldType)
			} else {
				fieldInstance = reflect.Zero(fieldType)
			}

			// Call the type's Field{TypeName}() method
			result := method.Func.Call([]reflect.Value{fieldInstance})
			if len(result) > 0 {
				holder := v.extractFieldOptions(result[0])
				v.fieldOptions[fieldName] = holder
			}
		}
	}
}

// extractFieldOptions extracts validation info from FieldOptions[T] using reflection
func (v *Validator[T]) extractFieldOptions(optsValue reflect.Value) *fieldOptionHolder {
				holder := &fieldOptionHolder{
					required:    optsValue.FieldByName("Required_").Bool(),
					validators:  []func(any) error{},
					constraints: make(map[string]any),
				}

				// Extract constraints map
				constraintsField := optsValue.FieldByName("Constraints_")
				if constraintsField.IsValid() && !constraintsField.IsNil() {
					// Copy constraints map
					iter := constraintsField.MapRange()
					for iter.Next() {
						key := iter.Key().String()
						value := iter.Value().Interface()
						holder.constraints[key] = value
					}
				}

				// Extract validators using reflection
				validatorsField := optsValue.FieldByName("Validators_")
				if validatorsField.IsValid() && validatorsField.Len() > 0 {
					for j := 0; j < validatorsField.Len(); j++ {
						validatorFunc := validatorsField.Index(j)
						// Wrap the typed validator in a type-erased one
						holder.validators = append(holder.validators, func(val any) error {
							// Call the validator using reflection
							results := validatorFunc.Call([]reflect.Value{reflect.ValueOf(val)})
							if len(results) > 0 && !results[0].IsNil() {
								return results[0].Interface().(error)
							}
							return nil
						})
					}
				}

	return holder
			}

func (v *Validator[T]) Validate(obj *T) []ValidationError {
	return v.validateWithPath(obj, []string{})
}

// validateWithPath validates the struct and tracks the field path for nested validation
func (v *Validator[T]) validateWithPath(obj *T, path []string) []ValidationError {
	var errs []ValidationError
	val := reflect.ValueOf(obj).Elem()

	for fieldName, opts := range v.fieldOptions {
		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}

		// Build current field path
		currentPath := append(append([]string{}, path...), fieldName)

		// Get the actual field value
		value := field.Interface()

		// Check for required fields (zero value check)
		if opts.required && field.IsZero() {
			// If field has a default value, it's not an error (will be applied via ApplyDefaults)
			if _, hasDefault := opts.constraints[ConstraintDefault]; !hasDefault {
				errs = append(errs, ValidationError{
					Loc:     currentPath,
					Message: "required field",
					Type:    "required",
				})
				continue
			}
		}

		// Skip validation for optional fields with zero values
		if field.IsZero() {
			continue
		}

		// Run validators for non-zero values
			for _, validator := range opts.validators {
				if err := validator(value); err != nil {
				errs = append(errs, ValidationError{
					Loc:     currentPath,
					Message: err.Error(),
					Type:    "constraint",
				})
			}
		}

		// Validate union constraints
		if unionErr := v.validateUnionConstraints(value, opts.constraints, currentPath); unionErr != nil {
			errs = append(errs, *unionErr)
		}

		// Recursively validate nested structs
		if field.Kind() == reflect.Struct && !isBasicType(field.Type()) {
			nestedErrs := v.validateNested(field, currentPath)
			errs = append(errs, nestedErrs...)
				}

		// Validate pointer to struct
		if field.Kind() == reflect.Pointer && !field.IsNil() && field.Elem().Kind() == reflect.Struct {
			nestedErrs := v.validateNested(field.Elem(), currentPath)
			errs = append(errs, nestedErrs...)
		}
	}

	return errs
}

// validateNested validates a nested struct by calling its validator if it has Field methods
func (v *Validator[T]) validateNested(field reflect.Value, parentPath []string) []ValidationError {
	// Get the type of the nested struct
	fieldType := field.Type()

	// Check if the nested struct has Field methods (has validation)
	ptrType := reflect.PointerTo(fieldType)
	hasValidation := false
	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		if len(method.Name) > 5 && method.Name[:5] == "Field" {
			hasValidation = true
			break
		}
	}

	if !hasValidation {
		return nil
	}

	// For nested validation, we need to scan the nested struct's Field methods
	// and validate recursively
	var errs []ValidationError
	ptrVal := field.Addr()

	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		if len(method.Name) <= 5 || method.Name[:5] != "Field" {
			continue
		}

		nestedFieldName := method.Name[5:]
		nestedField := field.FieldByName(nestedFieldName)
		if !nestedField.IsValid() {
			continue
		}

		// Get field options by calling the Field method
		result := method.Func.Call([]reflect.Value{ptrVal})
		if len(result) == 0 {
			continue
		}

		// Extract validation info from the result
		nestedFieldPath := append(append([]string{}, parentPath...), nestedFieldName)
		resultType := result[0].Type()

		if resultType.Kind() != reflect.Struct {
			continue
		}

		// Extract Required_
		requiredField := result[0].FieldByName("Required_")
		isRequired := requiredField.IsValid() && requiredField.Kind() == reflect.Bool && requiredField.Bool()

		// Extract Constraints_ to check for defaults
		constraintsField := result[0].FieldByName("Constraints_")
		hasDefault := false
		if constraintsField.IsValid() && constraintsField.Kind() == reflect.Map {
			defaultVal := constraintsField.MapIndex(reflect.ValueOf(ConstraintDefault))
			hasDefault = defaultVal.IsValid()
		}

		// Check if field is zero and required
		if nestedField.IsZero() {
			if isRequired && !hasDefault {
				errs = append(errs, ValidationError{
					Loc:     nestedFieldPath,
					Message: "required field",
					Type:    "required",
				})
				continue
			}
			// Skip validation for optional zero fields
			continue
		}

		// Extract and run validators
		validatorsField := result[0].FieldByName("Validators_")
		if validatorsField.IsValid() && validatorsField.Kind() == reflect.Slice {
			fieldValue := nestedField.Interface()
			for j := 0; j < validatorsField.Len(); j++ {
				validator := validatorsField.Index(j)
				if validator.Kind() == reflect.Func {
					// Call the validator function
					validatorResults := validator.Call([]reflect.Value{reflect.ValueOf(fieldValue)})
					if len(validatorResults) > 0 && !validatorResults[0].IsNil() {
						// Validator returned an error
						if err, ok := validatorResults[0].Interface().(error); ok {
							errs = append(errs, ValidationError{
								Loc:     nestedFieldPath,
								Message: err.Error(),
								Type:    "constraint",
							})
						}
					}
				}
			}
		}

		// Recursively validate nested structs
		if nestedField.Kind() == reflect.Struct && !isBasicType(nestedField.Type()) {
			nestedErrs := v.validateNested(nestedField, nestedFieldPath)
			errs = append(errs, nestedErrs...)
		}

		// Validate pointer to nested struct
		if nestedField.Kind() == reflect.Ptr && !nestedField.IsNil() && nestedField.Elem().Kind() == reflect.Struct {
			nestedErrs := v.validateNested(nestedField.Elem(), nestedFieldPath)
			errs = append(errs, nestedErrs...)
		}
	}
	return errs
}

// isBasicType checks if a type is a basic Go type (not a custom struct that needs validation)
func isBasicType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String,
		reflect.Slice, reflect.Map, reflect.Array:
		return true
	}

	// Check for time.Time and other standard library types we don't want to recurse into
	if t.PkgPath() == "time" || t.PkgPath() == "sync" {
		return true
	}

	return false
}

// validateUnionConstraints validates union (anyOf) and discriminated union (oneOf) constraints
func (v *Validator[T]) validateUnionConstraints(value any, constraints map[string]any, path []string) *ValidationError {
	// Check for discriminated union first (more specific)
	if discriminatorConstraint, ok := constraints[ConstraintDiscriminator]; ok {
		if discMap, ok := discriminatorConstraint.(map[string]any); ok {
			discriminatorField, _ := discMap["propertyName"].(string)
			mapping, _ := discMap["mapping"].(map[string]any)

			if discriminatorField != "" && mapping != nil {
				// Value must be a struct
				valReflect := reflect.ValueOf(value)
				if valReflect.Kind() != reflect.Struct {
					return &ValidationError{
						Loc:     path,
						Message: "discriminated union requires a struct type",
						Type:    "constraint",
					}
				}

				// Get the discriminator field value
				discField := valReflect.FieldByName(discriminatorField)
				if !discField.IsValid() {
					return &ValidationError{
						Loc:     path,
						Message: fmt.Sprintf("discriminator field '%s' not found", discriminatorField),
						Type:    "constraint",
					}
				}

				discValue := fmt.Sprintf("%v", discField.Interface())

				// Check if the discriminator value is in the allowed mapping
				if _, ok := mapping[discValue]; !ok {
					validValues := make([]string, 0, len(mapping))
					for k := range mapping {
						validValues = append(validValues, k)
					}
					return &ValidationError{
						Loc:     path,
						Message: fmt.Sprintf("invalid discriminator value '%s', expected one of: %v", discValue, validValues),
						Type:    "constraint",
					}
				}

				// Valid discriminated union value
				return nil
			}
		}
	}

	// Check for simple union (anyOf)
	var allowedTypes []string
	var complexTypes []any

	// Collect primitive type constraints
	if anyOf, ok := constraints[ConstraintAnyOf]; ok {
		if anyOfSlice, ok := anyOf.([]map[string]string); ok {
			for _, typeMap := range anyOfSlice {
				if typeName, ok := typeMap["type"]; ok {
					allowedTypes = append(allowedTypes, typeName)
				}
			}
		}
	}

	// Collect complex type constraints
	if anyOfTypes, ok := constraints["anyOfTypes"]; ok {
		if types, ok := anyOfTypes.([]any); ok {
			complexTypes = types
		}
	}

	// If no union constraints, validation passes
	if len(allowedTypes) == 0 && len(complexTypes) == 0 {
		return nil
	}

	// Get the actual type of the value
	valReflect := reflect.ValueOf(value)
	valType := valReflect.Type()

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
		if matchesJSONSchemaType(valReflect, allowedType) {
			return nil // Match found
		}
	}

	// No match found
	allTypes := append([]string{}, allowedTypes...)
	for _, ct := range complexTypes {
		allTypes = append(allTypes, reflect.TypeOf(ct).String())
	}

	return &ValidationError{
		Loc:     path,
		Message: fmt.Sprintf("value does not match any allowed type: %v", allTypes),
		Type:    "constraint",
	}
}

// matchesJSONSchemaType checks if a reflect.Value matches a JSON Schema type name
func matchesJSONSchemaType(val reflect.Value, schemaType string) bool {
	switch schemaType {
	case "string":
		return val.Kind() == reflect.String
	case "integer":
		switch val.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return true
		}
	case "number":
		switch val.Kind() {
		case reflect.Float32, reflect.Float64,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return true
		}
	case "boolean":
		return val.Kind() == reflect.Bool
	case "object":
		return val.Kind() == reflect.Map || val.Kind() == reflect.Struct
	case "array":
		return val.Kind() == reflect.Slice || val.Kind() == reflect.Array
	case "null":
		return !val.IsValid() || val.IsNil()
	}
	return false
}

// ApplyDefaults applies default values to zero-valued fields that have defaults defined.
// This should be called after JSON unmarshaling to set defaults for missing fields.
// Returns an error if reflection fails.
func (v *Validator[T]) ApplyDefaults(obj *T) error {
	val := reflect.ValueOf(obj).Elem()

	for fieldName, opts := range v.fieldOptions {
		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}

		// Only apply default if field is zero value
		if !field.IsZero() {
			continue
		}

		// Check if field can be set
		if !field.CanSet() {
			continue
		}

		// Get default from constraints
		defaultVal, ok := opts.constraints[ConstraintDefault]
		if !ok {
			continue
		}

		// Set the default value
		defaultReflect := reflect.ValueOf(defaultVal)
		if defaultReflect.Type().AssignableTo(field.Type()) {
			field.Set(defaultReflect)
		}
	}

	return nil
}

// ValidateJSON unmarshals JSON data, applies defaults, and validates.
// This is a convenience method that combines the three common steps:
// 1. Unmarshal JSON into the struct
// 2. Apply default values to zero-valued fields
// 3. Validate the struct
// Returns the populated struct and any validation errors.
func (v *Validator[T]) ValidateJSON(data []byte) (*T, []ValidationError) {
	var obj T
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, []ValidationError{{
			Loc:     []string{},
			Message: fmt.Sprintf("json unmarshal failed: %v", err),
			Type:    "json_decode",
		}}
	}

	if err := v.ApplyDefaults(&obj); err != nil {
		return nil, []ValidationError{{
			Loc:     []string{},
			Message: fmt.Sprintf("apply defaults failed: %v", err),
			Type:    "internal",
		}}
	}

	errs := v.Validate(&obj)
	if len(errs) > 0 {
		return &obj, errs
	}

	return &obj, nil
}

// FieldOptions returns the field options map (for schema generation)
func (v *Validator[T]) FieldOptions() map[string]any {
	result := make(map[string]any, len(v.fieldOptions))
	for k, v := range v.fieldOptions {
		result[k] = v
	}
	return result
}
