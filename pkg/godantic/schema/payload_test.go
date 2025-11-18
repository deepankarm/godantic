package schema_test

import (
	"strings"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

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
