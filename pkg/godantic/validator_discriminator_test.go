package godantic_test

import (
	"encoding/json"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Test types for discriminated union validation - using Animal example

type AnimalSpecies string

const (
	SpeciesCat  AnimalSpecies = "cat"
	SpeciesDog  AnimalSpecies = "dog"
	SpeciesBird AnimalSpecies = "bird"
)

// Animal interface - the discriminated union
type Animal interface {
	GetSpecies() AnimalSpecies
	isAnimal()
}

// Cat implementation
type Cat struct {
	Species   AnimalSpecies `json:"species"`
	Name      string        `json:"name"`
	LivesLeft int           `json:"lives_left"`
	IsIndoor  bool          `json:"is_indoor"`
}

func (c Cat) GetSpecies() AnimalSpecies { return c.Species }
func (c Cat) isAnimal()                 {}

func (c *Cat) FieldSpecies() godantic.FieldOptions[AnimalSpecies] {
	return godantic.Field(
		godantic.Required[AnimalSpecies](),
		godantic.Const(SpeciesCat),
	)
}

func (c *Cat) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(1),
	)
}

func (c *Cat) FieldLivesLeft() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.Min(0),
		godantic.Max(9),
	)
}

// Dog implementation
type Dog struct {
	Species AnimalSpecies `json:"species"`
	Name    string        `json:"name"`
	Breed   string        `json:"breed"`
	IsGood  bool          `json:"is_good"`
}

func (d Dog) GetSpecies() AnimalSpecies { return d.Species }
func (d Dog) isAnimal()                 {}

func (d *Dog) FieldSpecies() godantic.FieldOptions[AnimalSpecies] {
	return godantic.Field(
		godantic.Required[AnimalSpecies](),
		godantic.Const(SpeciesDog),
	)
}

func (d *Dog) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(1),
	)
}

func (d *Dog) FieldBreed() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
	)
}

// Bird implementation
type Bird struct {
	Species  AnimalSpecies `json:"species"`
	Name     string        `json:"name"`
	CanFly   bool          `json:"can_fly"`
	Wingspan float64       `json:"wingspan"`
}

func (b Bird) GetSpecies() AnimalSpecies { return b.Species }
func (b Bird) isAnimal()                 {}

func (b *Bird) FieldSpecies() godantic.FieldOptions[AnimalSpecies] {
	return godantic.Field(
		godantic.Required[AnimalSpecies](),
		godantic.Const(SpeciesBird),
	)
}

func (b *Bird) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(1),
	)
}

func (b *Bird) FieldWingspan() godantic.FieldOptions[float64] {
	return godantic.Field(
		godantic.Required[float64](),
		godantic.Min(0.0),
	)
}

// Tests

func TestDiscriminatedUnion_Cat(t *testing.T) {
	jsonData := `{
		"species": "cat",
		"name": "Whiskers",
		"lives_left": 7,
		"is_indoor": true
	}`

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  Cat{},
			SpeciesDog:  Dog{},
			SpeciesBird: Bird{},
		}),
	)

	animal, errs := validator.Marshal([]byte(jsonData))
	if errs != nil {
		t.Fatalf("Validation failed: %v", errs)
	}

	// Type assert to verify we got the right type
	cat, ok := (*animal).(Cat)
	if !ok {
		t.Fatalf("Expected Cat, got %T", *animal)
	}

	if cat.Name != "Whiskers" {
		t.Errorf("Expected name 'Whiskers', got %s", cat.Name)
	}

	if cat.LivesLeft != 7 {
		t.Errorf("Expected 7 lives, got %d", cat.LivesLeft)
	}

	if cat.GetSpecies() != SpeciesCat {
		t.Errorf("Expected species 'cat', got %s", cat.GetSpecies())
	}
}

