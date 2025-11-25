package walk

import (
	"fmt"
	"reflect"
	"testing"
)

// Test structs
type testUser struct {
	Name  string
	Email string
	Age   int
}

type testAddress struct {
	Street string
	City   string
	Zip    string
}

type testUserWithAddress struct {
	Name    string
	Email   string
	Address testAddress
}

type testUserWithSlice struct {
	Name      string
	Addresses []testAddress
}

// mockScanner returns predefined field options for testing
type mockScanner struct {
	options map[string]map[string]*FieldOptions // typeName -> fieldName -> options
}

func (m *mockScanner) ScanFieldOptions(t reflect.Type) map[string]*FieldOptions {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if opts, ok := m.options[t.Name()]; ok {
		return opts
	}
	return nil
}

func TestWalker_BasicValidation(t *testing.T) {
	scanner := &mockScanner{
		options: map[string]map[string]*FieldOptions{
			"testUser": {
				"Name": {
					Required: true,
				},
				"Email": {
					Required: true,
					Validators: []func(any) error{
						func(v any) error {
							s := v.(string)
							if len(s) < 5 {
								return fmt.Errorf("email too short")
							}
							return nil
						},
					},
				},
				"Age": {
					Validators: []func(any) error{
						func(v any) error {
							age := v.(int)
							if age < 0 || age > 150 {
								return fmt.Errorf("age must be between 0 and 150")
							}
							return nil
						},
					},
				},
			},
		},
	}

	vp := NewValidateProcessor()
	walker := NewWalker(scanner, vp)

	tests := []struct {
		name       string
		user       testUser
		wantErrors int
	}{
		{
			name:       "valid user",
			user:       testUser{Name: "John", Email: "john@example.com", Age: 30},
			wantErrors: 0,
		},
		{
			name:       "missing required name",
			user:       testUser{Email: "john@example.com", Age: 30},
			wantErrors: 1,
		},
		{
			name:       "missing required fields",
			user:       testUser{},
			wantErrors: 2, // Name and Email
		},
		{
			name:       "invalid age",
			user:       testUser{Name: "John", Email: "john@example.com", Age: 200},
			wantErrors: 1,
		},
		{
			name:       "email too short",
			user:       testUser{Name: "John", Email: "a@b", Age: 30},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vp.Errors = nil // Reset errors
			err := walker.Walk(reflect.ValueOf(tt.user), nil)
			if err != nil {
				t.Fatalf("Walk returned error: %v", err)
			}
			if len(vp.Errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d. Errors: %v", len(vp.Errors), tt.wantErrors, vp.Errors)
			}
		})
	}
}

func TestWalker_NestedStruct(t *testing.T) {
	scanner := &mockScanner{
		options: map[string]map[string]*FieldOptions{
			"testUserWithAddress": {
				"Name": {Required: true},
			},
			"testAddress": {
				"City": {Required: true},
				"Zip": {
					Required: true,
					Validators: []func(any) error{
						func(v any) error {
							s := v.(string)
							if len(s) != 5 {
								return fmt.Errorf("zip must be 5 characters")
							}
							return nil
						},
					},
				},
			},
		},
	}

	vp := NewValidateProcessor()
	walker := NewWalker(scanner, vp)

	tests := []struct {
		name       string
		user       testUserWithAddress
		wantErrors int
	}{
		{
			name: "valid nested",
			user: testUserWithAddress{
				Name:    "John",
				Address: testAddress{City: "NYC", Zip: "12345"},
			},
			wantErrors: 0,
		},
		{
			name: "missing nested city",
			user: testUserWithAddress{
				Name:    "John",
				Address: testAddress{Zip: "12345"},
			},
			wantErrors: 1,
		},
		{
			name: "invalid nested zip",
			user: testUserWithAddress{
				Name:    "John",
				Address: testAddress{City: "NYC", Zip: "123"},
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vp.Errors = nil
			err := walker.Walk(reflect.ValueOf(tt.user), nil)
			if err != nil {
				t.Fatalf("Walk returned error: %v", err)
			}
			if len(vp.Errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d. Errors: %v", len(vp.Errors), tt.wantErrors, vp.Errors)
			}
		})
	}
}

