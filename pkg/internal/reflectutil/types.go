package reflectutil

import "reflect"

// JSONSchemaType returns the JSON Schema type string for a Go type.
func JSONSchemaType(t reflect.Type) string {
	if t == nil {
		return ""
	}

	// Handle pointers recursively
	if t.Kind() == reflect.Pointer {
		return JSONSchemaType(t.Elem())
	}

	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	case reflect.Interface:
		return "object"
	}
	return ""
}

// MatchesJSONSchemaType checks if a reflect.Value matches a JSON Schema type name.
func MatchesJSONSchemaType(val reflect.Value, schemaType string) bool {
	// Handle null check
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

	got := JSONSchemaType(val.Type())
	if got == schemaType {
		return true
	}

	// "number" allows integer values
	if schemaType == "number" && got == "integer" {
		return true
	}

	return false
}

// IsBasicType checks if a type is a basic Go type (not a custom struct needing validation).
func IsBasicType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String,
		reflect.Slice, reflect.Map, reflect.Array:
		return true
	}

	// Skip standard library types
	if t.PkgPath() == "time" || t.PkgPath() == "sync" {
		return true
	}

	return false
}

// UnwrapPointer returns the element type if pointer, otherwise returns the type itself.
func UnwrapPointer(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Pointer {
		return t.Elem()
	}
	return t
}

// UnwrapPointerInfo returns the unwrapped type and whether it was a pointer.
func UnwrapPointerInfo(t reflect.Type) (unwrapped reflect.Type, isPointer bool) {
	if t.Kind() == reflect.Pointer {
		return t.Elem(), true
	}
	return t, false
}

// UnwrapValue unwraps pointers and interfaces to get the underlying value.
func UnwrapValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return v
		}
		v = v.Elem()
	}
	return v
}

// IsWalkableSliceElem checks if a slice's element type should be walked.
// Returns true for structs (non-basic) and interfaces (discriminated unions).
func IsWalkableSliceElem(sliceType reflect.Type) bool {
	elemType := UnwrapPointer(sliceType.Elem())
	// Interfaces (for discriminated unions) - actual elements are concrete structs
	if elemType.Kind() == reflect.Interface {
		return true
	}
	// Non-basic structs
	return elemType.Kind() == reflect.Struct && !IsBasicType(elemType)
}

// CollectStructTypes recursively collects all struct types from a type.
func CollectStructTypes(t reflect.Type, types map[string]reflect.Type) {
	if t == nil {
		return
	}

	t = UnwrapPointer(t)

	if t.Kind() != reflect.Struct {
		return
	}

	// Prevent infinite recursion
	typeName := t.Name()
	if typeName != "" {
		if _, exists := types[typeName]; exists {
			return
		}
		types[typeName] = t
	}

	// Process all fields
	for i := range t.NumField() {
		fieldType := t.Field(i).Type

		// Unwrap slices/arrays
		if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
			fieldType = fieldType.Elem()
		}

		fieldType = UnwrapPointer(fieldType)
		if fieldType.Kind() == reflect.Struct {
			CollectStructTypes(fieldType, types)
		}
	}
}

// MakeAddressable creates an addressable value from a possibly non-addressable value
func MakeAddressable(value reflect.Value, typ reflect.Type) reflect.Value {
	if value.CanAddr() {
		return value.Addr()
	}
	newValue := reflect.New(typ)
	newValue.Elem().Set(value)
	return newValue
}

// ConvertToInterfaceType converts a concrete value to interface type T
func ConvertToInterfaceType[T any](concretePtr reflect.Value, originalType reflect.Type) T {
	var result T
	if originalType.Kind() == reflect.Pointer {
		result = concretePtr.Interface().(T)
	} else {
		result = concretePtr.Elem().Interface().(T)
	}
	return result
}
