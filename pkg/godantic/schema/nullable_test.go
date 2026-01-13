package schema_test

import (
	"encoding/json"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

// NullableResponse tests nullable fields matching Python's Optional[T] behavior
type NullableResponse struct {
	Message    string    `json:"message"`
	Citations  *[]string `json:"citations"`   // nullable
	QuickHints *[]string `json:"quick_hints"` // nullable
	Count      *int      `json:"count"`       // nullable
}

func (r *NullableResponse) FieldMessage() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Description[string]("The main response message"),
	)
}

func (r *NullableResponse) FieldCitations() godantic.FieldOptions[*[]string] {
	return godantic.Field(
		godantic.Nullable[*[]string](),
		godantic.Description[*[]string]("Optional list of citations"),
	)
}

func (r *NullableResponse) FieldQuickHints() godantic.FieldOptions[*[]string] {
	return godantic.Field(
		godantic.Nullable[*[]string](),
	)
}

func (r *NullableResponse) FieldCount() godantic.FieldOptions[*int] {
	return godantic.Field(
		godantic.Nullable[*int](),
		godantic.Description[*int]("Optional count"),
	)
}

func TestNullableConstraint(t *testing.T) {
	sg := schema.NewGenerator[NullableResponse]()
	s, err := sg.Generate()
	if err != nil {
		t.Fatalf("failed to generate schema: %v", err)
	}

	def, ok := s.Definitions["NullableResponse"]
	if !ok {
		t.Fatal("NullableResponse definition not found")
	}

	tests := []struct {
		name      string
		fieldName string
		wantType  string // expected type in first anyOf option (or direct type if not nullable)
		wantNull  bool   // whether field should have anyOf with null
	}{
		{
			name:      "nullable array field",
			fieldName: "citations",
			wantType:  "array",
			wantNull:  true,
		},
		{
			name:      "another nullable array field",
			fieldName: "quick_hints",
			wantType:  "array",
			wantNull:  true,
		},
		{
			name:      "nullable integer field",
			fieldName: "count",
			wantType:  "integer",
			wantNull:  true,
		},
		{
			name:      "non-nullable string field",
			fieldName: "message",
			wantType:  "string",
			wantNull:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prop, ok := def.Properties.Get(tt.fieldName)
			if !ok {
				t.Fatalf("%s property not found", tt.fieldName)
			}

			if tt.wantNull {
				if prop.AnyOf == nil || len(prop.AnyOf) != 2 {
					schemaJSON, _ := json.MarshalIndent(prop, "", "  ")
					t.Fatalf("expected anyOf with 2 options, got:\n%s", string(schemaJSON))
				}

				if prop.AnyOf[0].Type != tt.wantType {
					t.Errorf("expected first anyOf type '%s', got '%s'", tt.wantType, prop.AnyOf[0].Type)
				}

				if prop.AnyOf[1].Type != "null" {
					t.Errorf("expected second anyOf type 'null', got '%s'", prop.AnyOf[1].Type)
				}
			} else {
				if prop.AnyOf != nil {
					t.Errorf("expected no anyOf for non-nullable field")
				}
				if prop.Type != tt.wantType {
					t.Errorf("expected type '%s', got '%s'", tt.wantType, prop.Type)
				}
			}
		})
	}
}

func TestNullablePreservesMetadata(t *testing.T) {
	sg := schema.NewGenerator[NullableResponse]()
	s, err := sg.Generate()
	if err != nil {
		t.Fatalf("failed to generate schema: %v", err)
	}

	def := s.Definitions["NullableResponse"]

	tests := []struct {
		name      string
		fieldName string
		wantTitle string
		wantDesc  string
		descInner bool // true if description is on inner schema
	}{
		{
			name:      "nullable field preserves title",
			fieldName: "citations",
			wantTitle: "Citations",
		},
		{
			name:      "nullable field preserves description in inner schema",
			fieldName: "citations",
			wantDesc:  "Optional list of citations",
			descInner: true,
		},
		{
			name:      "nullable field without description still has title",
			fieldName: "quick_hints",
			wantTitle: "Quick Hints",
		},
		{
			name:      "non-nullable field preserves title",
			fieldName: "message",
			wantTitle: "Message",
		},
		{
			name:      "non-nullable field preserves description",
			fieldName: "message",
			wantDesc:  "The main response message",
			descInner: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prop, _ := def.Properties.Get(tt.fieldName)

			if tt.wantTitle != "" && prop.Title != tt.wantTitle {
				t.Errorf("expected title '%s', got '%s'", tt.wantTitle, prop.Title)
			}

			if tt.wantDesc != "" {
				desc := prop.Description
				if tt.descInner && prop.AnyOf != nil && len(prop.AnyOf) > 0 {
					desc = prop.AnyOf[0].Description
				}
				if desc != tt.wantDesc {
					t.Errorf("expected description '%s', got '%s'", tt.wantDesc, desc)
				}
			}
		})
	}
}

func TestNullableSchemaJSON(t *testing.T) {
	sg := schema.NewGenerator[NullableResponse]()
	s, err := sg.Generate()
	if err != nil {
		t.Fatalf("failed to generate schema: %v", err)
	}

	schemaJSON, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal schema: %v", err)
	}

	if testing.Verbose() {
		t.Logf("Generated schema:\n%s", string(schemaJSON))
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	defs := schemaMap["$defs"].(map[string]any)
	response := defs["NullableResponse"].(map[string]any)
	props := response["properties"].(map[string]any)

	tests := []struct {
		name      string
		fieldName string
		wantAnyOf bool
	}{
		{"citations has anyOf", "citations", true},
		{"quick_hints has anyOf", "quick_hints", true},
		{"count has anyOf", "count", true},
		{"message has no anyOf", "message", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := props[tt.fieldName].(map[string]any)
			anyOf, hasAnyOf := field["anyOf"].([]any)

			if hasAnyOf != tt.wantAnyOf {
				t.Errorf("expected anyOf=%v, got %v", tt.wantAnyOf, hasAnyOf)
			}

			if tt.wantAnyOf {
				if len(anyOf) != 2 {
					t.Errorf("expected 2 anyOf options, got %d", len(anyOf))
				}
				nullOpt := anyOf[1].(map[string]any)
				if nullOpt["type"] != "null" {
					t.Errorf("expected second option type 'null', got %v", nullOpt["type"])
				}
			}
		})
	}
}
