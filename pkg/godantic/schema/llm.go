package schema

// LLM schema transformation utilities for OpenAI, Gemini, Claude structured output.
// These functions adapt JSON schemas to meet provider-specific requirements.

// TransformForOpenAI adapts a JSON schema for OpenAI's structured output requirements.
// OpenAI strict mode requires:
//   - type "object" at root level
//   - no anyOf/oneOf/allOf at root level (must be wrapped)
//   - no $ref references (must be inlined)
//   - all properties must be in required array
//   - additionalProperties must be false
//
// Union schemas from GenerateUnionSchema have "anyOf" at root, so they get wrapped
// in a "response" property.
func TransformForOpenAI(schema map[string]any) map[string]any {
	return TransformForOpenAIWithOptions(schema, OpenAITransformOptions{
		WrapperPropertyName: "response",
	})
}

// OpenAITransformOptions configures the OpenAI schema transformation
type OpenAITransformOptions struct {
	// WrapperPropertyName is the property name used to wrap union schemas.
	// Default is "response". Python/Pydantic uses "field".
	WrapperPropertyName string
}

// TransformForOpenAIWithOptions adapts a JSON schema with custom options.
func TransformForOpenAIWithOptions(schema map[string]any, opts OpenAITransformOptions) map[string]any {
	if opts.WrapperPropertyName == "" {
		opts.WrapperPropertyName = "response"
	}

	// First, resolve all $ref references (OpenAI strict mode doesn't support $ref)
	defs, _ := schema["$defs"].(map[string]any)
	resolved := resolveRefs(schema, defs)
	resolvedSchema, ok := resolved.(map[string]any)
	if !ok {
		// Fallback to original schema if resolution fails
		resolvedSchema = schema
	}

	// Remove $defs from the resolved schema since all refs are now inlined
	delete(resolvedSchema, "$defs")

	// Ensure all properties are in required arrays (OpenAI strict mode requirement)
	ensureAllPropertiesRequired(resolvedSchema)

	// Check if schema has anyOf/oneOf/allOf at root level
	_, hasAnyOf := resolvedSchema["anyOf"]
	_, hasOneOf := resolvedSchema["oneOf"]
	_, hasAllOf := resolvedSchema["allOf"]

	if !hasAnyOf && !hasOneOf && !hasAllOf {
		// Schema doesn't have union types at root, just ensure type is set
		if _, hasType := resolvedSchema["type"]; !hasType {
			resolvedSchema["type"] = "object"
		}
		return resolvedSchema
	}

	// Wrap the union schema in a property
	// This makes: {"anyOf": [...]} -> {"type": "object", "properties": {"response": {"anyOf": [...]}}}
	wrappedSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			opts.WrapperPropertyName: resolvedSchema,
		},
		"required":             []string{opts.WrapperPropertyName},
		"additionalProperties": false,
	}

	return wrappedSchema
}

// resolveRefs recursively resolves all $ref references in a JSON schema by inlining definitions.
// This is required for OpenAI strict mode which doesn't support JSON Schema $ref.
func resolveRefs(node any, defs map[string]any) any {
	switch v := node.(type) {
	case map[string]any:
		// Check if this is a $ref
		if ref, ok := v["$ref"].(string); ok {
			// Extract the definition name from "#/$defs/Name"
			const prefix = "#/$defs/"
			if len(ref) > len(prefix) && ref[:len(prefix)] == prefix {
				defName := ref[len(prefix):]
				if def, ok := defs[defName]; ok {
					// Recursively resolve refs in the definition
					if defMap, ok := def.(map[string]any); ok {
						resolved := resolveRefs(deepCopyMap(defMap), defs)
						if resolvedMap, ok := resolved.(map[string]any); ok {
							// Preserve sibling properties (description, title, etc.) from the $ref object
							// by merging them into the resolved definition
							for key, val := range v {
								if key != "$ref" {
									resolvedMap[key] = val
								}
							}
							return resolvedMap
						}
						return resolved
					}
				}
			}
			// If we can't resolve, return as-is (will cause OpenAI error)
			return v
		}

		// Recursively process all values
		result := make(map[string]any, len(v))
		for key, val := range v {
			if key == "$defs" {
				// Skip $defs - we're inlining everything
				continue
			}
			result[key] = resolveRefs(val, defs)
		}
		return result

	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = resolveRefs(item, defs)
		}
		return result

	case []map[string]any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = resolveRefs(item, defs)
		}
		return result

	default:
		return v
	}
}

// ensureAllPropertiesRequired recursively ensures all object properties are in the required array.
// This is required for OpenAI strict mode which mandates all properties be required.
func ensureAllPropertiesRequired(node any) {
	switch v := node.(type) {
	case map[string]any:
		// If this is an object with properties, ensure all are required
		if v["type"] == "object" {
			if props, ok := v["properties"].(map[string]any); ok {
				// Build list of all property names
				propNames := make([]string, 0, len(props))
				for name := range props {
					propNames = append(propNames, name)
				}

				// Set required to all property names
				v["required"] = propNames

				// Ensure additionalProperties is false
				v["additionalProperties"] = false

				// Recursively process each property
				for _, propSchema := range props {
					ensureAllPropertiesRequired(propSchema)
				}
			}
		}

		// Process anyOf/oneOf/allOf
		for _, key := range []string{"anyOf", "oneOf", "allOf"} {
			if items, ok := v[key].([]any); ok {
				for _, item := range items {
					ensureAllPropertiesRequired(item)
				}
			}
			if items, ok := v[key].([]map[string]any); ok {
				for _, item := range items {
					ensureAllPropertiesRequired(item)
				}
			}
		}

		// Process array items
		if v["type"] == "array" {
			if items, ok := v["items"].(map[string]any); ok {
				ensureAllPropertiesRequired(items)
			}
		}

	case []any:
		for _, item := range v {
			ensureAllPropertiesRequired(item)
		}

	case []map[string]any:
		for _, item := range v {
			ensureAllPropertiesRequired(item)
		}
	}
}
