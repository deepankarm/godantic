package walk

import (
	"reflect"
	"testing"
)

func TestAdapterScanner(t *testing.T) {
	type TestStruct struct {
		Name string
		Age  int
	}

	tests := []struct {
		name        string
		extractFunc FieldOptionExtractor
		checkResult func(t *testing.T, result map[string]*FieldOptions)
	}{
		{
			name: "returns field options from extract function",
			extractFunc: func(typ reflect.Type) map[string]*FieldOptions {
				if typ == reflect.TypeOf(TestStruct{}) {
					return map[string]*FieldOptions{
						"Name": {Required: true, Constraints: map[string]any{"minLength": 1}},
						"Age":  {Required: false, Constraints: map[string]any{"min": 0}},
					}
				}
				return nil
			},
			checkResult: func(t *testing.T, result map[string]*FieldOptions) {
				if result == nil || len(result) != 2 {
					t.Fatalf("expected 2 field options, got %v", result)
				}
				if !result["Name"].Required {
					t.Error("expected Name to be required")
				}
			},
		},
		{
			name:        "returns nil when extract returns nil",
			extractFunc: func(reflect.Type) map[string]*FieldOptions { return nil },
			checkResult: func(t *testing.T, result map[string]*FieldOptions) {
				if result != nil {
					t.Errorf("expected nil result, got %v", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewAdapterScanner(tt.extractFunc)
			result := scanner.ScanFieldOptions(reflect.TypeOf(TestStruct{}))
			tt.checkResult(t, result)
		})
	}
}
