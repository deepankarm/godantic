package schema

import (
	"reflect"
	"slices"
	"strings"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/invopop/jsonschema"
)

// enhanceSchemaWithValidation adds our validation metadata to the schema
// This is the entry point for Generator[T] path
func (g *Generator[T]) enhanceSchemaWithValidation(schema *jsonschema.Schema) {
	var zero T
	rootType := reflect.TypeOf(zero)
	enhanceSchema(schema, g.reflector, rootType)
}

// enhanceSchema is the unified enhancement function used by both Generator[T] and GenerateForType
// It handles all schema enhancement including union variants and field options
func enhanceSchema(schema *jsonschema.Schema, reflector *jsonschema.Reflector, rootType reflect.Type) {
	if rootType.Kind() == reflect.Pointer {
		rootType = rootType.Elem()
	}

	if rootType.Kind() != reflect.Struct {
		return
	}

	// Collect all struct types from the root type
	structTypes := make(map[string]reflect.Type)
	godantic.CollectStructTypes(rootType, structTypes)

	// Iteratively collect and reflect union variant types
	collectAndReflectUnionVariants(schema, reflector, structTypes)

	// Enhance each definition with field options
	if schema.Definitions != nil {
		for defName, defSchema := range schema.Definitions {
			if structType, ok := structTypes[defName]; ok {
				enhanceDefinition(defSchema, structType)
			}
		}
	}
}

// collectAndReflectUnionVariants iteratively collects and reflects all discriminated union variant types
// This is needed because variant types may themselves contain nested discriminated unions
func collectAndReflectUnionVariants(schema *jsonschema.Schema, reflector *jsonschema.Reflector, structTypes map[string]reflect.Type) {
	processedTypes := make(map[string]bool)
	for {
		newTypesFound := false

		// Reflect union variants from all collected types
		for typeName, structType := range structTypes {
			if processedTypes[typeName] {
				continue
			}
			reflectVariantsFromType(reflector, schema, structType)
			processedTypes[typeName] = true
			newTypesFound = true
		}

		// Discover newly added variant types from schema definitions
		if discoverNewVariantTypes(schema, structTypes) {
			newTypesFound = true
		}

		// Continue until no new types are found
		if !newTypesFound {
			break
		}
	}
}

// discoverNewVariantTypes searches for variant types in discriminated unions and adds them to structTypes
// Returns true if any new types were found
func discoverNewVariantTypes(schema *jsonschema.Schema, structTypes map[string]reflect.Type) bool {
	if schema.Definitions == nil {
		return false
	}

	newFound := false
	for defName := range schema.Definitions {
		if _, exists := structTypes[defName]; exists {
			continue // Already have this type
		}

		// Search for this type in discriminated union mappings
		if variantType := findVariantTypeByName(defName, structTypes); variantType != nil {
			structTypes[defName] = variantType
			godantic.CollectStructTypes(variantType, structTypes)
			newFound = true
		}
	}
	return newFound
}

// findVariantTypeByName searches for a variant type by name in all discriminated union constraints
func findVariantTypeByName(defName string, structTypes map[string]reflect.Type) reflect.Type {
	for _, structType := range structTypes {
		fieldOptions := godantic.ScanTypeFieldOptions(structType)
		for _, opts := range fieldOptions {
			if discriminator, ok := opts.Constraints[godantic.ConstraintDiscriminator].(map[string]any); ok {
				if mapping, ok := discriminator["mapping"].(map[string]any); ok {
					for _, variant := range mapping {
						variantType := reflect.TypeOf(variant)
						if variantType != nil {
							if variantType.Kind() == reflect.Pointer {
								variantType = variantType.Elem()
							}
							if variantType.Name() == defName {
								return variantType
							}
						}
					}
				}
			}
		}
	}
	return nil
}

// isEmptyInterfaceSchema checks if a schema is an "empty" schema that would serialize to `true`
// This happens when jsonschema encounters an interface type
func isEmptyInterfaceSchema(s *jsonschema.Schema) bool {
	// A schema is considered "empty" (interface) if it has no type, no properties, no ref, etc.
	return s.Type == "" &&
		s.Ref == "" &&
		s.Properties == nil &&
		s.Items == nil &&
		s.OneOf == nil &&
		s.AnyOf == nil &&
		s.AllOf == nil
}

// enhanceDefinition enhances a schema definition with field options from a type
func enhanceDefinition(defSchema *jsonschema.Schema, t reflect.Type) {
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

		// Handle interface types: when jsonschema encounters an interface,
		// it creates an empty schema that serializes to `true`
		// We need to ensure it's a proper schema object before applying constraints
		if prop != nil && isEmptyInterfaceSchema(prop) {
			// Create a new schema object to replace the empty one
			newProp := &jsonschema.Schema{}
			defSchema.Properties.Set(jsonName, newProp)
			prop = newProp
		}

		// Add required fields
		if opts.Required && !slices.Contains(defSchema.Required, jsonName) {
			defSchema.Required = append(defSchema.Required, jsonName)
		}

		// Apply all constraints to property
		applyConstraints(prop, opts.Constraints)
	}
}
