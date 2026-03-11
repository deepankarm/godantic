package schema

import "encoding/json"

// TransformForOpenAI adapts a JSON schema for OpenAI's structured output strict mode
// and returns json.RawMessage with struct field declaration order preserved.
//
// types must be the same Go struct instances used to generate the schema (needed
// to determine struct field declaration order for JSON key ordering).
//
// Applies the following transformations:
//   - Wraps root-level anyOf/oneOf/allOf in a "response" property (root must be type: object)
//   - Sets all properties as required (strict mode mandate)
//   - Sets additionalProperties: false on all objects
//   - Strips sibling properties (description, title) from $ref nodes ($ref + siblings is invalid)
//   - Strips null defaults
//   - Preserves $ref and $defs (named types help the model pick the correct union variant)
//   - Serializes with struct field declaration order (Go maps randomize key order, which
//     affects model behavior — the model is sensitive to property ordering in the schema)
func TransformForOpenAI(schema map[string]any, types ...any) (json.RawMessage, error) {
	transformed := transformForOpenAI(schema)
	reg := buildTypeRegistry(types...)
	return marshalOrdered(transformed, reg)
}

// transformForOpenAI applies OpenAI strict mode constraints to a schema,
// returning map[string]any (unordered). Used internally by TransformForOpenAI.
func transformForOpenAI(schema map[string]any) map[string]any {
	return transformForOpenAIWithOptions(schema, openAITransformOptions{
		WrapperPropertyName: "response",
	})
}

type openAITransformOptions struct {
	WrapperPropertyName string // Property name for wrapping root-level unions. Default: "response".
}

// transformForOpenAIWithOptions applies strict mode constraints with a configurable wrapper name.
func transformForOpenAIWithOptions(schema map[string]any, opts openAITransformOptions) map[string]any {
	if opts.WrapperPropertyName == "" {
		opts.WrapperPropertyName = "response"
	}

	result := deepCopyMap(schema)
	ensureStrictSchema(result)

	_, hasAnyOf := result["anyOf"]
	_, hasOneOf := result["oneOf"]
	_, hasAllOf := result["allOf"]

	if !hasAnyOf && !hasOneOf && !hasAllOf {
		if _, hasType := result["type"]; !hasType {
			result["type"] = "object"
		}
		return result
	}

	// Wrap union in a property so root is type: object.
	// $defs are hoisted to the wrapper root level.
	defs, hasDefs := result["$defs"]
	inner := make(map[string]any)
	for k, v := range result {
		if k != "$defs" {
			inner[k] = v
		}
	}

	wrapped := map[string]any{
		"type": "object",
		"properties": map[string]any{
			opts.WrapperPropertyName: inner,
		},
		"required":             []string{opts.WrapperPropertyName},
		"additionalProperties": false,
	}

	if hasDefs {
		wrapped["$defs"] = defs
	}

	return wrapped
}

// ensureStrictSchema recursively enforces OpenAI strict mode constraints:
//   - all properties listed in required
//   - additionalProperties: false on all objects
//   - $ref sibling properties stripped (OpenAI rejects $ref with description/title)
//   - null defaults stripped
func ensureStrictSchema(node any) {
	switch v := node.(type) {
	case map[string]any:
		if _, hasRef := v["$ref"]; hasRef && hasSiblingKeys(v) {
			for key := range v {
				if key != "$ref" {
					delete(v, key)
				}
			}
			return
		}

		if v["type"] == "object" {
			if props, ok := v["properties"].(map[string]any); ok {
				propNames := make([]string, 0, len(props))
				for name := range props {
					propNames = append(propNames, name)
				}
				v["required"] = propNames
				v["additionalProperties"] = false

				for _, propSchema := range props {
					ensureStrictSchema(propSchema)
				}
			}
		}

		if v["type"] == "array" {
			if items, ok := v["items"].(map[string]any); ok {
				ensureStrictSchema(items)
			}
		}

		for _, key := range []string{"anyOf", "oneOf", "allOf"} {
			if items, ok := v[key].([]any); ok {
				for _, item := range items {
					ensureStrictSchema(item)
				}
			}
			if items, ok := v[key].([]map[string]any); ok {
				for _, item := range items {
					ensureStrictSchema(item)
				}
			}
		}

		if defs, ok := v["$defs"].(map[string]any); ok {
			for _, def := range defs {
				ensureStrictSchema(def)
			}
		}

		if d, hasDefault := v["default"]; hasDefault && d == nil {
			delete(v, "default")
		}

	case []any:
		for _, item := range v {
			ensureStrictSchema(item)
		}

	case []map[string]any:
		for _, item := range v {
			ensureStrictSchema(item)
		}
	}
}

func hasSiblingKeys(node map[string]any) bool {
	for key := range node {
		if key != "$ref" {
			return true
		}
	}
	return false
}
