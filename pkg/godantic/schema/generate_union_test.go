package schema_test

import (
	"encoding/json"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

// Test types for union schema generation
type UnionTypeA struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type UnionTypeB struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

type UnionTypeC struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Type with nested struct
type UnionNested struct {
	Data NestedData `json:"data"`
}

type NestedData struct {
	Field string `json:"field"`
}

// Type with godantic field options
type UnionWithOptions struct {
	Required string `json:"required"`
	Optional string `json:"optional"`
}

func (u *UnionWithOptions) FieldRequired() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(1))
}

func (u *UnionWithOptions) FieldOptional() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Default("default_value"))
}

func TestGenerateUnionSchema_Empty(t *testing.T) {
	result, err := schema.GenerateUnionSchema()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestGenerateUnionSchema_NilType(t *testing.T) {
	_, err := schema.GenerateUnionSchema(nil)
	if err == nil {
		t.Error("expected error for nil type")
	}
}

func TestGenerateUnionSchema_SingleType(t *testing.T) {
	result, err := schema.GenerateUnionSchema(UnionTypeA{})
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Single type should NOT have anyOf
	if _, hasAnyOf := result["anyOf"]; hasAnyOf {
		t.Error("single type should not have anyOf wrapper")
	}

	// Should have properties from UnionTypeA
	props, ok := result["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in schema")
	}
	if _, ok := props["name"]; !ok {
		t.Error("expected 'name' property")
	}
	if _, ok := props["value"]; !ok {
		t.Error("expected 'value' property")
	}
}

func TestGenerateUnionSchema_TwoTypes(t *testing.T) {
	result, err := schema.GenerateUnionSchema(UnionTypeA{}, UnionTypeB{})
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}

	anyOf, ok := result["anyOf"].([]map[string]any)
	if !ok {
		t.Fatalf("expected anyOf array, got %T", result["anyOf"])
	}
	if len(anyOf) != 2 {
		t.Errorf("expected 2 schemas in anyOf, got %d", len(anyOf))
	}

	// Verify each schema has correct properties
	schema1 := anyOf[0]
	props1, _ := schema1["properties"].(map[string]any)
	if _, ok := props1["name"]; !ok {
		t.Error("first schema should have 'name' property")
	}

	schema2 := anyOf[1]
	props2, _ := schema2["properties"].(map[string]any)
	if _, ok := props2["id"]; !ok {
		t.Error("second schema should have 'id' property")
	}
}

func TestGenerateUnionSchema_ThreeTypes(t *testing.T) {
	result, err := schema.GenerateUnionSchema(UnionTypeA{}, UnionTypeB{}, UnionTypeC{})
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}

	anyOf, ok := result["anyOf"].([]map[string]any)
	if !ok {
		t.Fatalf("expected anyOf array, got %T", result["anyOf"])
	}
	if len(anyOf) != 3 {
		t.Errorf("expected 3 schemas in anyOf, got %d", len(anyOf))
	}
}

func TestGenerateUnionSchema_WithNestedTypes(t *testing.T) {
	result, err := schema.GenerateUnionSchema(UnionTypeA{}, UnionNested{})
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}

	anyOf, ok := result["anyOf"].([]map[string]any)
	if !ok {
		t.Fatalf("expected anyOf array, got %T", result["anyOf"])
	}
	if len(anyOf) != 2 {
		t.Errorf("expected 2 schemas, got %d", len(anyOf))
	}

	// Nested type should have its $defs merged
	defs, ok := result["$defs"].(map[string]any)
	if !ok {
		t.Fatal("expected $defs for nested types")
	}
	if _, ok := defs["NestedData"]; !ok {
		t.Error("expected NestedData in $defs")
	}
}

func TestGenerateUnionSchema_WithFieldOptions(t *testing.T) {
	result, err := schema.GenerateUnionSchema(UnionWithOptions{}, UnionTypeA{})
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}

	anyOf, ok := result["anyOf"].([]map[string]any)
	if !ok {
		t.Fatalf("expected anyOf array, got %T", result["anyOf"])
	}

	// First schema (UnionWithOptions) should have constraints
	optionsSchema := anyOf[0]
	props, _ := optionsSchema["properties"].(map[string]any)
	requiredField, _ := props["required"].(map[string]any)

	// Should have minLength constraint
	if minLen, ok := requiredField["minLength"]; !ok || minLen != float64(1) {
		t.Error("expected minLength: 1 on required field")
	}

	// Should have required field in required array
	required, _ := optionsSchema["required"].([]any)
	found := false
	for _, r := range required {
		if r == "required" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'required' in required array")
	}
}

func TestGenerateUnionSchema_PreservesTypeStructure(t *testing.T) {
	result, err := schema.GenerateUnionSchema(UnionTypeA{}, UnionTypeB{})
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}

	// Each type in anyOf should be type: object
	anyOf := result["anyOf"].([]map[string]any)
	for i, s := range anyOf {
		if s["type"] != "object" {
			t.Errorf("schema %d should be type: object, got %v", i, s["type"])
		}
	}
}

