package walk

import (
	"reflect"
	"testing"

	"github.com/deepankarm/godantic/pkg/internal/errors"
)

func TestUnmarshalProcessor_ProcessField(t *testing.T) {
	tests := []struct {
		name       string
		ctx        *FieldContext
		wantErrors int
	}{
		{"skips root", &FieldContext{IsRoot: true}, 0},
		{"skips empty JSON", &FieldContext{IsRoot: false, RawJSON: []byte{}}, 0},
		{"skips non-settable", &FieldContext{IsRoot: false, RawJSON: []byte(`"x"`), Value: reflect.ValueOf("test")}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewUnmarshalProcessor()
			_ = p.ProcessField(tt.ctx)
			if len(p.Errors) != tt.wantErrors {
				t.Errorf("got %d errors, want %d", len(p.Errors), tt.wantErrors)
			}
		})
	}
}

func TestUnmarshalProcessor_UnmarshalRegular(t *testing.T) {
	tests := []struct {
		name     string
		rawJSON  []byte
		initial  any
		expected any
	}{
		{"string", []byte(`"hello"`), "", "hello"},
		{"int", []byte(`42`), 0, 42},
		{"bool", []byte(`true`), false, true},
		{"slice", []byte(`[1,2,3]`), []int{}, []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewUnmarshalProcessor()
			val := reflect.New(reflect.TypeOf(tt.initial)).Elem()
			ctx := &FieldContext{Path: []string{"field"}, RawJSON: tt.rawJSON, Value: val}
			_ = p.unmarshalRegular(ctx)
			if len(p.Errors) != 0 {
				t.Fatalf("unexpected errors: %v", p.Errors)
			}
			if !reflect.DeepEqual(val.Interface(), tt.expected) {
				t.Errorf("got %v, want %v", val.Interface(), tt.expected)
			}
		})
	}
}

func TestUnmarshalProcessor_UnmarshalRegular_Error(t *testing.T) {
	p := NewUnmarshalProcessor()
	var value int
	ctx := &FieldContext{Path: []string{"field"}, RawJSON: []byte(`"not int"`), Value: reflect.ValueOf(&value).Elem()}
	_ = p.unmarshalRegular(ctx)
	if len(p.Errors) != 1 || p.Errors[0].Type != errors.ErrorTypeJSONDecode {
		t.Errorf("expected JSON decode error, got %v", p.Errors)
	}
}

func TestUnmarshalProcessor_UnmarshalDiscriminatedSingle(t *testing.T) {
	mapping := map[string]any{"a": testEvent{}, "b": testEventB{}}

	tests := []struct {
		name       string
		rawJSON    []byte
		wantErrors int
		errorType  errors.ErrorType
	}{
		{"success", []byte(`{"Type":"a","Data":"test"}`), 0, ""},
		{"missing discriminator", []byte(`{"Data":"test"}`), 1, errors.ErrorTypeDiscriminatorMissing},
		{"invalid discriminator", []byte(`{"Type":"invalid"}`), 1, errors.ErrorTypeDiscriminatorInvalid},
		{"invalid JSON", []byte(`{invalid}`), 1, errors.ErrorTypeJSONDecode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewUnmarshalProcessor()
			var value any
			ctx := &FieldContext{Path: []string{"event"}, RawJSON: tt.rawJSON, Value: reflect.ValueOf(&value).Elem()}
			_ = p.unmarshalDiscriminatedSingle(ctx, "Type", mapping)
			if len(p.Errors) != tt.wantErrors {
				t.Fatalf("got %d errors, want %d: %v", len(p.Errors), tt.wantErrors, p.Errors)
			}
			if tt.wantErrors > 0 && p.Errors[0].Type != tt.errorType {
				t.Errorf("error type = %v, want %v", p.Errors[0].Type, tt.errorType)
			}
		})
	}
}

func TestUnmarshalProcessor_UnmarshalDiscriminatedSlice(t *testing.T) {
	mapping := map[string]any{"a": testEvent{}, "b": testEvent{}}

	tests := []struct {
		name         string
		rawJSON      []byte
		wantErrors   int
		wantElements int
	}{
		{"success", []byte(`[{"Type":"a"},{"Type":"b"}]`), 0, 2},
		{"invalid JSON", []byte(`[invalid]`), 1, 0},
		{"mixed valid/invalid", []byte(`[{"Type":"a"},{"Type":"invalid"},{"Data":"x"}]`), 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewUnmarshalProcessor()
			value := make([]any, 0)
			ctx := &FieldContext{Path: []string{"events"}, RawJSON: tt.rawJSON, Value: reflect.ValueOf(&value).Elem()}
			_ = p.unmarshalDiscriminatedSlice(ctx, "Type", mapping)
			if len(p.Errors) != tt.wantErrors {
				t.Fatalf("got %d errors, want %d: %v", len(p.Errors), tt.wantErrors, p.Errors)
			}
			if len(value) != tt.wantElements {
				t.Errorf("got %d elements, want %d", len(value), tt.wantElements)
			}
		})
	}
}

func TestUnmarshalProcessor_UnmarshalDiscriminated_Fallback(t *testing.T) {
	p := NewUnmarshalProcessor()
	var value any
	ctx := &FieldContext{Path: []string{"field"}, RawJSON: []byte(`{"x":1}`), Value: reflect.ValueOf(&value).Elem()}
	// Missing propertyName should fall back to regular unmarshal
	constraint := map[string]any{"mapping": map[string]any{"a": struct{}{}}}
	_ = p.unmarshalDiscriminated(ctx, constraint)
	if len(p.Errors) != 0 {
		t.Errorf("expected fallback to succeed, got %v", p.Errors)
	}
}

func TestUnmarshalProcessor_UnmarshalDiscriminatedSingle_PointerMapping(t *testing.T) {
	p := NewUnmarshalProcessor()
	var value any
	ctx := &FieldContext{Path: []string{"event"}, RawJSON: []byte(`{"Type":"a"}`), Value: reflect.ValueOf(&value).Elem()}
	mapping := map[string]any{"a": &testEvent{}} // Pointer in mapping
	_ = p.unmarshalDiscriminatedSingle(ctx, "Type", mapping)
	if len(p.Errors) != 0 {
		t.Errorf("unexpected errors: %v", p.Errors)
	}
}

func TestUnmarshalProcessor_ShouldDescend(t *testing.T) {
	type CustomStruct struct{ Name string }
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"slice of structs", []CustomStruct{}, true},
		{"slice of strings", []string{}, false},
		{"struct", CustomStruct{}, true},
		{"primitive", 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewUnmarshalProcessor()
			ctx := &FieldContext{Value: reflect.ValueOf(tt.value)}
			if got := p.ShouldDescend(ctx); got != tt.expected {
				t.Errorf("ShouldDescend() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUnmarshalProcessor_GetErrors(t *testing.T) {
	p := NewUnmarshalProcessor()
	p.Errors = append(p.Errors, ValidationError{Message: "test"})
	if errs := p.GetErrors(); len(errs) != 1 {
		t.Errorf("GetErrors() returned %d errors", len(errs))
	}
}
