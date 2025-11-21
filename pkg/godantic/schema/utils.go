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
