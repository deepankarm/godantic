package reflectutil_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/deepankarm/godantic/pkg/internal/reflectutil"
)

func TestIsWalkableSliceElem(t *testing.T) {
	type CustomStruct struct {
		Name string
	}

	type CustomInterface interface {
		Method()
	}

	tests := []struct {
		name     string
		slice    any
		expected bool
	}{
		{
			name:     "slice of strings - not walkable",
			slice:    []string{},
			expected: false,
		},
		{
			name:     "slice of ints - not walkable",
			slice:    []int{},
			expected: false,
		},
		{
			name:     "slice of custom struct - walkable",
			slice:    []CustomStruct{},
			expected: true,
		},
		{
			name:     "slice of pointer to struct - walkable",
			slice:    []*CustomStruct{},
			expected: true,
		},
		{
			name:     "slice of interface - walkable (discriminated unions)",
			slice:    []CustomInterface{},
			expected: true,
		},
		{
			name:     "slice of time.Time - not walkable (basic type)",
			slice:    []time.Time{},
			expected: false,
		},
		{
			name:     "slice of maps - not walkable",
			slice:    []map[string]any{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sliceType := reflect.TypeOf(tt.slice)
			result := reflectutil.IsWalkableSliceElem(sliceType)
			if result != tt.expected {
				t.Errorf("IsWalkableSliceElem(%v) = %v, want %v", sliceType, result, tt.expected)
			}
		})
	}
}

func TestUnwrapValue(t *testing.T) {
	t.Run("unwraps pointer", func(t *testing.T) {
		x := 42
		ptr := &x
		val := reflect.ValueOf(ptr)
		unwrapped := reflectutil.UnwrapValue(val)
		if unwrapped.Kind() != reflect.Int {
			t.Errorf("expected int, got %v", unwrapped.Kind())
		}
		if unwrapped.Int() != 42 {
			t.Errorf("expected 42, got %d", unwrapped.Int())
		}
	})

	t.Run("unwraps interface", func(t *testing.T) {
		var iface any = "hello"
		val := reflect.ValueOf(&iface).Elem()
		unwrapped := reflectutil.UnwrapValue(val)
		if unwrapped.Kind() != reflect.String {
			t.Errorf("expected string, got %v", unwrapped.Kind())
		}
		if unwrapped.String() != "hello" {
			t.Errorf("expected 'hello', got %s", unwrapped.String())
		}
	})

	t.Run("handles nil pointer", func(t *testing.T) {
		var ptr *int
		val := reflect.ValueOf(ptr)
		unwrapped := reflectutil.UnwrapValue(val)
		if !unwrapped.IsNil() {
			t.Errorf("expected nil, got %v", unwrapped)
		}
	})

	t.Run("non-pointer unchanged", func(t *testing.T) {
		x := 42
		val := reflect.ValueOf(x)
		unwrapped := reflectutil.UnwrapValue(val)
		if unwrapped.Kind() != reflect.Int {
			t.Errorf("expected int, got %v", unwrapped.Kind())
		}
	})
}