func TestDiscriminatedUnion_Dog(t *testing.T) {
	jsonData := `{
		"species": "dog",
		"name": "Buddy",
		"breed": "Golden Retriever",
		"is_good": true
	}`

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  Cat{},
			SpeciesDog:  Dog{},
			SpeciesBird: Bird{},
		}),
	)

	animal, errs := validator.Marshal([]byte(jsonData))
	if errs != nil {
		t.Fatalf("Validation failed: %v", errs)
	}

	dog, ok := (*animal).(Dog)
	if !ok {
		t.Fatalf("Expected Dog, got %T", *animal)
	}

	if dog.Name != "Buddy" {
		t.Errorf("Expected name 'Buddy', got %s", dog.Name)
	}

	if dog.Breed != "Golden Retriever" {
		t.Errorf("Expected breed 'Golden Retriever', got %s", dog.Breed)
	}
}

func TestDiscriminatedUnion_Bird(t *testing.T) {
	jsonData := `{
		"species": "bird",
		"name": "Tweety",
		"can_fly": true,
		"wingspan": 0.25
	}`

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  Cat{},
			SpeciesDog:  Dog{},
			SpeciesBird: Bird{},
		}),
	)

	animal, errs := validator.Marshal([]byte(jsonData))
	if errs != nil {
		t.Fatalf("Validation failed: %v", errs)
	}

	bird, ok := (*animal).(Bird)
	if !ok {
		t.Fatalf("Expected Bird, got %T", *animal)
	}

	if bird.Name != "Tweety" {
		t.Errorf("Expected name 'Tweety', got %s", bird.Name)
	}

	if bird.Wingspan != 0.25 {
		t.Errorf("Expected wingspan 0.25, got %f", bird.Wingspan)
	}
}

func TestDiscriminatedUnion_InvalidDiscriminator(t *testing.T) {
	jsonData := `{
		"species": "fish",
		"name": "Nemo"
	}`

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  Cat{},
			SpeciesDog:  Dog{},
			SpeciesBird: Bird{},
		}),
	)

	_, errs := validator.Marshal([]byte(jsonData))
	if errs == nil {
		t.Fatal("Expected validation to fail for invalid discriminator")
	}

	if errs[0].Type != "discriminator_invalid" {
		t.Errorf("Expected error type 'discriminator_invalid', got %s", errs[0].Type)
	}
}

func TestDiscriminatedUnion_MissingDiscriminator(t *testing.T) {
	jsonData := `{
		"name": "Mystery Animal"
	}`

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  Cat{},
			SpeciesDog:  Dog{},
			SpeciesBird: Bird{},
		}),
	)

	_, errs := validator.Marshal([]byte(jsonData))
	if errs == nil {
		t.Fatal("Expected validation to fail for missing discriminator")
	}

	if errs[0].Type != "discriminator_missing" {
		t.Errorf("Expected error type 'discriminator_missing', got %s", errs[0].Type)
	}
}

func TestDiscriminatedUnion_ValidationFailure(t *testing.T) {
	jsonData := `{
		"species": "cat",
		"name": "Whiskers",
		"lives_left": 15
	}`

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  Cat{},
			SpeciesDog:  Dog{},
			SpeciesBird: Bird{},
		}),
	)

	_, errs := validator.Marshal([]byte(jsonData))
	if errs == nil {
		t.Fatal("Expected validation to fail for lives_left > 9")
	}

	if len(errs) == 0 {
		t.Fatal("Expected at least one validation error")
	}

	// Should fail on the Max(9) constraint
	found := false
	for _, err := range errs {
		if err.Type == "constraint" && len(err.Loc) > 0 && err.Loc[len(err.Loc)-1] == "LivesLeft" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected constraint error on LivesLeft field")
	}
}

func TestDiscriminatedUnion_MissingRequiredField(t *testing.T) {
	jsonData := `{
		"species": "dog",
		"name": "Buddy"
	}`

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  Cat{},
			SpeciesDog:  Dog{},
			SpeciesBird: Bird{},
		}),
	)

	_, errs := validator.Marshal([]byte(jsonData))
	if errs == nil {
		t.Fatal("Expected validation to fail for missing required 'breed' field")
	}

	// Should fail on required Breed field
	found := false
	for _, err := range errs {
		if err.Type == "required" && len(err.Loc) > 0 && err.Loc[len(err.Loc)-1] == "Breed" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected required field error on Breed field, got: %v", errs)
	}
}

