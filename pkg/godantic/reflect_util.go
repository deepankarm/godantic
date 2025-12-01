package godantic

import (
	"reflect"

	"github.com/deepankarm/godantic/pkg/internal/reflectutil"
)

// fieldScanner provides common reflection utilities for scanning Field{Name}() methods
type fieldScanner struct{}

// scanFieldOptionsFromType scans a type for Field{Name}() methods and returns field options
// This is the core shared logic used by both Validator and schema generation
func (fs *fieldScanner) scanFieldOptionsFromType(typ reflect.Type) map[string]*fieldOptionHolder {
	fieldOptions := make(map[string]*fieldOptionHolder)

	// First, look for Field{Name}() methods on the parent struct
	ptrType := reflect.PointerTo(typ)
	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		if len(method.Name) > 5 && method.Name[:5] == "Field" {
			fieldName := method.Name[5:]
			// Call method to get options
			zeroPtr := reflect.New(typ)
			result := method.Func.Call([]reflect.Value{zeroPtr})
			if len(result) > 0 {
				holder := fs.extractFieldOptions(result[0])
				fieldOptions[fieldName] = holder
			}
		}
	}

	// Second, check each struct field for type-level validation
	// Only if parent struct didn't define Field{Name}() method
	// Skip if not a struct (e.g., slice, map, etc.)
	if typ.Kind() != reflect.Struct {
		return fieldOptions
	}
	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		fieldName := structField.Name

		// Skip if parent struct already defined validation for this field
		if _, exists := fieldOptions[fieldName]; exists {
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
				holder := fs.extractFieldOptions(result[0])
				fieldOptions[fieldName] = holder
			}
		}
	}

	return fieldOptions
}

// extractFieldOptions extracts validation info from FieldOptions[T] using reflection
func (fs *fieldScanner) extractFieldOptions(optsValue reflect.Value) *fieldOptionHolder {
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

// Global scanner instance for use across the package
var scanner = &fieldScanner{}

// FieldOptionInfo provides field validation info for schema generation
type FieldOptionInfo struct {
	Required    bool
	Constraints map[string]any
}

// toPublic converts the internal holder to the public FieldOptionInfo
func (foh *fieldOptionHolder) toPublic() FieldOptionInfo {
	return FieldOptionInfo{
		Required:    foh.required,
		Constraints: foh.constraints,
	}
}

// ScanTypeFieldOptions scans a reflect.Type for Field{Name}() methods
// Returns a map of field name -> field option info
// This is the public API for schema generation and other external uses
func ScanTypeFieldOptions(t reflect.Type) map[string]FieldOptionInfo {
	t = reflectutil.UnwrapPointer(t)
	internalOpts := scanner.scanFieldOptionsFromType(t)

	// Convert to public API
	result := make(map[string]FieldOptionInfo, len(internalOpts))
	for fieldName, holder := range internalOpts {
		result[fieldName] = holder.toPublic()
	}

	return result
}
