package schema_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

// Test types for discriminated union
type TestCat struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type TestDog struct {
	Type  string `json:"type"`
	Breed string `json:"breed"`
}

type TestAnimal interface {
	GetType() string
}

type TestZoo struct {
	Animal TestAnimal `json:"animal"`
}

func (z *TestZoo) FieldAnimal() godantic.FieldOptions[TestAnimal] {
	return godantic.Field(
		godantic.DiscriminatedUnion[TestAnimal]("type", map[string]any{
			"cat": TestCat{},
			"dog": TestDog{},
		}),
	)
}

// TestGenerateForType_DiscriminatedUnion tests that GenerateForType correctly
// generates schemas for discriminated unions and includes all variant types
func TestGenerateForType_DiscriminatedUnion(t *testing.T) {

	// Generate schema using GenerateForType (not Generator[T])
	zooType := reflect.TypeOf(TestZoo{})
	schemaMap, err := schema.GenerateForType(zooType)
	if err != nil {
		t.Fatalf("GenerateForType failed: %v", err)
	}

	// Convert to JSON for inspection
	schemaJSON, err := json.MarshalIndent(schemaMap, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	t.Logf("Generated schema:\n%s", schemaJSON)

	// Check that $defs contains all variant types
	defs, ok := schemaMap["$defs"].(map[string]any)
	if !ok {
		t.Fatal("Expected $defs in schema")
	}

	// Check that Cat and Dog schemas are included
	if _, hasCat := defs["TestCat"]; !hasCat {
		t.Error("Expected TestCat schema in $defs")
	}

	if _, hasDog := defs["TestDog"]; !hasDog {
		t.Error("Expected TestDog schema in $defs")
	}

	// Check that Zoo schema is included
	if _, hasZoo := defs["TestZoo"]; !hasZoo {
		t.Error("Expected TestZoo schema in $defs")
	}

	// Verify discriminator is set correctly
	zooSchema, ok := defs["TestZoo"].(map[string]any)
	if !ok {
		t.Fatal("Zoo schema is not a map")
	}

	properties, ok := zooSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Zoo schema has no properties")
	}

	animalProp, ok := properties["animal"].(map[string]any)
	if !ok {
		t.Fatal("Animal property not found")
	}

	// Check oneOf
	oneOf, ok := animalProp["oneOf"].([]any)
	if !ok {
		t.Fatal("Animal property should have oneOf")
	}

	if len(oneOf) != 2 {
		t.Errorf("Expected 2 variants in oneOf, got %d", len(oneOf))
	}

	// Check discriminator
	discriminator, ok := animalProp["discriminator"].(map[string]any)
	if !ok {
		t.Fatal("Animal property should have discriminator")
	}

	propertyName, ok := discriminator["propertyName"].(string)
	if !ok || propertyName != "type" {
		t.Errorf("Expected discriminator propertyName 'type', got %v", propertyName)
	}
}

// Test types for nested discriminated union
type TestSuccessResult struct {
	Status string `json:"status"`
	Data   string `json:"data"`
}

type TestErrorResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type TestResult interface {
	GetStatus() string
}

type TestResponse struct {
	Result TestResult `json:"result"`
}

func (r *TestResponse) FieldResult() godantic.FieldOptions[TestResult] {
	return godantic.Field(
		godantic.DiscriminatedUnion[TestResult]("status", map[string]any{
			"success": TestSuccessResult{},
			"error":   TestErrorResult{},
		}),
	)
}

type TestAPICall struct {
	Response TestResponse `json:"response"`
}

// TestGenerateForType_NestedDiscriminatedUnion tests that nested discriminated unions work
func TestGenerateForType_NestedDiscriminatedUnion(t *testing.T) {

	// Generate schema for the outer type
	apiCallType := reflect.TypeOf(TestAPICall{})
	schemaMap, err := schema.GenerateForType(apiCallType)
	if err != nil {
		t.Fatalf("GenerateForType failed: %v", err)
	}

	// Check that all nested types are included
	defs, ok := schemaMap["$defs"].(map[string]any)
	if !ok {
		t.Fatal("Expected $defs in schema")
	}

	requiredTypes := []string{"TestAPICall", "TestResponse", "TestSuccessResult", "TestErrorResult"}
	for _, typeName := range requiredTypes {
		if _, has := defs[typeName]; !has {
			t.Errorf("Expected %s schema in $defs", typeName)
		}
	}
}

// TestGenerateForType_WithoutDiscriminatedUnion tests that regular structs still work
func TestGenerateForType_WithoutDiscriminatedUnion(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}

	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	personType := reflect.TypeOf(Person{})
	schemaMap, err := schema.GenerateForType(personType)
	if err != nil {
		t.Fatalf("GenerateForType failed: %v", err)
	}

	defs, ok := schemaMap["$defs"].(map[string]any)
	if !ok {
		t.Fatal("Expected $defs in schema")
	}

	if _, has := defs["Person"]; !has {
		t.Error("Expected Person schema in $defs")
	}

	if _, has := defs["Address"]; !has {
		t.Error("Expected Address schema in $defs")
	}
}