func TestDiscriminatedUnion_WithStringKeys(t *testing.T) {
	// Test the non-typed variant with plain string keys
	jsonData := `{
		"species": "cat",
		"name": "Mittens",
		"lives_left": 9,
		"is_indoor": false
	}`

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminator("species", map[string]any{
			"cat":  Cat{},
			"dog":  Dog{},
			"bird": Bird{},
		}),
	)

	animal, errs := validator.Marshal([]byte(jsonData))
	if errs != nil {
		t.Fatalf("Validation failed: %v", errs)
	}

	cat, ok := (*animal).(Cat)
	if !ok {
		t.Fatalf("Expected Cat, got %T", *animal)
	}

	if cat.Name != "Mittens" {
		t.Errorf("Expected name 'Mittens', got %s", cat.Name)
	}
}

// Tests for pointer types in discriminator map
// These are critical when interface methods are defined on pointer receivers

func TestDiscriminatedUnion_PointerVariants_Cat(t *testing.T) {
	jsonData := `{
		"species": "cat",
		"name": "Shadow",
		"lives_left": 5,
		"is_indoor": true
	}`

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  &Cat{},
			SpeciesDog:  &Dog{},
			SpeciesBird: &Bird{},
		}),
	)

	animal, errs := validator.Marshal([]byte(jsonData))
	if errs != nil {
		t.Fatalf("Validation failed: %v", errs)
	}

	// With pointer variants, result should be *Cat
	cat, ok := (*animal).(*Cat)
	if !ok {
		t.Fatalf("Expected *Cat, got %T", *animal)
	}

	if cat.Name != "Shadow" {
		t.Errorf("Expected name 'Shadow', got %s", cat.Name)
	}

	if cat.LivesLeft != 5 {
		t.Errorf("Expected 5 lives, got %d", cat.LivesLeft)
	}

	if cat.GetSpecies() != SpeciesCat {
		t.Errorf("Expected species 'cat', got %s", cat.GetSpecies())
	}
}

func TestDiscriminatedUnion_PointerVariants_Dog(t *testing.T) {
	jsonData := `{
		"species": "dog",
		"name": "Max",
		"breed": "Labrador",
		"is_good": true
	}`

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  &Cat{},
			SpeciesDog:  &Dog{},
			SpeciesBird: &Bird{},
		}),
	)

	animal, errs := validator.Marshal([]byte(jsonData))
	if errs != nil {
		t.Fatalf("Validation failed: %v", errs)
	}

	dog, ok := (*animal).(*Dog)
	if !ok {
		t.Fatalf("Expected *Dog, got %T", *animal)
	}

	if dog.Name != "Max" {
		t.Errorf("Expected name 'Max', got %s", dog.Name)
	}

	if dog.Breed != "Labrador" {
		t.Errorf("Expected breed 'Labrador', got %s", dog.Breed)
	}
}

func TestDiscriminatedUnion_PointerVariants_ValidationFailure(t *testing.T) {
	jsonData := `{
		"species": "cat",
		"name": "Fluffy",
		"lives_left": 15,
		"is_indoor": true
	}`

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  &Cat{},
			SpeciesDog:  &Dog{},
			SpeciesBird: &Bird{},
		}),
	)

	_, errs := validator.Marshal([]byte(jsonData))
	if errs == nil {
		t.Fatal("Expected validation to fail for lives_left > 9")
	}

	// Should have constraint validation error (Max constraint on lives_left)
	found := false
	for _, err := range errs {
		if err.Type == "constraint" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected constraint error, got: %v", errs)
	}
}

func TestDiscriminatedUnion_PointerVariants_InvalidDiscriminator(t *testing.T) {
	jsonData := `{
		"species": "dragon",
		"name": "Smaug"
	}`

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  &Cat{},
			SpeciesDog:  &Dog{},
			SpeciesBird: &Bird{},
		}),
	)

	_, errs := validator.Marshal([]byte(jsonData))
	if errs == nil {
		t.Fatal("Expected validation to fail for invalid discriminator")
	}

	// Should have discriminator_invalid error
	if errs[0].Type != "discriminator_invalid" {
		t.Errorf("Expected discriminator_invalid error, got: %s", errs[0].Type)
	}
}

