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

// Example with nested struct
type SchemaAddress struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	ZipCode string `json:"zipCode"`
}

func (a *SchemaAddress) FieldStreet() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(3),
	)
}

func (a *SchemaAddress) FieldCity() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(2),
	)
}

func (a *SchemaAddress) FieldZipCode() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Regex(`^[0-9]{5}$`),
	)
}

type SchemaCompany struct {
	Name    string        `json:"name"`
	Address SchemaAddress `json:"address"`
	Size    int           `json:"size"`
}

func (c *SchemaCompany) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
	)
}

func (c *SchemaCompany) FieldAddress() godantic.FieldOptions[SchemaAddress] {
	return godantic.Field(
		godantic.Required[SchemaAddress](),
	)
}

func (c *SchemaCompany) FieldSize() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.Min(1),
	)
}

func TestNestedSchemaGeneration(t *testing.T) {
	sg := schema.NewGenerator[SchemaCompany]()

	t.Run("generate schema with nested struct", func(t *testing.T) {
		schemaJSON, err := sg.GenerateJSON()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		// Verify nested structure
		if !strings.Contains(schemaJSON, "address") {
			t.Error("schema should contain 'address' field")
		}

		if !strings.Contains(schemaJSON, "street") {
			t.Error("schema should contain nested 'street' field")
		}
	})
}

// Test enum constraints from type-level validation
type OrderStatus string

const (
	OrderPending   OrderStatus = "pending"
	OrderConfirmed OrderStatus = "confirmed"
	OrderShipped   OrderStatus = "shipped"
	OrderDelivered OrderStatus = "delivered"
)

// Type-level validation for OrderStatus
func (OrderStatus) FieldOrderStatus() godantic.FieldOptions[OrderStatus] {
	return godantic.Field(
		godantic.Required[OrderStatus](),
		godantic.OneOf(OrderPending, OrderConfirmed, OrderShipped, OrderDelivered),
		godantic.Description[OrderStatus]("Current status of the order"),
	)
}

type Order struct {
	ID     string
	Status OrderStatus // Uses type-level validation
}

func (o *Order) FieldID() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
	)
}

func TestEnumConstraintsInSchema(t *testing.T) {
	t.Run("enum from type-level validation should be in schema", func(t *testing.T) {
		sg := schema.NewGenerator[Order]()
		schemaJSON, err := sg.GenerateJSON()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		// Check that enum constraint is present
		if !strings.Contains(schemaJSON, `"enum"`) {
			t.Error("schema should contain enum constraint from OrderStatus type")
		}

		// Check that all enum values are present
		expectedValues := []string{
			`"pending"`,
			`"confirmed"`,
			`"shipped"`,
			`"delivered"`,
		}
		for _, val := range expectedValues {
			if !strings.Contains(schemaJSON, val) {
				t.Errorf("schema should contain enum value %s", val)
			}
		}

		// Check that description is present
		if !strings.Contains(schemaJSON, "Current status of the order") {
			t.Error("schema should contain description from type-level validation")
		}

		// Verify the Status field is marked as required
		if !strings.Contains(schemaJSON, `"required"`) {
			t.Error("schema should mark Status as required")
		}
	})
}
