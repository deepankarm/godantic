package schema_test

import (
	"slices"
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

	t.Run("nested struct should have required fields", func(t *testing.T) {
		s, err := sg.Generate()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		// Check parent struct (SchemaCompany) has required fields
		companySchema := s.Definitions["SchemaCompany"]
		if companySchema == nil {
			t.Fatal("SchemaCompany definition not found")
		}

		expectedCompanyRequired := []string{"name", "address", "size"}
		for _, field := range expectedCompanyRequired {
			if !slices.Contains(companySchema.Required, field) {
				t.Errorf("SchemaCompany should have '%s' in required array", field)
			}
		}

		// Check nested struct (SchemaAddress) has required fields
		addressSchema := s.Definitions["SchemaAddress"]
		if addressSchema == nil {
			t.Fatal("SchemaAddress definition not found")
		}

		expectedAddressRequired := []string{"street", "city", "zipCode"}
		for _, field := range expectedAddressRequired {
			if !slices.Contains(addressSchema.Required, field) {
				t.Errorf("SchemaAddress should have '%s' in required array, got: %v", field, addressSchema.Required)
			}
		}
	})
}
