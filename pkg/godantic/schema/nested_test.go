package schema_test

import (
	"strings"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

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
