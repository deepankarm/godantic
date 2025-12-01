package godantic_test

import (
	"encoding/json"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// ═══════════════════════════════════════════════════════════════════════════
// Discriminated Union - Marshal (JSON → Struct)
// Uses shared TAnimal, TCat, TDog, TBird fixtures from testdata_test.go
// ═══════════════════════════════════════════════════════════════════════════

func TestUnion_Marshal(t *testing.T) {
	validator := NewTAnimalValidator()

	tests := []struct {
		name        string
		json        string
		wantSpecies TAnimalSpecies
		wantName    string
		wantType    string // "cat", "dog", "bird"
		wantErr     bool
		errType     string
	}{
		{
			name:        "cat",
			json:        `{"species": "cat", "name": "Whiskers", "lives_left": 7, "is_indoor": true}`,
			wantSpecies: TSpeciesCat,
			wantName:    "Whiskers",
			wantType:    "cat",
		},
		{
			name:        "dog",
			json:        `{"species": "dog", "name": "Buddy", "breed": "Golden Retriever", "is_good": true}`,
			wantSpecies: TSpeciesDog,
			wantName:    "Buddy",
			wantType:    "dog",
		},
		{
			name:        "bird",
			json:        `{"species": "bird", "name": "Tweety", "can_fly": true, "wingspan": 0.25}`,
			wantSpecies: TSpeciesBird,
			wantName:    "Tweety",
			wantType:    "bird",
		},
		{
			name:    "invalid_discriminator",
			json:    `{"species": "fish", "name": "Nemo"}`,
			wantErr: true,
			errType: "discriminator_invalid",
		},
		{
			name:    "missing_discriminator",
			json:    `{"name": "Mystery"}`,
			wantErr: true,
			errType: "discriminator_missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			animal, errs := validator.Unmarshal([]byte(tt.json))

			if tt.wantErr {
				if len(errs) == 0 {
					t.Fatal("expected error, got none")
				}
				if errs[0].Type != godantic.ErrorType(tt.errType) {
					t.Errorf("error type = %s, want %s", errs[0].Type, tt.errType)
				}
				return
			}

			if len(errs) != 0 {
				t.Fatalf("unexpected errors: %v", errs)
			}
			if animal == nil {
				t.Fatal("expected result")
			}

			// Type-specific assertions
			switch tt.wantType {
			case "cat":
				cat, ok := (*animal).(*TCat)
				if !ok {
					t.Fatalf("expected *TCat, got %T", *animal)
				}
				if cat.Name != tt.wantName {
					t.Errorf("Name = %q, want %q", cat.Name, tt.wantName)
				}
				if cat.GetSpecies() != tt.wantSpecies {
					t.Errorf("Species = %v, want %v", cat.GetSpecies(), tt.wantSpecies)
				}
			case "dog":
				dog, ok := (*animal).(*TDog)
				if !ok {
					t.Fatalf("expected *TDog, got %T", *animal)
				}
				if dog.Name != tt.wantName {
					t.Errorf("Name = %q, want %q", dog.Name, tt.wantName)
				}
			case "bird":
				bird, ok := (*animal).(*TBird)
				if !ok {
					t.Fatalf("expected *TBird, got %T", *animal)
				}
				if bird.Name != tt.wantName {
					t.Errorf("Name = %q, want %q", bird.Name, tt.wantName)
				}
			}
		})
	}
}

