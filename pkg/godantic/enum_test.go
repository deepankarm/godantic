package godantic_test

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Test enums/custom types
type Status string

const (
	StatusPending  Status = "pending"
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusArchived Status = "archived"
)

// Define validation on Status type itself - will be reused across all structs
func (Status) FieldStatus() godantic.FieldOptions[Status] {
	return godantic.Field(
		godantic.Required[Status](),
		godantic.OneOf(StatusPending, StatusActive, StatusInactive, StatusArchived),
	)
}

type Task struct {
	Name   string
	Status Status // Validator automatically uses Status.FieldStatus()
}

func (t *Task) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func TestEnums(t *testing.T) {
	validator := godantic.NewValidator[Task]()

	t.Run("valid status should pass", func(t *testing.T) {
		task := Task{
			Name:   "Test Task",
			Status: StatusActive,
		}
		errs := validator.Validate(&task)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("all valid statuses should pass", func(t *testing.T) {
		statuses := []Status{StatusPending, StatusActive, StatusInactive, StatusArchived}
		for _, status := range statuses {
			task := Task{
				Name:   "Test Task",
				Status: status,
			}
			errs := validator.Validate(&task)
			if len(errs) != 0 {
				t.Errorf("expected no errors for status %s, got %d: %v", status, len(errs), errs)
			}
		}
	})

	t.Run("invalid status should fail", func(t *testing.T) {
		task := Task{
			Name:   "Test Task",
			Status: Status("invalid"),
		}
		errs := validator.Validate(&task)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
	})

	t.Run("empty status should fail", func(t *testing.T) {
		task := Task{
			Name:   "Test Task",
			Status: Status(""),
		}
		errs := validator.Validate(&task)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Status: required field" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})
}

// Test that parent struct can override type-level validation
type Severity string

const (
	SeverityLow  Severity = "low"
	SeverityHigh Severity = "high"
)

// Type-level validation for Severity
func (Severity) FieldSeverity() godantic.FieldOptions[Severity] {
	return godantic.Field(
		godantic.Required[Severity](),
		godantic.OneOf(SeverityLow, SeverityHigh),
	)
}

type Bug struct {
	Description string
	Severity    Severity
}

func (b *Bug) FieldDescription() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
	)
}

// Override type-level validation - make Severity optional for Bug
func (b *Bug) FieldSeverity() godantic.FieldOptions[Severity] {
	return godantic.Field(
		// Not required - overrides Severity.FieldSeverity()
		godantic.OneOf(SeverityLow, SeverityHigh),
	)
}

func TestTypeValidationOverride(t *testing.T) {
	validator := godantic.NewValidator[Bug]()

	t.Run("bug with severity should pass", func(t *testing.T) {
		bug := Bug{
			Description: "Memory leak",
			Severity:    SeverityHigh,
		}
		errs := validator.Validate(&bug)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("bug without severity should pass when field is zero value", func(t *testing.T) {
		bug := Bug{
			Description: "Memory leak",
			// Severity not set (zero value) - not required, so validation skipped
		}
		errs := validator.Validate(&bug)
		if len(errs) != 0 {
			t.Errorf("expected no errors (Severity not required for Bug), got %d: %v", len(errs), errs)
		}
	})

	t.Run("bug with invalid severity should fail even when not required", func(t *testing.T) {
		bug := Bug{
			Description: "Memory leak",
			Severity:    Severity("invalid"), // Invalid value still validates
		}
		errs := validator.Validate(&bug)
		if len(errs) == 0 {
			t.Error("expected validation error for invalid severity")
		}
	})
}
