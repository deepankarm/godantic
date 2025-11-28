package walk

import (
	"reflect"
	"testing"

	"github.com/deepankarm/godantic/pkg/internal/errors"
)

// Test types for discriminated unions
type testEvent struct {
	Type string
	Data string
}

type testEventB struct {
	Type  string
	Count int
}

func TestUnionValidateProcessor_ProcessField(t *testing.T) {
	tests := []struct {
		name       string
		ctx        *FieldContext
		wantErrors int
	}{
		{
			name:       "skips root",
			ctx:        &FieldContext{IsRoot: true},
			wantErrors: 0,
		},
		{
			name:       "skips nil field options",
			ctx:        &FieldContext{IsRoot: false, FieldOptions: nil, Value: reflect.ValueOf("test")},
			wantErrors: 0,
		},
		{
			name: "skips zero values",
			ctx: &FieldContext{
				IsRoot:       false,
				FieldOptions: &FieldOptions{Constraints: map[string]any{"anyOf": []map[string]string{{"type": "string"}}}},
				Value:        reflect.ValueOf(""),
			},
			wantErrors: 0,
		},
		{
			name: "valid anyOf string",
			ctx: &FieldContext{
				IsRoot: false,
				Path:   []string{"field"},
				FieldOptions: &FieldOptions{
					Constraints: map[string]any{"anyOf": []map[string]string{{"type": "string"}}},
				},
				Value: reflect.ValueOf("hello"),
			},
			wantErrors: 0,
		},
		{
			name: "valid discriminator",
			ctx: &FieldContext{
				IsRoot: false,
				Path:   []string{"event"},
				FieldOptions: &FieldOptions{
					Constraints: map[string]any{
						"discriminator": map[string]any{
							"propertyName": "Type",
							"mapping":      map[string]any{"a": testEvent{}},
						},
					},
				},
				Value: reflect.ValueOf(testEvent{Type: "a", Data: "test"}),
			},
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewUnionValidateProcessor()
			_ = p.ProcessField(tt.ctx)
			if len(p.Errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d", len(p.Errors), tt.wantErrors)
			}
		})
	}
}

