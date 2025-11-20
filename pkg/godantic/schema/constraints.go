package schema

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/invopop/jsonschema"
)

// applyConstraints applies all constraints to a property
func applyConstraints(prop *jsonschema.Schema, constraints map[string]any) {
	if constraints == nil {
		return
	}

	applyMetadataConstraints(prop, constraints)
	applyNumericConstraints(prop, constraints)
	applyStringConstraints(prop, constraints)
	applyArrayConstraints(prop, constraints)
	applyObjectConstraints(prop, constraints)
	applyValueConstraints(prop, constraints)
	applyUnionConstraints(prop, constraints)
}

// applyMetadataConstraints applies metadata constraints (description, title, etc.)
func applyMetadataConstraints(prop *jsonschema.Schema, constraints map[string]any) {
	if desc, ok := constraints[godantic.ConstraintDescription].(string); ok {
		prop.Description = desc
	}
	if title, ok := constraints[godantic.ConstraintTitle].(string); ok {
		prop.Title = title
	}
	if example, ok := constraints[godantic.ConstraintExample]; ok {
		prop.Examples = []any{example}
	}
	if format, ok := constraints[godantic.ConstraintFormat].(string); ok {
		prop.Format = format
	}
	if readOnly, ok := constraints[godantic.ConstraintReadOnly].(bool); ok && readOnly {
		prop.ReadOnly = true
	}
	if writeOnly, ok := constraints[godantic.ConstraintWriteOnly].(bool); ok && writeOnly {
		prop.WriteOnly = true
	}
	if deprecated, ok := constraints[godantic.ConstraintDeprecated].(bool); ok && deprecated {
		prop.Deprecated = true
	}
}

// applyNumericConstraints applies numeric constraints (min, max, multipleOf, etc.)
func applyNumericConstraints(prop *jsonschema.Schema, constraints map[string]any) {
	if min, ok := constraints[godantic.ConstraintMinimum]; ok {
		prop.Minimum = toJSONNumber(min)
	}
	if max, ok := constraints[godantic.ConstraintMaximum]; ok {
		prop.Maximum = toJSONNumber(max)
	}
	if exclusiveMin, ok := constraints[godantic.ConstraintExclusiveMinimum]; ok {
		prop.ExclusiveMinimum = toJSONNumber(exclusiveMin)
	}
	if exclusiveMax, ok := constraints[godantic.ConstraintExclusiveMaximum]; ok {
		prop.ExclusiveMaximum = toJSONNumber(exclusiveMax)
	}
	if multipleOf, ok := constraints[godantic.ConstraintMultipleOf]; ok {
		prop.MultipleOf = toJSONNumber(multipleOf)
	}
}

// applyStringConstraints applies string constraints (minLength, maxLength, pattern, etc.)
func applyStringConstraints(prop *jsonschema.Schema, constraints map[string]any) {
	if minLen, ok := constraints[godantic.ConstraintMinLength].(int); ok {
		val := uint64(minLen)
		prop.MinLength = &val
	}
	if maxLen, ok := constraints[godantic.ConstraintMaxLength].(int); ok {
		val := uint64(maxLen)
		prop.MaxLength = &val
	}
	if pattern, ok := constraints[godantic.ConstraintPattern].(string); ok {
		prop.Pattern = pattern
	}
	if encoding, ok := constraints[godantic.ConstraintContentEncoding].(string); ok {
		prop.ContentEncoding = encoding
	}
	if mediaType, ok := constraints[godantic.ConstraintContentMediaType].(string); ok {
		prop.ContentMediaType = mediaType
	}
}

// applyArrayConstraints applies array constraints (minItems, maxItems, uniqueItems)
func applyArrayConstraints(prop *jsonschema.Schema, constraints map[string]any) {
	if minItems, ok := constraints[godantic.ConstraintMinItems].(int); ok {
		val := uint64(minItems)
		prop.MinItems = &val
	}
	if maxItems, ok := constraints[godantic.ConstraintMaxItems].(int); ok {
		val := uint64(maxItems)
		prop.MaxItems = &val
	}
	if uniqueItems, ok := constraints[godantic.ConstraintUniqueItems].(bool); ok && uniqueItems {
		prop.UniqueItems = true
	}
}

// applyObjectConstraints applies object/map constraints (minProperties, maxProperties)
func applyObjectConstraints(prop *jsonschema.Schema, constraints map[string]any) {
	if minProps, ok := constraints[godantic.ConstraintMinProperties].(int); ok {
		val := uint64(minProps)
		prop.MinProperties = &val
	}
	if maxProps, ok := constraints[godantic.ConstraintMaxProperties].(int); ok {
		val := uint64(maxProps)
		prop.MaxProperties = &val
	}
}

