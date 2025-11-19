package schema_test

import (
	"encoding/json"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

// Test Union (anyOf) with primitive types
type FlexibleConfig struct {
	Value any
}

func (c *FlexibleConfig) FieldValue() godantic.FieldOptions[any] {
	return godantic.Field(
		godantic.Required[any](),
		godantic.Union[any]("string", "integer", "object"),
		godantic.Description[any]("Can be a string, number, or configuration object"),
	)
}

func TestUnionSchema(t *testing.T) {
	sg := schema.NewGenerator[FlexibleConfig]()
	generatedSchema, err := sg.Generate()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	schemaJSON, err := json.MarshalIndent(generatedSchema, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	// Parse schema to verify structure
	var schemaMap map[string]any
	if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
		t.Fatalf("Failed to parse schema JSON: %v", err)
	}

	// Verify anyOf is present for Value field
	defs, ok := schemaMap["$defs"].(map[string]any)
	if !ok {
		t.Fatal("$defs not found in schema")
	}

	flexibleConfig, ok := defs["FlexibleConfig"].(map[string]any)
	if !ok {
		t.Fatal("FlexibleConfig not found in $defs")
	}

	properties, ok := flexibleConfig["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties not found in FlexibleConfig")
	}

	valueField, ok := properties["Value"].(map[string]any)
	if !ok {
		t.Fatal("Value field not found in properties")
	}

	anyOf, ok := valueField["anyOf"].([]any)
	if !ok {
		t.Fatal("anyOf not found in Value field")
	}

	if len(anyOf) != 3 {
		t.Errorf("Expected 3 types in anyOf, got %d", len(anyOf))
	}

	// Verify description is present
	if desc, ok := valueField["description"].(string); !ok || desc == "" {
		t.Error("Description not found or empty in Value field")
	}
}

// Test DiscriminatedUnion (oneOf with discriminator) - real-world API response example

type ResponseType interface {
	GetStatus() string
}

type SuccessResponse struct {
	Status string
	Data   map[string]string
}

func (s SuccessResponse) GetStatus() string { return s.Status }

type ErrorResponse struct {
	Status  string
	Message string
	Code    int
}

func (e ErrorResponse) GetStatus() string { return e.Status }

type PendingResponse struct {
	Status   string
	Progress int
}

func (p PendingResponse) GetStatus() string { return p.Status }

type APIResult struct {
	Response ResponseType
}

func (a *APIResult) FieldResponse() godantic.FieldOptions[ResponseType] {
	return godantic.Field(
		godantic.Required[ResponseType](),
		godantic.DiscriminatedUnion[ResponseType]("Status", map[string]any{
			"success": SuccessResponse{},
			"error":   ErrorResponse{},
			"pending": PendingResponse{},
		}),
		godantic.Description[ResponseType]("API response - structure depends on Status field"),
	)
}

// Test Union with complex types (slices of structs)
type TextInput struct {
	Type string
	Text string
}

type ImageInput struct {
	Type     string
	ImageURL string
}

type Payload struct {
	Query any
}

func (q *Payload) FieldQuery() godantic.FieldOptions[any] {
	return godantic.Field(
		godantic.Required[any](),
		godantic.Union[any]("", []TextInput{}, []ImageInput{}),
		godantic.Description[any]("Query can be a string or array of text/image inputs"),
	)
}

func TestUnionWithComplexTypes(t *testing.T) {
	sg := schema.NewGenerator[Payload]()
	generatedSchema, err := sg.Generate()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	schemaJSON, err := json.MarshalIndent(generatedSchema, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	// Parse schema to verify structure
	var schemaMap map[string]any
	if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
		t.Fatalf("Failed to parse schema JSON: %v", err)
	}

	// Verify anyOf is present for Query field
	defs, ok := schemaMap["$defs"].(map[string]any)
	if !ok {
		t.Fatal("$defs not found in schema")
	}

	payload, ok := defs["Payload"].(map[string]any)
	if !ok {
		t.Fatal("Payload not found in $defs")
	}

	properties, ok := payload["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties not found in Payload")
	}

	queryField, ok := properties["Query"].(map[string]any)
	if !ok {
		t.Fatal("Query field not found in properties")
	}

	anyOf, ok := queryField["anyOf"].([]any)
	if !ok {
		t.Fatal("anyOf not found in Query field")
	}

	if len(anyOf) != 3 {
		t.Errorf("Expected 3 types in anyOf (string, []TextInput, []ImageInput), got %d", len(anyOf))
	}

	// Verify TextInput and ImageInput definitions exist
	if _, ok := defs["TextInput"]; !ok {
		t.Error("TextInput definition not found in $defs")
	}

	if _, ok := defs["ImageInput"]; !ok {
		t.Error("ImageInput definition not found in $defs")
	}

	// Verify description is present
	if desc, ok := queryField["description"].(string); !ok || desc == "" {
		t.Error("Description not found or empty in Query field")
	}
}

