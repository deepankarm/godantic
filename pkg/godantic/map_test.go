package godantic_test

import (
	"fmt"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Test maps
type Configuration struct {
	Name     string
	Settings map[string]string
	Limits   map[string]int
}

func (c *Configuration) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
	)
}

func (c *Configuration) FieldSettings() godantic.FieldOptions[map[string]string] {
	return godantic.Field(
		godantic.Required[map[string]string](),
		godantic.Validate(func(settings map[string]string) error {
			if len(settings) == 0 {
				return fmt.Errorf("must have at least 1 setting")
			}
			// Validate required keys
			requiredKeys := []string{"host", "port"}
			for _, key := range requiredKeys {
				if _, exists := settings[key]; !exists {
					return fmt.Errorf("missing required setting: %s", key)
				}
			}
			return nil
		}),
	)
}

func (c *Configuration) FieldLimits() godantic.FieldOptions[map[string]int] {
	return godantic.Field(
		godantic.Validate(func(limits map[string]int) error {
			for key, value := range limits {
				if value < 0 {
					return fmt.Errorf("limit %s cannot be negative", key)
				}
			}
			return nil
		}),
	)
}

func TestMaps(t *testing.T) {
	validator := godantic.NewValidator[Configuration]()

	t.Run("valid configuration should pass", func(t *testing.T) {
		config := Configuration{
			Name: "prod-config",
			Settings: map[string]string{
				"host": "localhost",
				"port": "8080",
			},
			Limits: map[string]int{
				"max_connections": 100,
				"timeout":         30,
			},
		}
		errs := validator.Validate(&config)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("empty settings should fail", func(t *testing.T) {
		config := Configuration{
			Name:     "prod-config",
			Settings: map[string]string{},
		}
		errs := validator.Validate(&config)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Settings: must have at least 1 setting" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("missing required key should fail", func(t *testing.T) {
		config := Configuration{
			Name: "prod-config",
			Settings: map[string]string{
				"host": "localhost",
				// missing "port"
			},
		}
		errs := validator.Validate(&config)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Settings: missing required setting: port" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("negative limit should fail", func(t *testing.T) {
		config := Configuration{
			Name: "prod-config",
			Settings: map[string]string{
				"host": "localhost",
				"port": "8080",
			},
			Limits: map[string]int{
				"max_connections": -1, // invalid
			},
		}
		errs := validator.Validate(&config)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Limits: limit max_connections cannot be negative" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})
}
