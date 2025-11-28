package errors

import (
	"errors"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      ValidationError
		expected string
	}{
		{"with path", ValidationError{Loc: []string{"User", "Email"}, Message: "invalid"}, "User.Email: invalid"},
		{"empty path", ValidationError{Loc: []string{}, Message: "invalid"}, "invalid"},
		{"single path", ValidationError{Loc: []string{"Name"}, Message: "required"}, "Name: required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name     string
		errs     ValidationErrors
		expected string
	}{
		{"empty", ValidationErrors{}, "validation errors: (none)"},
		{"single", ValidationErrors{{Loc: []string{"Email"}, Message: "invalid"}}, "Email: invalid"},
		{
			"multiple",
			ValidationErrors{
				{Loc: []string{"Name"}, Message: "required"},
				{Loc: []string{"Age"}, Message: "invalid"},
			},
			"validation errors (2): Name: required; Age: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.errs.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestValidationErrors_Unwrap(t *testing.T) {
	errs := ValidationErrors{
		{Loc: []string{"A"}, Message: "err1"},
		{Loc: []string{"B"}, Message: "err2"},
	}
	unwrapped := errs.Unwrap()
	if len(unwrapped) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(unwrapped))
	}
}

func TestValidationErrors_HasJSONDecodeError(t *testing.T) {
	tests := []struct {
		name     string
		errs     ValidationErrors
		expected bool
	}{
		{"empty", ValidationErrors{}, false},
		{"has decode error", ValidationErrors{{Type: ErrorTypeJSONDecode}}, true},
		{"no decode error", ValidationErrors{{Type: ErrorTypeConstraint}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.errs.HasJSONDecodeError(); got != tt.expected {
				t.Errorf("HasJSONDecodeError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidationErrors_ErrorsAs(t *testing.T) {
	errs := ValidationErrors{{Loc: []string{"X"}, Message: "test"}}
	var target ValidationErrors
	if !errors.As(errs, &target) {
		t.Error("errors.As should succeed")
	}
}
