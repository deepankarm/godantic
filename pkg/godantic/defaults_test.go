package godantic_test

import (
	"encoding/json"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Test struct with various default values
type ServerSettings struct {
	Name        string
	Type        string
	Port        int
	Enabled     bool
	Tags        []string
	MaxRetries  int
	Description string
}

func (c *ServerSettings) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
	)
}

func (c *ServerSettings) FieldType() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Default("default"),
		godantic.Const("default"),
	)
}

func (c *ServerSettings) FieldPort() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.Default(8080),
		godantic.Min(1),
		godantic.Max(65535),
	)
}

func (c *ServerSettings) FieldEnabled() godantic.FieldOptions[bool] {
	return godantic.Field(
		godantic.Required[bool](),
		godantic.Default(true),
	)
}

func (c *ServerSettings) FieldTags() godantic.FieldOptions[[]string] {
	return godantic.Field(
		godantic.Required[[]string](),
		godantic.Default([]string{}),
	)
}

func (c *ServerSettings) FieldMaxRetries() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Default(3),
		godantic.Min(0),
	)
}

func (c *ServerSettings) FieldDescription() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Default(""),
	)
}

func TestApplyDefaults(t *testing.T) {
	validator := godantic.NewValidator[ServerSettings]()

	t.Run("apply defaults to empty struct", func(t *testing.T) {
		settings := ServerSettings{
			Name: "test", // Only set Name (required without default)
		}

		err := validator.ApplyDefaults(&settings)
		if err != nil {
			t.Fatalf("ApplyDefaults failed: %v", err)
		}

		if settings.Type != "default" {
			t.Errorf("expected Type='default', got '%s'", settings.Type)
		}
		if settings.Port != 8080 {
			t.Errorf("expected Port=8080, got %d", settings.Port)
		}
		if !settings.Enabled {
			t.Error("expected Enabled=true")
		}
		if settings.Tags == nil {
			t.Error("expected Tags to be initialized to empty slice")
		}
		if settings.MaxRetries != 3 {
			t.Errorf("expected MaxRetries=3, got %d", settings.MaxRetries)
		}
		if settings.Description != "" {
			t.Errorf("expected Description='', got '%s'", settings.Description)
		}
	})

	t.Run("do not override explicitly set non-zero values", func(t *testing.T) {
		settings := ServerSettings{
			Name:       "test",
			Type:       "default",
			Port:       9000, // Explicitly set, should not be overridden
			Enabled:    true, // Non-zero value
			MaxRetries: 10,
		}

		err := validator.ApplyDefaults(&settings)
		if err != nil {
			t.Fatalf("ApplyDefaults failed: %v", err)
		}

		if settings.Port != 9000 {
			t.Errorf("expected Port=9000 (not overridden), got %d", settings.Port)
		}
		if settings.Enabled != true {
			t.Error("expected Enabled=true (not overridden)")
		}
		if settings.MaxRetries != 10 {
			t.Errorf("expected MaxRetries=10 (not overridden), got %d", settings.MaxRetries)
		}
	})

	t.Run("zero values are indistinguishable from unset - will be overridden", func(t *testing.T) {
		// This is a known limitation: can't tell if user set Enabled=false or didn't set it
		settings := ServerSettings{
			Name:    "test",
			Type:    "default",
			Enabled: false, // Zero value - will be overridden by default
		}

		err := validator.ApplyDefaults(&settings)
		if err != nil {
			t.Fatalf("ApplyDefaults failed: %v", err)
		}

		// Enabled will be set to default (true) because false is the zero value
		if settings.Enabled != true {
			t.Error("expected Enabled=true (default applied to zero value)")
		}
	})

	t.Run("validation passes after applying defaults", func(t *testing.T) {
		settings := ServerSettings{
			Name: "test",
		}

		// Before applying defaults, validation should fail for required fields
		errs := validator.Validate(&settings)
		// Should NOT fail because fields have defaults
		if len(errs) != 0 {
			t.Errorf("expected 0 errors (fields have defaults), got %d: %v", len(errs), errs)
		}

		// Apply defaults
		err := validator.ApplyDefaults(&settings)
		if err != nil {
			t.Fatalf("ApplyDefaults failed: %v", err)
		}

		// After applying defaults, validation should pass
		errs = validator.Validate(&settings)
		if len(errs) != 0 {
			t.Errorf("expected no validation errors after applying defaults, got %d: %v", len(errs), errs)
		}
	})
}

