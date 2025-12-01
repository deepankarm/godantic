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
	if t.Kind() == reflect.Pointer {
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

func TestWalker_RootSlice(t *testing.T) {
	scanner := &mockScanner{
		options: map[string]map[string]*FieldOptions{
			"testUser": {
				"Name": {Required: true},
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

	t.Run("unmarshal root slice with validation", func(t *testing.T) {
		jsonData := []byte(`[{"Name":"Alice","Email":"alice@example.com","Age":30},{"Name":"Bob","Email":"bob@example.com","Age":25}]`)

		slice := make([]testUser, 0)
		sliceVal := reflect.ValueOf(&slice).Elem()

		up := NewUnmarshalProcessor()
		vp := NewValidateProcessor()
		walker := NewWalker(scanner, up, vp)

		err := walker.Walk(sliceVal, jsonData)
		if err != nil {
			t.Fatalf("Walk returned error: %v", err)
		}

		if len(up.Errors) > 0 {
			t.Fatalf("got unmarshal errors: %v", up.Errors)
		}
		if len(vp.Errors) > 0 {
			t.Fatalf("got validation errors: %v", vp.Errors)
		}

		if len(slice) != 2 {
			t.Fatalf("got %d elements, want 2", len(slice))
		}
		if slice[0].Name != "Alice" {
			t.Errorf("got first name %q, want 'Alice'", slice[0].Name)
		}
		if slice[1].Age != 25 {
			t.Errorf("got second age %d, want 25", slice[1].Age)
		}
	})

	t.Run("root slice validation errors have correct paths", func(t *testing.T) {
		jsonData := []byte(`[{"Email":"alice@example.com","Age":30},{"Name":"Bob","Email":"bob@example.com","Age":200}]`)

		slice := make([]testUser, 0)
		sliceVal := reflect.ValueOf(&slice).Elem()

		up := NewUnmarshalProcessor()
		vp := NewValidateProcessor()
		walker := NewWalker(scanner, up, vp)

		err := walker.Walk(sliceVal, jsonData)
		if err != nil {
			t.Fatalf("Walk returned error: %v", err)
		}

		if len(vp.Errors) != 2 {
			t.Fatalf("got %d validation errors, want 2: %v", len(vp.Errors), vp.Errors)
		}

		expectedPaths := map[string]bool{"[0].Name": true, "[1].Age": true}
		for _, verr := range vp.Errors {
			path := ""
			if len(verr.Loc) > 0 {
				path = verr.Loc[0]
				if len(verr.Loc) > 1 {
					path += "." + verr.Loc[1]
				}
			}
			if !expectedPaths[path] {
				t.Errorf("unexpected error path: %v", verr.Loc)
			}
		}
	})

	t.Run("root slice with pointer elements", func(t *testing.T) {
		jsonData := []byte(`[{"Name":"Alice","Email":"alice@example.com","Age":30},{"Name":"Bob","Email":"bob@example.com","Age":25}]`)

		slice := make([]*testUser, 0)
		sliceVal := reflect.ValueOf(&slice).Elem()

		up := NewUnmarshalProcessor()
		vp := NewValidateProcessor()
		walker := NewWalker(scanner, up, vp)

		err := walker.Walk(sliceVal, jsonData)
		if err != nil {
			t.Fatalf("Walk returned error: %v", err)
		}

		if len(slice) != 2 {
			t.Fatalf("got %d elements, want 2", len(slice))
		}
		if slice[0] == nil {
			t.Fatal("got nil first element, want non-nil")
		}
		if slice[0].Name != "Alice" {
			t.Errorf("got first name %q, want 'Alice'", slice[0].Name)
		}
	})

	t.Run("empty root slice", func(t *testing.T) {
		jsonData := []byte(`[]`)

		slice := make([]testUser, 0)
		sliceVal := reflect.ValueOf(&slice).Elem()

		up := NewUnmarshalProcessor()
		walker := NewWalker(scanner, up)

		err := walker.Walk(sliceVal, jsonData)
		if err != nil {
			t.Fatalf("Walk returned error: %v", err)
		}

		if len(slice) != 0 {
			t.Errorf("got %d elements, want 0", len(slice))
		}
	})

	t.Run("root slice with primitive types", func(t *testing.T) {
		jsonData := []byte(`["apple", "banana", "cherry"]`)

		slice := make([]string, 0)
		sliceVal := reflect.ValueOf(&slice).Elem()

		up := NewUnmarshalProcessor()
		walker := NewWalker(scanner, up)

		err := walker.Walk(sliceVal, jsonData)
		if err != nil {
			t.Fatalf("Walk returned error: %v", err)
		}

		if len(slice) != 3 {
			t.Errorf("got %d elements, want 3", len(slice))
		}
		if slice[0] != "apple" {
			t.Errorf("got first element %q, want 'apple'", slice[0])
		}
	})

	t.Run("root slice invalid JSON", func(t *testing.T) {
		jsonData := []byte(`not valid json`)

		slice := make([]testUser, 0)
		sliceVal := reflect.ValueOf(&slice).Elem()

		up := NewUnmarshalProcessor()
		walker := NewWalker(scanner, up)

		err := walker.Walk(sliceVal, jsonData)
		if err != nil {
			t.Fatalf("Walk returned error: %v", err)
		}

		if len(up.Errors) == 0 {
			t.Fatal("want JSON decode error, got none")
		}
		if up.Errors[0].Type != "json_decode" {
			t.Errorf("got error type %q, want 'json_decode'", up.Errors[0].Type)
		}
	})

	t.Run("root slice without unmarshal processor", func(t *testing.T) {
		jsonData := []byte(`[{"Name":"Alice","Email":"alice@example.com","Age":30}]`)

		slice := make([]testUser, 0)
		sliceVal := reflect.ValueOf(&slice).Elem()

		vp := NewValidateProcessor()
		walker := NewWalker(scanner, vp)

		err := walker.Walk(sliceVal, jsonData)
		if err != nil {
			t.Fatalf("Walk returned error: %v", err)
		}
	})
}
