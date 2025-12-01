package godantic_test

import (
	"fmt"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Test slices of native types
type Article struct {
	Title    string
	Tags     []string
	Scores   []int
	Keywords []string
}

func (a *Article) FieldTitle() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (a *Article) FieldTags() godantic.FieldOptions[[]string] {
	return godantic.Field(
		godantic.Required[[]string](),
		godantic.Validate(func(tags []string) error {
			if len(tags) < 1 {
				return fmt.Errorf("must have at least 1 tag")
			}
			if len(tags) > 10 {
				return fmt.Errorf("cannot have more than 10 tags")
			}
			return nil
		}),
	)
}

func (a *Article) FieldScores() godantic.FieldOptions[[]int] {
	return godantic.Field(
		godantic.Validate(func(scores []int) error {
			for i, score := range scores {
				if score < 0 || score > 100 {
					return fmt.Errorf("score at index %d must be between 0 and 100", i)
				}
			}
			return nil
		}),
	)
}

func (a *Article) FieldKeywords() godantic.FieldOptions[[]string] {
	return godantic.Field(
		godantic.Validate(func(keywords []string) error {
			// Check for unique keywords
			seen := make(map[string]bool)
			for _, keyword := range keywords {
				if seen[keyword] {
					return fmt.Errorf("duplicate keyword: %s", keyword)
				}
				seen[keyword] = true
			}
			return nil
		}),
	)
}

func TestSlicesOfNativeTypes(t *testing.T) {
	validator := godantic.NewValidator[Article]()

	t.Run("valid article with tags should pass", func(t *testing.T) {
		article := Article{
			Title:    "Test Article",
			Tags:     []string{"go", "testing", "validation"},
			Scores:   []int{85, 90, 95},
			Keywords: []string{"golang", "pydantic", "validation"},
		}
		errs := validator.Validate(&article)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("empty tags should fail", func(t *testing.T) {
		article := Article{
			Title: "Test Article",
			Tags:  []string{},
		}
		errs := validator.Validate(&article)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Tags: must have at least 1 tag" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("too many tags should fail", func(t *testing.T) {
		article := Article{
			Title: "Test Article",
			Tags:  []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"},
		}
		errs := validator.Validate(&article)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Tags: cannot have more than 10 tags" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("invalid score should fail", func(t *testing.T) {
		article := Article{
			Title:  "Test Article",
			Tags:   []string{"go"},
			Scores: []int{85, 150, 95}, // 150 is invalid
		}
		errs := validator.Validate(&article)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Scores: score at index 1 must be between 0 and 100" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("duplicate keywords should fail", func(t *testing.T) {
		article := Article{
			Title:    "Test Article",
			Tags:     []string{"go"},
			Keywords: []string{"golang", "testing", "golang"}, // duplicate
		}
		errs := validator.Validate(&article)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Keywords: duplicate keyword: golang" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})
}

// Test slices of structs
type Contact struct {
	Name  string
	Email string
}

type Company struct {
	Name     string
	Contacts []Contact
}

func (c *Company) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (c *Company) FieldContacts() godantic.FieldOptions[[]Contact] {
	return godantic.Field(
		godantic.Required[[]Contact](),
		godantic.Validate(func(contacts []Contact) error {
			if len(contacts) < 1 {
				return fmt.Errorf("must have at least 1 contact")
			}
			// Validate each contact
			for i, contact := range contacts {
				if contact.Name == "" {
					return fmt.Errorf("contact at index %d must have a name", i)
				}
				if contact.Email == "" {
					return fmt.Errorf("contact at index %d must have an email", i)
				}
			}
			return nil
		}),
	)
}

func TestSlicesOfStructs(t *testing.T) {
	validator := godantic.NewValidator[Company]()

	t.Run("valid company with contacts should pass", func(t *testing.T) {
		company := Company{
			Name: "Tech Corp",
			Contacts: []Contact{
				{Name: "Alice", Email: "alice@example.com"},
				{Name: "Bob", Email: "bob@example.com"},
			},
		}
		errs := validator.Validate(&company)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("empty contacts should fail", func(t *testing.T) {
		company := Company{
			Name:     "Tech Corp",
			Contacts: []Contact{},
		}
		errs := validator.Validate(&company)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Contacts: must have at least 1 contact" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("contact with empty name should fail", func(t *testing.T) {
		company := Company{
			Name: "Tech Corp",
			Contacts: []Contact{
				{Name: "", Email: "alice@example.com"},
			},
		}
		errs := validator.Validate(&company)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Contacts: contact at index 0 must have a name" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})
}

// Test automatic validation of slice elements with Field methods
type Employee struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (e *Employee) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (e *Employee) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

type Organization struct {
	Name      string     `json:"name"`
	Employees []Employee `json:"employees"`
}

func (o *Organization) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (o *Organization) FieldEmployees() godantic.FieldOptions[[]Employee] {
	return godantic.Field(godantic.Required[[]Employee]())
}

func TestSlicesOfStructsWithFieldMethods(t *testing.T) {
	validator := godantic.NewValidator[Organization]()

	t.Run("valid organization with employees should pass", func(t *testing.T) {
		org := Organization{
			Name: "Tech Corp",
			Employees: []Employee{
				{Name: "Alice", Email: "alice@example.com"},
				{Name: "Bob", Email: "bob@example.com"},
			},
		}
		errs := validator.Validate(&org)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("employee with missing name should fail", func(t *testing.T) {
		org := Organization{
			Name: "Tech Corp",
			Employees: []Employee{
				{Name: "", Email: "alice@example.com"}, // Missing required Name
			},
		}
		errs := validator.Validate(&org)
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if errs[0].Error() != "Employees.[0].Name: required field" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("unmarshal JSON with missing employee email should fail", func(t *testing.T) {
		jsonStr := `{
			"name": "Tech Corp",
			"employees": [
				{"name": "Alice", "email": "alice@example.com"},
				{"name": "Bob", "email": ""}
			]
		}`

		_, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if errs[0].Error() != "Employees.[1].Email: required field" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})
}
