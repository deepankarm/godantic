// Package walk provides a generic tree walker for struct traversal.
package walk

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/deepankarm/godantic/pkg/internal/errors"
	"github.com/deepankarm/godantic/pkg/internal/reflectutil"
)

// FieldContext provides context for processing a single field during tree walk.
type FieldContext struct {
	// Path is the location in the struct tree, e.g., ["User", "Address", "ZipCode"]
	Path []string

	// StructField is the reflect.StructField metadata (nil for root)
	StructField *reflect.StructField

	// Value is the reflect.Value of this field
	Value reflect.Value

	// RawJSON is the raw JSON for this field (nil if not unmarshaling or field not in JSON)
	RawJSON json.RawMessage

	// FieldOptions contains validation options from Field{Name}() method (nil if none)
	FieldOptions *FieldOptions

	// IsRoot is true for the root struct being walked
	IsRoot bool
}

// FieldOptions holds validation info extracted from Field{Name}() methods.
// This is a simplified view for the walker - validators are stored separately.
type FieldOptions struct {
	Required    bool
	Constraints map[string]any
	Validators  []func(any) error
}

// Processor handles fields during tree walk.
type Processor interface {
	// ProcessField is called for each struct field (and root).
	// Return non-nil error to stop traversal immediately.
	// To collect errors without stopping, store them internally and return nil.
	ProcessField(ctx *FieldContext) error

	// GetErrors returns any validation errors collected by this processor.
	GetErrors() []ValidationError
}

// DescentController optionally controls whether to descend into nested structs.
// If a Processor doesn't implement this, walker descends into all non-basic structs.
type DescentController interface {
	// ShouldDescend returns true if walker should recurse into this field's value.
	ShouldDescend(ctx *FieldContext) bool
}

// Walker traverses struct trees with pluggable processors.
type Walker struct {
	processors []Processor
	scanner    FieldScanner
	visited    map[uintptr]bool // Track visited pointers to prevent cycles
}

// FieldScanner scans types for field options. Allows dependency injection for testing.
type FieldScanner interface {
	ScanFieldOptions(t reflect.Type) map[string]*FieldOptions
}

// NewWalker creates a walker with the given processors.
func NewWalker(scanner FieldScanner, processors ...Processor) *Walker {
	return &Walker{
		processors: processors,
		scanner:    scanner,
	}
}

// Walk traverses a struct value, calling processors for each field.
// val should be the struct value (not pointer). data is optional raw JSON.
func (w *Walker) Walk(val reflect.Value, data []byte) error {
	// Reset visited map for each walk
	w.visited = make(map[uintptr]bool)

	// Parse JSON once at root if provided
	var rawFields map[string]json.RawMessage
	var jsonParseErr error
	if len(data) > 0 {
		jsonParseErr = json.Unmarshal(data, &rawFields)
	}

	// Check if any processor needs JSON (UnmarshalProcessor does)
	hasUnmarshalProcessor := false
	for _, p := range w.processors {
		if _, ok := p.(*UnmarshalProcessor); ok {
			hasUnmarshalProcessor = true
			break
		}
	}

	// If we have an unmarshal processor and JSON parsing failed, report error
	if hasUnmarshalProcessor && jsonParseErr != nil {
		if up, ok := w.processors[0].(*UnmarshalProcessor); ok {
			up.Errors = append(up.Errors, ValidationError{
				Loc:     []string{},
				Message: "json unmarshal failed: " + jsonParseErr.Error(),
				Type:    errors.ErrorTypeJSONDecode,
			})
		}
		return nil // Don't continue with invalid JSON
	}

	return w.walkStruct(val, rawFields, []string{}, true)
}

