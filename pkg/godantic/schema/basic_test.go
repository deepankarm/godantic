package schema_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
	"github.com/invopop/jsonschema"
)

// Simple user for schema generation
type SchemaUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	Username string `json:"username"`
	Active   bool   `json:"active"`
}

func (u *SchemaUser) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Description[string]("User's full name"),
		godantic.MinLen(2),
		godantic.MaxLen(50),
	)
}

func (u *SchemaUser) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Description[string]("User's email address"),
		godantic.Regex(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
	)
}

func (u *SchemaUser) FieldAge() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.Description[int]("User's age in years"),
		godantic.Min(0),
		godantic.Max(130),
	)
}

func (u *SchemaUser) FieldUsername() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Description[string]("Unique username"),
		godantic.Regex(`^[a-zA-Z0-9_]+$`),
	)
}

func (u *SchemaUser) FieldActive() godantic.FieldOptions[bool] {
	return godantic.Field(
		godantic.Required[bool](),
		godantic.Description[bool]("Whether the user account is active"),
	)
}

func TestSchemaGeneration(t *testing.T) {
	sg := schema.NewGenerator[SchemaUser]()

	t.Run("generate basic schema", func(t *testing.T) {
		s, err := sg.Generate()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		if s == nil {
			t.Fatal("schema is nil")
		}
		if s.Definitions == nil {
			t.Fatal("definitions is nil")
		}
		if len(s.Definitions) == 0 {
			t.Fatal("schema definitions is empty")
		}
		var actualSchema *jsonschema.Schema
		for _, def := range s.Definitions {
			actualSchema = def
			break
		}

		if actualSchema == nil || actualSchema.Properties == nil {
			t.Fatal("actual schema properties not found")
		}
		nameProp, ok := actualSchema.Properties.Get("name")
		if !ok || nameProp == nil {
			t.Error("name property not found")
		}

		emailProp, ok := actualSchema.Properties.Get("email")
		if !ok || emailProp == nil {
			t.Error("email property not found")
		}
	})

	t.Run("generate schema as JSON", func(t *testing.T) {
		schemaJSON, err := sg.GenerateJSON()
		if err != nil {
			t.Fatalf("failed to generate schema JSON: %v", err)
		}

		if schemaJSON == "" {
			t.Fatal("schema JSON is empty")
		}

		// Parse to verify it's valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(schemaJSON), &parsed); err != nil {
			t.Fatalf("schema JSON is not valid: %v", err)
		}

		// Check for expected fields (schema uses $defs)
		if _, ok := parsed["$defs"]; !ok {
			t.Error("schema missing '$defs' field")
		}

		// Verify it contains our validation constraints
		if !strings.Contains(schemaJSON, "minLength") {
			t.Error("schema should contain minLength constraint")
		}
		if !strings.Contains(schemaJSON, "maxLength") {
			t.Error("schema should contain maxLength constraint")
		}
		if !strings.Contains(schemaJSON, "pattern") {
			t.Error("schema should contain pattern constraint")
		}
	})

	t.Run("required fields are marked", func(t *testing.T) {
		s, err := sg.Generate()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		// Find the actual schema in definitions
		var actualSchema *jsonschema.Schema
		for _, def := range s.Definitions {
			actualSchema = def
			break
		}

		if actualSchema == nil {
			t.Fatal("actual schema not found in definitions")
		}

		// Check required fields
		requiredFields := []string{"name", "email", "username"}
		for _, field := range requiredFields {
			found := slices.Contains(actualSchema.Required, field)
			if !found {
				t.Errorf("field '%s' should be marked as required", field)
			}
		}
	})
}

func TestSchemaWithOptions(t *testing.T) {
	t.Run("generate schema with custom options", func(t *testing.T) {
		s, err := schema.GenerateWithOptions[SchemaUser](schema.Options{
			Title:       "User Schema",
			Description: "Schema for user objects in the system",
			Version:     "1.0.0",
		})

		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		if s.Title != "User Schema" {
			t.Errorf("expected title 'User Schema', got '%s'", s.Title)
		}

		if s.Description != "Schema for user objects in the system" {
			t.Errorf("unexpected description: %s", s.Description)
		}

		if s.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got '%s'", s.Version)
		}
	})
}

func TestGenerateFlattened(t *testing.T) {
	sg := schema.NewGenerator[SchemaUser]()

	t.Run("flattened schema has root object at top level", func(t *testing.T) {
		flatSchema, err := sg.GenerateFlattened()
		if err != nil {
			t.Fatalf("failed to generate flattened schema: %v", err)
		}

		// Should have type at root
		schemaType, ok := flatSchema["type"].(string)
		if !ok || schemaType != "object" {
			t.Errorf("expected type 'object' at root, got: %v", flatSchema["type"])
		}

		// Should have properties at root
		_, hasProperties := flatSchema["properties"]
		if !hasProperties {
			t.Error("flattened schema should have 'properties' at root level")
		}

		// Should have required at root
		_, hasRequired := flatSchema["required"]
		if !hasRequired {
			t.Error("flattened schema should have 'required' at root level")
		}

		// Should NOT have $ref at root
		if _, hasRef := flatSchema["$ref"]; hasRef {
			t.Error("flattened schema should not have '$ref' at root level")
		}
	})

	t.Run("flattened schema preserves validation constraints", func(t *testing.T) {
		flatSchema, err := sg.GenerateFlattened()
		if err != nil {
			t.Fatalf("failed to generate flattened schema: %v", err)
		}

		properties, ok := flatSchema["properties"].(map[string]any)
		if !ok {
			t.Fatal("properties is not a map")
		}

		// Check that name field has constraints
		nameProp, ok := properties["name"].(map[string]any)
		if !ok {
			t.Fatal("name property not found")
		}

		if _, hasMinLength := nameProp["minLength"]; !hasMinLength {
			t.Error("name property should have minLength constraint")
		}

		if _, hasMaxLength := nameProp["maxLength"]; !hasMaxLength {
			t.Error("name property should have maxLength constraint")
		}

		// Check that email field has pattern
		emailProp, ok := properties["email"].(map[string]any)
		if !ok {
			t.Fatal("email property not found")
		}

		if _, hasPattern := emailProp["pattern"]; !hasPattern {
			t.Error("email property should have pattern constraint")
		}
	})
}

func TestGeneratorWithOptions(t *testing.T) {
	sg := schema.NewGenerator[SchemaUser]().WithOptions(schema.SchemaOptions{
		AutoGenerateTitles: false,
	})

	s, err := sg.Generate()
	if err != nil {
		t.Fatalf("failed to generate schema: %v", err)
	}

	if s == nil {
		t.Error("expected non-nil schema")
	}
}

func TestGeneratorWithAutoTitles(t *testing.T) {
	sg := schema.NewGenerator[SchemaUser]().WithAutoTitles(false)

	s, err := sg.Generate()
	if err != nil {
		t.Fatalf("failed to generate schema: %v", err)
	}

	if s == nil {
		t.Error("expected non-nil schema")
	}
}

func TestGenerateJSON(t *testing.T) {
	sg := schema.NewGenerator[SchemaUser]()

	jsonStr, err := sg.GenerateJSON()
	if err != nil {
		t.Fatalf("GenerateJSON failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("expected non-empty JSON string")
	}

	// Should be valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Errorf("GenerateJSON produced invalid JSON: %v", err)
	}
}
