package schema

import (
	"reflect"
	"slices"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/internal/reflectutil"
	"github.com/invopop/jsonschema"
)

// enhance adds godantic metadata (constraints, titles, etc.) to the schema
func (g *Generator[T]) enhance(schema *jsonschema.Schema) {
	var zero T
	rootType := reflect.TypeOf(zero)
	enhanceSchema(schema, g.reflector, rootType, g.options)
}

// enhanceSchema is the unified enhancement function used by both Generator[T] and GenerateForType
// It handles all schema enhancement including union variants and field options
func enhanceSchema(schema *jsonschema.Schema, reflector *jsonschema.Reflector, rootType reflect.Type, opts SchemaOptions) {
	if rootType.Kind() == reflect.Pointer {
		rootType = rootType.Elem()
	}

	if rootType.Kind() != reflect.Struct {
		return
	}

	// Collect all struct types from the root type
	structTypes := make(map[string]reflect.Type)
	reflectutil.CollectStructTypes(rootType, structTypes)

	// Iteratively collect and reflect union variant types
	collectAndReflectUnionVariants(schema, reflector, structTypes)

	// Enhance each definition with field options
	if schema.Definitions != nil {
		for defName, defSchema := range schema.Definitions {
			if structType, ok := structTypes[defName]; ok {
				enhanceDefinition(defSchema, structType, opts.AutoGenerateTitles)
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
			reflectutil.CollectStructTypes(variantType, structTypes)
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
// This happens when jsonschema encounters an interface or any type
func isEmptyInterfaceSchema(s *jsonschema.Schema) bool {
	// A schema is considered "empty" (interface/any) if it has no type, no properties, no ref, etc.
	// Note: OneOf/AnyOf/AllOf may be empty slices rather than nil
	return s.Type == "" &&
		s.Ref == "" &&
		s.Properties == nil &&
		s.Items == nil &&
		len(s.OneOf) == 0 &&
		len(s.AnyOf) == 0 &&
		len(s.AllOf) == 0
}

// enhanceDefinition enhances a schema definition with field options from a type.
// Single pass over properties - applies constraints, required, and titles.
func enhanceDefinition(defSchema *jsonschema.Schema, t reflect.Type, autoGenerateTitles bool) {
	if defSchema.Properties == nil {
		return
	}

	// Collect field options from type and embedded structs
	fieldOptions := godantic.ScanTypeFieldOptions(t)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			for name, opts := range godantic.ScanTypeFieldOptions(field.Type) {
				if _, exists := fieldOptions[name]; !exists {
					fieldOptions[name] = opts
				}
			}
		}
	}

	// Track which properties have field options
	enhanced := make(map[string]bool)

	// Apply field options to properties with Field{Name}() methods
	for fieldName, opts := range fieldOptions {
		jsonName := reflectutil.GoFieldToJSONName(t, fieldName)
		prop, _ := defSchema.Properties.Get(jsonName)
		if prop == nil {
			prop, _ = defSchema.Properties.Get(fieldName)
			if prop == nil {
				continue
			}
			jsonName = fieldName
		}

		// Replace empty interface schemas
		if isEmptyInterfaceSchema(prop) {
			prop = &jsonschema.Schema{}
			defSchema.Properties.Set(jsonName, prop)
		}

		// Mark as required
		if opts.Required && !slices.Contains(defSchema.Required, jsonName) {
			defSchema.Required = append(defSchema.Required, jsonName)
		}

		// Apply constraints
		applyConstraints(prop, opts.Constraints)

		// Add title
		if prop.Title == "" {
			prop.Title = toTitleCase(fieldName)
		}

		enhanced[jsonName] = true
	}

	// Handle remaining properties without field options (auto-titles)
	if autoGenerateTitles {
		for pair := defSchema.Properties.Oldest(); pair != nil; pair = pair.Next() {
			if enhanced[pair.Key] {
				continue
			}
			prop := pair.Value
			if isEmptyInterfaceSchema(prop) {
				defSchema.Properties.Set(pair.Key, &jsonschema.Schema{Title: toTitleCase(pair.Key)})
			} else if prop.Title == "" {
				prop.Title = toTitleCase(pair.Key)
			}
		}
	}
}

// toTitleCase converts a field name to a human-readable title
// e.g., "userName" -> "User Name", "ma_user_query" -> "Ma User Query", "BranchID" -> "Branch ID"
func toTitleCase(fieldName string) string {
	if fieldName == "" {
		return ""
	}

	var result []rune
	capitalizeNext := true // Capitalize the first letter

	for i, r := range fieldName {
		if r == '_' {
			// Replace underscore with space and capitalize next letter
			result = append(result, ' ')
			capitalizeNext = true
		} else if i > 0 && r >= 'A' && r <= 'Z' && fieldName[i-1] >= 'a' && fieldName[i-1] <= 'z' {
			// Add space before capital letters in camelCase (except at start)
			result = append(result, ' ')
			result = append(result, r)
			capitalizeNext = false
		} else if capitalizeNext && r >= 'a' && r <= 'z' {
			// Capitalize this lowercase letter
			result = append(result, r-'a'+'A')
			capitalizeNext = false
		} else {
			// Append the character as-is (uppercase letters, numbers, etc.)
			result = append(result, r)
			// If it's an uppercase letter or space, mark that we've handled the capital
			if r >= 'A' && r <= 'Z' || r == ' ' {
				capitalizeNext = r == ' ' // Only capitalize after spaces, not after uppercase
			} else {
				capitalizeNext = false
			}
		}
	}

	return string(result)
}
