package schema

import (
	"encoding/json"
	"strings"
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

func TestTransformForOpenAI_Map(t *testing.T) {
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
			name:   "keeps $ref in $defs and anyOf",
			schema: unionSchema,
			check: func(t *testing.T, transformed map[string]any) {
				// $defs should exist at root
				defs, ok := transformed["$defs"].(map[string]any)
				if !ok {
					t.Fatal("expected $defs at root")
				}

				// Should have SuccessResponse and ErrorResponse in $defs
				if _, ok := defs["SuccessResponse"]; !ok {
					t.Error("expected SuccessResponse in $defs")
				}
				if _, ok := defs["ErrorResponse"]; !ok {
					t.Error("expected ErrorResponse in $defs")
				}

				// response property should have anyOf with $ref entries
				props := transformed["properties"].(map[string]any)
				response := props["response"].(map[string]any)
				anyOf, ok := response["anyOf"].([]any)
				if !ok {
					t.Fatal("expected anyOf in response property")
				}

				for _, item := range anyOf {
					m, ok := item.(map[string]any)
					if !ok {
						t.Fatalf("expected map in anyOf, got %T", item)
					}
					ref, ok := m["$ref"].(string)
					if !ok {
						t.Fatalf("expected $ref in anyOf item, got %v", m)
					}
					if !strings.HasPrefix(ref, "#/$defs/") {
						t.Errorf("expected $ref to start with #/$defs/, got %s", ref)
					}
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
			name: "strips sibling properties from $ref nodes",
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
				// Siblings should be stripped from $ref node
				if _, has := transformed["description"]; has {
					t.Error("expected description to be stripped from $ref node")
				}
				if _, has := transformed["title"]; has {
					t.Error("expected title to be stripped from $ref node")
				}
				// $ref should be preserved
				if _, has := transformed["$ref"]; !has {
					t.Error("expected $ref to be preserved")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Deep copy schema to avoid mutation between tests
			schemaCopy := deepCopyMap(tt.schema)
			transformed := transformForOpenAI(schemaCopy)

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
			transformed := transformForOpenAIWithOptions(schemaCopy, openAITransformOptions{
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

func TestTransformForOpenAI_Ordered(t *testing.T) {
	unionSchema, err := GenerateUnionSchema(SuccessResponse{}, ErrorResponse{})
	if err != nil {
		t.Fatalf("failed to generate union schema: %v", err)
	}

	raw, err := TransformForOpenAI(unionSchema, SuccessResponse{}, ErrorResponse{})
	if err != nil {
		t.Fatalf("TransformForOpenAI failed: %v", err)
	}

	// Must return valid JSON
	if !json.Valid(raw) {
		t.Fatal("TransformForOpenAI returned invalid JSON")
	}

	// Must be parseable back to a map
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Should have the same structure as the map version
	if parsed["type"] != "object" {
		t.Errorf("expected type 'object' at root, got %v", parsed["type"])
	}
	props, ok := parsed["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties at root")
	}
	if _, has := props["response"]; !has {
		t.Error("expected 'response' property")
	}

	// Check field order in the raw JSON string (not re-marshaled, which loses order)
	s := string(raw)

	// Find the SuccessPayload section in the raw output to verify field order.
	// The ordered serializer should output properties in struct declaration order:
	// message, data, tags (matching SuccessPayload struct field order).
	spStart := strings.Index(s, `"SuccessPayload"`)
	if spStart < 0 {
		t.Fatal("expected SuccessPayload in raw output")
	}
	spSection := s[spStart:]

	msgIdx := strings.Index(spSection, `"message"`)
	dataIdx := strings.Index(spSection, `"data"`)
	tagsIdx := strings.Index(spSection, `"tags"`)
	if msgIdx < 0 || dataIdx < 0 || tagsIdx < 0 {
		t.Fatalf("missing expected fields in SuccessPayload section: %s", spSection[:200])
	}
	if !(msgIdx < dataIdx && dataIdx < tagsIdx) {
		t.Errorf("expected field order message < data < tags in SuccessPayload, got indices %d, %d, %d", msgIdx, dataIdx, tagsIdx)
	}

	// Verify the overall output has $defs before properties (preferred order)
	defsIdx := strings.Index(s, `"$defs"`)
	propsIdx := strings.Index(s, `"properties"`)
	if defsIdx < 0 || propsIdx < 0 {
		t.Fatalf("missing $defs or properties in output")
	}
	if defsIdx > propsIdx {
		t.Errorf("expected $defs before properties in ordered output, got $defs at %d, properties at %d", defsIdx, propsIdx)
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
		// Recurse into $defs
		if defs, ok := v["$defs"].(map[string]any); ok {
			for name, def := range defs {
				missing = append(missing, findObjectsWithoutRequired(def, path+".$defs."+name)...)
			}
		}
	case []any:
		for i, item := range v {
			missing = append(missing, findObjectsWithoutRequired(item, path+"["+string(rune('0'+i))+"]")...)
		}
	}

	return missing
}
