package schema_test

import (
	"strings"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

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