func TestDiscriminatedUnion_PointerVariants_MissingRequiredField(t *testing.T) {
	jsonData := `{
		"species": "cat",
		"lives_left": 3,
		"is_indoor": false
	}`

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  &Cat{},
			SpeciesDog:  &Dog{},
			SpeciesBird: &Bird{},
		}),
	)

	_, errs := validator.Marshal([]byte(jsonData))
	if errs == nil {
		t.Fatal("Expected validation to fail for missing required 'name' field")
	}

	// Should have required field error
	found := false
	for _, err := range errs {
		if err.Type == "required" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected required error, got: %v", errs)
	}
}

// Tests for Unmarshal (struct → JSON) with discriminated unions

func TestDiscriminatedUnion_Unmarshal_Cat(t *testing.T) {
	cat := Cat{
		Species:   SpeciesCat,
		Name:      "Whiskers",
		LivesLeft: 7,
		IsIndoor:  true,
	}

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  Cat{},
			SpeciesDog:  Dog{},
			SpeciesBird: Bird{},
		}),
	)

	var animal Animal = cat
	jsonData, errs := validator.Unmarshal(&animal)
	if errs != nil {
		t.Fatalf("Unmarshal failed: %v", errs)
	}

	if len(jsonData) == 0 {
		t.Fatal("Expected non-empty JSON data")
	}

	// Verify it's valid JSON
	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Invalid JSON produced: %v", err)
	}

	if result["species"] != string(SpeciesCat) {
		t.Errorf("Expected species 'cat', got %v", result["species"])
	}
	if result["name"] != "Whiskers" {
		t.Errorf("Expected name 'Whiskers', got %v", result["name"])
	}
	if result["lives_left"].(float64) != 7 {
		t.Errorf("Expected lives_left 7, got %v", result["lives_left"])
	}
}

func TestDiscriminatedUnion_Unmarshal_Dog(t *testing.T) {
	dog := Dog{
		Species: SpeciesDog,
		Name:    "Buddy",
		Breed:   "Golden Retriever",
		IsGood:  true,
	}

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  Cat{},
			SpeciesDog:  Dog{},
			SpeciesBird: Bird{},
		}),
	)

	var animal Animal = dog
	jsonData, errs := validator.Unmarshal(&animal)
	if errs != nil {
		t.Fatalf("Unmarshal failed: %v", errs)
	}

	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Invalid JSON produced: %v", err)
	}

	if result["species"] != string(SpeciesDog) {
		t.Errorf("Expected species 'dog', got %v", result["species"])
	}
	if result["name"] != "Buddy" {
		t.Errorf("Expected name 'Buddy', got %v", result["name"])
	}
	if result["breed"] != "Golden Retriever" {
		t.Errorf("Expected breed 'Golden Retriever', got %v", result["breed"])
	}
}

func TestDiscriminatedUnion_Unmarshal_ValidationFailure(t *testing.T) {
	cat := Cat{
		Species:   SpeciesCat,
		Name:      "Whiskers",
		LivesLeft: 15, // Invalid: exceeds Max(9)
		IsIndoor:  true,
	}

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  Cat{},
			SpeciesDog:  Dog{},
			SpeciesBird: Bird{},
		}),
	)

	var animal Animal = cat
	_, errs := validator.Unmarshal(&animal)
	if errs == nil {
		t.Fatal("Expected validation to fail for lives_left > 9")
	}

	// Should have constraint error
	found := false
	for _, err := range errs {
		if err.Type == "constraint" && len(err.Loc) > 0 && err.Loc[len(err.Loc)-1] == "LivesLeft" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected constraint error on LivesLeft, got: %v", errs)
	}
}

