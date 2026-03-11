package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// typeRegistry maps JSON schema $defs type names to their Go reflect.Type.
// Used to determine struct field declaration order for JSON serialization.
type typeRegistry map[string]reflect.Type

// buildTypeRegistry builds a registry from the types used in GenerateUnionSchema.
func buildTypeRegistry(types ...any) typeRegistry {
	reg := make(typeRegistry)
	for _, t := range types {
		rt := reflect.TypeOf(t)
		if rt.Kind() == reflect.Pointer {
			rt = rt.Elem()
		}
		reg.register(rt)
	}
	return reg
}

// register recursively registers a type and all its struct field types.
func (r typeRegistry) register(t reflect.Type) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Slice {
		r.register(t.Elem())
		return
	}
	if t.Kind() != reflect.Struct {
		return
	}
	name := t.Name()
	if name == "" {
		return
	}
	if _, exists := r[name]; exists {
		return
	}
	r[name] = t
	for i := 0; i < t.NumField(); i++ {
		r.register(t.Field(i).Type)
	}
}

// fieldOrder returns json tag names of a struct in declaration order.
func (r typeRegistry) fieldOrder(typeName string) []string {
	t, ok := r[typeName]
	if !ok {
		return nil
	}
	var order []string
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		order = append(order, strings.Split(tag, ",")[0])
	}
	return order
}

// GenerateUnionSchemaOrdered generates a JSON schema with anyOf from multiple types,
// returning json.RawMessage with struct field declaration order preserved.
// This is required for OpenAI structured output where property ordering
// affects model behavior.
func GenerateUnionSchemaOrdered(types ...any) (json.RawMessage, error) {
	schema, err := GenerateUnionSchema(types...)
	if err != nil {
		return nil, fmt.Errorf("generating union schema: %w", err)
	}
	if schema == nil {
		return nil, nil
	}
	reg := buildTypeRegistry(types...)
	return marshalOrdered(schema, reg)
}

// marshalOrdered serializes a schema with properties ordered by struct field
// declaration order where type info is available.
func marshalOrdered(schema map[string]any, reg typeRegistry) (json.RawMessage, error) {
	var buf bytes.Buffer
	if err := writeValue(&buf, schema, reg, ""); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeValue writes any JSON value with ordered keys for objects.
// typeName is the Go type name context (used to look up field order for "properties").
func writeValue(buf *bytes.Buffer, v any, reg typeRegistry, typeName string) error {
	switch val := v.(type) {
	case map[string]any:
		return writeObject(buf, val, reg, typeName)
	case []any:
		return writeArray(buf, val, reg)
	default:
		data, err := json.Marshal(val)
		if err != nil {
			return err
		}
		buf.Write(data)
		return nil
	}
}

func writeObject(buf *bytes.Buffer, obj map[string]any, reg typeRegistry, typeName string) error {
	buf.WriteByte('{')

	keys := objectKeyOrder(obj, reg, typeName)
	first := true
	for _, key := range keys {
		val, ok := obj[key]
		if !ok {
			continue
		}
		if !first {
			buf.WriteByte(',')
		}
		first = false

		keyJSON, _ := json.Marshal(key)
		buf.Write(keyJSON)
		buf.WriteByte(':')

		// Determine child type context
		childType := childTypeName(key, typeName, reg)
		if err := writeValue(buf, val, reg, childType); err != nil {
			return err
		}
	}

	buf.WriteByte('}')
	return nil
}

func writeArray(buf *bytes.Buffer, arr []any, reg typeRegistry) error {
	buf.WriteByte('[')
	for i, item := range arr {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := writeValue(buf, item, reg, ""); err != nil {
			return err
		}
	}
	buf.WriteByte(']')
	return nil
}

// childTypeName determines the Go type context for a child value.
// parentKey is the key being written, parentType is the current type context.
func childTypeName(parentKey, parentType string, reg typeRegistry) string {
	// "properties" maps inherit the parent type for field ordering
	if parentKey == "properties" {
		return parentType
	}
	// Keys inside $defs: the key itself is the type name
	if _, ok := reg[parentKey]; ok {
		return parentKey
	}
	return ""
}

// objectKeyOrder returns keys in the desired order for a JSON object.
func objectKeyOrder(obj map[string]any, reg typeRegistry, typeName string) []string {
	// If we have a known type, use struct field order for properties
	if typeName != "" {
		if order := reg.fieldOrder(typeName); order != nil {
			return appendMissing(order, obj)
		}
	}

	// For $defs: sort by key name for stability
	// For everything else: use preferred schema key order
	return preferredKeyOrder(obj)
}

// preferredKeyOrder returns keys in an order matching typical pydantic/OpenAI output.
func preferredKeyOrder(obj map[string]any) []string {
	preferred := []string{
		"$defs",
		"additionalProperties",
		"properties",
		"required",
		"title",
		"description",
		"type",
		"const",
		"default",
		"anyOf", "oneOf", "allOf",
		"$ref",
		"items",
		"enum",
	}
	return appendMissing(preferred, obj)
}

// appendMissing returns keys from `order` that exist in `obj`, followed by
// remaining keys from `obj` not in `order` (sorted for stability).
func appendMissing(order []string, obj map[string]any) []string {
	seen := make(map[string]struct{}, len(order))
	for _, k := range order {
		seen[k] = struct{}{}
	}
	result := make([]string, 0, len(obj))
	for _, k := range order {
		if _, ok := obj[k]; ok {
			result = append(result, k)
		}
	}
	for _, k := range sortedKeys(obj) {
		if _, ok := seen[k]; !ok {
			result = append(result, k)
		}
	}
	return result
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
