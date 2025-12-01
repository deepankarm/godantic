// Package reflectutil provides shared reflection utilities for godantic.
package reflectutil

import (
	"reflect"
	"strings"
)

// JSONFieldName returns the JSON field name for a struct field.
// Returns the json tag name if present, otherwise the Go field name.
// Returns "-" for ignored fields.
func JSONFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx]
	}
	return tag
}

// FieldByJSONName finds a struct field value by its JSON name.
// Searches by exact match, capitalized version, and json tags.
func FieldByJSONName(val reflect.Value, typ reflect.Type, jsonName string) reflect.Value {
	// Unwrap pointers
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return reflect.Value{}
		}
		val = val.Elem()
		typ = typ.Elem()
	}

	// Try direct field name match
	if field := val.FieldByName(jsonName); field.IsValid() {
		return field
	}

	// Try capitalized version (common: "event" -> "Event")
	if len(jsonName) > 0 {
		capitalized := strings.ToUpper(jsonName[:1]) + jsonName[1:]
		if field := val.FieldByName(capitalized); field.IsValid() {
			return field
		}
	}

	// Search by JSON tag
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if JSONFieldName(field) == jsonName {
			return val.Field(i)
		}
	}

	return reflect.Value{}
}

// FieldByGoName finds a struct field by Go field name and returns its JSON name.
// Returns the default name if field not found.
func GoFieldToJSONName(typ reflect.Type, goFieldName string) string {
	typ = UnwrapPointer(typ)

	// Try direct field
	if field, ok := typ.FieldByName(goFieldName); ok {
		return JSONFieldName(field)
	}

	// Try embedded structs
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if embField, ok := field.Type.FieldByName(goFieldName); ok {
				return JSONFieldName(embField)
			}
		}
	}

	return goFieldName
}
