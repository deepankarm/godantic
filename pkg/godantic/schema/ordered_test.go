package schema_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

// Types for ordered schema tests
type OrderedTypeA struct {
	FirstField  string `json:"first_field"`
	SecondField int    `json:"second_field"`
	ThirdField  bool   `json:"third_field"`
}

type OrderedTypeB struct {
	Alpha string `json:"alpha"`
	Beta  int    `json:"beta"`
}

type OrderedNested struct {
	Name    string          `json:"name"`
	Details OrderedDetails  `json:"details"`
	Tags    *[]string       `json:"tags"`
}

type OrderedDetails struct {
	Value   string `json:"value"`
	Count   int    `json:"count"`
}

func TestGenerateUnionSchemaOrdered(t *testing.T) {
	result, err := schema.GenerateUnionSchemaOrdered(OrderedTypeA{}, OrderedTypeB{})
	if err != nil {
		t.Fatalf("GenerateUnionSchemaOrdered failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Must be valid JSON
	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// Must have anyOf and $defs
	if _, ok := parsed["anyOf"]; !ok {
		t.Error("expected anyOf in result")
	}
	if _, ok := parsed["$defs"]; !ok {
		t.Error("expected $defs in result")
	}
}

func TestGenerateUnionSchemaOrdered_SingleType(t *testing.T) {
	result, err := schema.GenerateUnionSchemaOrdered(OrderedTypeA{})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil")
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Single type should have properties, not anyOf
	if _, ok := parsed["anyOf"]; ok {
		t.Error("single type should not have anyOf")
	}
	if _, ok := parsed["properties"]; !ok {
		t.Error("expected properties for single type")
	}
}

func TestGenerateUnionSchemaOrdered_Empty(t *testing.T) {
	result, err := schema.GenerateUnionSchemaOrdered()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for empty, got %s", string(result))
	}
}

func TestGenerateUnionSchemaOrdered_PreservesFieldOrder(t *testing.T) {
	result, err := schema.GenerateUnionSchemaOrdered(OrderedTypeA{})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	raw := string(result)
	firstIdx := strings.Index(raw, `"first_field"`)
	secondIdx := strings.Index(raw, `"second_field"`)
	thirdIdx := strings.Index(raw, `"third_field"`)

	if firstIdx < 0 || secondIdx < 0 || thirdIdx < 0 {
		t.Fatalf("missing fields in output: %s", raw)
	}
	if firstIdx >= secondIdx || secondIdx >= thirdIdx {
		t.Errorf("fields not in declaration order: first=%d, second=%d, third=%d", firstIdx, secondIdx, thirdIdx)
	}
}

func TestTransformForOpenAI_PreservesFieldOrder(t *testing.T) {
	unionSchema, err := schema.GenerateUnionSchema(OrderedTypeA{}, OrderedTypeB{})
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}

	result, err := schema.TransformForOpenAI(unionSchema, OrderedTypeA{}, OrderedTypeB{})
	if err != nil {
		t.Fatalf("TransformForOpenAI failed: %v", err)
	}

	raw := string(result)

	// OrderedTypeA fields should appear in declaration order
	firstIdx := strings.Index(raw, `"first_field"`)
	secondIdx := strings.Index(raw, `"second_field"`)
	thirdIdx := strings.Index(raw, `"third_field"`)

	if firstIdx < 0 || secondIdx < 0 || thirdIdx < 0 {
		t.Fatalf("missing OrderedTypeA fields in output")
	}
	if firstIdx >= secondIdx || secondIdx >= thirdIdx {
		t.Errorf("OrderedTypeA fields not in declaration order: first=%d, second=%d, third=%d",
			firstIdx, secondIdx, thirdIdx)
	}

	// OrderedTypeB fields should appear in declaration order
	alphaIdx := strings.Index(raw, `"alpha"`)
	betaIdx := strings.Index(raw, `"beta"`)
	if alphaIdx < 0 || betaIdx < 0 {
		t.Fatalf("missing OrderedTypeB fields")
	}
	if alphaIdx >= betaIdx {
		t.Errorf("OrderedTypeB fields not in declaration order: alpha=%d, beta=%d", alphaIdx, betaIdx)
	}

	// Must be valid JSON
	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Must have wrapper
	if parsed["type"] != "object" {
		t.Errorf("expected root type object, got %v", parsed["type"])
	}
	props, _ := parsed["properties"].(map[string]any)
	if _, ok := props["response"]; !ok {
		t.Error("expected response wrapper property")
	}
}

func TestTransformForOpenAI_WithNestedTypes(t *testing.T) {
	unionSchema, err := schema.GenerateUnionSchema(OrderedNested{}, OrderedTypeB{})
	if err != nil {
		t.Fatalf("GenerateUnionSchema failed: %v", err)
	}

	result, err := schema.TransformForOpenAI(unionSchema, OrderedNested{}, OrderedTypeB{})
	if err != nil {
		t.Fatalf("TransformForOpenAI failed: %v", err)
	}

	// OrderedNested field order: name, details, tags
	raw := string(result)
	nameIdx := strings.Index(raw, `"name"`)
	detailsIdx := strings.Index(raw, `"details"`)
	tagsIdx := strings.Index(raw, `"tags"`)

	if nameIdx < 0 || detailsIdx < 0 || tagsIdx < 0 {
		t.Fatalf("missing fields")
	}
	if nameIdx >= detailsIdx || detailsIdx >= tagsIdx {
		t.Errorf("OrderedNested fields not in order: name=%d, details=%d, tags=%d",
			nameIdx, detailsIdx, tagsIdx)
	}

	// OrderedDetails field order: value, count
	valueIdx := strings.Index(raw, `"value"`)
	countIdx := strings.Index(raw, `"count"`)
	if valueIdx < 0 || countIdx < 0 {
		t.Fatalf("missing OrderedDetails fields")
	}
	if valueIdx >= countIdx {
		t.Errorf("OrderedDetails fields not in order: value=%d, count=%d", valueIdx, countIdx)
	}
}

func TestTransformForOpenAI_NonUnionSchema(t *testing.T) {
	// Single type schema (no anyOf)
	unionSchema, err := schema.GenerateUnionSchema(OrderedTypeA{})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	result, err := schema.TransformForOpenAI(unionSchema, OrderedTypeA{})
	if err != nil {
		t.Fatalf("TransformForOpenAI failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Non-union should NOT be wrapped
	if _, ok := parsed["properties"].(map[string]any)["response"]; ok {
		t.Error("non-union schema should not have response wrapper")
	}
}
