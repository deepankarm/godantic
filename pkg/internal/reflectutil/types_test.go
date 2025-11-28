package reflectutil_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/deepankarm/godantic/pkg/internal/reflectutil"
)

type customStruct struct{ Name string }

func TestJSONSchemaType(t *testing.T) {
	tests := []struct {
		name     string
		typ      reflect.Type
		expected string
	}{
		{"nil", nil, ""},
		{"string", reflect.TypeOf(""), "string"},
		{"int", reflect.TypeOf(0), "integer"},
		{"float64", reflect.TypeOf(0.0), "number"},
		{"bool", reflect.TypeOf(true), "boolean"},
		{"slice", reflect.TypeOf([]int{}), "array"},
		{"map", reflect.TypeOf(map[string]int{}), "object"},
		{"struct", reflect.TypeOf(struct{}{}), "object"},
		{"pointer to string", reflect.TypeOf((*string)(nil)), "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reflectutil.JSONSchemaType(tt.typ); got != tt.expected {
				t.Errorf("JSONSchemaType() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMatchesJSONSchemaType(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		schemaType string
		expected   bool
	}{
		{"string matches string", "hello", "string", true},
		{"int matches integer", 42, "integer", true},
		{"int matches number", 42, "number", true},
		{"float matches number", 3.14, "number", true},
		{"string doesn't match integer", "x", "integer", false},
		{"nil pointer matches null", (*int)(nil), "null", true},
		{"non-nil doesn't match null", 42, "null", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reflectutil.MatchesJSONSchemaType(reflect.ValueOf(tt.value), tt.schemaType); got != tt.expected {
				t.Errorf("MatchesJSONSchemaType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsBasicType(t *testing.T) {
	tests := []struct {
		name     string
		typ      reflect.Type
		expected bool
	}{
		{"int", reflect.TypeOf(0), true},
		{"string", reflect.TypeOf(""), true},
		{"slice", reflect.TypeOf([]int{}), true},
		{"map", reflect.TypeOf(map[string]int{}), true},
		{"time.Time", reflect.TypeOf(time.Time{}), true},
		{"custom struct", reflect.TypeOf(customStruct{}), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reflectutil.IsBasicType(tt.typ); got != tt.expected {
				t.Errorf("IsBasicType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUnwrapPointer(t *testing.T) {
	tests := []struct {
		name     string
		typ      reflect.Type
		expected reflect.Kind
	}{
		{"pointer", reflect.TypeOf((*int)(nil)), reflect.Int},
		{"non-pointer", reflect.TypeOf(0), reflect.Int},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reflectutil.UnwrapPointer(tt.typ).Kind(); got != tt.expected {
				t.Errorf("UnwrapPointer().Kind() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUnwrapValue(t *testing.T) {
	t.Run("unwraps pointer", func(t *testing.T) {
		x := 42
		if got := reflectutil.UnwrapValue(reflect.ValueOf(&x)); got.Kind() != reflect.Int || got.Int() != 42 {
			t.Errorf("failed to unwrap pointer: %v", got)
		}
	})

	t.Run("returns nil for nil pointer", func(t *testing.T) {
		var ptr *int
		if got := reflectutil.UnwrapValue(reflect.ValueOf(ptr)); !got.IsNil() {
			t.Error("expected nil")
		}
	})

	t.Run("unchanged for non-pointer", func(t *testing.T) {
		if got := reflectutil.UnwrapValue(reflect.ValueOf(42)); got.Kind() != reflect.Int {
			t.Errorf("expected int, got %v", got.Kind())
		}
	})
}

func TestIsWalkableSliceElem(t *testing.T) {
	type myInterface interface{ Method() }

	tests := []struct {
		name     string
		slice    any
		expected bool
	}{
		{"strings", []string{}, false},
		{"custom struct", []customStruct{}, true},
		{"pointer to struct", []*customStruct{}, true},
		{"interface", []myInterface{}, true},
		{"time.Time", []time.Time{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reflectutil.IsWalkableSliceElem(reflect.TypeOf(tt.slice)); got != tt.expected {
				t.Errorf("IsWalkableSliceElem() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCollectStructTypes(t *testing.T) {
	type Inner struct{ Value string }
	type Outer struct {
		Inner Inner
		Items []Inner
	}

	t.Run("collects nested structs", func(t *testing.T) {
		types := make(map[string]reflect.Type)
		reflectutil.CollectStructTypes(reflect.TypeOf(Outer{}), types)
		if len(types) != 2 {
			t.Errorf("expected 2 types, got %d", len(types))
		}
		if _, ok := types["Outer"]; !ok {
			t.Error("missing Outer")
		}
		if _, ok := types["Inner"]; !ok {
			t.Error("missing Inner")
		}
	})

	t.Run("handles nil", func(t *testing.T) {
		types := make(map[string]reflect.Type)
		reflectutil.CollectStructTypes(nil, types)
		if len(types) != 0 {
			t.Error("expected empty map")
		}
	})

	t.Run("handles non-struct", func(t *testing.T) {
		types := make(map[string]reflect.Type)
		reflectutil.CollectStructTypes(reflect.TypeOf(42), types)
		if len(types) != 0 {
			t.Error("expected empty map")
		}
	})

	t.Run("prevents infinite recursion", func(t *testing.T) {
		type Node struct {
			Value string
			Next  *Node
		}
		types := make(map[string]reflect.Type)
		reflectutil.CollectStructTypes(reflect.TypeOf(Node{}), types)
		if len(types) != 1 {
			t.Errorf("expected 1 type, got %d", len(types))
		}
	})
}

func TestMakeAddressable(t *testing.T) {
	val := reflect.ValueOf(42)
	addr := reflectutil.MakeAddressable(val, reflect.TypeOf(42))
	if addr.Kind() != reflect.Ptr {
		t.Errorf("expected pointer, got %v", addr.Kind())
	}
	if addr.Elem().Int() != 42 {
		t.Errorf("expected 42, got %d", addr.Elem().Int())
	}
}

type testEvent interface{ GetType() string }
type concreteEvent struct{ Type string }

func (e concreteEvent) GetType() string { return e.Type }

func TestConvertToInterfaceType(t *testing.T) {
	t.Run("pointer type", func(t *testing.T) {
		concrete := &concreteEvent{Type: "test"}
		result := reflectutil.ConvertToInterfaceType[testEvent](reflect.ValueOf(concrete), reflect.TypeOf(concrete))
		if result.GetType() != "test" {
			t.Errorf("got %q, want %q", result.GetType(), "test")
		}
	})

	t.Run("value type", func(t *testing.T) {
		concrete := concreteEvent{Type: "test"}
		result := reflectutil.ConvertToInterfaceType[testEvent](reflect.ValueOf(&concrete), reflect.TypeOf(concrete))
		if result.GetType() != "test" {
			t.Errorf("got %q, want %q", result.GetType(), "test")
		}
	})
}
