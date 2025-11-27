package godantic_test

import "github.com/deepankarm/godantic/pkg/godantic"

// ═══════════════════════════════════════════════════════════════════════════
// SHARED TEST FIXTURES
// All test types use "T" prefix for easy identification and grep-ability.
// Import this file's types across all test files in the package.
// ═══════════════════════════════════════════════════════════════════════════

// ───────────────────────────────────────────────────────────────────────────
// Basic Types
// ───────────────────────────────────────────────────────────────────────────

// TUser is a basic user struct with common fields.
type TUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func (u *TUser) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (u *TUser) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// TUserCustomTags has snake_case JSON tags.
type TUserCustomTags struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	EmailAddr string `json:"email_addr"`
}

func (u *TUserCustomTags) FieldFirstName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (u *TUserCustomTags) FieldEmailAddr() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// ───────────────────────────────────────────────────────────────────────────
// Nested Structs
// ───────────────────────────────────────────────────────────────────────────

// TAddress is a nested struct for testing.
type TAddress struct {
	Street string `json:"street"`
	City   string `json:"city"`
}

func (a *TAddress) FieldStreet() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (a *TAddress) FieldCity() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// TUserWithAddress has a nested TAddress.
type TUserWithAddress struct {
	Name    string   `json:"name"`
	Address TAddress `json:"address"`
}

func (u *TUserWithAddress) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (u *TUserWithAddress) FieldAddress() godantic.FieldOptions[TAddress] {
	return godantic.Field(godantic.Required[TAddress]())
}

// TUserWithPointers has pointer fields.
type TUserWithPointers struct {
	Name    *string   `json:"name"`
	Age     *int      `json:"age"`
	Address *TAddress `json:"address"`
}

func (u *TUserWithPointers) FieldName() godantic.FieldOptions[*string] {
	return godantic.Field(godantic.Required[*string]())
}

// TDeepConfig has 3+ levels of nesting.
type TDeepConfig struct {
	Level1 struct {
		Level2 struct {
			Level3 struct {
				Value string `json:"value"`
			} `json:"level3"`
		} `json:"level2"`
	} `json:"level1"`
}

func (d *TDeepConfig) FieldLevel1() godantic.FieldOptions[struct {
	Level2 struct {
		Level3 struct {
			Value string `json:"value"`
		} `json:"level3"`
	} `json:"level2"`
}] {
	return godantic.Field(godantic.Required[struct {
		Level2 struct {
			Level3 struct {
				Value string `json:"value"`
			} `json:"level3"`
		} `json:"level2"`
	}]())
}

// ───────────────────────────────────────────────────────────────────────────
// Collections (Slices & Maps)
// ───────────────────────────────────────────────────────────────────────────

// TItem is used in slice tests.
type TItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TUserWithSlice has slice fields.
type TUserWithSlice struct {
	Name  string   `json:"name"`
	Tags  []string `json:"tags"`
	IDs   []int    `json:"ids"`
	Items []TItem  `json:"items"`
}

func (u *TUserWithSlice) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// TUserWithMap has map fields.
type TUserWithMap struct {
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
	Scores   map[string]int    `json:"scores"`
}

func (u *TUserWithMap) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// TConfigWithNestedMap has map with nested struct values.
type TConfigWithNestedMap struct {
	Settings map[string]struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	} `json:"settings"`
}

// ───────────────────────────────────────────────────────────────────────────
// Discriminated Unions
// ───────────────────────────────────────────────────────────────────────────

// TAnimalSpecies is the discriminator enum.
type TAnimalSpecies string

const (
	TSpeciesCat  TAnimalSpecies = "cat"
	TSpeciesDog  TAnimalSpecies = "dog"
	TSpeciesBird TAnimalSpecies = "bird"
)

// TAnimal is the discriminated union interface.
type TAnimal interface {
	GetSpecies() TAnimalSpecies
	isAnimal()
}

// TCat implements TAnimal.
type TCat struct {
	Species   TAnimalSpecies `json:"species"`
	Name      string         `json:"name"`
	LivesLeft int            `json:"lives_left"`
	IsIndoor  bool           `json:"is_indoor"`
}

func (c TCat) GetSpecies() TAnimalSpecies { return c.Species }
func (c TCat) isAnimal()                  {}

