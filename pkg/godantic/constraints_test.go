package godantic_test

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Test URL validation
type URLProfile struct {
	Website string
	API     string
}

func (p *URLProfile) FieldWebsite() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.URL(),
	)
}

func (p *URLProfile) FieldAPI() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.URL(),
	)
}

func TestURLValidation(t *testing.T) {
	validator := godantic.NewValidator[URLProfile]()

	t.Run("valid URLs should pass", func(t *testing.T) {
		profile := URLProfile{
			Website: "https://example.com",
			API:     "https://api.example.com/v1",
		}
		errs := validator.Validate(&profile)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("invalid URL should fail", func(t *testing.T) {
		profile := URLProfile{
			Website: "not-a-url",
		}
		errs := validator.Validate(&profile)
		if len(errs) == 0 {
			t.Error("expected validation error for invalid URL")
		}
	})
}

// Test OneOf (enum validation)
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

type Issue struct {
	Title    string
	Priority Priority
}

func (i *Issue) FieldTitle() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
	)
}

func (i *Issue) FieldPriority() godantic.FieldOptions[Priority] {
	return godantic.Field(
		godantic.Required[Priority](),
		godantic.OneOf(PriorityLow, PriorityMedium, PriorityHigh),
	)
}

func TestOneOfValidation(t *testing.T) {
	validator := godantic.NewValidator[Issue]()

	t.Run("valid enum value should pass", func(t *testing.T) {
		issue := Issue{
			Title:    "Bug Report",
			Priority: PriorityHigh,
		}
		errs := validator.Validate(&issue)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("invalid enum value should fail", func(t *testing.T) {
		issue := Issue{
			Title:    "Bug Report",
			Priority: Priority("critical"), // Not in allowed values
		}
		errs := validator.Validate(&issue)
		if len(errs) == 0 {
			t.Error("expected validation error for invalid enum value")
		}
	})
}

// Test MultipleOf
type Dimension struct {
	Width  int
	Height int
}

func (d *Dimension) FieldWidth() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.MultipleOf(5),
	)
}

func (d *Dimension) FieldHeight() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.MultipleOf(10),
	)
}

func TestMultipleOfValidation(t *testing.T) {
	validator := godantic.NewValidator[Dimension]()

	t.Run("valid multiples should pass", func(t *testing.T) {
		dim := Dimension{
			Width:  25,  // 5 * 5
			Height: 100, // 10 * 10
		}
		errs := validator.Validate(&dim)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("invalid multiple should fail", func(t *testing.T) {
		dim := Dimension{
			Width:  23, // Not a multiple of 5
			Height: 100,
		}
		errs := validator.Validate(&dim)
		if len(errs) == 0 {
			t.Error("expected validation error for invalid multiple")
		}
	})
}

// Test ExclusiveMin and ExclusiveMax
type Score struct {
	Percentage float64
	Grade      int
}

func (s *Score) FieldPercentage() godantic.FieldOptions[float64] {
	return godantic.Field(
		godantic.Required[float64](),
		godantic.ExclusiveMin(0.0),   // Must be > 0, not >= 0
		godantic.ExclusiveMax(100.0), // Must be < 100, not <= 100
	)
}

func (s *Score) FieldGrade() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.ExclusiveMin(0),
		godantic.ExclusiveMax(5),
	)
}

