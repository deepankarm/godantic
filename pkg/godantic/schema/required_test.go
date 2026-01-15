package schema_test

import (
	"slices"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

// TestAutoRequiredNonPointerFields tests that non-pointer struct fields are automatically
// added to the "required" array in the JSON schema, matching Pydantic's behavior.
// This is important for OpenAI's structured output (strict: true) where required fields
// must be explicitly listed.

// SimpleStruct has fields WITHOUT explicit Required() - non-pointer fields should still be required
type SimpleStruct struct {
	Name    string  `json:"name"`    // non-pointer, should be required
	Age     int     `json:"age"`     // non-pointer, should be required
	Active  bool    `json:"active"`  // non-pointer, should be required
	Salary  float64 `json:"salary"`  // non-pointer, should be required
	Pointer *string `json:"pointer"` // pointer, should NOT be required
}

// No Field methods defined - relying on auto-required for non-pointer fields

func TestAutoRequiredNonPointerFields(t *testing.T) {
	sg := schema.NewGenerator[SimpleStruct]()
	s, err := sg.Generate()
	if err != nil {
		t.Fatalf("failed to generate schema: %v", err)
	}

	// Find the actual schema definition
	var actualSchema = s.Definitions["SimpleStruct"]
	if actualSchema == nil {
		t.Fatal("SimpleStruct definition not found")
	}

	// Non-pointer fields should be required
	nonPointerFields := []string{"name", "age", "active", "salary"}
	for _, field := range nonPointerFields {
		if !slices.Contains(actualSchema.Required, field) {
			t.Errorf("non-pointer field '%s' should be in required array, but required=%v", field, actualSchema.Required)
		}
	}

	// Pointer field should NOT be required
	if slices.Contains(actualSchema.Required, "pointer") {
		t.Errorf("pointer field 'pointer' should NOT be in required array")
	}
}

// MixedStruct has some fields with Field methods and some without
type MixedStruct struct {
	// Has Field method with explicit Required
	Name string `json:"name"`
	// Has Field method WITHOUT explicit Required - should still be required (non-pointer)
	Email string `json:"email"`
	// No Field method, non-pointer - should be required
	Age int `json:"age"`
	// No Field method, pointer - should NOT be required
	Nickname *string `json:"nickname"`
}

func (m *MixedStruct) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](), // explicit required
		godantic.Description[string]("User's name"),
	)
}

func (m *MixedStruct) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(
		// NO explicit Required() - but should still be required because non-pointer
		godantic.Description[string]("User's email"),
	)
}

func TestMixedStructRequired(t *testing.T) {
	sg := schema.NewGenerator[MixedStruct]()
	s, err := sg.Generate()
	if err != nil {
		t.Fatalf("failed to generate schema: %v", err)
	}

	var actualSchema = s.Definitions["MixedStruct"]
	if actualSchema == nil {
		t.Fatal("MixedStruct definition not found")
	}

	// All non-pointer fields should be required
	expectedRequired := []string{"name", "email", "age"}
	for _, field := range expectedRequired {
		if !slices.Contains(actualSchema.Required, field) {
			t.Errorf("non-pointer field '%s' should be in required array, but required=%v", field, actualSchema.Required)
		}
	}

	// Pointer field should NOT be required
	if slices.Contains(actualSchema.Required, "nickname") {
		t.Errorf("pointer field 'nickname' should NOT be in required array")
	}
}

// NullableStruct tests that Nullable() constraint properly marks fields as NOT required
type NullableStruct struct {
	// Non-pointer with Nullable - should NOT be required
	OptionalName string `json:"optional_name"`
	// Non-pointer without Nullable - should be required
	RequiredAge int `json:"required_age"`
}

func (n *NullableStruct) FieldOptionalName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Nullable[string](),
		godantic.Description[string]("Optional name field"),
	)
}

func TestNullableExcludesFromRequired(t *testing.T) {
	sg := schema.NewGenerator[NullableStruct]()
	s, err := sg.Generate()
	if err != nil {
		t.Fatalf("failed to generate schema: %v", err)
	}

	var actualSchema = s.Definitions["NullableStruct"]
	if actualSchema == nil {
		t.Fatal("NullableStruct definition not found")
	}

	// Non-pointer field WITH Nullable should NOT be required
	if slices.Contains(actualSchema.Required, "optional_name") {
		t.Errorf("field with Nullable() should NOT be in required array")
	}

	// Non-pointer field without Nullable should be required
	if !slices.Contains(actualSchema.Required, "required_age") {
		t.Errorf("non-pointer field 'required_age' should be in required array, but required=%v", actualSchema.Required)
	}
}

// NestedStruct tests required fields in nested structs
type InnerStruct struct {
	Value string `json:"value"` // should be required
}

type OuterStruct struct {
	Inner  InnerStruct  `json:"inner"`  // should be required (non-pointer)
	Nested *InnerStruct `json:"nested"` // should NOT be required (pointer)
}

func TestNestedStructRequired(t *testing.T) {
	sg := schema.NewGenerator[OuterStruct]()
	s, err := sg.Generate()
	if err != nil {
		t.Fatalf("failed to generate schema: %v", err)
	}

	// Check outer struct required fields
	outerSchema := s.Definitions["OuterStruct"]
	if outerSchema == nil {
		t.Fatal("OuterStruct definition not found")
	}

	if !slices.Contains(outerSchema.Required, "inner") {
		t.Errorf("non-pointer nested struct 'inner' should be in required array, but required=%v", outerSchema.Required)
	}

	if slices.Contains(outerSchema.Required, "nested") {
		t.Errorf("pointer nested struct 'nested' should NOT be in required array")
	}

	// Check inner struct required fields
	innerSchema := s.Definitions["InnerStruct"]
	if innerSchema == nil {
		t.Fatal("InnerStruct definition not found")
	}

	if !slices.Contains(innerSchema.Required, "value") {
		t.Errorf("non-pointer field 'value' in InnerStruct should be in required array, but required=%v", innerSchema.Required)
	}
}
