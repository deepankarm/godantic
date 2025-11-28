package reflectutil

import (
	"reflect"
	"testing"
)

type testTagStruct struct {
	UserName string `json:"username"`
	Email    string `json:"email,omitempty"`
	Age      int
	Internal string `json:"-"`
}

type testEmbedded struct {
	EmbeddedField string `json:"embedded_field"`
}

type testWithEmbedded struct {
	Name string `json:"name"`
	testEmbedded
}

func TestJSONFieldName(t *testing.T) {
	tests := []struct {
		name     string
		field    reflect.StructField
		expected string
	}{
		{"no tag", reflect.StructField{Name: "UserName", Type: reflect.TypeOf("")}, "UserName"},
		{"simple tag", reflect.StructField{Name: "X", Type: reflect.TypeOf(""), Tag: `json:"x"`}, "x"},
		{"with omitempty", reflect.StructField{Name: "X", Type: reflect.TypeOf(""), Tag: `json:"x,omitempty"`}, "x"},
		{"ignored", reflect.StructField{Name: "X", Type: reflect.TypeOf(""), Tag: `json:"-"`}, "-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JSONFieldName(tt.field); got != tt.expected {
				t.Errorf("JSONFieldName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFieldByJSONName(t *testing.T) {
	val := reflect.ValueOf(testTagStruct{UserName: "john", Email: "j@x.com", Age: 25})
	typ := reflect.TypeOf(testTagStruct{})

	tests := []struct {
		name      string
		jsonName  string
		wantValid bool
		wantValue any
	}{
		{"by json tag", "username", true, "john"},
		{"by field name", "Age", true, 25},
		{"by lowercase (capitalized)", "age", true, 25},
		{"not found", "nonexistent", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FieldByJSONName(val, typ, tt.jsonName)
			if got.IsValid() != tt.wantValid {
				t.Errorf("valid = %v, want %v", got.IsValid(), tt.wantValid)
				return
			}
			if tt.wantValid && got.Interface() != tt.wantValue {
				t.Errorf("value = %v, want %v", got.Interface(), tt.wantValue)
			}
		})
	}
}

func TestFieldByJSONName_Pointer(t *testing.T) {
	val := &testTagStruct{UserName: "test"}
	field := FieldByJSONName(reflect.ValueOf(val), reflect.TypeOf(val), "username")
	if !field.IsValid() || field.String() != "test" {
		t.Error("expected to find field through pointer")
	}
}

func TestFieldByJSONName_NilPointer(t *testing.T) {
	var val *testTagStruct
	field := FieldByJSONName(reflect.ValueOf(val), reflect.TypeOf(val), "username")
	if field.IsValid() {
		t.Error("expected invalid field for nil pointer")
	}
}

func TestGoFieldToJSONName(t *testing.T) {
	tests := []struct {
		name        string
		typ         reflect.Type
		goFieldName string
		expected    string
	}{
		{"with json tag", reflect.TypeOf(testTagStruct{}), "UserName", "username"},
		{"no json tag", reflect.TypeOf(testTagStruct{}), "Age", "Age"},
		{"not found", reflect.TypeOf(testTagStruct{}), "Unknown", "Unknown"},
		{"embedded", reflect.TypeOf(testWithEmbedded{}), "EmbeddedField", "embedded_field"},
		{"pointer type", reflect.TypeOf(&testTagStruct{}), "UserName", "username"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GoFieldToJSONName(tt.typ, tt.goFieldName); got != tt.expected {
				t.Errorf("GoFieldToJSONName() = %q, want %q", got, tt.expected)
			}
		})
	}
}
