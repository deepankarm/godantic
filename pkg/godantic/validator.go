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

type ValidationErrors []ValidationError

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

func (es ValidationErrors) Unwrap() []error {
	errs := make([]error, len(es))
	for i, e := range es {
		errs[i] = e
	}
	return errs
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

// Validator validates structs or discriminated union interfaces
type Validator[T any] struct {
	fieldOptions map[string]*fieldOptionHolder
	config       validatorConfig
}

// NewValidator creates a new validator for type T.
// For concrete structs, use without options: NewValidator[MyStruct]()
// For discriminated unions (interfaces), use with WithDiscriminator option:
//
//	NewValidator[MyInterface](WithDiscriminator("type", map[string]any{...}))
func NewValidator[T any](opts ...ValidatorOption) *Validator[T] {
	v := &Validator[T]{
		fieldOptions: make(map[string]*fieldOptionHolder),
	}

	// Apply options
	for _, opt := range opts {
		opt.apply(&v.config)
	}

	// Only scan field options if this is a concrete struct (not a discriminated union interface)
	if v.config.discriminator == nil {
		v.scanFieldOptions()
	}

	return v
}

func (v *Validator[T]) scanFieldOptions() {
	var zero T
	typ := reflect.TypeOf(zero)
	v.fieldOptions = scanner.scanFieldOptionsFromType(typ)
}

func (v *Validator[T]) Validate(obj *T) ValidationErrors {
	return v.validateWithPath(obj, []string{})
}

// validateWithPath validates the struct and tracks the field path for nested validation
func (v *Validator[T]) validateWithPath(obj *T, path []string) ValidationErrors {
	objPtr := reflect.ValueOf(obj)
	return validateFieldsWithReflection(objPtr, v.fieldOptions, path, v.validateUnionConstraints)
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
	// Handle null check (value-based)
	if schemaType == "null" {
		if !val.IsValid() {
			return true
		}
		switch val.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
			return val.IsNil()
		default:
			return false
		}
	}

	if !val.IsValid() {
		return false
	}

	// Use shared type mapping
	got := GetJSONSchemaType(val.Type())
	if got == schemaType {
		return true
	}

	// Special case: "number" schema type allows integer values
	if schemaType == "number" && got == "integer" {
		return true
	}

	return false
}

// ApplyDefaults applies default values to zero-valued fields that have defaults defined.
// This should be called after JSON unmarshaling to set defaults for missing fields.
// Returns an error if reflection fails.
func (v *Validator[T]) ApplyDefaults(obj *T) error {
	objPtr := reflect.ValueOf(obj)
	return scanner.applyDefaultsToStruct(objPtr, v.fieldOptions)
}

// Marshal unmarshals JSON data, applies defaults, and validates.
// This is a convenience method that combines the three common steps:
// 1. Unmarshal JSON into the struct (or route to correct type for discriminated unions)
// 2. Apply default values to zero-valued fields
// 3. Validate the struct
// Returns the populated struct and any validation errors.
func (v *Validator[T]) Marshal(data []byte) (*T, ValidationErrors) {
	// Check if this is a discriminated union validator
	if v.config.discriminator != nil {
		return v.validateDiscriminatedUnion(data, v.config.discriminator)
	}

	// Standard struct validation
	var obj T
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("json unmarshal failed: %v", err),
			Type:    "json_decode",
		}}
	}

	if err := v.ApplyDefaults(&obj); err != nil {
		return nil, ValidationErrors{{
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

// Unmarshal validates the struct, applies defaults, and marshals to JSON.
// This is a convenience method that combines the three common steps:
// 1. Validate the struct
// 2. Apply default values to zero-valued fields
// 3. Marshal the struct to JSON
// Returns the JSON bytes and any validation errors.
func (v *Validator[T]) Unmarshal(obj *T) ([]byte, ValidationErrors) {
	// Check if this is a discriminated union validator
	if v.config.discriminator != nil {
		return v.unmarshalDiscriminatedUnion(obj, v.config.discriminator)
	}

	// Standard struct validation
	// Validate first
	errs := v.Validate(obj)
	if len(errs) > 0 {
		return nil, errs
	}

	// Apply defaults to ensure all default values are set
	if err := v.ApplyDefaults(obj); err != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("apply defaults failed: %v", err),
			Type:    "internal",
		}}
	}

	// Marshal to JSON
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("json marshal failed: %v", err),
			Type:    "json_encode",
		}}
	}

	return data, nil
}

// FieldOptions returns the field options map (for schema generation)
func (v *Validator[T]) FieldOptions() map[string]any {
	result := make(map[string]any, len(v.fieldOptions))
	for k, v := range v.fieldOptions {
		result[k] = v
	}
	return result
}
