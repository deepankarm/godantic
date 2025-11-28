package godantic

import (
	"reflect"
	"testing"
)

// Test fixtures at package level
type testStructWithMethods struct {
	Name  string
	Email string
}

func (ts *testStructWithMethods) FieldName() FieldOptions[string] {
	return Field(Required[string]())
}

func (ts *testStructWithMethods) FieldEmail() FieldOptions[string] {
	return Field(Required[string]())
}

type testEmailType string

func (e testEmailType) FieldtestEmailType() FieldOptions[testEmailType] {
	return Field(Required[testEmailType]())
}

type testUserWithEmail struct {
	Email testEmailType
}

type testUserWithEmailOverride struct {
	Email testEmailType
}

func (u *testUserWithEmailOverride) FieldEmail() FieldOptions[testEmailType] {
	return Field[testEmailType]() // Not required - overrides type-level
}

func TestFieldScanner_ScanFieldOptionsFromType(t *testing.T) {
	tests := []struct {
		name           string
		typ            reflect.Type
		expectedFields []string
		checkRequired  map[string]bool
	}{
		{
			name:           "struct with Field methods",
			typ:            reflect.TypeOf(testStructWithMethods{}),
			expectedFields: []string{"Name", "Email"},
			checkRequired:  map[string]bool{"Name": true, "Email": true},
		},
		{
			name:           "type-level validation",
			typ:            reflect.TypeOf(testUserWithEmail{}),
			expectedFields: []string{"Email"},
			checkRequired:  map[string]bool{"Email": true},
		},
		{
			name:           "parent overrides type-level",
			typ:            reflect.TypeOf(testUserWithEmailOverride{}),
			expectedFields: []string{"Email"},
			checkRequired:  map[string]bool{"Email": false},
		},
		{
			name:           "struct without Field methods",
			typ:            reflect.TypeOf(struct{ Name string }{}),
			expectedFields: nil,
			checkRequired:  nil,
		},
	}

	scanner := &fieldScanner{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := scanner.scanFieldOptionsFromType(tt.typ)
			if opts == nil {
				t.Fatal("expected non-nil options")
			}
			if len(opts) != len(tt.expectedFields) {
				t.Errorf("got %d fields, want %d", len(opts), len(tt.expectedFields))
			}
			for field, wantRequired := range tt.checkRequired {
				if opt, ok := opts[field]; !ok {
					t.Errorf("missing field %s", field)
				} else if opt.required != wantRequired {
					t.Errorf("%s.required = %v, want %v", field, opt.required, wantRequired)
				}
			}
		})
	}
}

func TestFieldScanner_ExtractFieldOptions(t *testing.T) {
	scanner := &fieldScanner{}

	t.Run("extracts required", func(t *testing.T) {
		opts := Field(Required[string]())
		holder := scanner.extractFieldOptions(reflect.ValueOf(opts))
		if !holder.required {
			t.Error("expected required=true")
		}
	})

	t.Run("extracts validators", func(t *testing.T) {
		called := false
		opts := Field(Validate(func(s string) error {
			called = true
			return nil
		}))
		holder := scanner.extractFieldOptions(reflect.ValueOf(opts))
		if len(holder.validators) != 1 {
			t.Fatalf("expected 1 validator, got %d", len(holder.validators))
		}
		_ = holder.validators[0]("test")
		if !called {
			t.Error("validator was not called")
		}
	})

	t.Run("empty options", func(t *testing.T) {
		opts := Field[string]()
		holder := scanner.extractFieldOptions(reflect.ValueOf(opts))
		if holder.required || len(holder.validators) != 0 {
			t.Error("expected empty holder")
		}
	})
}

func TestFieldOptionHolder_ToPublic(t *testing.T) {
	holder := &fieldOptionHolder{
		required:    true,
		constraints: map[string]any{"min": 1},
	}
	public := holder.toPublic()
	if !public.Required {
		t.Error("expected Required=true")
	}
	if public.Constraints["min"] != 1 {
		t.Error("constraints not copied")
	}
}

func TestScanTypeFieldOptions(t *testing.T) {
	t.Run("struct type", func(t *testing.T) {
		result := ScanTypeFieldOptions(reflect.TypeOf(testStructWithMethods{}))
		if !result["Name"].Required {
			t.Error("expected Name.Required=true")
		}
	})

	t.Run("pointer type", func(t *testing.T) {
		result := ScanTypeFieldOptions(reflect.TypeOf(&testStructWithMethods{}))
		if !result["Name"].Required {
			t.Error("expected Name.Required=true for pointer type")
		}
	})

	t.Run("empty struct", func(t *testing.T) {
		result := ScanTypeFieldOptions(reflect.TypeOf(struct{}{}))
		if len(result) != 0 {
			t.Errorf("expected empty result, got %d fields", len(result))
		}
	})
}
