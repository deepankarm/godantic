package schema

import (
	"reflect"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/invopop/jsonschema"
)

// reflectVariantsFromType reflects all union variant types from a struct type and adds them to schema definitions
func reflectVariantsFromType(reflector *jsonschema.Reflector, schema *jsonschema.Schema, t reflect.Type) {
	fieldOptions := godantic.ScanTypeFieldOptions(t)
	for _, opts := range fieldOptions {
		reflectUnionConstraints(reflector, schema, opts.Constraints)
	}
}

// reflectUnionConstraints reflects union variant types from field constraints
func reflectUnionConstraints(reflector *jsonschema.Reflector, schema *jsonschema.Schema, constraints map[string]any) {
	// Handle DiscriminatedUnion variants
	if discriminator, ok := constraints[godantic.ConstraintDiscriminator].(map[string]any); ok {
		if mapping, ok := discriminator["mapping"].(map[string]any); ok {
			for _, variant := range mapping {
				reflectVariant(reflector, schema, variant)
			}
		}
	}

	// Handle UnionOf complex types
	if anyOfTypes, ok := constraints["anyOfTypes"]; ok {
		if types, ok := anyOfTypes.([]any); ok {
			for _, typeInstance := range types {
				reflectUnionOf(reflector, schema, typeInstance)
			}
		}
	}
}

// reflectUnionOf reflects types from UnionOf constraints and adds them to schema definitions
func reflectUnionOf(reflector *jsonschema.Reflector, schema *jsonschema.Schema, typeInstance any) {
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
			reflectVariant(reflector, schema, elemInstance)
		}
	} else if t.Kind() == reflect.Struct {
		// Directly reflect struct types
		reflectVariant(reflector, schema, typeInstance)
	}
	// Primitives and maps don't need definition reflection
}

// reflectVariant reflects a single variant type and adds it to the schema definitions
func reflectVariant(reflector *jsonschema.Reflector, schema *jsonschema.Schema, variant any) {
	variantType := reflect.TypeOf(variant)
	if variantType == nil {
		return
	}

	// Reflect the variant type to ensure it's in the schema
	variantSchema := reflector.Reflect(variant)

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
