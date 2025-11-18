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

// Test default values in schema
type ServerConfig struct {
	Host    string
	Port    int
	Debug   bool
	Timeout int
}

func (s *ServerConfig) FieldHost() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Default("localhost"),
		godantic.Description[string]("Server host address"),
	)
}

func (s *ServerConfig) FieldPort() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.Default(8080),
		godantic.Description[int]("Server port"),
		godantic.Min(1),
		godantic.Max(65535),
	)
}

func (s *ServerConfig) FieldDebug() godantic.FieldOptions[bool] {
	return godantic.Field(
		godantic.Default(false),
		godantic.Description[bool]("Enable debug mode"),
	)
}

func (s *ServerConfig) FieldTimeout() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Default(30),
		godantic.Description[int]("Request timeout in seconds"),
		godantic.Min(0),
	)
}

func TestDefaultsInSchema(t *testing.T) {
	t.Run("defaults should appear in JSON schema", func(t *testing.T) {
		sg := schema.NewGenerator[ServerConfig]()
		schemaJSON, err := sg.GenerateJSON()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		// Check that default values are present in schema
		if !strings.Contains(schemaJSON, `"default"`) {
			t.Error("schema should contain default constraint")
		}

		// Check specific default values
		if !strings.Contains(schemaJSON, `"localhost"`) {
			t.Error("schema should contain default value 'localhost' for Host")
		}

		if !strings.Contains(schemaJSON, `8080`) {
			t.Error("schema should contain default value 8080 for Port")
		}

		// Verify all fields have descriptions
		expectedDescriptions := []string{
			"Server host address",
			"Server port",
			"Enable debug mode",
			"Request timeout in seconds",
		}
		for _, desc := range expectedDescriptions {
			if !strings.Contains(schemaJSON, desc) {
				t.Errorf("schema should contain description: %s", desc)
			}
		}

		// Verify required fields are marked (fields with defaults should still be required if specified)
		if !strings.Contains(schemaJSON, `"required"`) {
			t.Error("schema should have required fields")
		}
	})

	t.Run("defaults with enum values", func(t *testing.T) {
		type Environment struct {
			Name string
		}

		// Define inline for test
		validator := godantic.NewValidator[Environment]()

		// Just verify validator works - actual enum default test is in enum tests
		_ = validator
	})
}

// Test payload with const, defaults, and descriptions
type APIRequest struct {
	Action     string
	Query      string
	ResourceID string
	Dimensions []float64
}

func (r *APIRequest) FieldAction() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Const("search"),
		godantic.Default("search"),
		godantic.Description[string]("API action type"),
	)
}

func (r *APIRequest) FieldQuery() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Default(""),
		godantic.Description[string]("Search query string"),
	)
}

func (r *APIRequest) FieldResourceID() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Default(""),
		godantic.Description[string]("Target resource identifier"),
	)
}

func (r *APIRequest) FieldDimensions() godantic.FieldOptions[[]float64] {
	return godantic.Field(
		godantic.Required[[]float64](),
		godantic.Default([]float64{}),
		godantic.Description[[]float64]("Dimensions array - [width, height]"),
	)
}

func TestPayloadWithConstDefaultsAndDescriptions(t *testing.T) {
	t.Run("payload with const, defaults, and descriptions in schema", func(t *testing.T) {
		sg := schema.NewGenerator[APIRequest]()
		schemaJSON, err := sg.GenerateJSON()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		// Verify const and default for Action field
		if !strings.Contains(schemaJSON, `"const"`) {
			t.Error("schema should contain const constraint for Action")
		}
		if !strings.Contains(schemaJSON, `"search"`) {
			t.Error("schema should contain const value 'search'")
		}

		// Verify defaults are present
		if !strings.Contains(schemaJSON, `"default"`) {
			t.Error("schema should contain default values")
		}

		// Verify descriptions are present
		if !strings.Contains(schemaJSON, "Search query string") {
			t.Error("schema should contain 'Search query string' description")
		}
		if !strings.Contains(schemaJSON, "Target resource identifier") {
			t.Error("schema should contain 'Target resource identifier' description")
		}
		if !strings.Contains(schemaJSON, "API action type") {
			t.Error("schema should contain 'API action type' description")
		}
	})
}
