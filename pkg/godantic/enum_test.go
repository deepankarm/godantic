package godantic_test

import (
	"fmt"
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

type Task struct {
	Name   string
	Status Status
}

func (t *Task) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
	)
}

func (t *Task) FieldStatus() godantic.FieldOptions[Status] {
	return godantic.Field(
		godantic.Required[Status](),
		godantic.Validate(func(status Status) error {
			validStatuses := map[Status]bool{
				StatusPending:  true,
				StatusActive:   true,
				StatusInactive: true,
				StatusArchived: true,
			}
			if !validStatuses[status] {
				return fmt.Errorf("invalid status: %s (must be one of: pending, active, inactive, archived)", status)
			}
			return nil
		}),
	)
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
		if errs[0].Error() != "Status: invalid status: invalid (must be one of: pending, active, inactive, archived)" {
			t.Errorf("unexpected error: %v", errs[0])
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