func TestExclusiveMinMaxValidation(t *testing.T) {
	validator := godantic.NewValidator[Score]()

	t.Run("valid values within exclusive bounds should pass", func(t *testing.T) {
		score := Score{
			Percentage: 85.5,
			Grade:      4,
		}
		errs := validator.Validate(&score)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("value equal to exclusive minimum should fail", func(t *testing.T) {
		score := Score{
			Percentage: 0.0, // Equal to exclusive min
			Grade:      2,
		}
		errs := validator.Validate(&score)
		if len(errs) == 0 {
			t.Error("expected validation error for value equal to exclusive minimum")
		}
	})

	t.Run("value equal to exclusive maximum should fail", func(t *testing.T) {
		score := Score{
			Percentage: 100.0, // Equal to exclusive max
			Grade:      2,
		}
		errs := validator.Validate(&score)
		if len(errs) == 0 {
			t.Error("expected validation error for value equal to exclusive maximum")
		}
	})
}

// Test MinItems, MaxItems, UniqueItems
type Playlist struct {
	Songs     []string
	TopFive   []string
	UniqueIDs []int
}

func (p *Playlist) FieldSongs() godantic.FieldOptions[[]string] {
	return godantic.Field(
		godantic.Required[[]string](),
		godantic.MinItems[string](1),
	)
}

func (p *Playlist) FieldTopFive() godantic.FieldOptions[[]string] {
	return godantic.Field(
		godantic.MaxItems[string](5),
	)
}

func (p *Playlist) FieldUniqueIDs() godantic.FieldOptions[[]int] {
	return godantic.Field(
		godantic.UniqueItems[int](),
	)
}

func TestArrayConstraints(t *testing.T) {
	validator := godantic.NewValidator[Playlist]()

	t.Run("valid arrays should pass", func(t *testing.T) {
		playlist := Playlist{
			Songs:     []string{"Song 1", "Song 2"},
			TopFive:   []string{"A", "B", "C"},
			UniqueIDs: []int{1, 2, 3, 4},
		}
		errs := validator.Validate(&playlist)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("empty array should fail MinItems", func(t *testing.T) {
		playlist := Playlist{
			Songs:     []string{}, // Empty, violates MinItems(1)
			UniqueIDs: []int{1},
		}
		errs := validator.Validate(&playlist)
		if len(errs) == 0 {
			t.Error("expected validation error for empty array")
		}
	})

	t.Run("array exceeding MaxItems should fail", func(t *testing.T) {
		playlist := Playlist{
			Songs:     []string{"A"},
			TopFive:   []string{"1", "2", "3", "4", "5", "6"}, // 6 items, max is 5
			UniqueIDs: []int{1},
		}
		errs := validator.Validate(&playlist)
		if len(errs) == 0 {
			t.Error("expected validation error for array exceeding max items")
		}
	})

	t.Run("duplicate items should fail UniqueItems", func(t *testing.T) {
		playlist := Playlist{
			Songs:     []string{"A"},
			UniqueIDs: []int{1, 2, 3, 2}, // Duplicate 2
		}
		errs := validator.Validate(&playlist)
		if len(errs) == 0 {
			t.Error("expected validation error for duplicate items")
		}
	})
}

// Test MinProperties, MaxProperties
type Config struct {
	Settings map[string]any
	Limits   map[string]any
}

func (c *Config) FieldSettings() godantic.FieldOptions[map[string]any] {
	return godantic.Field(
		godantic.Required[map[string]any](),
		godantic.MinProperties(1),
	)
}

func (c *Config) FieldLimits() godantic.FieldOptions[map[string]any] {
	return godantic.Field(
		godantic.MaxProperties(3),
	)
}

func TestMapConstraints(t *testing.T) {
	validator := godantic.NewValidator[Config]()

	t.Run("valid maps should pass", func(t *testing.T) {
		config := Config{
			Settings: map[string]any{"key1": "value1"},
			Limits:   map[string]any{"max": 100, "min": 0},
		}
		errs := validator.Validate(&config)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("empty map should fail MinProperties", func(t *testing.T) {
		config := Config{
			Settings: map[string]any{}, // Empty, violates MinProperties(1)
		}
		errs := validator.Validate(&config)
		if len(errs) == 0 {
			t.Error("expected validation error for empty map")
		}
	})

	t.Run("map exceeding MaxProperties should fail", func(t *testing.T) {
		config := Config{
			Settings: map[string]any{"key1": "value1"},
			Limits:   map[string]any{"a": 1, "b": 2, "c": 3, "d": 4}, // 4 props, max is 3
		}
		errs := validator.Validate(&config)
		if len(errs) == 0 {
			t.Error("expected validation error for map exceeding max properties")
		}
	})
}

// Test Const
type Environment struct {
	Type string
}

func (e *Environment) FieldType() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Const("production"),
	)
}

func TestConstValidation(t *testing.T) {
	validator := godantic.NewValidator[Environment]()

	t.Run("value matching const should pass", func(t *testing.T) {
		env := Environment{
			Type: "production",
		}
		errs := validator.Validate(&env)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("value not matching const should fail", func(t *testing.T) {
		env := Environment{
			Type: "development",
		}
		errs := validator.Validate(&env)
		if len(errs) == 0 {
			t.Error("expected validation error for value not matching const")
		}
	})
}

// Test ContentEncoding and ContentMediaType
type Document struct {
	Base64Data string
	JSONData   string
}

func (d *Document) FieldBase64Data() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.ContentEncoding("base64"),
	)
}

func (d *Document) FieldJSONData() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.ContentMediaType("application/json"),
	)
}

func TestContentConstraints(t *testing.T) {
	validator := godantic.NewValidator[Document]()

	// These are schema metadata only, no validation
	t.Run("content constraints should not fail validation", func(t *testing.T) {
		doc := Document{
			Base64Data: "SGVsbG8gV29ybGQ=",
			JSONData:   `{"key": "value"}`,
		}
		errs := validator.Validate(&doc)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("constraints should be stored in field options", func(t *testing.T) {
		doc := &Document{}
		opts := doc.FieldBase64Data()
		if opts.Constraints_[godantic.ConstraintContentEncoding] != "base64" {
			t.Error("expected ContentEncoding constraint to be stored")
		}

		opts2 := doc.FieldJSONData()
		if opts2.Constraints_[godantic.ConstraintContentMediaType] != "application/json" {
			t.Error("expected ContentMediaType constraint to be stored")
		}
	})
}

// Test schema metadata constraints (ReadOnly, WriteOnly, Deprecated, Title, Format, Default)
type APISchema struct {
	ID          string
	Password    string
	OldField    string
	DisplayName string
	CreatedAt   string
	Status      string
}

func (a *APISchema) FieldID() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.ReadOnly[string](),
		godantic.Title[string]("Unique Identifier"),
	)
}

func (a *APISchema) FieldPassword() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.WriteOnly[string](),
		godantic.MinLen(8),
	)
}