func TestDefaultsWithJSON(t *testing.T) {
	// Test the manual 3-step workflow (for users who need more control)
	validator := godantic.NewValidator[ServerSettings]()

	t.Run("manual workflow - unmarshal, apply defaults, validate", func(t *testing.T) {
		jsonData := `{"name": "production"}`

		var settings ServerSettings
		err := json.Unmarshal([]byte(jsonData), &settings)
		if err != nil {
			t.Fatalf("JSON unmarshal failed: %v", err)
		}

		err = validator.ApplyDefaults(&settings)
		if err != nil {
			t.Fatalf("ApplyDefaults failed: %v", err)
		}

		errs := validator.Validate(&settings)
		if len(errs) != 0 {
			t.Errorf("validation should pass after applying defaults, got errors: %v", errs)
		}

		// Verify defaults were applied
		if settings.Type != "default" {
			t.Errorf("expected Type='default' (default applied), got '%s'", settings.Type)
		}
		if settings.Port != 8080 {
			t.Errorf("expected Port=8080 (default applied), got %d", settings.Port)
		}
	})
}

// Test Marshal convenience method
func TestMarshal(t *testing.T) {
	validator := godantic.NewValidator[ServerSettings]()

	t.Run("valid JSON with all fields", func(t *testing.T) {
		jsonData := []byte(`{
			"name": "production",
			"type": "default",
			"port": 3000,
			"enabled": true,
			"tags": ["api", "v2"],
			"maxRetries": 5,
			"description": "Production server"
		}`)

		settings, errs := validator.Unmarshal(jsonData)
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got: %v", errs)
		}

		if settings.Name != "production" {
			t.Errorf("expected Name='production', got '%s'", settings.Name)
		}
		if settings.Port != 3000 {
			t.Errorf("expected Port=3000, got %d", settings.Port)
		}
		if !settings.Enabled {
			t.Error("expected Enabled=true")
		}
	})

	t.Run("JSON with missing fields gets defaults applied", func(t *testing.T) {
		jsonData := []byte(`{"name": "staging"}`)

		settings, errs := validator.Unmarshal(jsonData)
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got: %v", errs)
		}

		// Defaults should be applied
		if settings.Type != "default" {
			t.Errorf("expected Type='default' (default applied), got '%s'", settings.Type)
		}
		if settings.Port != 8080 {
			t.Errorf("expected Port=8080 (default applied), got %d", settings.Port)
		}
		if !settings.Enabled {
			t.Error("expected Enabled=true (default applied)")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		jsonData := []byte(`{invalid json}`)

		settings, errs := validator.Unmarshal(jsonData)
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d", len(errs))
		}

		if settings != nil {
			t.Error("expected nil settings on JSON error")
		}

		errMsg := errs[0].Error()
		if !contains(errMsg, "json unmarshal failed") {
			t.Errorf("expected JSON unmarshal error, got: %v", errMsg)
		}
	})

	t.Run("validation errors are returned with populated struct", func(t *testing.T) {
		// Missing required Name field
		jsonData := []byte(`{"port": 3000}`)

		settings, errs := validator.Unmarshal(jsonData)
		if len(errs) != 1 {
			t.Fatalf("expected 1 validation error, got %d: %v", len(errs), errs)
		}

		// Struct should still be returned even with validation errors
		if settings == nil {
			t.Fatal("expected settings to be returned even with validation errors")
		}

		// Port should be set from JSON
		if settings.Port != 3000 {
			t.Errorf("expected Port=3000 from JSON, got %d", settings.Port)
		}

		// Defaults should have been applied
		if settings.Type != "default" {
			t.Errorf("expected Type='default' (default applied), got '%s'", settings.Type)
		}

		// Error should be about missing Name
		errMsg := errs[0].Error()
		if !contains(errMsg, "Name") || !contains(errMsg, "required") {
			t.Errorf("expected Name required error, got: %v", errMsg)
		}
	})

	t.Run("one-liner usage example", func(t *testing.T) {
		jsonData := []byte(`{"name": "test", "port": 9000}`)

		// One-liner: unmarshal + defaults + validate
		settings, errs := validator.Unmarshal(jsonData)

		if len(errs) != 0 {
			t.Fatalf("validation failed: %v", errs)
		}

		// Ready to use
		if settings.Name != "test" {
			t.Errorf("expected Name='test', got '%s'", settings.Name)
		}
		if settings.Port != 9000 {
			t.Errorf("expected Port=9000, got %d", settings.Port)
		}
		if settings.Type != "default" {
			t.Errorf("expected Type='default' (default), got '%s'", settings.Type)
		}
	})

	t.Run("zero values get defaults applied (known limitation)", func(t *testing.T) {
		// JSON with explicit false - indistinguishable from omitted field
		jsonData := []byte(`{"name": "test", "enabled": false}`)

		settings, errs := validator.Unmarshal(jsonData)
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got: %v", errs)
		}

		// False is zero value, so default (true) is applied
		// Use *bool if you need to distinguish nil vs false
		if settings.Enabled != true {
			t.Error("expected Enabled=true (default applied to zero value)")
		}
	})
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test Unmarshal convenience method
func TestUnmarshal(t *testing.T) {
	validator := godantic.NewValidator[ServerSettings]()

	t.Run("valid struct marshals to JSON", func(t *testing.T) {
		settings := ServerSettings{
			Name:        "production",
			Type:        "default",
			Port:        3000,
			Enabled:     true,
			Tags:        []string{"api", "v2"},
			MaxRetries:  5,
			Description: "Production server",
		}

		jsonData, errs := validator.Marshal(&settings)
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got: %v", errs)
		}

		// Verify JSON is valid
		var result map[string]any
		err := json.Unmarshal(jsonData, &result)
		if err != nil {
			t.Fatalf("failed to unmarshal result JSON: %v", err)
		}

		if result["Name"] != "production" {
			t.Errorf("expected Name='production', got '%v'", result["Name"])
		}
		if result["Port"] != float64(3000) {
			t.Errorf("expected Port=3000, got %v", result["Port"])
		}
	})

	t.Run("struct with missing fields gets defaults applied before marshaling", func(t *testing.T) {
		settings := ServerSettings{
			Name: "staging",
			// Other fields are zero values, should get defaults
		}

		jsonData, errs := validator.Marshal(&settings)
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got: %v", errs)
		}

		// Parse JSON to verify defaults were applied
		var result ServerSettings
		err := json.Unmarshal(jsonData, &result)
		if err != nil {
			t.Fatalf("failed to unmarshal result JSON: %v", err)
		}

		if result.Type != "default" {
			t.Errorf("expected Type='default' (default applied), got '%s'", result.Type)
		}
		if result.Port != 8080 {
			t.Errorf("expected Port=8080 (default applied), got %d", result.Port)
		}
		if !result.Enabled {
			t.Error("expected Enabled=true (default applied)")
		}
	})

	t.Run("invalid struct returns validation errors", func(t *testing.T) {
		settings := ServerSettings{
			// Missing required Name field
			Type: "default",
			Port: 3000,
		}

		jsonData, errs := validator.Marshal(&settings)
		if len(errs) != 1 {
			t.Fatalf("expected 1 validation error, got %d: %v", len(errs), errs)
		}

		if jsonData != nil {
			t.Error("expected nil JSON data on validation error")
		}

		// Error should be about missing Name
		errMsg := errs[0].Error()
		if !contains(errMsg, "Name") || !contains(errMsg, "required") {
			t.Errorf("expected Name required error, got: %v", errMsg)
		}
	})

	t.Run("struct with constraint violations returns errors", func(t *testing.T) {
		settings := ServerSettings{
			Name: "test",
			Type: "default",
			Port: 70000, // Exceeds max of 65535
		}

		jsonData, errs := validator.Marshal(&settings)
		if len(errs) != 1 {
			t.Fatalf("expected 1 validation error, got %d: %v", len(errs), errs)
		}

		if jsonData != nil {
			t.Error("expected nil JSON data on validation error")
		}

		// Error should be about Port constraint
		errMsg := errs[0].Error()
		if !contains(errMsg, "Port") {
			t.Errorf("expected Port constraint error, got: %v", errMsg)
		}
	})

	t.Run("round-trip: Marshal then Unmarshal", func(t *testing.T) {
		// Start with JSON
		originalJSON := []byte(`{"name": "test-server", "port": 9000}`)

		// Marshal: JSON -> struct (with defaults and validation)
		settings, errs := validator.Unmarshal(originalJSON)
		if len(errs) != 0 {
			t.Fatalf("Marshal failed: %v", errs)
		}

		// Unmarshal: struct -> JSON (with validation)
		resultJSON, errs := validator.Marshal(settings)
		if len(errs) != 0 {
			t.Fatalf("Unmarshal failed: %v", errs)
		}

		// Parse result to verify
		var result ServerSettings
		err := json.Unmarshal(resultJSON, &result)
		if err != nil {
			t.Fatalf("failed to parse result JSON: %v", err)
		}

		// Verify data integrity
		if result.Name != "test-server" {
			t.Errorf("expected Name='test-server', got '%s'", result.Name)
		}
		if result.Port != 9000 {
			t.Errorf("expected Port=9000, got %d", result.Port)
		}
		// Defaults should be present
		if result.Type != "default" {
			t.Errorf("expected Type='default', got '%s'", result.Type)
		}
	})
}

