package godantic_test

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// ═══════════════════════════════════════════════════════════════════════════
// MarshalPartial - Basic Types
// ═══════════════════════════════════════════════════════════════════════════

func TestUnmarshalPartial_Basic(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantName       string
		wantEmail      string
		wantAge        int
		wantComplete   bool
		wantIncomplete []string // expected incomplete JSON paths
	}{
		{
			name:           "truncated_string",
			input:          `{"name": "Jo`,
			wantName:       "Jo",
			wantComplete:   false,
			wantIncomplete: []string{"name"},
		},
		{
			name:           "truncated_after_colon",
			input:          `{"name": "John", "age":`,
			wantName:       "John",
			wantAge:        0,
			wantComplete:   false,
			wantIncomplete: []string{"age"},
		},
		{
			name:         "complete",
			input:        `{"name": "John", "email": "john@example.com", "age": 30}`,
			wantName:     "John",
			wantEmail:    "john@example.com",
			wantAge:      30,
			wantComplete: true,
		},
		{
			name:         "complete_minimal",
			input:        `{"name": "A", "email": "b@c.com"}`,
			wantName:     "A",
			wantEmail:    "b@c.com",
			wantComplete: true,
		},
	}

	validator := godantic.NewValidator[TUser]()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, state, _ := validator.UnmarshalPartial([]byte(tt.input))

			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}
			if tt.wantEmail != "" && result.Email != tt.wantEmail {
				t.Errorf("Email = %q, want %q", result.Email, tt.wantEmail)
			}
			if result.Age != tt.wantAge {
				t.Errorf("Age = %d, want %d", result.Age, tt.wantAge)
			}
			if state.IsComplete != tt.wantComplete {
				t.Errorf("IsComplete = %v, want %v", state.IsComplete, tt.wantComplete)
			}

			for _, path := range tt.wantIncomplete {
				found := false
				for _, field := range state.IncompleteFields {
					if field.JSONPath == path {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected incomplete field %q, got: %v", path, state.IncompleteFields)
				}
			}
		})
	}
}

func TestUnmarshalPartial_CustomTags(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantFirstName  string
		wantComplete   bool
		wantIncomplete []string
	}{
		{
			name:           "truncated_snake_case",
			input:          `{"first_name": "Jo`,
			wantFirstName:  "Jo",
			wantComplete:   false,
			wantIncomplete: []string{"first_name"},
		},
		{
			name:          "complete_snake_case",
			input:         `{"first_name": "John", "last_name": "Doe", "email_addr": "j@d.com"}`,
			wantFirstName: "John",
			wantComplete:  true,
		},
	}

	validator := godantic.NewValidator[TUserCustomTags]()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, state, _ := validator.UnmarshalPartial([]byte(tt.input))

			if result == nil {
				t.Fatal("expected result")
			}
			if result.FirstName != tt.wantFirstName {
				t.Errorf("FirstName = %q, want %q", result.FirstName, tt.wantFirstName)
			}
			if state.IsComplete != tt.wantComplete {
				t.Errorf("IsComplete = %v, want %v", state.IsComplete, tt.wantComplete)
			}

			for _, path := range tt.wantIncomplete {
				found := false
				for _, field := range state.IncompleteFields {
					if field.JSONPath == path {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected incomplete field %q", path)
				}
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// MarshalPartial - Nested Structs
// ═══════════════════════════════════════════════════════════════════════════

func TestUnmarshalPartial_Nested(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantName       string
		wantStreet     string
		wantCity       string
		wantComplete   bool
		wantIncomplete []string
	}{
		{
			name:           "truncated_nested_field",
			input:          `{"name": "John", "address": {"street": "123 Main St", "city": "Ne`,
			wantName:       "John",
			wantStreet:     "123 Main St",
			wantCity:       "Ne",
			wantComplete:   false,
			wantIncomplete: []string{"address.city"},
		},
		{
			name:         "complete_nested",
			input:        `{"name": "John", "address": {"street": "123 Main St", "city": "New York"}}`,
			wantName:     "John",
			wantStreet:   "123 Main St",
			wantCity:     "New York",
			wantComplete: true,
		},
	}

	validator := godantic.NewValidator[TUserWithAddress]()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, state, _ := validator.UnmarshalPartial([]byte(tt.input))

			if result == nil {
				t.Fatal("expected result")
			}
			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}
			if result.Address.Street != tt.wantStreet {
				t.Errorf("Street = %q, want %q", result.Address.Street, tt.wantStreet)
			}
			if result.Address.City != tt.wantCity {
				t.Errorf("City = %q, want %q", result.Address.City, tt.wantCity)
			}
			if state.IsComplete != tt.wantComplete {
				t.Errorf("IsComplete = %v, want %v", state.IsComplete, tt.wantComplete)
			}

			for _, path := range tt.wantIncomplete {
				found := false
				for _, field := range state.IncompleteFields {
					if field.JSONPath == path {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected incomplete field %q, got: %v", path, state.IncompleteFields)
				}
			}
		})
	}
}