func TestDiscriminatedUnion_Unmarshal_InvalidDiscriminator(t *testing.T) {
	// Create a struct with an invalid discriminator value
	// This tests the type mismatch detection
	cat := Cat{
		Species:   "invalid", // Not one of the allowed values
		Name:      "Whiskers",
		LivesLeft: 7,
		IsIndoor:  true,
	}

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  Cat{},
			SpeciesDog:  Dog{},
			SpeciesBird: Bird{},
		}),
	)

	var animal Animal = cat
	_, errs := validator.Unmarshal(&animal)
	if errs == nil {
		t.Fatal("Expected validation to fail for invalid discriminator value")
	}

	if errs[0].Type != "discriminator_invalid" {
		t.Errorf("Expected discriminator_invalid error, got: %s", errs[0].Type)
	}
}

func TestDiscriminatedUnion_Unmarshal_MissingRequiredField(t *testing.T) {
	dog := Dog{
		Species: SpeciesDog,
		Name:    "Buddy",
		// Breed is missing (required field)
		IsGood: true,
	}

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  Cat{},
			SpeciesDog:  Dog{},
			SpeciesBird: Bird{},
		}),
	)

	var animal Animal = dog
	_, errs := validator.Unmarshal(&animal)
	if errs == nil {
		t.Fatal("Expected validation to fail for missing required 'breed' field")
	}

	found := false
	for _, err := range errs {
		if err.Type == "required" && len(err.Loc) > 0 && err.Loc[len(err.Loc)-1] == "Breed" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected required error on Breed, got: %v", errs)
	}
}

func TestDiscriminatedUnion_Unmarshal_PointerVariants(t *testing.T) {
	cat := &Cat{
		Species:   SpeciesCat,
		Name:      "Shadow",
		LivesLeft: 5,
		IsIndoor:  true,
	}

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat:  &Cat{},
			SpeciesDog:  &Dog{},
			SpeciesBird: &Bird{},
		}),
	)

	var animal Animal = cat
	jsonData, errs := validator.Unmarshal(&animal)
	if errs != nil {
		t.Fatalf("Unmarshal failed: %v", errs)
	}

	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Invalid JSON produced: %v", err)
	}

	if result["name"] != "Shadow" {
		t.Errorf("Expected name 'Shadow', got %v", result["name"])
	}
	if result["lives_left"].(float64) != 5 {
		t.Errorf("Expected lives_left 5, got %v", result["lives_left"])
	}
}

// Test with defaults to ensure they're applied during Unmarshal
type CatWithDefaults struct {
	Species   AnimalSpecies `json:"species"`
	Name      string        `json:"name"`
	LivesLeft int           `json:"lives_left"`
	Color     string        `json:"color"`
}

func (c CatWithDefaults) GetSpecies() AnimalSpecies { return c.Species }
func (c CatWithDefaults) isAnimal()                 {}

func (c *CatWithDefaults) FieldSpecies() godantic.FieldOptions[AnimalSpecies] {
	return godantic.Field(
		godantic.Required[AnimalSpecies](),
		godantic.Const(SpeciesCat),
	)
}

func (c *CatWithDefaults) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
	)
}

func (c *CatWithDefaults) FieldLivesLeft() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Default(9), // Default value
		godantic.Min(0),
		godantic.Max(9),
	)
}

func (c *CatWithDefaults) FieldColor() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Default("orange"), // Default value
	)
}

func TestDiscriminatedUnion_Unmarshal_AppliesDefaults(t *testing.T) {
	cat := CatWithDefaults{
		Species: SpeciesCat,
		Name:    "Garfield",
		// LivesLeft not set - should default to 9
		// Color not set - should default to "orange"
	}

	validator := godantic.NewValidator[Animal](
		godantic.WithDiscriminatorTyped("species", map[AnimalSpecies]any{
			SpeciesCat: CatWithDefaults{},
		}),
	)

	var animal Animal = cat
	jsonData, errs := validator.Unmarshal(&animal)
	if errs != nil {
		t.Fatalf("Unmarshal failed: %v", errs)
	}

	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Invalid JSON produced: %v", err)
	}

	// Verify defaults were applied
	if result["lives_left"].(float64) != 9 {
		t.Errorf("Expected lives_left default 9, got %v", result["lives_left"])
	}
	if result["color"] != "orange" {
		t.Errorf("Expected color default 'orange', got %v", result["color"])
	}
}