// Test that defaults work with type-level validation
type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
)

func (TaskPriority) FieldTaskPriority() godantic.FieldOptions[TaskPriority] {
	return godantic.Field(
		godantic.Required[TaskPriority](),
		godantic.OneOf(TaskPriorityLow, TaskPriorityMedium, TaskPriorityHigh),
		godantic.Default(TaskPriorityMedium),
	)
}

type TaskWithDefaults struct {
	Name     string
	Priority TaskPriority
}

func (t *TaskWithDefaults) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
	)
}

// Test types for nested defaults
type NestedAddress struct {
	City    string
	Country string
}

type PersonWithAddress struct {
	Name    string
	Address NestedAddress
}

func (a *NestedAddress) FieldCity() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Default("Unknown City"))
}

func (a *NestedAddress) FieldCountry() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Default("Unknown Country"))
}

func (p *PersonWithAddress) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func TestNestedStructDefaults(t *testing.T) {
	t.Run("nested struct defaults are applied", func(t *testing.T) {
		jsonData := []byte(`{"name": "John"}`)
		validator := godantic.NewValidator[PersonWithAddress]()

		person, errs := validator.Unmarshal(jsonData)
		if errs != nil {
			t.Fatalf("Validation failed: %v", errs)
		}

		if person.Address.City != "Unknown City" {
			t.Errorf("Expected City 'Unknown City', got '%s'", person.Address.City)
		}

		if person.Address.Country != "Unknown Country" {
			t.Errorf("Expected Country 'Unknown Country', got '%s'", person.Address.Country)
		}
	})
}

func TestDefaultsWithTypeValidation(t *testing.T) {
	validator := godantic.NewValidator[TaskWithDefaults]()

	t.Run("apply default from type-level validation", func(t *testing.T) {
		task := TaskWithDefaults{
			Name: "test task",
		}

		err := validator.ApplyDefaults(&task)
		if err != nil {
			t.Fatalf("ApplyDefaults failed: %v", err)
		}

		if task.Priority != TaskPriorityMedium {
			t.Errorf("expected Priority='medium' (from type default), got '%s'", task.Priority)
		}

		// Validate
		errs := validator.Validate(&task)
		if len(errs) != 0 {
			t.Errorf("validation should pass after applying defaults, got errors: %v", errs)
		}
	})
}