func TestUnion_Marshal_Validation(t *testing.T) {
	validator := NewTAnimalValidator()

	tests := []struct {
		name     string
		json     string
		errType  string
		errField string
	}{
		{
			name:     "cat_lives_exceed_max",
			json:     `{"species": "cat", "name": "Whiskers", "lives_left": 15}`,
			errType:  "constraint",
			errField: "LivesLeft",
		},
		{
			name:     "dog_missing_breed",
			json:     `{"species": "dog", "name": "Buddy"}`,
			errType:  "required",
			errField: "Breed",
		},
		{
			name:     "cat_missing_name",
			json:     `{"species": "cat", "lives_left": 5}`,
			errType:  "required",
			errField: "Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := validator.Unmarshal([]byte(tt.json))
			if len(errs) == 0 {
				t.Fatal("expected validation error")
			}

			found := false
			for _, err := range errs {
				if string(err.Type) == tt.errType && len(err.Loc) > 0 && err.Loc[len(err.Loc)-1] == tt.errField {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %s error on %s, got: %v", tt.errType, tt.errField, errs)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Discriminated Union - Unmarshal (Struct → JSON)
// ═══════════════════════════════════════════════════════════════════════════

func TestUnion_Unmarshal(t *testing.T) {
	validator := NewTAnimalValidator()

	tests := []struct {
		name        string
		animal      TAnimal
		wantSpecies string
		wantName    string
		wantErr     bool
		errType     string
	}{
		{
			name:        "cat",
			animal:      &TCat{Species: TSpeciesCat, Name: "Whiskers", LivesLeft: 7, IsIndoor: true},
			wantSpecies: "cat",
			wantName:    "Whiskers",
		},
		{
			name:        "dog",
			animal:      &TDog{Species: TSpeciesDog, Name: "Buddy", Breed: "Golden", IsGood: true},
			wantSpecies: "dog",
			wantName:    "Buddy",
		},
		{
			name:        "bird",
			animal:      &TBird{Species: TSpeciesBird, Name: "Tweety", CanFly: true, Wingspan: 0.25},
			wantSpecies: "bird",
			wantName:    "Tweety",
		},
		{
			name:    "invalid_discriminator",
			animal:  &TCat{Species: "invalid", Name: "Bad", LivesLeft: 5},
			wantErr: true,
			errType: "discriminator_invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, errs := validator.Marshal(&tt.animal)

			if tt.wantErr {
				if len(errs) == 0 {
					t.Fatal("expected error")
				}
				if string(errs[0].Type) != tt.errType {
					t.Errorf("error type = %s, want %s", errs[0].Type, tt.errType)
				}
				return
			}

			if len(errs) != 0 {
				t.Fatalf("unexpected errors: %v", errs)
			}

			var result map[string]any
			if err := json.Unmarshal(jsonData, &result); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			if result["species"] != tt.wantSpecies {
				t.Errorf("species = %v, want %s", result["species"], tt.wantSpecies)
			}
			if result["name"] != tt.wantName {
				t.Errorf("name = %v, want %s", result["name"], tt.wantName)
			}
		})
	}
}

func TestUnion_Unmarshal_Validation(t *testing.T) {
	validator := NewTAnimalValidator()

	tests := []struct {
		name     string
		animal   TAnimal
		errType  string
		errField string
	}{
		{
			name:     "cat_lives_exceed_max",
			animal:   &TCat{Species: TSpeciesCat, Name: "Bad", LivesLeft: 15},
			errType:  "constraint",
			errField: "LivesLeft",
		},
		{
			name:     "dog_missing_breed",
			animal:   &TDog{Species: TSpeciesDog, Name: "Buddy"},
			errType:  "required",
			errField: "Breed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := validator.Marshal(&tt.animal)
			if len(errs) == 0 {
				t.Fatal("expected validation error")
			}

			found := false
			for _, err := range errs {
				if string(err.Type) == tt.errType && len(err.Loc) > 0 && err.Loc[len(err.Loc)-1] == tt.errField {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %s error on %s, got: %v", tt.errType, tt.errField, errs)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Discriminated Union - String Keys (non-typed variant)
// ═══════════════════════════════════════════════════════════════════════════

func TestUnion_StringKeys(t *testing.T) {
	validator := godantic.NewValidator[TAnimal](
		godantic.WithDiscriminator("species", map[string]any{
			"cat":  &TCat{},
			"dog":  &TDog{},
			"bird": &TBird{},
		}),
	)

	animal, errs := validator.Unmarshal([]byte(`{"species": "cat", "name": "Mittens", "lives_left": 9}`))
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	cat, ok := (*animal).(*TCat)
	if !ok {
		t.Fatalf("expected *TCat, got %T", *animal)
	}
	if cat.Name != "Mittens" {
		t.Errorf("Name = %q, want 'Mittens'", cat.Name)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Discriminated Union - Defaults
// ═══════════════════════════════════════════════════════════════════════════

// TCatWithDefaults has default values for testing
type TCatWithDefaults struct {
	Species   TAnimalSpecies `json:"species"`
	Name      string         `json:"name"`
	LivesLeft int            `json:"lives_left"`
	Color     string         `json:"color"`
}

func (c TCatWithDefaults) GetSpecies() TAnimalSpecies { return c.Species }
func (c TCatWithDefaults) isAnimal()                  {}

func (c *TCatWithDefaults) FieldSpecies() godantic.FieldOptions[TAnimalSpecies] {
	return godantic.Field(godantic.Required[TAnimalSpecies](), godantic.Const(TSpeciesCat))
}

func (c *TCatWithDefaults) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (c *TCatWithDefaults) FieldLivesLeft() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Default(9), godantic.Min(0), godantic.Max(9))
}

func (c *TCatWithDefaults) FieldColor() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Default("orange"))
}

func TestUnion_Unmarshal_AppliesDefaults(t *testing.T) {
	validator := godantic.NewValidator[TAnimal](
		godantic.WithDiscriminatorTyped("species", map[TAnimalSpecies]any{
			TSpeciesCat: &TCatWithDefaults{},
		}),
	)

	cat := TCatWithDefaults{Species: TSpeciesCat, Name: "Garfield"}
	var animal TAnimal = &cat

	jsonData, errs := validator.Marshal(&animal)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["lives_left"].(float64) != 9 {
		t.Errorf("lives_left = %v, want 9 (default)", result["lives_left"])
	}
	if result["color"] != "orange" {
		t.Errorf("color = %v, want 'orange' (default)", result["color"])
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Discriminated Union - Slice of Unions
// Uses types from testdata_test.go where possible
// ═══════════════════════════════════════════════════════════════════════════

// TContentBlock for slice-of-union tests
type TContentBlock interface {
	GetBlockType() string
}

type TTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (t TTextBlock) GetBlockType() string { return t.Type }

type TImageBlock struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func (i TImageBlock) GetBlockType() string { return i.Type }

type TCodeBlock struct {
	Type     string `json:"type"`
	Language string `json:"language"`
	Code     string `json:"code"`
}

func (c TCodeBlock) GetBlockType() string { return c.Type }

type TDocument struct {
	Title  string          `json:"title"`
	Blocks []TContentBlock `json:"blocks"`
}

func (d *TDocument) FieldTitle() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (d *TDocument) FieldBlocks() godantic.FieldOptions[[]TContentBlock] {
	return godantic.Field(
		godantic.Required[[]TContentBlock](),
		godantic.DiscriminatedUnion[[]TContentBlock]("type", map[string]any{
			"text":  TTextBlock{},
			"image": TImageBlock{},
			"code":  TCodeBlock{},
		}),
	)
}

func TestUnion_SliceOfUnions(t *testing.T) {
	validator := godantic.NewValidator[TDocument]()

	t.Run("validate", func(t *testing.T) {
		doc := TDocument{
			Title: "Test",
			Blocks: []TContentBlock{
				TTextBlock{Type: "text", Text: "Hello"},
				TImageBlock{Type: "image", URL: "http://example.com/img.png"},
				TCodeBlock{Type: "code", Language: "go", Code: "fmt.Println()"},
			},
		}
		errs := validator.Validate(&doc)
		if len(errs) != 0 {
			t.Errorf("unexpected errors: %v", errs)
		}
	})

	t.Run("marshal", func(t *testing.T) {
		doc, errs := validator.Unmarshal([]byte(`{
			"title": "Test",
			"blocks": [
				{"type": "text", "text": "Hello"},
				{"type": "image", "url": "http://example.com"},
				{"type": "code", "language": "go", "code": "x := 1"}
			]
		}`))
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}

		if doc.Title != "Test" {
			t.Errorf("Title = %q, want 'Test'", doc.Title)
		}
		if len(doc.Blocks) != 3 {
			t.Fatalf("Blocks len = %d, want 3", len(doc.Blocks))
		}

		if _, ok := doc.Blocks[0].(TTextBlock); !ok {
			t.Errorf("Blocks[0] = %T, want TTextBlock", doc.Blocks[0])
		}
		if _, ok := doc.Blocks[1].(TImageBlock); !ok {
			t.Errorf("Blocks[1] = %T, want TImageBlock", doc.Blocks[1])
		}
		if _, ok := doc.Blocks[2].(TCodeBlock); !ok {
			t.Errorf("Blocks[2] = %T, want TCodeBlock", doc.Blocks[2])
		}
	})

	t.Run("unmarshal", func(t *testing.T) {
		doc := TDocument{
			Title: "Round Trip",
			Blocks: []TContentBlock{
				TTextBlock{Type: "text", Text: "World"},
				TCodeBlock{Type: "code", Language: "python", Code: "print('hi')"},
			},
		}

		jsonData, errs := validator.Marshal(&doc)
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}

		var result map[string]any
		if err := json.Unmarshal(jsonData, &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		blocks := result["blocks"].([]any)
		if len(blocks) != 2 {
			t.Fatalf("blocks len = %d, want 2", len(blocks))
		}

		block0 := blocks[0].(map[string]any)
		if block0["type"] != "text" || block0["text"] != "World" {
			t.Errorf("block[0] = %v, want text/World", block0)
		}
	})
}

// TBlockWithRequired has a required field for validation testing
type TBlockWithRequired struct {
	Type string `json:"type"`
	Data string `json:"data"`
	ID   string `json:"id"`
}

func (b TBlockWithRequired) GetBlockType() string { return b.Type }

func (b *TBlockWithRequired) FieldID() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

type TBlockContainer struct {
	Title string          `json:"title"`
	Items []TContentBlock `json:"items"`
}

func (b *TBlockContainer) FieldTitle() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (b *TBlockContainer) FieldItems() godantic.FieldOptions[[]TContentBlock] {
	return godantic.Field(
		godantic.Required[[]TContentBlock](),
		godantic.DiscriminatedUnion[[]TContentBlock]("type", map[string]any{
			"text":    &TTextBlock{},
			"complex": &TBlockWithRequired{},
		}),
	)
}

func TestUnion_SliceValidation(t *testing.T) {
	validator := godantic.NewValidator[TBlockContainer]()

	t.Run("valid_items", func(t *testing.T) {
		_, errs := validator.Unmarshal([]byte(`{
			"title": "Test",
			"items": [
				{"type": "text", "text": "Hello"},
				{"type": "complex", "data": "Some data", "id": "item-123"}
			]
		}`))
		if len(errs) != 0 {
			t.Errorf("unexpected errors: %v", errs)
		}
	})

	t.Run("missing_required_in_element", func(t *testing.T) {
		_, errs := validator.Unmarshal([]byte(`{
			"title": "Test",
			"items": [{"type": "complex", "data": "Missing ID"}]
		}`))
		if len(errs) == 0 {
			t.Fatal("expected error for missing ID")
		}

		found := false
		for _, err := range errs {
			if len(err.Loc) > 0 && err.Loc[len(err.Loc)-1] == "ID" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error on ID field, got: %v", errs)
		}
	})

	t.Run("multiple_validation_errors", func(t *testing.T) {
		_, errs := validator.Unmarshal([]byte(`{
			"title": "Test",
			"items": [
				{"type": "complex", "data": "First"},
				{"type": "text", "text": "OK"},
				{"type": "complex", "data": "Second"}
			]
		}`))
		if len(errs) < 2 {
			t.Errorf("expected >= 2 errors, got %d: %v", len(errs), errs)
		}
	})
}