// walkStruct walks a struct value and its fields.
func (w *Walker) walkStruct(val reflect.Value, rawFields map[string]json.RawMessage, path []string, isRoot bool) error {
	// Unwrap pointers/interfaces and check for cycles
	for val.Kind() == reflect.Pointer || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return nil
		}
		// Track pointer address to detect cycles (only for pointers, not interfaces)
		if val.Kind() == reflect.Pointer {
			ptr := val.Pointer()
			if w.visited[ptr] {
				return nil // Already visited this address, skip to prevent infinite recursion
			}
			w.visited[ptr] = true
		}
		val = val.Elem()
	}

	// Must be a struct
	if val.Kind() != reflect.Struct {
		return nil
	}

	t := val.Type()

	// Process root first
	if isRoot {
		rootCtx := &FieldContext{
			Path:   path,
			Value:  val,
			IsRoot: true,
		}
		for _, p := range w.processors {
			if err := p.ProcessField(rootCtx); err != nil {
				return err
			}
		}
	}

	// Scan field options for this type
	fieldOpts := w.scanner.ScanFieldOptions(t)

	// Process each field
	for i := 0; i < t.NumField(); i++ {
		structField := t.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !structField.IsExported() {
			continue
		}

		// Get JSON field name
		jsonName := reflectutil.JSONFieldName(structField)
		if jsonName == "-" {
			continue // Skip ignored fields
		}

		// Build context - lookup RawJSON with case-insensitive fallback (like json.Unmarshal)
		fieldPath := appendPath(path, structField.Name)
		ctx := &FieldContext{
			Path:         fieldPath,
			StructField:  &structField,
			Value:        fieldVal,
			RawJSON:      lookupRawField(rawFields, jsonName, structField.Name),
			FieldOptions: fieldOpts[structField.Name],
			IsRoot:       false,
		}

		// Run all processors
		for _, p := range w.processors {
			if err := p.ProcessField(ctx); err != nil {
				return err
			}
		}

		// Check if we should descend
		if w.shouldDescend(ctx) {
			// Handle slices
			if fieldVal.Kind() == reflect.Slice {
				if err := w.walkSlice(fieldVal, ctx.RawJSON, fieldPath); err != nil {
					return err
				}
			} else {
				// Nested struct - parse its JSON if available
				var nestedRaw map[string]json.RawMessage
				if len(ctx.RawJSON) > 0 {
					json.Unmarshal(ctx.RawJSON, &nestedRaw)
				}
				if err := w.walkStruct(fieldVal, nestedRaw, fieldPath, false); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// walkSlice walks each element of a slice.
func (w *Walker) walkSlice(slice reflect.Value, rawJSON json.RawMessage, path []string) error {
	slice = reflectutil.UnwrapValue(slice)
	if slice.Kind() != reflect.Slice || slice.IsNil() {
		return nil
	}

	// Check if elements are worth walking (structs or interfaces)
	if !reflectutil.IsWalkableSliceElem(slice.Type()) {
		return nil
	}

	// Parse slice JSON if available
	var rawElements []json.RawMessage
	if len(rawJSON) > 0 {
		json.Unmarshal(rawJSON, &rawElements)
	}

	// Walk each element
	for i := 0; i < slice.Len(); i++ {
		elemPath := appendPathIndex(path, i)
		elemVal := slice.Index(i)

		var elemRaw json.RawMessage
		if i < len(rawElements) {
			elemRaw = rawElements[i]
		}

		var rawFields map[string]json.RawMessage
		if len(elemRaw) > 0 {
			json.Unmarshal(elemRaw, &rawFields)
		}

		if err := w.walkStruct(elemVal, rawFields, elemPath, false); err != nil {
			return err
		}
	}

	return nil
}

// shouldDescend checks if we should recurse into this field.
func (w *Walker) shouldDescend(ctx *FieldContext) bool {
	// Check if any processor wants to control descent
	for _, p := range w.processors {
		if dc, ok := p.(DescentController); ok {
			return dc.ShouldDescend(ctx)
		}
	}

	// Default: descend into non-basic struct types
	val := reflectutil.UnwrapValue(ctx.Value)
	if val.Kind() == reflect.Slice {
		return true // Let walkSlice decide
	}
	if val.Kind() != reflect.Struct {
		return false
	}
	return !reflectutil.IsBasicType(val.Type())
}

// Errors collects all errors from all processors.
func (w *Walker) Errors() []ValidationError {
	var errs []ValidationError
	for _, p := range w.processors {
		errs = append(errs, p.GetErrors()...)
	}
	return errs
}

// appendPath appends a field name to the path.
func appendPath(path []string, name string) []string {
	result := make([]string, len(path)+1)
	copy(result, path)
	result[len(path)] = name
	return result
}

// appendPathIndex appends an array index to the path.
func appendPathIndex(path []string, index int) []string {
	result := make([]string, len(path)+1)
	copy(result, path)
	result[len(path)] = "[" + strconv.Itoa(index) + "]"
	return result
}

// lookupRawField looks up a field in rawFields with case-insensitive fallback.
// This mimics json.Unmarshal's behavior: exact match first, then case-insensitive.
func lookupRawField(rawFields map[string]json.RawMessage, jsonName, fieldName string) json.RawMessage {
	if rawFields == nil {
		return nil
	}

	// Try exact JSON tag name first
	if raw, ok := rawFields[jsonName]; ok {
		return raw
	}

	// Try exact field name (for fields without json tag)
	if raw, ok := rawFields[fieldName]; ok {
		return raw
	}

	// Case-insensitive fallback (like json.Unmarshal)
	lowerJSON := strings.ToLower(jsonName)
	lowerField := strings.ToLower(fieldName)
	for key, raw := range rawFields {
		lowerKey := strings.ToLower(key)
		if lowerKey == lowerJSON || lowerKey == lowerField {
			return raw
		}
	}

	return nil
}
