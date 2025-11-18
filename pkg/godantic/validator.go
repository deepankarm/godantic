package godantic

import (
	"fmt"
	"reflect"
)

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

func (v *Validator[T]) Validate(obj *T) []error {
	var errs []error
	val := reflect.ValueOf(obj).Elem()

	for fieldName, opts := range v.fieldOptions {
		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}

		// Get the actual field value
		value := field.Interface()

		// Check for required fields (zero value check)
		if opts.required && field.IsZero() {
			errs = append(errs, fmt.Errorf("%s: required field", fieldName))
			continue
		}

		// Skip validation for optional fields with zero values
		if field.IsZero() {
			continue
		}

		// Run validators for non-zero values
		for _, validator := range opts.validators {
			if err := validator(value); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", fieldName, err))
			}
		}
	}

	return errs
}

// FieldOptions returns the field options map (for schema generation)
func (v *Validator[T]) FieldOptions() map[string]*fieldOptionHolder {
	return v.fieldOptions
}