func TestWalker_SliceOfStructs(t *testing.T) {
	scanner := &mockScanner{
		options: map[string]map[string]*FieldOptions{
			"testUserWithSlice": {
				"Name": {Required: true},
			},
			"testAddress": {
				"City": {Required: true},
			},
		},
	}

	vp := NewValidateProcessor()
	walker := NewWalker(scanner, vp)

	tests := []struct {
		name       string
		user       testUserWithSlice
		wantErrors int
	}{
		{
			name: "valid slice",
			user: testUserWithSlice{
				Name: "John",
				Addresses: []testAddress{
					{City: "NYC"},
					{City: "LA"},
				},
			},
			wantErrors: 0,
		},
		{
			name: "invalid element in slice",
			user: testUserWithSlice{
				Name: "John",
				Addresses: []testAddress{
					{City: "NYC"},
					{}, // Missing City
				},
			},
			wantErrors: 1,
		},
		{
			name: "multiple invalid elements",
			user: testUserWithSlice{
				Name: "John",
				Addresses: []testAddress{
					{}, // Missing City
					{}, // Missing City
				},
			},
			wantErrors: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vp.Errors = nil
			err := walker.Walk(reflect.ValueOf(tt.user), nil)
			if err != nil {
				t.Fatalf("Walk returned error: %v", err)
			}
			if len(vp.Errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d. Errors: %v", len(vp.Errors), tt.wantErrors, vp.Errors)
			}
		})
	}
}

func TestDefaultsProcessor(t *testing.T) {
	scanner := &mockScanner{
		options: map[string]map[string]*FieldOptions{
			"testUser": {
				"Name": {
					Constraints: map[string]any{"default": "Anonymous"},
				},
				"Age": {
					Constraints: map[string]any{"default": 18},
				},
			},
		},
	}

	dp := NewDefaultsProcessor()
	walker := NewWalker(scanner, dp)

	user := testUser{}
	err := walker.Walk(reflect.ValueOf(&user).Elem(), nil)
	if err != nil {
		t.Fatalf("Walk returned error: %v", err)
	}

	if user.Name != "Anonymous" {
		t.Errorf("Name = %q, want %q", user.Name, "Anonymous")
	}
	if user.Age != 18 {
		t.Errorf("Age = %d, want %d", user.Age, 18)
	}
}

func TestUnmarshalProcessor_Basic(t *testing.T) {
	type SimpleStruct struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	jsonData := []byte(`{"name":"test","count":42}`)

	// Create struct to populate
	var obj SimpleStruct
	objPtr := reflect.ValueOf(&obj)

	// Create walker with unmarshal processor
	scanner := &mockScanner{options: map[string]map[string]*FieldOptions{}}
	up := NewUnmarshalProcessor()
	walker := NewWalker(scanner, up)

	err := walker.Walk(objPtr.Elem(), jsonData)
	if err != nil {
		t.Fatalf("Walk returned error: %v", err)
	}

	if len(up.Errors) > 0 {
		t.Fatalf("Unmarshal had errors: %v", up.Errors)
	}

	if obj.Name != "test" {
		t.Errorf("Expected Name='test', got '%s'", obj.Name)
	}
	if obj.Count != 42 {
		t.Errorf("Expected Count=42, got %d", obj.Count)
	}
}

func TestDefaultsProcessor_DoesNotOverrideExisting(t *testing.T) {
	scanner := &mockScanner{
		options: map[string]map[string]*FieldOptions{
			"testUser": {
				"Name": {
					Constraints: map[string]any{"default": "Anonymous"},
				},
			},
		},
	}

	dp := NewDefaultsProcessor()
	walker := NewWalker(scanner, dp)

	user := testUser{Name: "John"}
	err := walker.Walk(reflect.ValueOf(&user).Elem(), nil)
	if err != nil {
		t.Fatalf("Walk returned error: %v", err)
	}

	if user.Name != "John" {
		t.Errorf("Name = %q, want %q (should not override)", user.Name, "John")
	}
}
