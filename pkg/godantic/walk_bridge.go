package godantic

import (
	"reflect"
	"sync"

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
