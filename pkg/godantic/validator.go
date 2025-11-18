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

	// Look for Field{Name}() methods
	ptrType := reflect.PointerTo(typ)
	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		if len(method.Name) > 5 && method.Name[:5] == "Field" {
			fieldName := method.Name[5:]
			// Call method to get options
			result := method.Func.Call([]reflect.Value{reflect.New(typ)})
			if len(result) > 0 {
				// Use reflection to extract validators
				optsValue := result[0]
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

				v.fieldOptions[fieldName] = holder
			}
		}
	}
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

		// Run validators if field is not zero or if validators should run on zero values
		if !field.IsZero() || len(opts.validators) > 0 {
			for _, validator := range opts.validators {
				if err := validator(value); err != nil {
					errs = append(errs, fmt.Errorf("%s: %w", fieldName, err))
				}
			}
		}
	}

	return errs
}

// FieldOptions returns the field options map (for schema generation)
func (v *Validator[T]) FieldOptions() map[string]*fieldOptionHolder {
	return v.fieldOptions
}
