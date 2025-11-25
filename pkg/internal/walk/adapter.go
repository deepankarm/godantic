package walk

import (
	"reflect"
)

// FieldOptionExtractor is a function that extracts field options from a type.
// This allows the walker to work with different field option implementations.
type FieldOptionExtractor func(t reflect.Type) map[string]*FieldOptions

// AdapterScanner wraps a FieldOptionExtractor to implement FieldScanner.
type AdapterScanner struct {
	extract FieldOptionExtractor
}

// NewAdapterScanner creates a scanner from an extraction function.
func NewAdapterScanner(extract FieldOptionExtractor) *AdapterScanner {
	return &AdapterScanner{extract: extract}
}

// ScanFieldOptions implements FieldScanner.
func (s *AdapterScanner) ScanFieldOptions(t reflect.Type) map[string]*FieldOptions {
	return s.extract(t)
}
