package godantic_test

import (
	"strings"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Test regex validation
type UserProfile struct {
	Username    string
	Email       string
	PhoneNumber string
	Website     string
}

func (u *UserProfile) FieldUsername() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(3),
		godantic.MaxLen(20),
		godantic.Regex(`^[a-zA-Z0-9_]+$`),
	)
}

func (u *UserProfile) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Regex(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
	)
}

func (u *UserProfile) FieldPhoneNumber() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Regex(`^\+?[1-9]\d{1,14}$`),
	)
}

func (u *UserProfile) FieldWebsite() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Regex(`^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*)?$`),
	)
}

func TestRegexValidation(t *testing.T) {
	validator := godantic.NewValidator[UserProfile]()

	t.Run("valid profile should pass", func(t *testing.T) {
		profile := UserProfile{
			Username:    "john_doe123",
			Email:       "john@example.com",
			PhoneNumber: "+12125551234",
			Website:     "https://example.com",
		}
		errs := validator.Validate(&profile)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("invalid username with special chars should fail", func(t *testing.T) {
		profile := UserProfile{
			Username:    "john-doe!", // contains - and !
			Email:       "john@example.com",
			PhoneNumber: "+12125551234",
			Website:     "https://example.com",
		}
		errs := validator.Validate(&profile)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if !strings.Contains(errs[0].Error(), "value does not match pattern") {
			t.Errorf("expected error to contain 'value does not match pattern', got %v", errs[0])
		}
	})

	t.Run("invalid email should fail", func(t *testing.T) {
		profile := UserProfile{
			Username:    "john_doe",
			Email:       "not-an-email",
			PhoneNumber: "+12125551234",
			Website:     "https://example.com",
		}
		errs := validator.Validate(&profile)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if !strings.Contains(errs[0].Error(), "value does not match pattern") {
			t.Errorf("expected error to contain 'value does not match pattern', got %v", errs[0])
		}
	})

	t.Run("invalid phone number should fail", func(t *testing.T) {
		profile := UserProfile{
			Username:    "john_doe",
			Email:       "john@example.com",
			PhoneNumber: "123-456-7890", // contains dashes
			Website:     "https://example.com",
		}
		errs := validator.Validate(&profile)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if !strings.Contains(errs[0].Error(), "value does not match pattern") {
			t.Errorf("expected error to contain 'value does not match pattern', got %v", errs[0])
		}
	})

	t.Run("invalid website should fail", func(t *testing.T) {
		profile := UserProfile{
			Username:    "john_doe",
			Email:       "john@example.com",
			PhoneNumber: "+12125551234",
			Website:     "not-a-url",
		}
		errs := validator.Validate(&profile)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if !strings.Contains(errs[0].Error(), "value does not match pattern") {
			t.Errorf("expected error to contain 'value does not match pattern', got %v", errs[0])
		}
	})

	t.Run("multiple validation errors", func(t *testing.T) {
		profile := UserProfile{
			Username: "jo!", // too short and has special char
			Email:    "bad-email",
			Website:  "not-a-url",
		}
		errs := validator.Validate(&profile)
		if len(errs) < 3 {
			t.Errorf("expected at least 3 errors, got %d: %v", len(errs), errs)
		}
		for _, err := range errs {
			if !strings.Contains(err.Error(), "value does not match pattern") && !strings.Contains(err.Error(), "length must be") {
				t.Errorf("unexpected error message: %v", err)
			}
		}
	})
}
