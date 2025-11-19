package schema

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/invopop/jsonschema"
)

// Generator generates JSON Schema from validated structs
type Generator[T any] struct {
	validator *godantic.Validator[T]
	reflector *jsonschema.Reflector
}

// NewGenerator creates a new schema generator
func NewGenerator[T any]() *Generator[T] {
	return &Generator[T]{
		validator: godantic.NewValidator[T](),
		reflector: &jsonschema.Reflector{
			AllowAdditionalProperties:  false,
			RequiredFromJSONSchemaTags: true,
		},
	}
}

// Generate generates JSON Schema for the type
func (g *Generator[T]) Generate() (*jsonschema.Schema, error) {
	var zero T
	schema := g.reflector.Reflect(zero)
	g.enhanceSchemaWithValidation(schema)
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

	// Convert schema to map for manipulation
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	// If there's no $ref at root, return as-is
	ref, hasRef := schemaMap["$ref"].(string)
	if !hasRef {
		return schemaMap, nil
	}

	// Get the $defs
	defs, ok := schemaMap["$defs"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("$defs not found in schema")
	}

	// Extract the root type name from $ref (e.g., "#/$defs/TypeName" -> "TypeName")
	if !strings.HasPrefix(ref, "#/$defs/") {
		return nil, fmt.Errorf("unexpected $ref format: %s", ref)
	}
	rootTypeName := ref[len("#/$defs/"):]

	// Get the root definition
	rootDef, ok := defs[rootTypeName].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("root definition %s not found in $defs", rootTypeName)
	}

	// Create flattened schema with root definition at top level
	result := make(map[string]any)
	maps.Copy(result, rootDef)

	// Add $defs for nested types (excluding root type to avoid duplication)
	if len(defs) > 1 {
		result["$defs"] = defs
	}

	return result, nil
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

// fieldOption is a local interface for accessing field option properties
type fieldOption interface {
	Required() bool
	Constraints() map[string]any
}