// Test Union with mixed primitive and complex types
type MixedPayload struct {
	Data any
}

func (m *MixedPayload) FieldData() godantic.FieldOptions[any] {
	return godantic.Field(
		godantic.Required[any](),
		godantic.Union[any]("string", "integer", []TextInput{}),
		godantic.Description[any]("Data can be string, integer, or array of text inputs"),
	)
}

func TestUnionWithMixedTypes(t *testing.T) {
	sg := schema.NewGenerator[MixedPayload]()
	generatedSchema, err := sg.Generate()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	schemaJSON, err := json.MarshalIndent(generatedSchema, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	// Parse schema to verify structure
	var schemaMap map[string]any
	if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
		t.Fatalf("Failed to parse schema JSON: %v", err)
	}

	// Verify anyOf is present for Data field
	defs, ok := schemaMap["$defs"].(map[string]any)
	if !ok {
		t.Fatal("$defs not found in schema")
	}

	mixedPayload, ok := defs["MixedPayload"].(map[string]any)
	if !ok {
		t.Fatal("MixedPayload not found in $defs")
	}

	properties, ok := mixedPayload["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties not found in MixedPayload")
	}

	dataField, ok := properties["Data"].(map[string]any)
	if !ok {
		t.Fatal("Data field not found in properties")
	}

	anyOf, ok := dataField["anyOf"].([]any)
	if !ok {
		t.Fatal("anyOf not found in Data field")
	}

	if len(anyOf) != 3 {
		t.Errorf("Expected 3 types in anyOf (string, integer, []TextInput), got %d", len(anyOf))
	}

	// Verify first is string primitive
	firstType := anyOf[0].(map[string]any)
	if firstType["type"] != "string" {
		t.Errorf("Expected first type to be 'string', got %v", firstType["type"])
	}

	// Verify second is integer primitive
	secondType := anyOf[1].(map[string]any)
	if secondType["type"] != "integer" {
		t.Errorf("Expected second type to be 'integer', got %v", secondType["type"])
	}

	// Verify third is array with TextInput items
	thirdType := anyOf[2].(map[string]any)
	if thirdType["type"] != "array" {
		t.Errorf("Expected third type to be 'array', got %v", thirdType["type"])
	}

	// Verify TextInput definition exists
	if _, ok := defs["TextInput"]; !ok {
		t.Error("TextInput definition not found in $defs")
	}
}

func TestDiscriminatedUnionSchema(t *testing.T) {
	sg := schema.NewGenerator[APIResult]()
	generatedSchema, err := sg.Generate()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	schemaJSON, err := json.MarshalIndent(generatedSchema, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	// Parse schema to verify structure
	var schemaMap map[string]any
	if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
		t.Fatalf("Failed to parse schema JSON: %v", err)
	}

	// Verify oneOf and discriminator are present for Response field
	defs, ok := schemaMap["$defs"].(map[string]any)
	if !ok {
		t.Fatal("$defs not found in schema")
	}

	apiResult, ok := defs["APIResult"].(map[string]any)
	if !ok {
		t.Fatal("APIResult not found in $defs")
	}

	properties, ok := apiResult["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties not found in APIResult")
	}

	responseField, ok := properties["Response"].(map[string]any)
	if !ok {
		t.Fatal("Response field not found in properties")
	}

	oneOf, ok := responseField["oneOf"].([]any)
	if !ok {
		t.Fatal("oneOf not found in Response field")
	}

	if len(oneOf) != 3 {
		t.Errorf("Expected 3 types in oneOf, got %d", len(oneOf))
	}

	discriminator, ok := responseField["discriminator"].(map[string]any)
	if !ok {
		t.Fatal("discriminator not found in Response field")
	}

	propertyName, ok := discriminator["propertyName"].(string)
	if !ok || propertyName != "Status" {
		t.Errorf("Expected discriminator propertyName to be 'Status', got %v", propertyName)
	}

	// Verify description is present
	if desc, ok := responseField["description"].(string); !ok || desc == "" {
		t.Error("Description not found or empty in Response field")
}

	// Verify all response type definitions exist
	if _, ok := defs["SuccessResponse"]; !ok {
		t.Error("SuccessResponse definition not found in $defs")
	}
	if _, ok := defs["ErrorResponse"]; !ok {
		t.Error("ErrorResponse definition not found in $defs")
	}
	if _, ok := defs["PendingResponse"]; !ok {
		t.Error("PendingResponse definition not found in $defs")
	}
}