func TestGenerateUnionSchema_JSONSerializable(t *testing.T) {
	result, err := schema.GenerateUnionSchema(UnionTypeA{}, UnionTypeB{}, UnionTypeC{})
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}

	// Should be JSON serializable
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal to JSON: %v", err)
	}

	// Should be parseable back
	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Verify structure preserved
	anyOf, ok := parsed["anyOf"].([]any)
	if !ok {
		t.Fatal("anyOf not preserved in JSON round-trip")
	}
	if len(anyOf) != 3 {
		t.Errorf("expected 3 schemas after round-trip, got %d", len(anyOf))
	}
}

// Test with pointer types
func TestGenerateUnionSchema_PointerTypes(t *testing.T) {
	result, err := schema.GenerateUnionSchema(&UnionTypeA{}, &UnionTypeB{})
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}

	anyOf, ok := result["anyOf"].([]map[string]any)
	if !ok {
		t.Fatalf("expected anyOf array, got %T", result["anyOf"])
	}
	if len(anyOf) != 2 {
		t.Errorf("expected 2 schemas, got %d", len(anyOf))
	}
}

// Test types with same field names but different types
type SameFieldA struct {
	Status string `json:"status"`
	Value  int    `json:"value"`
}

type SameFieldB struct {
	Status string `json:"status"`
	Value  string `json:"value"`
}

func TestGenerateUnionSchema_OverlappingFields(t *testing.T) {
	result, err := schema.GenerateUnionSchema(SameFieldA{}, SameFieldB{})
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}

	anyOf := result["anyOf"].([]map[string]any)
	if len(anyOf) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(anyOf))
	}

	// First schema: value should be integer
	propsA := anyOf[0]["properties"].(map[string]any)
	valueA := propsA["value"].(map[string]any)
	if valueA["type"] != "integer" {
		t.Errorf("SameFieldA.Value should be integer, got %v", valueA["type"])
	}

	// Second schema: value should be string
	propsB := anyOf[1]["properties"].(map[string]any)
	valueB := propsB["value"].(map[string]any)
	if valueB["type"] != "string" {
		t.Errorf("SameFieldB.Value should be string, got %v", valueB["type"])
	}
}

// Test for API response union (e.g., success | error | pending)
type SuccessResult struct {
	Status string            `json:"status"`
	Data   map[string]string `json:"data"`
}

type ErrorResult struct {
	Status  string `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type PendingResult struct {
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	ETA      string `json:"eta"`
}

func TestGenerateUnionSchema_APIResponse(t *testing.T) {
	result, err := schema.GenerateUnionSchema(
		SuccessResult{},
		ErrorResult{},
		PendingResult{},
	)
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}

	anyOf := result["anyOf"].([]map[string]any)
	if len(anyOf) != 3 {
		t.Fatalf("expected 3 variants, got %d", len(anyOf))
	}

	// All should have status field (common discriminator pattern)
	for i, s := range anyOf {
		props := s["properties"].(map[string]any)
		if _, ok := props["status"]; !ok {
			t.Errorf("schema %d missing status field", i)
		}
	}

	// Should be valid JSON Schema
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var schemaCheck map[string]any
	if err := json.Unmarshal(jsonBytes, &schemaCheck); err != nil {
		t.Fatalf("invalid JSON Schema: %v", err)
	}
}

// Test with multiple nested types to ensure $defs merging works
type ParentA struct {
	Child ChildType `json:"child"`
}

type ParentB struct {
	Child ChildType `json:"child"`
	Extra string    `json:"extra"`
}

type ChildType struct {
	Value int `json:"value"`
}

func TestGenerateUnionSchema_SharedNestedTypes(t *testing.T) {
	result, err := schema.GenerateUnionSchema(ParentA{}, ParentB{})
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}

	anyOf := result["anyOf"].([]map[string]any)
	if len(anyOf) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(anyOf))
	}

	// Both reference same ChildType, should be in merged $defs
	defs, ok := result["$defs"].(map[string]any)
	if !ok {
		t.Fatal("expected $defs for shared nested type")
	}
	if _, ok := defs["ChildType"]; !ok {
		t.Error("expected ChildType in merged $defs")
	}
}

// Test simple types without $ref (primitives don't need flattening)
type SimpleFlat struct {
	Name string `json:"name"`
}

func TestGenerateUnionSchema_NoRefNeeded(t *testing.T) {
	// Simple struct should still work even if schema has no $ref
	result, err := schema.GenerateUnionSchema(SimpleFlat{})
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}

	// Should have properties directly
	props, ok := result["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in schema")
	}
	if _, ok := props["name"]; !ok {
		t.Error("expected 'name' property")
	}
}
