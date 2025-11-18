package godantic_test

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

type User struct {
	Email    string
	Age      int
	Username string
}

func (u *User) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Description[string]("User email address"),
		godantic.Example("user@example.com"),
		godantic.Email(),
	)
}

func (u *User) FieldAge() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Description[int]("User age in years"),
		godantic.Min(0),
		godantic.Max(130),
	)
}

func (u *User) FieldUsername() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(3),
		godantic.MaxLen(20),
	)
}

func TestBasicValidation(t *testing.T) {
	validator := godantic.NewValidator[User]()

	t.Run("empty user should fail required fields", func(t *testing.T) {
		user := User{}
		errs := validator.Validate(&user)
		if len(errs) != 2 {
			t.Errorf("expected 2 errors, got %d", len(errs))
		}

		// Check specific errors
		errMap := make(map[string]bool)
		for _, err := range errs {
			errMap[err.Error()] = true
		}
		if !errMap["Email: required field"] {
			t.Error("expected Email required error")
		}
		if !errMap["Username: required field"] {
			t.Error("expected Username required error")
		}
	})

	t.Run("valid user should pass", func(t *testing.T) {
		user := User{
			Email:    "user@example.com",
			Username: "john_doe",
			Age:      25,
		}
		errs := validator.Validate(&user)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("invalid age should fail", func(t *testing.T) {
		user := User{
			Email:    "user@example.com",
			Username: "john",
			Age:      150,
		}
		errs := validator.Validate(&user)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Age: value must be <= 130" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("username too short should fail", func(t *testing.T) {
		user := User{
			Email:    "user@example.com",
			Username: "ab",
		}
		errs := validator.Validate(&user)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Username: length must be >= 3" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})
}

func TestMinMaxConstraints(t *testing.T) {
	t.Run("min constraint on int", func(t *testing.T) {
		validator := godantic.NewValidator[User]()
		user := User{
			Email:    "test@example.com",
			Username: "testuser",
			Age:      -1,
		}
		errs := validator.Validate(&user)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Age: value must be >= 0" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("max constraint on int", func(t *testing.T) {
		validator := godantic.NewValidator[User]()
		user := User{
			Email:    "test@example.com",
			Username: "testuser",
			Age:      200,
		}
		errs := validator.Validate(&user)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Age: value must be <= 130" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})
}

func TestStringLengthConstraints(t *testing.T) {
	t.Run("minlen constraint", func(t *testing.T) {
		validator := godantic.NewValidator[User]()
		user := User{
			Email:    "test@example.com",
			Username: "ab",
			Age:      25,
		}
		errs := validator.Validate(&user)
		if len(errs) != 1 {
			t.Errorf("expected 1 error for minlen, got %d", len(errs))
		}
		if errs[0].Error() != "Username: length must be >= 3" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("maxlen constraint", func(t *testing.T) {
		validator := godantic.NewValidator[User]()
		user := User{
			Email:    "test@example.com",
			Username: "this_is_a_very_long_username_that_exceeds_limit",
			Age:      25,
		}
		errs := validator.Validate(&user)
		if len(errs) != 1 {
			t.Errorf("expected 1 error for maxlen, got %d", len(errs))
		}
		if errs[0].Error() != "Username: length must be <= 20" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})
}
