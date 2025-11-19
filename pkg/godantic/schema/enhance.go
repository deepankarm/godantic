package schema

import (
	"reflect"
	"slices"
	"strings"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/invopop/jsonschema"
)

// enhanceSchemaWithValidation adds our validation metadata to the schema
func (g *Generator[T]) enhanceSchemaWithValidation(schema *jsonschema.Schema) {
	fieldOptions := g.validator.FieldOptions()
	actualSchema := findActualSchema(schema)

	if actualSchema == nil || actualSchema.Properties == nil {
		return
	}

	// Reflect discriminated union variant types
	g.reflectUnionVariants(schema, fieldOptions)

	// Enhance each property with validation metadata
	g.enhanceProperties(actualSchema, fieldOptions)

	// Enhance all nested definitions
	g.enhanceAllDefinitions(schema)
}

// enhanceAllDefinitions enhances all definitions in the schema with validation metadata
func (g *Generator[T]) enhanceAllDefinitions(schema *jsonschema.Schema) {
	if schema.Definitions == nil {
		return
	}

	// Get the root type to walk through
	var zero T
	rootType := reflect.TypeOf(zero)

	// Collect all struct types from the root type
	structTypes := make(map[string]reflect.Type)
	godantic.CollectStructTypes(rootType, structTypes)

	// Reflect union variants from all nested types (not just root)
	for _, structType := range structTypes {
		reflectUnionVariantsFromType(g, schema, structType)
	}

	// Enhance each definition with its field options
	for defName, defSchema := range schema.Definitions {
		if structType, ok := structTypes[defName]; ok {
			enhanceDefinitionWithType(defSchema, structType)
		} else {
			// Definition exists but we don't have its type - skip silently
			// This can happen for union variant types added dynamically
		}
	}
}

// enhanceDefinitionWithType enhances a schema definition with field options from a type
func enhanceDefinitionWithType(defSchema *jsonschema.Schema, t reflect.Type) {
	if defSchema.Properties == nil {
		return
	}

	// Use shared reflection utility to scan Field{Name}() methods
	fieldOptions := godantic.ScanTypeFieldOptions(t)

	// Apply field options to schema properties
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Name

		// Get field options (if any)
		opts, hasOpts := fieldOptions[fieldName]
		if !hasOpts {
			continue
		}

		// Get JSON name from tag or use lowercase first letter
		jsonTag := field.Tag.Get("json")
		jsonName := fieldName
		if jsonTag != "" {
			// Parse JSON tag (handle cases like "field_name,omitempty")
			if idx := strings.Index(jsonTag, ","); idx != -1 {
				jsonName = jsonTag[:idx]
			} else {
				jsonName = jsonTag
			}
		} else {
			jsonName = toLowerFirst(fieldName)
		}

		// Get the property from schema
		prop, ok := defSchema.Properties.Get(jsonName)
		if !ok || prop == nil {
			// Try with original field name
			prop, ok = defSchema.Properties.Get(fieldName)
			if !ok || prop == nil {
				continue
			}
			jsonName = fieldName
		}

		// Add required fields
		if opts.Required && !slices.Contains(defSchema.Required, jsonName) {
			defSchema.Required = append(defSchema.Required, jsonName)
		}

		// Apply all constraints to property
		applyConstraints(prop, opts.Constraints)
	}
}

// enhanceProperties enhances each property with validation metadata
func (g *Generator[T]) enhanceProperties(actualSchema *jsonschema.Schema, fieldOptions map[string]any) {
	for fieldName, optsAny := range fieldOptions {
		opts := optsAny.(fieldOption)

		// Try both the original field name and lowercase first letter
		jsonName := toLowerFirst(fieldName)

		prop, ok := actualSchema.Properties.Get(jsonName)
		if !ok || prop == nil {
			// Try with original field name (jsonschema lib may use original case)
			prop, ok = actualSchema.Properties.Get(fieldName)
			if !ok || prop == nil {
				continue
			}
			jsonName = fieldName // Use the one that worked
		}

		// Add required fields
		if opts.Required() && !slices.Contains(actualSchema.Required, jsonName) {
			actualSchema.Required = append(actualSchema.Required, jsonName)
		}

		// Apply all constraints to property
		applyConstraints(prop, opts.Constraints())
	}
}