func (c *TCat) FieldSpecies() godantic.FieldOptions[TAnimalSpecies] {
	return godantic.Field(godantic.Required[TAnimalSpecies](), godantic.Const(TSpeciesCat))
}

func (c *TCat) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(1))
}

func (c *TCat) FieldLivesLeft() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Required[int](), godantic.Min(0), godantic.Max(9))
}

// TDog implements TAnimal.
type TDog struct {
	Species TAnimalSpecies `json:"species"`
	Name    string         `json:"name"`
	Breed   string         `json:"breed"`
	IsGood  bool           `json:"is_good"`
}

func (d TDog) GetSpecies() TAnimalSpecies { return d.Species }
func (d TDog) isAnimal()                  {}

func (d *TDog) FieldSpecies() godantic.FieldOptions[TAnimalSpecies] {
	return godantic.Field(godantic.Required[TAnimalSpecies](), godantic.Const(TSpeciesDog))
}

func (d *TDog) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(1))
}

func (d *TDog) FieldBreed() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// TBird implements TAnimal.
type TBird struct {
	Species  TAnimalSpecies `json:"species"`
	Name     string         `json:"name"`
	CanFly   bool           `json:"can_fly"`
	Wingspan float64        `json:"wingspan"`
}

func (b TBird) GetSpecies() TAnimalSpecies { return b.Species }
func (b TBird) isAnimal()                  {}

func (b *TBird) FieldSpecies() godantic.FieldOptions[TAnimalSpecies] {
	return godantic.Field(godantic.Required[TAnimalSpecies](), godantic.Const(TSpeciesBird))
}

func (b *TBird) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(1))
}

func (b *TBird) FieldWingspan() godantic.FieldOptions[float64] {
	return godantic.Field(godantic.Required[float64](), godantic.Min(0.0))
}

// TSimpleAnimal is a non-interface union type for array tests.
type TSimpleAnimal struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func (a *TSimpleAnimal) FieldType() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// TAnimalList contains a slice of TSimpleAnimal.
type TAnimalList struct {
	Animals []TSimpleAnimal `json:"animals"`
}

func (a *TAnimalList) FieldAnimals() godantic.FieldOptions[[]TSimpleAnimal] {
	return godantic.Field(godantic.Required[[]TSimpleAnimal]())
}

// TNestedAnimal has nested unions.
type TNestedAnimal struct {
	Type    string         `json:"type"`
	Name    string         `json:"name"`
	Details TSimpleAnimal  `json:"details"`
	Parent  *TSimpleAnimal `json:"parent"`
}

func (n *TNestedAnimal) FieldType() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// TCustomTaggedAnimal has snake_case discriminator.
type TCustomTaggedAnimal struct {
	AnimalType string `json:"animal_type"`
	Name       string `json:"name"`
}

func (c *TCustomTaggedAnimal) FieldAnimalType() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// ───────────────────────────────────────────────────────────────────────────
// Map Validator Types (for ValidateFromStringMap / ValidateFromMultiValueMap)
// ───────────────────────────────────────────────────────────────────────────

// TPathParams represents typical URL path parameters.
type TPathParams struct {
	ID     int    `json:"id"`
	Slug   string `json:"slug"`
	Active bool   `json:"active"`
}

func (p *TPathParams) FieldID() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Required[int](), godantic.Min(1))
}

func (p *TPathParams) FieldSlug() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(1))
}

// TQueryParams represents typical URL query parameters.
type TQueryParams struct {
	Page    int      `json:"page"`
	Limit   int      `json:"limit"`
	Tags    []string `json:"tags"`
	Enabled bool     `json:"enabled"`
	Score   float64  `json:"score"`
}

func (q *TQueryParams) FieldPage() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Default(1), godantic.Min(1))
}

func (q *TQueryParams) FieldLimit() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Default(10), godantic.Min(1), godantic.Max(100))
}

// ───────────────────────────────────────────────────────────────────────────
// Helper functions for creating test validators
// ───────────────────────────────────────────────────────────────────────────

// NewTAnimalValidator creates a validator for TAnimal discriminated union.
func NewTAnimalValidator() *godantic.Validator[TAnimal] {
	return godantic.NewValidator[TAnimal](
		godantic.WithDiscriminatorTyped("species", map[TAnimalSpecies]any{
			TSpeciesCat:  &TCat{},
			TSpeciesDog:  &TDog{},
			TSpeciesBird: &TBird{},
		}),
	)
}