// applyValueConstraints applies value constraints (enum, const, default)
func applyValueConstraints(prop *jsonschema.Schema, constraints map[string]any) {
	if enum, ok := constraints[godantic.ConstraintEnum]; ok {
		// Convert enum to []any (it may be []T from OneOf[T])
		if enumSlice, ok := enum.([]any); ok {
			prop.Enum = enumSlice
		} else {
			// Handle typed slices by converting to []any using reflection
			v := reflect.ValueOf(enum)
			if v.Kind() == reflect.Slice {
				enumAny := make([]any, v.Len())
				for i := 0; i < v.Len(); i++ {
					enumAny[i] = v.Index(i).Interface()
				}
				prop.Enum = enumAny
			}
		}
	}
	if constVal, ok := constraints[godantic.ConstraintConst]; ok {
		prop.Const = constVal
	}
	if defaultVal, ok := constraints[godantic.ConstraintDefault]; ok {
		prop.Default = defaultVal
	}
}

// applyUnionConstraints applies union constraints (anyOf, oneOf with discriminator)
func applyUnionConstraints(prop *jsonschema.Schema, constraints map[string]any) {
	// Collect all anyOf schemas (both primitive and complex types)
	var allSchemas []*jsonschema.Schema

	// Handle primitive type names (from string arguments)
	if anyOf, ok := constraints[godantic.ConstraintAnyOf]; ok {
		if anyOfSlice, ok := anyOf.([]map[string]string); ok {
			for _, typeMap := range anyOfSlice {
				allSchemas = append(allSchemas, &jsonschema.Schema{
					Type: typeMap["type"],
				})
			}
		}
	}

	// Handle complex Go types (from non-string arguments)
	if anyOfTypes, ok := constraints["anyOfTypes"]; ok {
		if types, ok := anyOfTypes.([]any); ok {
			for _, typeInstance := range types {
				schema := createSchemaForType(reflect.TypeOf(typeInstance))
				if schema != nil {
					allSchemas = append(allSchemas, schema)
				}
			}
		}
	}

	// Set the combined anyOf if we have any schemas
	if len(allSchemas) > 0 {
		prop.AnyOf = allSchemas
	}

	// Handle DiscriminatedUnion (oneOf with discriminator)
	if discriminator, ok := constraints[godantic.ConstraintDiscriminator].(map[string]any); ok {
		propertyName, _ := discriminator["propertyName"].(string)
		mapping, _ := discriminator["mapping"].(map[string]any)

		if propertyName != "" && mapping != nil {
			// Create oneOf schemas for each variant
			schemas := make([]*jsonschema.Schema, 0, len(mapping))
			for _, variant := range mapping {
				// Use reflection to get the type of the variant
				variantType := reflect.TypeOf(variant)
				if variantType != nil {
					// Create a temporary schema for this variant type
					variantSchema := &jsonschema.Schema{
						Ref: fmt.Sprintf("#/$defs/%s", variantType.Name()),
					}
					schemas = append(schemas, variantSchema)
				}
			}
			prop.OneOf = schemas

			// Add discriminator as an OpenAPI extension
			// This is stored in Extras since it's OpenAPI-specific, not core JSON Schema
			if prop.Extras == nil {
				prop.Extras = make(map[string]any)
			}
			prop.Extras["discriminator"] = map[string]any{
				"propertyName": propertyName,
			}
		}
	}
}

// createSchemaForType creates a JSON Schema from a reflect.Type
func createSchemaForType(t reflect.Type) *jsonschema.Schema {
	if t == nil {
		return nil
	}

	// Handle pointer recursion
	if t.Kind() == reflect.Pointer {
		return createSchemaForType(t.Elem())
	}

	// Handle structs (use Ref)
	if t.Kind() == reflect.Struct {
		return &jsonschema.Schema{
			Ref: fmt.Sprintf("#/$defs/%s", t.Name()),
		}
	}

	// Handle arrays/slices
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		elemType := t.Elem()
		itemSchema := createSchemaForType(elemType)
		return &jsonschema.Schema{
			Type:  "array",
			Items: itemSchema,
		}
	}

	// Handle primitives and maps using shared type mapping
	typeName := godantic.GetJSONSchemaType(t)
	if typeName != "" {
		return &jsonschema.Schema{Type: typeName}
	}

	return &jsonschema.Schema{}
}

// toJSONNumber converts numeric values to json.Number
func toJSONNumber(v any) json.Number {
	switch val := v.(type) {
	case int:
		return json.Number(fmt.Sprintf("%d", val))
	case int64:
		return json.Number(fmt.Sprintf("%d", val))
	case float64:
		return json.Number(fmt.Sprintf("%g", val))
	default:
		return json.Number("0")
	}
}