func TestUnmarshalPartial_DeepNested(t *testing.T) {
	validator := godantic.NewValidator[TDeepConfig]()

	tests := []struct {
		name           string
		input          string
		wantValue      string
		wantComplete   bool
		wantIncomplete []string
	}{
		{
			name:           "truncated_deep",
			input:          `{"level1": {"level2": {"level3": {"value": "tes`,
			wantValue:      "tes",
			wantComplete:   false,
			wantIncomplete: []string{"level1.level2.level3.value"},
		},
		{
			name:         "complete_deep",
			input:        `{"level1": {"level2": {"level3": {"value": "test"}}}}`,
			wantValue:    "test",
			wantComplete: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, state, _ := validator.UnmarshalPartial([]byte(tt.input))

			if result == nil {
				t.Fatal("expected result")
			}
			if result.Level1.Level2.Level3.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", result.Level1.Level2.Level3.Value, tt.wantValue)
			}
			if state.IsComplete != tt.wantComplete {
				t.Errorf("IsComplete = %v, want %v", state.IsComplete, tt.wantComplete)
			}

			for _, path := range tt.wantIncomplete {
				found := false
				for _, field := range state.IncompleteFields {
					if field.JSONPath == path {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected incomplete field %q, got: %v", path, state.IncompleteFields)
				}
			}
		})
	}
}

