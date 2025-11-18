package schema_test

import (
	"encoding/json"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

// Test Union (anyOf) - field can be multiple types

type FlexibleConfig struct {
	// Value can be string, integer, or object
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

// Test DiscriminatedUnion (oneOf with discriminator) - common pattern for API responses

type Cat struct {
	Type string
	Meow string
}

type Dog struct {
	Type string
	Bark string
}

type Bird struct {
	Type  string
	Chirp string
}

type AnimalResponse struct {
	Animal any
}

func (a *AnimalResponse) FieldAnimal() godantic.FieldOptions[any] {
	return godantic.Field(
		godantic.Required[any](),
		godantic.DiscriminatedUnion[any]("Type", map[string]any{
			"cat":  Cat{},
			"dog":  Dog{},
			"bird": Bird{},
		}),
		godantic.Description[any]("The animal can be a cat, dog, or bird"),
	)
}

func TestDiscriminatedUnionSchema(t *testing.T) {
	sg := schema.NewGenerator[AnimalResponse]()
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

	// Verify oneOf and discriminator are present for Animal field
	defs, ok := schemaMap["$defs"].(map[string]any)
	if !ok {
		t.Fatal("$defs not found in schema")
	}

	animalResponse, ok := defs["AnimalResponse"].(map[string]any)
	if !ok {
		t.Fatal("AnimalResponse not found in $defs")
	}

	properties, ok := animalResponse["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties not found in AnimalResponse")
	}

	animalField, ok := properties["Animal"].(map[string]any)
	if !ok {
		t.Fatal("Animal field not found in properties")
	}

	oneOf, ok := animalField["oneOf"].([]any)
	if !ok {
		t.Fatal("oneOf not found in Animal field")
	}

	if len(oneOf) != 3 {
		t.Errorf("Expected 3 types in oneOf, got %d", len(oneOf))
	}

	discriminator, ok := animalField["discriminator"].(map[string]any)
	if !ok {
		t.Fatal("discriminator not found in Animal field")
	}

	propertyName, ok := discriminator["propertyName"].(string)
	if !ok || propertyName != "Type" {
		t.Errorf("Expected discriminator propertyName to be 'Type', got %v", propertyName)
	}

	// Verify description is present
	if desc, ok := animalField["description"].(string); !ok || desc == "" {
		t.Error("Description not found or empty in Animal field")
	}
}