func TestUnionValidateProcessor_ValidateAnyOf(t *testing.T) {
	tests := []struct {
		name      string
		ctx       *FieldContext
		wantError bool
	}{
		{
			name: "string matches string type",
			ctx: &FieldContext{
				Path:         []string{"field"},
				FieldOptions: &FieldOptions{Constraints: map[string]any{"anyOf": []map[string]string{{"type": "string"}}}},
				Value:        reflect.ValueOf("hello"),
			},
			wantError: false,
		},
		{
			name: "int matches integer type",
			ctx: &FieldContext{
				Path:         []string{"field"},
				FieldOptions: &FieldOptions{Constraints: map[string]any{"anyOf": []map[string]string{{"type": "integer"}}}},
				Value:        reflect.ValueOf(42),
			},
			wantError: false,
		},
		{
			name: "int matches number type",
			ctx: &FieldContext{
				Path:         []string{"field"},
				FieldOptions: &FieldOptions{Constraints: map[string]any{"anyOf": []map[string]string{{"type": "number"}}}},
				Value:        reflect.ValueOf(42),
			},
			wantError: false,
		},
		{
			name: "type mismatch",
			ctx: &FieldContext{
				Path:         []string{"field"},
				FieldOptions: &FieldOptions{Constraints: map[string]any{"anyOf": []map[string]string{{"type": "integer"}}}},
				Value:        reflect.ValueOf("not an int"),
			},
			wantError: true,
		},
		{
			name: "no constraints passes",
			ctx: &FieldContext{
				Path:         []string{"field"},
				FieldOptions: &FieldOptions{Constraints: map[string]any{}},
				Value:        reflect.ValueOf("anything"),
			},
			wantError: false,
		},
		{
			name: "complex type match",
			ctx: &FieldContext{
				Path:         []string{"field"},
				FieldOptions: &FieldOptions{Constraints: map[string]any{"anyOfTypes": []any{testEvent{}}}},
				Value:        reflect.ValueOf(testEvent{Type: "a"}),
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewUnionValidateProcessor()
			err := p.validateAnyOf(tt.ctx)
			if (err != nil) != tt.wantError {
				t.Errorf("validateAnyOf() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestUnionValidateProcessor_ValidateDiscriminator(t *testing.T) {
	mapping := map[string]any{"a": testEvent{}, "b": testEventB{}}
	constraint := map[string]any{"propertyName": "Type", "mapping": mapping}

	tests := []struct {
		name       string
		value      any
		constraint map[string]any
		wantError  bool
	}{
		{"valid discriminator", testEvent{Type: "a", Data: "test"}, constraint, false},
		{"invalid discriminator value", testEvent{Type: "invalid"}, constraint, true},
		{"valid slice", []testEvent{{Type: "a"}, {Type: "a"}}, constraint, false},
		{"slice with invalid element", []testEvent{{Type: "a"}, {Type: "invalid"}}, constraint, true},
		{"empty constraint", testEvent{Type: "a"}, map[string]any{}, false},
		{"missing propertyName", testEvent{Type: "a"}, map[string]any{"mapping": mapping}, false},
		{"missing mapping", testEvent{Type: "a"}, map[string]any{"propertyName": "Type"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewUnionValidateProcessor()
			ctx := &FieldContext{Path: []string{"event"}, Value: reflect.ValueOf(tt.value)}
			err := p.validateDiscriminator(ctx, tt.constraint)
			if (err != nil) != tt.wantError {
				t.Errorf("validateDiscriminator() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestUnionValidateProcessor_ValidateSingleDiscriminator(t *testing.T) {
	mapping := map[string]any{"a": testEvent{}}

	tests := []struct {
		name      string
		value     any
		wantError bool
		checkMsg  string
	}{
		{"valid struct", testEvent{Type: "a"}, false, ""},
		{"pointer to struct", &testEvent{Type: "a"}, false, ""},
		{"nil pointer", (*testEvent)(nil), true, ""},
		{"non-struct", "not a struct", true, "discriminated union requires a struct type"},
		{"missing field", struct{ Data string }{Data: "test"}, true, ""},
		{"invalid value", testEvent{Type: "invalid"}, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewUnionValidateProcessor()
			err := p.validateSingleDiscriminator(reflect.ValueOf(tt.value), "Type", mapping, []string{"event"})
			if (err != nil) != tt.wantError {
				t.Errorf("error = %v, wantError %v", err, tt.wantError)
			}
			if tt.checkMsg != "" && err != nil && err.Message != tt.checkMsg {
				t.Errorf("message = %q, want %q", err.Message, tt.checkMsg)
			}
		})
	}
}

func TestUnionValidateProcessor_ShouldDescend(t *testing.T) {
	p := NewUnionValidateProcessor()
	if p.ShouldDescend(&FieldContext{}) {
		t.Error("union processor should not descend")
	}
}

func TestUnionValidateProcessor_GetErrors(t *testing.T) {
	p := NewUnionValidateProcessor()
	p.Errors = append(p.Errors, ValidationError{Message: "test", Type: errors.ErrorTypeConstraint})
	if errs := p.GetErrors(); len(errs) != 1 || errs[0].Message != "test" {
		t.Errorf("GetErrors() = %v", errs)
	}
}

func TestIsZero(t *testing.T) {
	tests := []struct {
		value    any
		expected bool
	}{
		{0, true},
		{42, false},
		{"", true},
		{"hello", false},
		{(*int)(nil), true},
		{false, true},
		{true, false},
	}

	for _, tt := range tests {
		if got := isZero(reflect.ValueOf(tt.value)); got != tt.expected {
			t.Errorf("isZero(%v) = %v, want %v", tt.value, got, tt.expected)
		}
	}
}
