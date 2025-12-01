package godantic_test

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

func TestSlicesOfNativeTypes(t *testing.T) {
	validator := godantic.NewValidator[TArticle]()

	t.Run("valid article with tags should pass", func(t *testing.T) {
		article := TArticle{
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
		article := TArticle{
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
		article := TArticle{
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
		article := TArticle{
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
		article := TArticle{
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

func TestSlicesOfStructsWithFieldMethods(t *testing.T) {
	validator := godantic.NewValidator[TOrganization]()

	t.Run("valid organization with employees should pass", func(t *testing.T) {
		org := TOrganization{
			Name: "Tech Corp",
			Employees: []TEmployee{
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
		org := TOrganization{
			Name: "Tech Corp",
			Employees: []TEmployee{
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

func TestRootSliceUnmarshal(t *testing.T) {
	validator := godantic.NewValidator[[]TUser]()

	t.Run("valid root slice should pass", func(t *testing.T) {
		jsonStr := `[
			{"name": "Alice", "email": "alice@example.com", "age": 30},
			{"name": "Bob", "email": "bob@example.com", "age": 25}
		]`

		people, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %d: %v", len(errs), errs)
		}
		if len(*people) != 2 {
			t.Errorf("expected 2 people, got %d", len(*people))
		}
		if (*people)[0].Name != "Alice" {
			t.Errorf("expected first person name 'Alice', got '%s'", (*people)[0].Name)
		}
		if (*people)[1].Age != 25 {
			t.Errorf("expected second person age 25, got %d", (*people)[1].Age)
		}
	})

	t.Run("missing required field should fail with correct path", func(t *testing.T) {
		jsonStr := `[
			{"name": "Alice", "email": "alice@example.com", "age": 30},
			{"email": "bob@example.com", "age": 25}
		]`

		_, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if errs[0].Error() != "[1].Name: required field" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("validation error should have correct path", func(t *testing.T) {
		jsonStr := `[
			{"name": "Alice", "email": "alice@example.com", "age": 30},
			{"name": "Bob", "email": "bob@example.com", "age": 200}
		]`

		_, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if errs[0].Error() != "[1].Age: age must be between 0 and 150" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("multiple elements with errors", func(t *testing.T) {
		jsonStr := `[
			{"email": "alice@example.com", "age": 30},
			{"name": "Bob", "email": "bob@example.com", "age": 200},
			{"name": "Charlie", "email": "charlie@example.com"}
		]`

		_, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 3 {
			t.Fatalf("expected 3 errors, got %d: %v", len(errs), errs)
		}
		// Check error paths
		expectedErrors := map[string]bool{
			"[0].Name: required field":               true,
			"[1].Age: age must be between 0 and 150": true,
			"[2].Age: required field":                true,
		}
		for _, err := range errs {
			if !expectedErrors[err.Error()] {
				t.Errorf("unexpected error: %v", err)
			}
		}
	})

	t.Run("invalid JSON should fail", func(t *testing.T) {
		jsonStr := `not valid json`
		_, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Type != "json_decode" {
			t.Errorf("expected json_decode error, got %s", errs[0].Type)
		}
	})

	t.Run("empty array should pass", func(t *testing.T) {
		jsonStr := `[]`
		people, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %d: %v", len(errs), errs)
		}
		if len(*people) != 0 {
			t.Errorf("expected empty slice, got %d elements", len(*people))
		}
	})
}

func TestRootSliceWithBeforeValidateHook(t *testing.T) {
	validator := godantic.NewValidator[[]TMessage]()

	t.Run("hook should transform each element", func(t *testing.T) {
		jsonStr := `[
			{"type": "text", "text": "Hello"},
			{"content": "World"}
		]`

		messages, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %d: %v", len(errs), errs)
		}
		if len(*messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(*messages))
		}
		if (*messages)[0].Text != "Hello" {
			t.Errorf("expected first message text 'Hello', got '%s'", (*messages)[0].Text)
		}
		if (*messages)[1].Type != "text" {
			t.Errorf("expected second message type 'text', got '%s'", (*messages)[1].Type)
		}
		if (*messages)[1].Text != "World" {
			t.Errorf("expected second message text 'World', got '%s'", (*messages)[1].Text)
		}
	})

	t.Run("validation after hook transformation", func(t *testing.T) {
		jsonStr := `[
			{"content": "Valid"},
			{"invalid": "data"}
		]`

		_, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 2 {
			t.Fatalf("expected 2 errors, got %d: %v", len(errs), errs)
		}
		// Both Type and Text should be missing for the second element
		errorMessages := map[string]bool{}
		for _, err := range errs {
			errorMessages[err.Error()] = true
		}
		if !errorMessages["[1].Type: required field"] || !errorMessages["[1].Text: required field"] {
			t.Errorf("unexpected errors: %v", errs)
		}
	})
}

func TestRootSliceWithPointerElements(t *testing.T) {
	validator := godantic.NewValidator[[]*TUser]()

	t.Run("unmarshal pointer slice", func(t *testing.T) {
		jsonStr := `[
			{"name": "Alice", "email": "alice@example.com", "age": 30},
			{"name": "Bob", "email": "bob@example.com", "age": 25}
		]`

		users, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %d: %v", len(errs), errs)
		}
		if len(*users) != 2 {
			t.Errorf("expected 2 users, got %d", len(*users))
		}
		if (*users)[0].Name != "Alice" {
			t.Errorf("expected first user name 'Alice', got '%s'", (*users)[0].Name)
		}
	})

	t.Run("validation error with pointer elements", func(t *testing.T) {
		jsonStr := `[
			{"name": "Alice", "email": "alice@example.com", "age": 30},
			{"email": "bob@example.com", "age": 25}
		]`

		_, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}
		if errs[0].Error() != "[1].Name: required field" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})
}
