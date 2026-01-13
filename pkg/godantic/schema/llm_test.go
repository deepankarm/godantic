package schema

import (
	"encoding/json"
	"testing"
)

// Test types for OpenAI schema transformation tests
// These simulate union types commonly used with LLM structured output

// SuccessResponse represents a successful operation result
type SuccessResponse struct {
	Status  string         `json:"status"`
	Payload SuccessPayload `json:"payload"`
}

type SuccessPayload struct {
	Message string   `json:"message"`
	Data    []string `json:"data"`
	Tags    []string `json:"tags"`
}

// ErrorResponse represents an error result
type ErrorResponse struct {
	Status  string       `json:"status"`
	Payload ErrorPayload `json:"payload"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

func TestTransformForOpenAI(t *testing.T) {
	// Generate base union schema for tests
	unionSchema, err := GenerateUnionSchema(SuccessResponse{}, ErrorResponse{})
	if err != nil {
		t.Fatalf("failed to generate union schema: %v", err)
	}

	tests := []struct {
		name   string
		schema map[string]any
		check  func(t *testing.T, transformed map[string]any)
	}{
		{
			name:   "wraps union schema with response property",
			schema: unionSchema,
			check: func(t *testing.T, transformed map[string]any) {
				if transformed["type"] != "object" {
					t.Errorf("expected type 'object' at root, got %v", transformed["type"])
				}

				props, ok := transformed["properties"].(map[string]any)
				if !ok {
					t.Fatal("expected properties at root")
				}

				if _, hasResponse := props["response"]; !hasResponse {
					t.Error("expected 'response' property (union wrapper)")
				}
			},
		},
		{
			name:   "sets required array correctly",
			schema: unionSchema,
			check: func(t *testing.T, transformed map[string]any) {
				required, ok := transformed["required"].([]string)
				if !ok {
					t.Fatal("expected required array at root")
				}
				if len(required) != 1 || required[0] != "response" {
					t.Errorf("expected required=['response'], got %v", required)
				}
			},
		},
		{
			name:   "sets additionalProperties to false",
			schema: unionSchema,
			check: func(t *testing.T, transformed map[string]any) {
				if transformed["additionalProperties"] != false {
					t.Errorf("expected additionalProperties=false, got %v", transformed["additionalProperties"])
				}
			},
		},
		{
			name:   "removes all $ref references",
			schema: unionSchema,
			check: func(t *testing.T, transformed map[string]any) {
				refs := findRefs(transformed, "root")
				if len(refs) > 0 {
					t.Errorf("found $ref in transformed schema: %v", refs)
				}
			},
		},
		{
			name:   "ensures all properties are required",
			schema: unionSchema,
			check: func(t *testing.T, transformed map[string]any) {
				missing := findObjectsWithoutRequired(transformed, "root")
				if len(missing) > 0 {
					t.Errorf("objects without required arrays: %v", missing)
				}
			},
		},
		{
			name: "preserves sibling properties on $ref",
			schema: map[string]any{
				"$defs": map[string]any{
					"TestType": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name": map[string]any{"type": "string"},
						},
					},
				},
				"$ref":        "#/$defs/TestType",
				"description": "This is a test description",
				"title":       "Test Title",
			},
			check: func(t *testing.T, transformed map[string]any) {
				if transformed["description"] != "This is a test description" {
					t.Errorf("expected description to be preserved, got %v", transformed["description"])
				}
				if transformed["title"] != "Test Title" {
					t.Errorf("expected title to be preserved, got %v", transformed["title"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Deep copy schema to avoid mutation between tests
			schemaCopy := deepCopyMap(tt.schema)
			transformed := TransformForOpenAI(schemaCopy)

			if testing.Verbose() {
				transformedJSON, _ := json.MarshalIndent(transformed, "", "  ")
				t.Logf("Transformed schema:\n%s", string(transformedJSON))
			}

			tt.check(t, transformed)
		})
	}
}

func TestTransformForOpenAIWithOptions(t *testing.T) {
	unionSchema, err := GenerateUnionSchema(SuccessResponse{}, ErrorResponse{})
	if err != nil {
		t.Fatalf("failed to generate union schema: %v", err)
	}

	tests := []struct {
		name        string
		wrapperName string
		wantProp    string
	}{
		{
			name:        "default wrapper name",
			wrapperName: "",
			wantProp:    "response",
		},
		{
			name:        "custom wrapper name 'field'",
			wrapperName: "field",
			wantProp:    "field",
		},
		{
			name:        "custom wrapper name 'result'",
			wrapperName: "result",
			wantProp:    "result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaCopy := deepCopyMap(unionSchema)
			transformed := TransformForOpenAIWithOptions(schemaCopy, OpenAITransformOptions{
				WrapperPropertyName: tt.wrapperName,
			})

			props, ok := transformed["properties"].(map[string]any)
			if !ok {
				t.Fatal("expected properties at root")
			}

			if _, has := props[tt.wantProp]; !has {
				t.Errorf("expected '%s' property, got keys: %v", tt.wantProp, keys(props))
			}

			required := transformed["required"].([]string)
			if len(required) != 1 || required[0] != tt.wantProp {
				t.Errorf("expected required=['%s'], got %v", tt.wantProp, required)
			}
		})
	}
}

// Helper to get map keys for error messages
func keys(m map[string]any) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

// Helper to find all $ref in a schema
func findRefs(node any, path string) []string {
	var refs []string

	switch v := node.(type) {
	case map[string]any:
		if ref, ok := v["$ref"]; ok {
			refs = append(refs, path+": $ref="+ref.(string))
		}
		for key, val := range v {
			refs = append(refs, findRefs(val, path+"."+key)...)
		}
	case []any:
		for i, item := range v {
			refs = append(refs, findRefs(item, path+"["+string(rune('0'+i))+"]")...)
		}
	}

	return refs
}

// Helper to find objects without required arrays
func findObjectsWithoutRequired(node any, path string) []string {
	var missing []string

	switch v := node.(type) {
	case map[string]any:
		if v["type"] == "object" {
			if props, hasProps := v["properties"].(map[string]any); hasProps {
				if _, hasReq := v["required"]; !hasReq {
					missing = append(missing, path)
				}
				for name, prop := range props {
					missing = append(missing, findObjectsWithoutRequired(prop, path+"."+name)...)
				}
			}
		}
		for _, key := range []string{"anyOf", "oneOf", "allOf"} {
			if items, ok := v[key].([]any); ok {
				for i, item := range items {
					missing = append(missing, findObjectsWithoutRequired(item, path+"."+key+"["+string(rune('0'+i))+"]")...)
				}
			}
		}
		if v["type"] == "array" {
			if items, ok := v["items"].(map[string]any); ok {
				missing = append(missing, findObjectsWithoutRequired(items, path+".items")...)
			}
		}
	case []any:
		for i, item := range v {
			missing = append(missing, findObjectsWithoutRequired(item, path+"["+string(rune('0'+i))+"]")...)
		}
	}

	return missing
}
