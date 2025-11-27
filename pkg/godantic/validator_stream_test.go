package godantic_test

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// ═══════════════════════════════════════════════════════════════════════════
// StreamParser - Basic Streaming
// ═══════════════════════════════════════════════════════════════════════════

func TestStreamParser_Basic(t *testing.T) {
	parser := godantic.NewStreamParser[TUser]()

	// Feed first chunk
	result1, state1, _ := parser.Feed([]byte(`{"name": "Jo`))
	if result1 == nil {
		t.Fatal("expected result")
	}
	if result1.Name != "Jo" {
		t.Errorf("Name = %q, want 'Jo'", result1.Name)
	}
	if state1.IsComplete {
		t.Error("expected incomplete")
	}

	// Feed second chunk
	result2, state2, _ := parser.Feed([]byte(`hn", "email": "john@example.com"`))
	if result2 == nil {
		t.Fatal("expected result")
	}
	if result2.Name != "John" {
		t.Errorf("Name = %q, want 'John'", result2.Name)
	}
	if result2.Email != "john@example.com" {
		t.Errorf("Email = %q, want 'john@example.com'", result2.Email)
	}
	_ = state2 // might be complete or not depending on parser

	// Feed final chunk
	result3, state3, errs := parser.Feed([]byte(`, "age": 30}`))
	if result3 == nil {
		t.Fatal("expected result")
	}
	if result3.Name != "John" {
		t.Errorf("Name = %q, want 'John'", result3.Name)
	}
	if result3.Age != 30 {
		t.Errorf("Age = %d, want 30", result3.Age)
	}
	if !state3.IsComplete {
		t.Error("expected complete")
	}
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestStreamParser_Reset(t *testing.T) {
	parser := godantic.NewStreamParser[TUser]()

	parser.Feed([]byte(`{"name": "Jo`))
	if len(parser.Buffer()) == 0 {
		t.Error("expected buffer to have data")
	}

	parser.Reset()
	if len(parser.Buffer()) != 0 {
		t.Error("expected buffer to be empty after reset")
	}

	result, state, _ := parser.Feed([]byte(`{"name": "John", "email": "j@e.com"}`))
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Name != "John" {
		t.Errorf("Name = %q, want 'John'", result.Name)
	}
	if !state.IsComplete {
		t.Error("expected complete")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// StreamParser - Collections
// ═══════════════════════════════════════════════════════════════════════════

func TestStreamParser_Slices(t *testing.T) {
	parser := godantic.NewStreamParser[TUserWithSlice]()

	// Feed chunks with slice
	result1, state1, _ := parser.Feed([]byte(`{"name": "John", "tags": ["tag1", "tag`))
	if result1 == nil {
		t.Fatal("expected result")
	}
	if result1.Name != "John" {
		t.Errorf("Name = %q, want 'John'", result1.Name)
	}
	if len(result1.Tags) == 0 {
		t.Error("expected at least one tag")
	}
	if state1.IsComplete {
		t.Error("expected incomplete")
	}

	// Continue feeding
	result2, state2, _ := parser.Feed([]byte(`2", "tag3"], "ids": [1, 2,`))
	if result2 == nil {
		t.Fatal("expected result")
	}
	if len(result2.Tags) < 2 {
		t.Errorf("Tags len = %d, want >= 2", len(result2.Tags))
	}
	_ = state2

	// Complete
	result3, state3, errs := parser.Feed([]byte(` 3]}`))
	if result3 == nil {
		t.Fatal("expected result")
	}
	if len(result3.Tags) != 3 {
		t.Errorf("Tags len = %d, want 3", len(result3.Tags))
	}
	if len(result3.IDs) != 3 {
		t.Errorf("IDs len = %d, want 3", len(result3.IDs))
	}
	if !state3.IsComplete {
		t.Error("expected complete")
	}
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestStreamParser_Maps(t *testing.T) {
	parser := godantic.NewStreamParser[TUserWithMap]()

	// Feed chunks with map
	result1, state1, _ := parser.Feed([]byte(`{"name": "John", "metadata": {"ke`))
	if result1 == nil {
		t.Fatal("expected result")
	}
	if result1.Name != "John" {
		t.Errorf("Name = %q, want 'John'", result1.Name)
	}
	if state1.IsComplete {
		t.Error("expected incomplete")
	}

	// Continue
	result2, state2, _ := parser.Feed([]byte(`y": "valu`))
	if result2 == nil {
		t.Fatal("expected result")
	}
	if result2.Metadata == nil {
		t.Error("expected metadata map")
	}
	_ = state2

	// Complete
	result3, state3, errs := parser.Feed([]byte(`e"}, "scores": {"math": 95}}`))
	if result3 == nil {
		t.Fatal("expected result")
	}
	if result3.Metadata["key"] != "value" {
		t.Errorf("Metadata[key] = %q, want 'value'", result3.Metadata["key"])
	}
	if result3.Scores["math"] != 95 {
		t.Errorf("Scores[math] = %d, want 95", result3.Scores["math"])
	}
	if !state3.IsComplete {
		t.Error("expected complete")
	}
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// StreamParser - Nested Structs
// ═══════════════════════════════════════════════════════════════════════════

func TestStreamParser_Nested(t *testing.T) {
	parser := godantic.NewStreamParser[TUserWithAddress]()

	// Feed chunks with nested struct
	result1, state1, _ := parser.Feed([]byte(`{"name": "John", "address": {"street": "123 Main St", "city": "Ne`))
	if result1 == nil {
		t.Fatal("expected result")
	}
	if result1.Name != "John" {
		t.Errorf("Name = %q, want 'John'", result1.Name)
	}
	if result1.Address.Street != "123 Main St" {
		t.Errorf("Street = %q, want '123 Main St'", result1.Address.Street)
	}
	if result1.Address.City != "Ne" {
		t.Errorf("City = %q, want 'Ne'", result1.Address.City)
	}
	if state1.IsComplete {
		t.Error("expected incomplete")
	}

	// Complete
	result2, state2, errs := parser.Feed([]byte(`w York"}}`))
	if result2 == nil {
		t.Fatal("expected result")
	}
	if result2.Address.City != "New York" {
		t.Errorf("City = %q, want 'New York'", result2.Address.City)
	}
	if !state2.IsComplete {
		t.Error("expected complete")
	}
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestStreamParser_DeepNested(t *testing.T) {
	parser := godantic.NewStreamParser[TDeepConfig]()

	// Feed chunks
	result1, state1, _ := parser.Feed([]byte(`{"level1": {"level2": {"level3": {"value": "tes`))
	if result1 == nil {
		t.Fatal("expected result")
	}
	if result1.Level1.Level2.Level3.Value != "tes" {
		t.Errorf("Value = %q, want 'tes'", result1.Level1.Level2.Level3.Value)
	}
	if state1.IsComplete {
		t.Error("expected incomplete")
	}

	// Complete
	result2, state2, errs := parser.Feed([]byte(`t"}}}}`))
	if result2 == nil {
		t.Fatal("expected result")
	}
	if result2.Level1.Level2.Level3.Value != "test" {
		t.Errorf("Value = %q, want 'test'", result2.Level1.Level2.Level3.Value)
	}
	if !state2.IsComplete {
		t.Error("expected complete")
	}
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestStreamParser_Pointers(t *testing.T) {
	parser := godantic.NewStreamParser[TUserWithPointers]()

	// Feed chunks with pointer fields
	result1, state1, _ := parser.Feed([]byte(`{"name": "Jo`))
	if result1 == nil {
		t.Fatal("expected result")
	}
	if result1.Name == nil || *result1.Name != "Jo" {
		t.Errorf("Name = %v, want 'Jo'", result1.Name)
	}
	if state1.IsComplete {
		t.Error("expected incomplete")
	}

	// Complete
	result2, state2, errs := parser.Feed([]byte(`hn", "age": 30}`))
	if result2 == nil {
		t.Fatal("expected result")
	}
	if result2.Name == nil || *result2.Name != "John" {
		t.Errorf("Name = %v, want 'John'", result2.Name)
	}
	if result2.Age == nil || *result2.Age != 30 {
		t.Errorf("Age = %v, want 30", result2.Age)
	}
	if !state2.IsComplete {
		t.Error("expected complete")
	}
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// StreamParser - Discriminated Unions
// ═══════════════════════════════════════════════════════════════════════════

func TestStreamParser_Union(t *testing.T) {
	validator := NewTAnimalValidator()
	parser := godantic.NewStreamParserWithValidator(validator)

	// Feed partial discriminator
	result1, state1, _ := parser.Feed([]byte(`{"species": "do`))
	if state1.IsComplete {
		t.Error("expected incomplete")
	}
	_ = result1

	// Complete discriminator, partial fields
	result2, state2, _ := parser.Feed([]byte(`g", "name": "Bu`))
	if result2 == nil {
		t.Fatal("expected result")
	}
	dog, ok := (*result2).(*TDog)
	if !ok {
		t.Fatalf("expected *TDog, got %T", *result2)
	}
	if dog.Species != TSpeciesDog {
		t.Errorf("Species = %v, want 'dog'", dog.Species)
	}
	if dog.Name != "Bu" {
		t.Errorf("Name = %q, want 'Bu'", dog.Name)
	}
	if state2.IsComplete {
		t.Error("expected incomplete")
	}

	// Complete
	result3, state3, errs := parser.Feed([]byte(`ddy", "breed": "Golden Retriever", "is_good": true}`))
	if result3 == nil {
		t.Fatal("expected result")
	}
	dog3, ok := (*result3).(*TDog)
	if !ok {
		t.Fatalf("expected *TDog, got %T", *result3)
	}
	if dog3.Name != "Buddy" {
		t.Errorf("Name = %q, want 'Buddy'", dog3.Name)
	}
	if dog3.Breed != "Golden Retriever" {
		t.Errorf("Breed = %q, want 'Golden Retriever'", dog3.Breed)
	}
	if !state3.IsComplete {
		t.Error("expected complete")
	}
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

func TestStreamParser_UnionArray(t *testing.T) {
	parser := godantic.NewStreamParser[TAnimalList]()

	// Feed chunks with array of unions
	result1, state1, _ := parser.Feed([]byte(`{"animals": [{"type": "dog", "name": "Re`))
	if result1 == nil {
		t.Fatal("expected result")
	}
	if len(result1.Animals) == 0 {
		t.Error("expected at least one animal")
	}
	if result1.Animals[0].Name != "Re" {
		t.Errorf("Name = %q, want 'Re'", result1.Animals[0].Name)
	}
	if state1.IsComplete {
		t.Error("expected incomplete")
	}

	// Continue
	result2, state2, _ := parser.Feed([]byte(`x"}, {"type": "ca`))
	if result2 == nil {
		t.Fatal("expected result")
	}
	if len(result2.Animals) < 1 {
		t.Error("expected at least one animal")
	}
	_ = state2

	// Complete
	result3, state3, errs := parser.Feed([]byte(`t", "name": "Fluffy"}]}`))
	if result3 == nil {
		t.Fatal("expected result")
	}
	if len(result3.Animals) != 2 {
		t.Errorf("Animals len = %d, want 2", len(result3.Animals))
	}
	if !state3.IsComplete {
		t.Error("expected complete")
	}
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// StreamParser - Map with Nested Struct
// ═══════════════════════════════════════════════════════════════════════════

func TestStreamParser_MapWithNestedStruct(t *testing.T) {
	parser := godantic.NewStreamParser[TConfigWithNestedMap]()

	// Feed chunks
	result1, state1, _ := parser.Feed([]byte(`{"settings": {"key1": {"value": "tes`))
	if result1 == nil {
		t.Fatal("expected result")
	}
	if result1.Settings == nil {
		t.Error("expected settings map")
	}
	if state1.IsComplete {
		t.Error("expected incomplete")
	}

	// Complete
	result2, state2, errs := parser.Feed([]byte(`t", "count": 42}}}`))
	if result2 == nil {
		t.Fatal("expected result")
	}
	if result2.Settings["key1"].Value != "test" {
		t.Errorf("Value = %q, want 'test'", result2.Settings["key1"].Value)
	}
	if result2.Settings["key1"].Count != 42 {
		t.Errorf("Count = %d, want 42", result2.Settings["key1"].Count)
	}
	if !state2.IsComplete {
		t.Error("expected complete")
	}
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}
