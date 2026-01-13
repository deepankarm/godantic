package schema

import (
	"github.com/invopop/jsonschema"
)

// findActualSchema finds the actual schema definition (might be in $defs)
func findActualSchema(schema *jsonschema.Schema) *jsonschema.Schema {
	if len(schema.Definitions) > 0 {
		// Get first definition
		for _, def := range schema.Definitions {
			return def
		}
	}
	return schema
}

// deepCopyMap creates a deep copy of a map[string]any to avoid modifying originals
func deepCopyMap(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case map[string]any:
			result[k] = deepCopyMap(val)
		case []any:
			result[k] = deepCopySlice(val)
		case []map[string]any:
			arr := make([]any, len(val))
			for i, item := range val {
				arr[i] = deepCopyMap(item)
			}
			result[k] = arr
		default:
			result[k] = v
		}
	}
	return result
}

// deepCopySlice creates a deep copy of a []any slice
func deepCopySlice(s []any) []any {
	result := make([]any, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]any:
			result[i] = deepCopyMap(val)
		case []any:
			result[i] = deepCopySlice(val)
		default:
			result[i] = v
		}
	}
	return result
}
