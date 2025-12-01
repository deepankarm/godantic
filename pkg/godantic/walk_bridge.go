package godantic

import (
	"reflect"
	"sync"

	"github.com/deepankarm/godantic/pkg/internal/partialjson"
	"github.com/deepankarm/godantic/pkg/internal/reflectutil"
	"github.com/deepankarm/godantic/pkg/internal/walk"
)

// walkScanner adapts godantic's field scanning to the walker's interface.
// It caches results to avoid repeated reflection calls.
type walkScanner struct {
	cache sync.Map // map[reflect.Type]map[string]*walk.FieldOptions
}

// ScanFieldOptions implements walk.FieldScanner with caching.
func (s *walkScanner) ScanFieldOptions(t reflect.Type) map[string]*walk.FieldOptions {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Check cache first
	if cached, ok := s.cache.Load(t); ok {
		return cached.(map[string]*walk.FieldOptions)
	}

	// Use existing scanner
	internalOpts := scanner.scanFieldOptionsFromType(t)

	// Convert to walk.FieldOptions
	result := make(map[string]*walk.FieldOptions, len(internalOpts))
	for fieldName, holder := range internalOpts {
		result[fieldName] = &walk.FieldOptions{
			Required:    holder.required,
			Constraints: holder.constraints,
			Validators:  holder.validators,
		}
	}

	// Cache the result
	s.cache.Store(t, result)
	return result
}

// cachedScanner is the shared scanner instance with caching.
var cachedScanner = &walkScanner{}

// walkValidate runs validation processors on a struct.
func walkValidate(objPtr reflect.Value) ValidationErrors {
	w := walk.NewWalker(cachedScanner,
		walk.NewValidateProcessor(),
		walk.NewUnionValidateProcessor(),
	)
	if err := w.Walk(objPtr.Elem(), nil); err != nil {
		return ValidationErrors{{Loc: []string{}, Message: err.Error(), Type: ErrorTypeInternal}}
	}
	return w.Errors()
}

// walkDefaults applies default values to zero fields.
func walkDefaults(objPtr reflect.Value) error {
	w := walk.NewWalker(cachedScanner, walk.NewDefaultsProcessor())
	return w.Walk(objPtr.Elem(), nil)
}

// walkParse unmarshals JSON, applies defaults, and validates.
func walkParse(objPtr reflect.Value, data []byte) ValidationErrors {
	w := walk.NewWalker(cachedScanner,
		walk.NewUnmarshalProcessor(),
		walk.NewDefaultsProcessor(),
		walk.NewValidateProcessor(),
		walk.NewUnionValidateProcessor(),
	)
	if err := w.Walk(objPtr.Elem(), data); err != nil {
		return ValidationErrors{{Loc: []string{}, Message: err.Error(), Type: ErrorTypeInternal}}
	}
	return w.Errors()
}

// prefixErrors prepends a path segment to all error locations.
func prefixErrors(errs ValidationErrors, prefix string) ValidationErrors {
	result := make(ValidationErrors, len(errs))
	for i, e := range errs {
		result[i] = ValidationError{
			Loc:     append([]string{prefix}, e.Loc...),
			Message: e.Message,
			Type:    e.Type,
		}
	}
	return result
}

// walkParsePartial unmarshals potentially incomplete JSON, applies defaults, and validates.
// Returns the result with incomplete field paths tracked.
func walkParsePartial(objPtr reflect.Value, data []byte) (*PartialUnmarshalResult, ValidationErrors) {
	// First parse to get incomplete paths
	parser := partialjson.NewParser(false)
	parseResult, err := parser.Parse(data)
	if err != nil {
		return nil, ValidationErrors{{Loc: []string{}, Message: err.Error(), Type: ErrorTypeJSONDecode}}
	}

	// Use normal processors - we'll filter validation errors after
	unmarshalProcessor := walk.NewUnmarshalProcessor()
	defaultsProcessor := walk.NewDefaultsProcessor()
	validateProcessor := walk.NewValidateProcessor()
	unionValidateProcessor := walk.NewUnionValidateProcessor()

	w := walk.NewWalker(cachedScanner,
		unmarshalProcessor,
		defaultsProcessor,
		validateProcessor,
		unionValidateProcessor,
	)

	// Walk with repaired JSON
	if err := w.Walk(objPtr.Elem(), parseResult.Repaired); err != nil {
		return nil, ValidationErrors{{Loc: []string{}, Message: err.Error(), Type: ErrorTypeInternal}}
	}

	// Filter out validation errors for incomplete fields using actual JSON tags
	typ := objPtr.Elem().Type()
	validationErrors := filterIncompleteFieldErrors(validateProcessor.GetErrors(), parseResult.Incomplete, typ)

	return &PartialUnmarshalResult{
		Value:           objPtr.Elem(),
		IncompletePaths: parseResult.Incomplete,
		TruncatedAt:     parseResult.TruncatedAt,
		Errors:          unmarshalProcessor.GetErrors(),
	}, validationErrors
}

// filterIncompleteFieldErrors removes validation errors for fields that are incomplete.
// Uses the struct type to properly map Go field names to JSON field names.
func filterIncompleteFieldErrors(errs []walk.ValidationError, incompletePaths [][]string, typ reflect.Type) ValidationErrors {
	if len(incompletePaths) == 0 {
		// Fast path: nothing incomplete, keep all errors
		result := make(ValidationErrors, len(errs))
		for i, e := range errs {
			result[i] = ValidationError{Loc: e.Loc, Message: e.Message, Type: ErrorType(e.Type)}
		}
		return result
	}

	// Build set of incomplete JSON paths using partialjson utility
	incompleteSet := partialjson.BuildIncompleteSet(incompletePaths)

	var filtered ValidationErrors
	for _, e := range errs {
		// Convert struct path to JSON path using actual JSON tags
		jsonPath := structPathToJSONPath(e.Loc, typ)
		if !partialjson.IsPathOrParentIncomplete(jsonPath, incompleteSet) {
			filtered = append(filtered, ValidationError{
				Loc:     e.Loc,
				Message: e.Message,
				Type:    ErrorType(e.Type),
			})
		}
	}
	return filtered
}

// structPathToJSONPath converts struct field path to JSON path using actual JSON tags.
// Uses reflectutil.GoFieldToJSONName for proper tag lookup.
func structPathToJSONPath(structPath []string, typ reflect.Type) string {
	if len(structPath) == 0 {
		return ""
	}

	// Unwrap pointer types
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	var result string
	currentType := typ

	for i, fieldName := range structPath {
		// Handle array indices
		if len(fieldName) > 0 && fieldName[0] == '[' {
			if result != "" {
				result += fieldName
			} else {
				result = fieldName
			}
			// For array elements, try to get element type
			if currentType.Kind() == reflect.Slice || currentType.Kind() == reflect.Array {
				currentType = currentType.Elem()
			}
			continue
		}

		// Get JSON name from struct tag
		jsonName := reflectutil.GoFieldToJSONName(currentType, fieldName)

		if i == 0 {
			result = jsonName
		} else {
			result += "." + jsonName
		}

		// Update current type for nested fields
		if currentType.Kind() == reflect.Struct {
			if field, ok := currentType.FieldByName(fieldName); ok {
				currentType = field.Type
				if currentType.Kind() == reflect.Pointer {
					currentType = currentType.Elem()
				}
			}
		}
	}

	return result
}
