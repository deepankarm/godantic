package schema

import (
	"testing"

	"github.com/invopop/jsonschema"
)

func TestFindActualSchema(t *testing.T) {
	tests := []struct {
		name         string
		schema       *jsonschema.Schema
		wantOriginal bool
		wantFromDefs bool
	}{
		{
			name:         "no definitions returns original",
			schema:       &jsonschema.Schema{Type: "object"},
			wantOriginal: true,
		},
		{
			name:         "with definitions returns definition",
			schema:       &jsonschema.Schema{Definitions: map[string]*jsonschema.Schema{"User": {Type: "object"}}},
			wantFromDefs: true,
		},
		{
			name:         "empty schema returns original",
			schema:       &jsonschema.Schema{},
			wantOriginal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findActualSchema(tt.schema)
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if tt.wantOriginal && got != tt.schema {
				t.Error("expected original schema")
			}
			if tt.wantFromDefs {
				found := false
				for _, def := range tt.schema.Definitions {
					if got == def {
						found = true
						break
					}
				}
				if !found {
					t.Error("expected schema from definitions")
				}
			}
		})
	}
}
