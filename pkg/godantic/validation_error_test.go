package godantic_test

import (
	"strings"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Test structs for nested validation
type Location struct {
	Street  string
	City    string
	ZipCode string
}

func (a *Location) FieldStreet() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(3),
	)
}

func (a *Location) FieldCity() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(2),
	)
}

func (a *Location) FieldZipCode() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Regex(`^[0-9]{5}$`),
	)
}

type Business struct {
	Name     string
	Location Location
}

func (c *Business) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
	)
}

func (c *Business) FieldLocation() godantic.FieldOptions[Location] {
	return godantic.Field(
		godantic.Required[Location](),
	)
}

func TestValidationErrorPaths(t *testing.T) {
	t.Run("top-level field error has simple path", func(t *testing.T) {
		validator := godantic.NewValidator[Business]()

		company := Business{
			// Missing required Name
			Location: Location{
				Street:  "123 Main St",
				City:    "Boston",
				ZipCode: "02101",
			},
		}

		errs := validator.Validate(&company)
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}

		err := errs[0]
		if len(err.Loc) != 1 || err.Loc[0] != "Name" {
			t.Errorf("expected Loc=[Name], got %v", err.Loc)
		}
		if err.Message != "required field" {
			t.Errorf("expected 'required field', got '%s'", err.Message)
		}
		if err.Type != "required" {
			t.Errorf("expected 'required', got '%s'", err.Type)
		}

		// Check Error() string representation
		errStr := err.Error()
		if errStr != "Name: required field" {
			t.Errorf("expected 'Name: required field', got '%s'", errStr)
		}
	})

	t.Run("nested field error has full path", func(t *testing.T) {
		validator := godantic.NewValidator[Business]()

		company := Business{
			Name: "Acme Corp",
			Location: Location{
				Street: "123 Main St",
				City:   "Boston",
				// Missing required ZipCode
			},
		}

		errs := validator.Validate(&company)
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}

		err := errs[0]
		if len(err.Loc) != 2 || err.Loc[0] != "Location" || err.Loc[1] != "ZipCode" {
			t.Errorf("expected Loc=[Location, ZipCode], got %v", err.Loc)
		}

		// Check Error() string representation shows full path
		errStr := err.Error()
		if errStr != "Location.ZipCode: required field" {
			t.Errorf("expected 'Location.ZipCode: required field', got '%s'", errStr)
		}
	})

	t.Run("multiple nested errors show distinct paths", func(t *testing.T) {
		validator := godantic.NewValidator[Business]()

		company := Business{
			Name: "Acme Corp",
			Location: Location{
				Street:  "12",  // Too short
				City:    "B",   // Too short
				ZipCode: "abc", // Invalid format
			},
		}

		errs := validator.Validate(&company)
		if len(errs) != 3 {
			t.Fatalf("expected 3 errors, got %d: %v", len(errs), errs)
		}

		// Check that each error has the correct path
		pathsFound := make(map[string]bool)
		for _, err := range errs {
			if len(err.Loc) != 2 {
				t.Errorf("expected path length 2, got %d: %v", len(err.Loc), err.Loc)
			}
			if err.Loc[0] != "Location" {
				t.Errorf("expected first element 'Location', got '%s'", err.Loc[0])
			}

			pathStr := strings.Join(err.Loc, ".")
			pathsFound[pathStr] = true

			// Verify string representation
			if !strings.HasPrefix(err.Error(), pathStr+":") {
				t.Errorf("error string should start with path: %s", err.Error())
			}
		}

		// Should have errors for all three fields
		expectedPaths := []string{"Location.Street", "Location.City", "Location.ZipCode"}
		for _, path := range expectedPaths {
			if !pathsFound[path] {
				t.Errorf("expected error for path '%s'", path)
			}
		}
	})

	t.Run("validation error type indicates error category", func(t *testing.T) {
		validator := godantic.NewValidator[Business]()

		company := Business{
			// Missing Name
			Location: Location{
				Street:  "123 Main St",
				City:    "Boston",
				ZipCode: "abc", // Wrong format
			},
		}

		errs := validator.Validate(&company)
		if len(errs) < 2 {
			t.Fatalf("expected at least 2 errors, got %d", len(errs))
		}

		// Find the missing field error
		var missingErr *godantic.ValidationError
		var formatErr *godantic.ValidationError
		for i := range errs {
			if strings.Contains(errs[i].Message, "required") {
				missingErr = &errs[i]
			}
			if strings.Contains(errs[i].Message, "match") || strings.Contains(errs[i].Message, "pattern") {
				formatErr = &errs[i]
			}
		}

		if missingErr != nil && missingErr.Type != "required" {
			t.Errorf("missing field should have type 'required', got '%s'", missingErr.Type)
		}

		if formatErr != nil && formatErr.Type != "constraint" {
			t.Errorf("format error should have type 'constraint', got '%s'", formatErr.Type)
		}
	})
}

type Person struct {
	Name     string
	Location *Location
}

func (p *Person) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
	)
}

func (p *Person) FieldLocation() godantic.FieldOptions[*Location] {
	return godantic.Field[*Location]()
}

func TestValidationErrorWithPointers(t *testing.T) {
	validator := godantic.NewValidator[Person]()

	t.Run("nil pointer doesn't cause nested validation", func(t *testing.T) {
		person := Person{
			Name:     "John",
			Location: nil,
		}

		// Should not panic or try to validate nil pointer
		errs := validator.Validate(&person)
		// No errors expected for nil pointer (it's not required in this case)
		if len(errs) != 0 {
			t.Errorf("expected no errors for nil pointer, got %d: %v", len(errs), errs)
		}
	})

	t.Run("non-nil pointer validates nested struct with path", func(t *testing.T) {
		person := Person{
			Name: "John",
			Location: &Location{
				Street: "123 Main St",
				City:   "Boston",
				// Missing ZipCode
			},
		}

		errs := validator.Validate(&person)
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}

		// Path should include Location even though it's a pointer
		err := errs[0]
		if len(err.Loc) != 2 || err.Loc[0] != "Location" || err.Loc[1] != "ZipCode" {
			t.Errorf("expected Loc=[Location, ZipCode], got %v", err.Loc)
		}
	})
}