func (a *APISchema) FieldOldField() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Deprecated[string](),
	)
}

func (a *APISchema) FieldDisplayName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Title[string]("Display Name"),
	)
}

func (a *APISchema) FieldCreatedAt() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Format[string]("date-time"),
	)
}

func (a *APISchema) FieldStatus() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Default("active"),
	)
}

func TestSchemaMetadataConstraints(t *testing.T) {
	validator := godantic.NewValidator[APISchema]()

	// These are schema metadata only, no validation
	t.Run("metadata constraints should not affect validation", func(t *testing.T) {
		api := APISchema{
			ID:          "123",
			Password:    "secure-password",
			OldField:    "deprecated-value",
			DisplayName: "Test User",
			CreatedAt:   "2024-01-01T00:00:00Z",
			Status:      "active",
		}
		errs := validator.Validate(&api)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("metadata should be stored in constraints", func(t *testing.T) {
		api := &APISchema{}

		idOpts := api.FieldID()
		if idOpts.Constraints_[godantic.ConstraintReadOnly] != true {
			t.Error("expected ReadOnly to be true")
		}
		if idOpts.Constraints_[godantic.ConstraintTitle] != "Unique Identifier" {
			t.Error("expected Title to be set")
		}

		pwOpts := api.FieldPassword()
		if pwOpts.Constraints_[godantic.ConstraintWriteOnly] != true {
			t.Error("expected WriteOnly to be true")
		}

		oldOpts := api.FieldOldField()
		if oldOpts.Constraints_[godantic.ConstraintDeprecated] != true {
			t.Error("expected Deprecated to be true")
		}

		nameOpts := api.FieldDisplayName()
		if nameOpts.Constraints_[godantic.ConstraintTitle] != "Display Name" {
			t.Error("expected Title to be set")
		}

		dateOpts := api.FieldCreatedAt()
		if dateOpts.Constraints_[godantic.ConstraintFormat] != "date-time" {
			t.Error("expected Format to be date-time")
		}

		statusOpts := api.FieldStatus()
		if statusOpts.Constraints_[godantic.ConstraintDefault] != "active" {
			t.Error("expected Default to be active")
		}
	})
}