func TestUnmarshalPartial_Pointers(t *testing.T) {
	validator := godantic.NewValidator[TUserWithPointers]()

	t.Run("truncated_pointer", func(t *testing.T) {
		result, state, _ := validator.UnmarshalPartial([]byte(`{"name": "Jo`))
		if result == nil {
			t.Fatal("expected result")
		}
		if result.Name == nil || *result.Name != "Jo" {
			t.Errorf("Name = %v, want 'Jo'", result.Name)
		}
		if state.IsComplete {
			t.Error("expected incomplete")
		}
	})

	t.Run("nil_pointer", func(t *testing.T) {
		result, _, _ := validator.UnmarshalPartial([]byte(`{"age": 30}`))
		if result == nil {
			t.Fatal("expected result")
		}
		if result.Name != nil {
			t.Error("expected name to be nil")
		}
		if result.Age == nil || *result.Age != 30 {
			t.Errorf("Age = %v, want 30", result.Age)
		}
	})

	t.Run("complete_pointers", func(t *testing.T) {
		result, state, errs := validator.UnmarshalPartial([]byte(`{"name": "John", "age": 30}`))
		if result == nil {
			t.Fatal("expected result")
		}
		if result.Name == nil || *result.Name != "John" {
			t.Errorf("Name = %v, want 'John'", result.Name)
		}
		if result.Age == nil || *result.Age != 30 {
			t.Errorf("Age = %v, want 30", result.Age)
		}
		if !state.IsComplete {
			t.Error("expected complete")
		}
		if len(errs) != 0 {
			t.Errorf("unexpected errors: %v", errs)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// MarshalPartial - Collections (Slices & Maps)
// ═══════════════════════════════════════════════════════════════════════════

func TestUnmarshalPartial_Slices(t *testing.T) {
	validator := godantic.NewValidator[TUserWithSlice]()

	t.Run("truncated_slice_element", func(t *testing.T) {
		result, state, _ := validator.UnmarshalPartial([]byte(`{"name": "John", "tags": ["tag1", "tag`))
		if result == nil {
			t.Fatal("expected result")
		}
		if result.Name != "John" {
			t.Errorf("Name = %q, want 'John'", result.Name)
		}
		if len(result.Tags) == 0 {
			t.Error("expected at least one tag")
		}
		if state.IsComplete {
			t.Error("expected incomplete")
		}
	})

	t.Run("truncated_int_slice", func(t *testing.T) {
		result, _, _ := validator.UnmarshalPartial([]byte(`{"name": "John", "ids": [1, 2,`))
		if result == nil {
			t.Fatal("expected result")
		}
		if len(result.IDs) != 2 {
			t.Errorf("IDs len = %d, want 2", len(result.IDs))
		}
	})

	t.Run("complete_slices", func(t *testing.T) {
		result, state, errs := validator.UnmarshalPartial([]byte(`{"name": "John", "tags": ["a", "b"], "ids": [1, 2, 3]}`))
		if result == nil {
			t.Fatal("expected result")
		}
		if len(result.Tags) != 2 {
			t.Errorf("Tags len = %d, want 2", len(result.Tags))
		}
		if len(result.IDs) != 3 {
			t.Errorf("IDs len = %d, want 3", len(result.IDs))
		}
		if !state.IsComplete {
			t.Error("expected complete")
		}
		if len(errs) != 0 {
			t.Errorf("unexpected errors: %v", errs)
		}
	})

	t.Run("truncated_struct_in_slice", func(t *testing.T) {
		result, state, _ := validator.UnmarshalPartial([]byte(`{"name": "John", "items": [{"id": 1, "name": "Item1"}, {"id": 2, "name": "Ite`))
		if result == nil {
			t.Fatal("expected result")
		}
		if len(result.Items) == 0 {
			t.Error("expected at least one item")
		}
		if state.IsComplete {
			t.Error("expected incomplete")
		}
	})
}

func TestUnmarshalPartial_Maps(t *testing.T) {
	validator := godantic.NewValidator[TUserWithMap]()

	t.Run("truncated_map_key", func(t *testing.T) {
		result, state, _ := validator.UnmarshalPartial([]byte(`{"name": "John", "metadata": {"ke`))
		if result == nil {
			t.Fatal("expected result")
		}
		if result.Name != "John" {
			t.Errorf("Name = %q, want 'John'", result.Name)
		}
		if state.IsComplete {
			t.Error("expected incomplete")
		}
	})

	t.Run("truncated_map_value", func(t *testing.T) {
		result, state, _ := validator.UnmarshalPartial([]byte(`{"name": "John", "metadata": {"key": "valu`))
		if result == nil {
			t.Fatal("expected result")
		}
		if result.Metadata == nil {
			t.Error("expected metadata map")
		}
		if state.IsComplete {
			t.Error("expected incomplete")
		}
	})

	t.Run("complete_maps", func(t *testing.T) {
		result, state, errs := validator.UnmarshalPartial([]byte(`{"name": "John", "metadata": {"key": "value"}, "scores": {"math": 95}}`))
		if result == nil {
			t.Fatal("expected result")
		}
		if result.Metadata["key"] != "value" {
			t.Errorf("Metadata[key] = %q, want 'value'", result.Metadata["key"])
		}
		if result.Scores["math"] != 95 {
			t.Errorf("Scores[math] = %d, want 95", result.Scores["math"])
		}
		if !state.IsComplete {
			t.Error("expected complete")
		}
		if len(errs) != 0 {
			t.Errorf("unexpected errors: %v", errs)
		}
	})
}

func TestUnmarshalPartial_MapWithNestedStruct(t *testing.T) {
	validator := godantic.NewValidator[TConfigWithNestedMap]()

	t.Run("truncated", func(t *testing.T) {
		result, state, _ := validator.UnmarshalPartial([]byte(`{"settings": {"key1": {"value": "tes`))
		if result == nil {
			t.Fatal("expected result")
		}
		if result.Settings == nil {
			t.Error("expected settings map")
		}
		if state.IsComplete {
			t.Error("expected incomplete")
		}
	})

	t.Run("complete", func(t *testing.T) {
		result, state, errs := validator.UnmarshalPartial([]byte(`{"settings": {"key1": {"value": "test", "count": 42}}}`))
		if result == nil {
			t.Fatal("expected result")
		}
		if result.Settings["key1"].Value != "test" {
			t.Errorf("Value = %q, want 'test'", result.Settings["key1"].Value)
		}
		if result.Settings["key1"].Count != 42 {
			t.Errorf("Count = %d, want 42", result.Settings["key1"].Count)
		}
		if !state.IsComplete {
			t.Error("expected complete")
		}
		if len(errs) != 0 {
			t.Errorf("unexpected errors: %v", errs)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// MarshalPartial - Discriminated Unions
// ═══════════════════════════════════════════════════════════════════════════

func TestUnmarshalPartial_Union(t *testing.T) {
	validator := NewTAnimalValidator()

	t.Run("partial_discriminator", func(t *testing.T) {
		_, state, _ := validator.UnmarshalPartial([]byte(`{"species": "do`))
		if state.IsComplete {
			t.Error("expected incomplete")
		}
		// Should track discriminator as incomplete
		found := false
		for _, f := range state.IncompleteFields {
			if f.JSONPath == "species" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected incomplete 'species', got: %v", state.IncompleteFields)
		}
	})

	t.Run("complete_discriminator_partial_field", func(t *testing.T) {
		result, state, _ := validator.UnmarshalPartial([]byte(`{"species": "dog", "name": "Bu`))
		if result == nil {
			t.Fatal("expected result")
		}
		dog, ok := (*result).(*TDog)
		if !ok {
			t.Fatalf("expected *TDog, got %T", *result)
		}
		if dog.Species != TSpeciesDog {
			t.Errorf("Species = %v, want 'dog'", dog.Species)
		}
		if dog.Name != "Bu" {
			t.Errorf("Name = %q, want 'Bu'", dog.Name)
		}
		if state.IsComplete {
			t.Error("expected incomplete")
		}
	})

	t.Run("complete", func(t *testing.T) {
		result, state, errs := validator.UnmarshalPartial([]byte(`{"species": "dog", "name": "Buddy", "breed": "Golden", "is_good": true}`))
		if result == nil {
			t.Fatal("expected result")
		}
		dog, ok := (*result).(*TDog)
		if !ok {
			t.Fatalf("expected *TDog, got %T", *result)
		}
		if dog.Name != "Buddy" {
			t.Errorf("Name = %q, want 'Buddy'", dog.Name)
		}
		if dog.Breed != "Golden" {
			t.Errorf("Breed = %q, want 'Golden'", dog.Breed)
		}
		if !state.IsComplete {
			t.Error("expected complete")
		}
		if len(errs) != 0 {
			t.Errorf("unexpected errors: %v", errs)
		}
	})

	t.Run("missing_discriminator", func(t *testing.T) {
		_, state, errs := validator.UnmarshalPartial([]byte(`{"name": "Bu`))
		if state.IsComplete {
			t.Error("expected incomplete")
		}
		// Should have discriminator missing error
		found := false
		for _, err := range errs {
			if err.Type == godantic.ErrorTypeDiscriminatorMissing {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected discriminator_missing error, got: %v", errs)
		}
	})
}

func TestUnmarshalPartial_UnionArray(t *testing.T) {
	validator := godantic.NewValidator[TAnimalList]()

	t.Run("truncated_element", func(t *testing.T) {
		result, state, _ := validator.UnmarshalPartial([]byte(`{"animals": [{"type": "dog", "name": "Re`))
		if result == nil {
			t.Fatal("expected result")
		}
		if len(result.Animals) == 0 {
			t.Error("expected at least one animal")
		}
		if result.Animals[0].Name != "Re" {
			t.Errorf("Name = %q, want 'Re'", result.Animals[0].Name)
		}
		if state.IsComplete {
			t.Error("expected incomplete")
		}
	})

	t.Run("complete", func(t *testing.T) {
		result, state, errs := validator.UnmarshalPartial([]byte(`{"animals": [{"type": "dog", "name": "Rex"}, {"type": "cat", "name": "Fluffy"}]}`))
		if result == nil {
			t.Fatal("expected result")
		}
		if len(result.Animals) != 2 {
			t.Errorf("Animals len = %d, want 2", len(result.Animals))
		}
		if !state.IsComplete {
			t.Error("expected complete")
		}
		if len(errs) != 0 {
			t.Errorf("unexpected errors: %v", errs)
		}
	})
}

func TestUnmarshalPartial_NestedUnion(t *testing.T) {
	validator := godantic.NewValidator[TNestedAnimal]()

	t.Run("truncated_nested", func(t *testing.T) {
		result, state, _ := validator.UnmarshalPartial([]byte(`{"type": "dog", "name": "Rex", "details": {"type": "ca`))
		if result == nil {
			t.Fatal("expected result")
		}
		if result.Type != "dog" {
			t.Errorf("Type = %q, want 'dog'", result.Type)
		}
		if state.IsComplete {
			t.Error("expected incomplete")
		}
	})

	t.Run("complete_nested", func(t *testing.T) {
		result, state, errs := validator.UnmarshalPartial([]byte(`{"type": "dog", "name": "Rex", "details": {"type": "cat", "name": "Fluffy"}}`))
		if result == nil {
			t.Fatal("expected result")
		}
		if result.Details.Type != "cat" {
			t.Errorf("Details.Type = %q, want 'cat'", result.Details.Type)
		}
		if !state.IsComplete {
			t.Error("expected complete")
		}
		if len(errs) != 0 {
			t.Errorf("unexpected errors: %v", errs)
		}
	})
}

func TestUnmarshalPartial_CustomTaggedUnion(t *testing.T) {
	validator := godantic.NewValidator[TCustomTaggedAnimal]()

	t.Run("truncated_custom_tag", func(t *testing.T) {
		result, state, _ := validator.UnmarshalPartial([]byte(`{"animal_type": "do`))
		if result == nil {
			t.Fatal("expected result")
		}
		if state.IsComplete {
			t.Error("expected incomplete")
		}
		// Check uses custom JSON tag
		found := false
		for _, f := range state.IncompleteFields {
			if f.JSONPath == "animal_type" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected incomplete 'animal_type', got: %v", state.IncompleteFields)
		}
	})

	t.Run("complete_custom_tag", func(t *testing.T) {
		result, state, errs := validator.UnmarshalPartial([]byte(`{"animal_type": "dog", "name": "Rex"}`))
		if result == nil {
			t.Fatal("expected result")
		}
		if result.AnimalType != "dog" {
			t.Errorf("AnimalType = %q, want 'dog'", result.AnimalType)
		}
		if !state.IsComplete {
			t.Error("expected complete")
		}
		if len(errs) != 0 {
			t.Errorf("unexpected errors: %v", errs)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// PartialState API Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestPartialState_WaitingFor(t *testing.T) {
	validator := godantic.NewValidator[TUser]()
	_, state, _ := validator.UnmarshalPartial([]byte(`{"name": "John", "age":`))

	waiting := state.WaitingFor()
	if len(waiting) == 0 {
		t.Error("expected waiting fields")
	}

	found := false
	for _, path := range waiting {
		if path == "age" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'age' in waiting list, got: %v", waiting)
	}
}

func TestPartialState_IsFieldComplete(t *testing.T) {
	validator := godantic.NewValidator[TUser]()
	_, state, _ := validator.UnmarshalPartial([]byte(`{"name": "John", "age":`))

	if !state.IsFieldComplete("name") {
		t.Error("expected 'name' to be complete")
	}

	if state.IsFieldComplete("age") {
		t.Error("expected 'age' to be incomplete")
	}
}
