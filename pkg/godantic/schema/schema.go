package schema

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/invopop/jsonschema"
)

// SchemaOptions configures schema generation behavior
type SchemaOptions struct {
	AutoGenerateTitles bool // Generate titles for all fields (Pydantic-style, default: true)
}

// DefaultSchemaOptions returns default options matching Pydantic behavior
func DefaultSchemaOptions() SchemaOptions {
	return SchemaOptions{
		AutoGenerateTitles: true,
	}
}

// Generator generates JSON Schema from validated structs
type Generator[T any] struct {
	validator *godantic.Validator[T]
	reflector *jsonschema.Reflector
	options   SchemaOptions
}

// NewGenerator creates a new schema generator with default options
func NewGenerator[T any]() *Generator[T] {
	return &Generator[T]{
		validator: godantic.NewValidator[T](),
		reflector: &jsonschema.Reflector{
			AllowAdditionalProperties:  false,
			RequiredFromJSONSchemaTags: true,
		},
		options: DefaultSchemaOptions(),
	}
}

// WithOptions configures the schema generator with custom options
func (g *Generator[T]) WithOptions(opts SchemaOptions) *Generator[T] {
	g.options = opts
	return g
}

// WithAutoTitles is a convenience method to configure auto-title generation
func (g *Generator[T]) WithAutoTitles(enabled bool) *Generator[T] {
	g.options.AutoGenerateTitles = enabled
	return g
}

// Generate generates JSON Schema for the type
func (g *Generator[T]) Generate() (*jsonschema.Schema, error) {
	var zero T
	schema := g.reflector.Reflect(zero)
	g.enhance(schema)
	return schema, nil
}

// GenerateFlattened generates a flattened JSON Schema suitable for LLM APIs
// (OpenAI, Gemini, Claude, etc.) that require the root object definition
// at the top level instead of a $ref
func (g *Generator[T]) GenerateFlattened() (map[string]any, error) {
	schema, err := g.Generate()
	if err != nil {
		return nil, err
	}

	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	return flattenSchemaMap(schemaMap)
}

// GenerateJSON generates JSON Schema as JSON string
func (g *Generator[T]) GenerateJSON() (string, error) {
	schema, err := g.Generate()
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Options allows customizing schema generation
type Options struct {
	Title       string
	Description string
	Version     string
}

// GenerateWithOptions generates schema with custom options
func GenerateWithOptions[T any](opts Options) (*jsonschema.Schema, error) {
	g := NewGenerator[T]()
	schema, err := g.Generate()
	if err != nil {
		return nil, err
	}

	if opts.Title != "" {
		schema.Title = opts.Title
	}
	if opts.Description != "" {
		schema.Description = opts.Description
	}
	if opts.Version != "" {
		schema.Version = opts.Version
	}

	return schema, nil
}

// GenerateForType generates a JSON schema for any reflect.Type with default options
func GenerateForType(t reflect.Type) (map[string]any, error) {
	return GenerateForTypeWithOptions(t, DefaultSchemaOptions())
}

// GenerateForTypeWithOptions generates a JSON schema for any reflect.Type with custom options
func GenerateForTypeWithOptions(t reflect.Type, opts SchemaOptions) (map[string]any, error) {
	var instance any
	if t.Kind() == reflect.Pointer {
		instance = reflect.New(t.Elem()).Interface()
	} else {
		instance = reflect.New(t).Interface()
	}

	// Create reflector with godantic settings
	reflector := &jsonschema.Reflector{
		AllowAdditionalProperties:  false,
		RequiredFromJSONSchemaTags: true,
	}

	schema := reflector.Reflect(instance)
	enhanceSchema(schema, reflector, t, opts)

	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
		return nil, err
	}

	return schemaMap, nil
}

// GenerateUnionSchema generates a JSON schema with anyOf from multiple types.
// This is the Go equivalent of Python's `TypeA | TypeB | TypeC`.
//
// Each type's schema is generated and flattened (no $ref at root).
// All $defs are merged into a single definitions block.
//
// Usage:
//
//	schema, err := GenerateUnionSchema(TypeA{}, TypeB{}, TypeC{})
//	// Returns: {"anyOf": [...], "$defs": {...}}
func GenerateUnionSchema(types ...any) (map[string]any, error) {
	if len(types) == 0 {
		return nil, nil
	}

	// Single type - just return its flattened schema
	if len(types) == 1 {
		return generateFlattenedForValue(types[0])
	}

	anyOf := make([]map[string]any, 0, len(types))
	mergedDefs := make(map[string]any)

	for _, t := range types {
		schema, err := generateFlattenedForValue(t)
		if err != nil {
			return nil, err
		}

		// Extract and merge $defs
		if defs, ok := schema["$defs"].(map[string]any); ok {
			maps.Copy(mergedDefs, defs)
			delete(schema, "$defs")
		}

		anyOf = append(anyOf, schema)
	}

	result := map[string]any{"anyOf": anyOf}
	if len(mergedDefs) > 0 {
		result["$defs"] = mergedDefs
	}

	return result, nil
}

// generateFlattenedForValue generates a flattened schema for a value instance
func generateFlattenedForValue(v any) (map[string]any, error) {
	t := reflect.TypeOf(v)
	if t == nil {
		return nil, fmt.Errorf("nil type provided")
	}

	schema, err := GenerateForType(t)
	if err != nil {
		return nil, err
	}

	return flattenSchemaMap(schema)
}

// flattenSchemaMap inlines the root $ref definition at the top level.
// If schema has {"$ref": "#/$defs/TypeName", "$defs": {...}}, it becomes
// the TypeName definition with $defs for any nested types.
func flattenSchemaMap(schema map[string]any) (map[string]any, error) {
	ref, hasRef := schema["$ref"].(string)
	if !hasRef {
		return schema, nil
	}

	defs, ok := schema["$defs"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("$defs not found in schema with $ref")
	}

	if !strings.HasPrefix(ref, "#/$defs/") {
		return nil, fmt.Errorf("unexpected $ref format: %s", ref)
	}
	typeName := ref[len("#/$defs/"):]

	rootDef, ok := defs[typeName].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("definition %s not found in $defs", typeName)
	}

	result := make(map[string]any)
	maps.Copy(result, rootDef)

	// Keep $defs for nested types (remove self-reference to avoid duplication)
	delete(defs, typeName)
	if len(defs) > 0 {
		result["$defs"] = defs
	}

	return result, nil
}
