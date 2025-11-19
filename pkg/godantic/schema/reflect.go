package schema

import (
	"reflect"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/invopop/jsonschema"
)

// reflectUnionVariants reflects all discriminated union variant types and adds them to schema definitions
func (g *Generator[T]) reflectUnionVariants(schema *jsonschema.Schema, fieldOptions map[string]any) {
	for _, optsAny := range fieldOptions {
		opts := optsAny.(fieldOption)

		// Handle DiscriminatedUnion variants
		if discriminator, ok := opts.Constraints()[godantic.ConstraintDiscriminator].(map[string]any); ok {
			if mapping, ok := discriminator["mapping"].(map[string]any); ok {
				// Reflect each variant type and add to schema definitions
				for _, variant := range mapping {
					g.reflectVariantType(schema, variant)
				}
			}
		}

		// Handle UnionOf complex types
		if anyOfTypes, ok := opts.Constraints()["anyOfTypes"]; ok {
			if types, ok := anyOfTypes.([]any); ok {
				for _, typeInstance := range types {
					g.reflectUnionOfType(schema, typeInstance)
				}
			}
		}
	}
}

// reflectUnionOfType reflects types from UnionOf and adds them to schema definitions
func (g *Generator[T]) reflectUnionOfType(schema *jsonschema.Schema, typeInstance any) {
	t := reflect.TypeOf(typeInstance)
	if t == nil {
		return
	}

	// Handle slices/arrays - reflect the element type
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		elemType := t.Elem()
		if elemType.Kind() == reflect.Struct {
			// Reflect the struct type
			elemInstance := reflect.New(elemType).Elem().Interface()
			g.reflectVariantType(schema, elemInstance)
		}
	} else if t.Kind() == reflect.Struct {
		// Directly reflect struct types
		g.reflectVariantType(schema, typeInstance)
	}
	// Primitives and maps don't need definition reflection
}

// reflectVariantType reflects a single variant type and adds it to the schema
func (g *Generator[T]) reflectVariantType(schema *jsonschema.Schema, variant any) {
	variantType := reflect.TypeOf(variant)
	if variantType == nil {
		return
	}

	// Reflect the variant type to ensure it's in the schema
	variantSchema := g.reflector.Reflect(variant)

	// Add to definitions if not already present
	if schema.Definitions == nil {
		schema.Definitions = make(jsonschema.Definitions)
	}

	// Find the actual variant definition (it might be nested)
	variantDefName := variantType.Name()
	if variantSchema.Definitions != nil {
		// If the reflected schema has definitions, merge them
		for defName, defSchema := range variantSchema.Definitions {
			if _, exists := schema.Definitions[defName]; !exists {
				schema.Definitions[defName] = defSchema
			}
		}
	}

	// Add the main variant schema if not already there
	if _, exists := schema.Definitions[variantDefName]; !exists {
		if variantSchema.Ref != "" {
			// If it's a reference, we need to get the actual schema
			actualVariantSchema := findActualSchema(variantSchema)
			if actualVariantSchema != nil {
				schema.Definitions[variantDefName] = actualVariantSchema
			}
		} else {
			schema.Definitions[variantDefName] = variantSchema
		}
	}
}

// reflectUnionVariantsFromType reflects union variants from a specific type's fields
func reflectUnionVariantsFromType[T any](g *Generator[T], schema *jsonschema.Schema, t reflect.Type) {
	// Create a zero value instance of the type
	zeroValue := reflect.New(t).Interface()
	v := reflect.ValueOf(zeroValue)

	// Process each field's Field* method
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		methodName := "Field" + field.Name
		method := v.MethodByName(methodName)
		if !method.IsValid() {
			continue
		}

		// Call the method to get field options
		results := method.Call(nil)
		if len(results) != 1 {
			continue
		}

		// Access the Constraints_ field
		optsValue := results[0]
		constraintsField := optsValue.FieldByName("Constraints_")
		if !constraintsField.IsValid() {
			continue
		}

		constraints, ok := constraintsField.Interface().(map[string]any)
		if !ok {
			continue
		}

		// Handle Union complex types (anyOfTypes)
		if anyOfTypes, ok := constraints["anyOfTypes"]; ok {
			if types, ok := anyOfTypes.([]any); ok {
				for _, typeInstance := range types {
					g.reflectUnionOfType(schema, typeInstance)
				}
			}
		}

		// Handle DiscriminatedUnion variants
		if discriminator, ok := constraints[godantic.ConstraintDiscriminator].(map[string]any); ok {
			if mapping, ok := discriminator["mapping"].(map[string]any); ok {
				for _, variant := range mapping {
					g.reflectVariantType(schema, variant)
				}
			}
		}
	}
}

// collectStructTypes recursively collects all struct types from a type
func collectStructTypes(t reflect.Type, types map[string]reflect.Type) {
	if t == nil {
		return
	}

	// Handle pointers
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Only process structs
	if t.Kind() != reflect.Struct {
		return
	}

	// Add this type to the map
	if t.Name() != "" {
		types[t.Name()] = t
	}

	// Recursively process all struct fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldType := field.Type

		// Handle slices/arrays
		if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
			fieldType = fieldType.Elem()
		}

		// Handle pointers
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}

		// Recursively collect struct types
		if fieldType.Kind() == reflect.Struct {
			collectStructTypes(fieldType, types)
		}
	}
}
