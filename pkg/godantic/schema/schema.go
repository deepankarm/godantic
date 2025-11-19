package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
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
	for k, v := range rootDef {
		result[k] = v
	}

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

// fieldOption is a local interface for accessing field option properties
type fieldOption interface {
	Required() bool
	Constraints() map[string]any
}

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
	collectStructTypes(rootType, structTypes)

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

// collectStructTypes recursively collects all struct types from a type
func collectStructTypes(t reflect.Type, types map[string]reflect.Type) {
	if t == nil {
		return
	}

	// Handle pointers
	if t.Kind() == reflect.Ptr {
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
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		// Recursively collect struct types
		if fieldType.Kind() == reflect.Struct {
			collectStructTypes(fieldType, types)
		}
	}
}

// enhanceDefinitionWithType enhances a schema definition with field options from a type
func enhanceDefinitionWithType(defSchema *jsonschema.Schema, t reflect.Type) {
	if defSchema.Properties == nil {
		return
	}

	// Create a zero value instance of the type
	zeroValue := reflect.New(t).Interface()

	// Use reflection to call Field* methods and get field options
	v := reflect.ValueOf(zeroValue)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Name

		// Look for Field{Name}() method
		methodName := "Field" + fieldName
		method := v.MethodByName(methodName)
		if !method.IsValid() {
			continue
		}

		// Call the method to get field options
		results := method.Call(nil)
		if len(results) != 1 {
			continue
		}

		// Get the result value and access Required_ and Constraints_ fields
		optsValue := results[0]

		// Access the Required_ field
		requiredField := optsValue.FieldByName("Required_")
		if !requiredField.IsValid() {
			continue
		}

		isRequired, ok := requiredField.Interface().(bool)
		if !ok {
			continue
		}

		// Access the Constraints_ field
		constraintsField := optsValue.FieldByName("Constraints_")
		if !constraintsField.IsValid() {
			continue
		}

		constraints, ok := constraintsField.Interface().(map[string]any)
		if !ok {
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
		if isRequired && !contains(defSchema.Required, jsonName) {
			defSchema.Required = append(defSchema.Required, jsonName)
		}

		// Apply all constraints to property
		applyConstraints(prop, constraints)
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
		if opts.Required() && !contains(actualSchema.Required, jsonName) {
			actualSchema.Required = append(actualSchema.Required, jsonName)
		}

		// Apply all constraints to property
		applyConstraints(prop, opts.Constraints())
	}
}

// findActualSchema finds the actual schema definition (might be in $defs)
func findActualSchema(schema *jsonschema.Schema) *jsonschema.Schema {
	if len(schema.Definitions) > 0 {
		// Get first definition
		for _, def := range schema.Definitions {
			return def
		}
	}
	return schema
}

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

	switch t.Kind() {
	case reflect.String:
		return &jsonschema.Schema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &jsonschema.Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return &jsonschema.Schema{Type: "number"}
	case reflect.Bool:
		return &jsonschema.Schema{Type: "boolean"}
	case reflect.Slice, reflect.Array:
		elemType := t.Elem()
		itemSchema := createSchemaForType(elemType)
		return &jsonschema.Schema{
			Type:  "array",
			Items: itemSchema,
		}
	case reflect.Map:
		return &jsonschema.Schema{Type: "object"}
	case reflect.Struct:
		return &jsonschema.Schema{
			Ref: fmt.Sprintf("#/$defs/%s", t.Name()),
		}
	case reflect.Pointer:
		return createSchemaForType(t.Elem())
	default:
		return &jsonschema.Schema{}
	}
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

// toLowerFirst converts first letter to lowercase
func toLowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]+32) + s[1:]
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
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
