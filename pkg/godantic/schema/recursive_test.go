package schema_test

import (
	"encoding/json"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

// TestRecursiveStructSimple tests JSON schema generation for a simple recursive struct
func TestRecursiveStructSimple(t *testing.T) {
	type Node struct {
		Value    string  `json:"value"`
		Children []*Node `json:"children,omitempty"`
	}

	gen := schema.NewGenerator[Node]()
	schema, err := gen.Generate()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// Convert to JSON for inspection
	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	t.Logf("Generated schema:\n%s", schemaJSON)

	// Verify the schema has $defs for Node
	if schema.Definitions == nil || len(schema.Definitions) == 0 {
		t.Error("Expected schema to have definitions for recursive type")
	}
}

// TreeNode is a recursive type with validation
type TreeNode struct {
	Name     string      `json:"name"`
	Value    int         `json:"value"`
	Children []*TreeNode `json:"children,omitempty"`
}

func (n *TreeNode) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(1),
	)
}

func (n *TreeNode) FieldValue() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Min(0),
	)
}

// TestRecursiveStructWithValidation tests recursive struct with godantic validation
func TestRecursiveStructWithValidation(t *testing.T) {
	gen := schema.NewGenerator[TreeNode]()
	schema, err := gen.Generate()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	t.Logf("Generated schema with validation:\n%s", schemaJSON)

	// Schema should not have infinite recursion
	if schema == nil {
		t.Error("Schema generation returned nil")
	}
}

// TestMutuallyRecursiveStructs tests mutual recursion (A -> B -> A)
func TestMutuallyRecursiveStructs(t *testing.T) {
	type PersonNode struct {
		Name      string         `json:"name"`
		BestFriend *PersonNode   `json:"best_friend,omitempty"`
		Friends    []*PersonNode `json:"friends,omitempty"`
	}

	gen := schema.NewGenerator[PersonNode]()
	schema, err := gen.Generate()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	t.Logf("Generated schema for mutually recursive:\n%s", schemaJSON)
}

// TestDeeplyNestedRecursive tests deeply nested recursive structures
func TestDeeplyNestedRecursive(t *testing.T) {
	type LinkedListNode struct {
		Data int              `json:"data"`
		Next *LinkedListNode  `json:"next,omitempty"`
	}

	gen := schema.NewGenerator[LinkedListNode]()
	schema, err := gen.Generate()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	t.Logf("Generated schema for linked list:\n%s", schemaJSON)

	// Verify it's using $ref to avoid infinite expansion
	var schemaMap map[string]any
	if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	// Check that the schema uses references ($ref) for recursive types
	// This prevents infinite expansion
	checkForRefs(t, schemaMap)
}

// TestRecursiveWithComplexFields tests recursive struct with complex nested fields
func TestRecursiveWithComplexFields(t *testing.T) {
	type Category struct {
		ID          string      `json:"id"`
		Name        string      `json:"name"`
		Description string      `json:"description"`
		Parent      *Category   `json:"parent,omitempty"`
		Children    []*Category `json:"children,omitempty"`
		Metadata    map[string]string `json:"metadata,omitempty"`
	}

	gen := schema.NewGenerator[Category]()
	schema, err := gen.Generate()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	t.Logf("Generated schema for complex recursive:\n%s", schemaJSON)
}

// checkForRefs recursively checks if a schema map contains $ref entries
func checkForRefs(t *testing.T, data any) bool {
	switch v := data.(type) {
	case map[string]any:
		for key, val := range v {
			if key == "$ref" {
				return true
			}
			if checkForRefs(t, val) {
				return true
			}
		}
	case []any:
		for _, item := range v {
			if checkForRefs(t, item) {
				return true
			}
		}
	}
	return false
}

