package godantic_test

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

func TestSlicesOfNativeTypes(t *testing.T) {
	validator := godantic.NewValidator[TArticle]()

	tests := []struct {
		name         string
		article      TArticle
		wantErrCount int
		wantErrMsg   string
	}{
		{
			name: "valid article with tags should pass",
			article: TArticle{
				Title:    "Test Article",
				Tags:     []string{"go", "testing", "validation"},
				Scores:   []int{85, 90, 95},
				Keywords: []string{"golang", "pydantic", "validation"},
			},
			wantErrCount: 0,
		},
		{
			name: "empty tags should fail",
			article: TArticle{
				Title: "Test Article",
				Tags:  []string{},
			},
			wantErrCount: 1,
			wantErrMsg:   "Tags: must have at least 1 tag",
		},
		{
			name: "too many tags should fail",
			article: TArticle{
				Title: "Test Article",
				Tags:  []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"},
			},
			wantErrCount: 1,
			wantErrMsg:   "Tags: cannot have more than 10 tags",
		},
		{
			name: "invalid score should fail",
			article: TArticle{
				Title:  "Test Article",
				Tags:   []string{"go"},
				Scores: []int{85, 150, 95},
			},
			wantErrCount: 1,
			wantErrMsg:   "Scores: score at index 1 must be between 0 and 100",
		},
		{
			name: "duplicate keywords should fail",
			article: TArticle{
				Title:    "Test Article",
				Tags:     []string{"go"},
				Keywords: []string{"golang", "testing", "golang"},
			},
			wantErrCount: 1,
			wantErrMsg:   "Keywords: duplicate keyword: golang",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.Validate(&tt.article)
			if len(errs) != tt.wantErrCount {
				t.Errorf("got %d errors, want %d: %v", len(errs), tt.wantErrCount, errs)
			}
			if tt.wantErrMsg != "" && len(errs) > 0 && errs[0].Error() != tt.wantErrMsg {
				t.Errorf("got error %q, want %q", errs[0].Error(), tt.wantErrMsg)
			}
		})
	}
}

func TestSlicesOfStructsWithFieldMethods(t *testing.T) {
	validator := godantic.NewValidator[TOrganization]()

	tests := []struct {
		name         string
		org          TOrganization
		wantErrCount int
		wantErrMsg   string
	}{
		{
			name: "valid organization with employees should pass",
			org: TOrganization{
				Name: "Tech Corp",
				Employees: []TEmployee{
					{Name: "Alice", Email: "alice@example.com"},
					{Name: "Bob", Email: "bob@example.com"},
				},
			},
			wantErrCount: 0,
		},
		{
			name: "employee with missing name should fail",
			org: TOrganization{
				Name: "Tech Corp",
				Employees: []TEmployee{
					{Name: "", Email: "alice@example.com"},
				},
			},
			wantErrCount: 1,
			wantErrMsg:   "Employees.[0].Name: required field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.Validate(&tt.org)
			if len(errs) != tt.wantErrCount {
				t.Fatalf("got %d errors, want %d: %v", len(errs), tt.wantErrCount, errs)
			}
			if tt.wantErrMsg != "" && len(errs) > 0 && errs[0].Error() != tt.wantErrMsg {
				t.Errorf("got error %q, want %q", errs[0].Error(), tt.wantErrMsg)
			}
		})
	}

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
			t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
		}
		if errs[0].Error() != "Employees.[1].Email: required field" {
			t.Errorf("got error %q, want %q", errs[0].Error(), "Employees.[1].Email: required field")
		}
	})
}

func TestRootSliceUnmarshal(t *testing.T) {
	validator := godantic.NewValidator[[]TUser]()

	tests := []struct {
		name         string
		jsonStr      string
		wantErrCount int
		wantErrMsg   string
		wantErrType  godantic.ErrorType
		validate     func(*testing.T, *[]TUser)
	}{
		{
			name:    "valid root slice should pass",
			jsonStr: `[{"name": "Alice", "email": "alice@example.com", "age": 30}, {"name": "Bob", "email": "bob@example.com", "age": 25}]`,
			validate: func(t *testing.T, people *[]TUser) {
				if len(*people) != 2 {
					t.Errorf("got %d people, want 2", len(*people))
				}
				if (*people)[0].Name != "Alice" {
					t.Errorf("got first name %q, want 'Alice'", (*people)[0].Name)
				}
				if (*people)[1].Age != 25 {
					t.Errorf("got second age %d, want 25", (*people)[1].Age)
				}
			},
		},
		{
			name:         "missing required field should fail with correct path",
			jsonStr:      `[{"name": "Alice", "email": "alice@example.com", "age": 30}, {"email": "bob@example.com", "age": 25}]`,
			wantErrCount: 1,
			wantErrMsg:   "[1].Name: required field",
		},
		{
			name:         "validation error should have correct path",
			jsonStr:      `[{"name": "Alice", "email": "alice@example.com", "age": 30}, {"name": "Bob", "email": "bob@example.com", "age": 200}]`,
			wantErrCount: 1,
			wantErrMsg:   "[1].Age: age must be between 0 and 150",
		},
		{
			name:         "invalid JSON should fail",
			jsonStr:      `not valid json`,
			wantErrCount: 1,
			wantErrType:  godantic.ErrorTypeJSONDecode,
		},
		{
			name:    "empty array should pass",
			jsonStr: `[]`,
			validate: func(t *testing.T, people *[]TUser) {
				if len(*people) != 0 {
					t.Errorf("got %d elements, want 0", len(*people))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			people, errs := validator.Unmarshal([]byte(tt.jsonStr))
			if len(errs) != tt.wantErrCount {
				t.Fatalf("got %d errors, want %d: %v", len(errs), tt.wantErrCount, errs)
			}
			if tt.wantErrMsg != "" && len(errs) > 0 && errs[0].Error() != tt.wantErrMsg {
				t.Errorf("got error %q, want %q", errs[0].Error(), tt.wantErrMsg)
			}
			if tt.wantErrType != "" && len(errs) > 0 && errs[0].Type != tt.wantErrType {
				t.Errorf("got error type %q, want %q", errs[0].Type, tt.wantErrType)
			}
			if tt.validate != nil {
				tt.validate(t, people)
			}
		})
	}

	t.Run("multiple elements with errors", func(t *testing.T) {
		jsonStr := `[
			{"email": "alice@example.com", "age": 30},
			{"name": "Bob", "email": "bob@example.com", "age": 200},
			{"name": "Charlie", "email": "charlie@example.com"}
		]`

		_, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 3 {
			t.Fatalf("got %d errors, want 3: %v", len(errs), errs)
		}
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
}

func TestRootSliceWithBeforeValidateHook(t *testing.T) {
	validator := godantic.NewValidator[[]TMessage]()

	tests := []struct {
		name         string
		jsonStr      string
		wantErrCount int
		wantErrMsgs  map[string]bool
		validate     func(*testing.T, *[]TMessage)
	}{
		{
			name:    "hook should transform each element",
			jsonStr: `[{"type": "text", "text": "Hello"}, {"content": "World"}]`,
			validate: func(t *testing.T, messages *[]TMessage) {
				if len(*messages) != 2 {
					t.Fatalf("got %d messages, want 2", len(*messages))
				}
				if (*messages)[0].Text != "Hello" {
					t.Errorf("got first text %q, want 'Hello'", (*messages)[0].Text)
				}
				if (*messages)[1].Type != "text" {
					t.Errorf("got second type %q, want 'text'", (*messages)[1].Type)
				}
				if (*messages)[1].Text != "World" {
					t.Errorf("got second text %q, want 'World'", (*messages)[1].Text)
				}
			},
		},
		{
			name:         "validation after hook transformation",
			jsonStr:      `[{"content": "Valid"}, {"invalid": "data"}]`,
			wantErrCount: 2,
			wantErrMsgs: map[string]bool{
				"[1].Type: required field": true,
				"[1].Text: required field": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, errs := validator.Unmarshal([]byte(tt.jsonStr))
			if len(errs) != tt.wantErrCount {
				t.Fatalf("got %d errors, want %d: %v", len(errs), tt.wantErrCount, errs)
			}
			if tt.wantErrMsgs != nil {
				errorMessages := make(map[string]bool)
				for _, err := range errs {
					errorMessages[err.Error()] = true
				}
				for wantMsg := range tt.wantErrMsgs {
					if !errorMessages[wantMsg] {
						t.Errorf("missing expected error: %q", wantMsg)
					}
				}
			}
			if tt.validate != nil {
				tt.validate(t, messages)
			}
		})
	}
}

func TestRootSliceWithPointerElements(t *testing.T) {
	validator := godantic.NewValidator[[]*TUser]()

	tests := []struct {
		name         string
		jsonStr      string
		wantErrCount int
		wantErrMsg   string
		validate     func(*testing.T, *[]*TUser)
	}{
		{
			name:    "unmarshal pointer slice",
			jsonStr: `[{"name": "Alice", "email": "alice@example.com", "age": 30}, {"name": "Bob", "email": "bob@example.com", "age": 25}]`,
			validate: func(t *testing.T, users *[]*TUser) {
				if len(*users) != 2 {
					t.Fatalf("got %d users, want 2", len(*users))
				}
				if (*users)[0].Name != "Alice" {
					t.Errorf("got first name %q, want 'Alice'", (*users)[0].Name)
				}
			},
		},
		{
			name:         "validation error with pointer elements",
			jsonStr:      `[{"name": "Alice", "email": "alice@example.com", "age": 30}, {"email": "bob@example.com", "age": 25}]`,
			wantErrCount: 1,
			wantErrMsg:   "[1].Name: required field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, errs := validator.Unmarshal([]byte(tt.jsonStr))
			if len(errs) != tt.wantErrCount {
				t.Fatalf("got %d errors, want %d: %v", len(errs), tt.wantErrCount, errs)
			}
			if tt.wantErrMsg != "" && len(errs) > 0 && errs[0].Error() != tt.wantErrMsg {
				t.Errorf("got error %q, want %q", errs[0].Error(), tt.wantErrMsg)
			}
			if tt.validate != nil {
				tt.validate(t, users)
			}
		})
	}
}

func TestRootSlicePrimitiveTypes(t *testing.T) {
	t.Run("unmarshal string slice", func(t *testing.T) {
		validator := godantic.NewValidator[[]string]()
		jsonStr := `["apple", "banana", "cherry"]`

		fruits, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 0 {
			t.Fatalf("got %d errors, want 0: %v", len(errs), errs)
		}
		if len(*fruits) != 3 {
			t.Errorf("got %d items, want 3", len(*fruits))
		}
		if (*fruits)[0] != "apple" {
			t.Errorf("got first item %q, want 'apple'", (*fruits)[0])
		}
	})

	t.Run("unmarshal int slice", func(t *testing.T) {
		validator := godantic.NewValidator[[]int]()
		jsonStr := `[1, 2, 3, 4, 5]`

		nums, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) != 0 {
			t.Fatalf("got %d errors, want 0: %v", len(errs), errs)
		}
		if len(*nums) != 5 {
			t.Errorf("got %d items, want 5", len(*nums))
		}
		if (*nums)[2] != 3 {
			t.Errorf("got third item %d, want 3", (*nums)[2])
		}
	})
}

func TestRootSliceHookErrors(t *testing.T) {
	t.Run("multiple hook errors should be prefixed with indices", func(t *testing.T) {
		validator := godantic.NewValidator[[]TMessage]()
		// Second element missing both required fields after hook runs
		jsonStr := `[
			{"type": "text", "text": "valid"},
			{"unknown": "field"}
		]`

		_, errs := validator.Unmarshal([]byte(jsonStr))
		if len(errs) == 0 {
			t.Fatal("expected validation errors, got none")
		}

		// Check that errors are properly indexed
		hasIndexedError := false
		for _, err := range errs {
			if len(err.Loc) > 0 && (err.Loc[0] == "[1]" || err.Loc[0] == "[0]") {
				hasIndexedError = true
				break
			}
		}
		if !hasIndexedError {
			t.Errorf("expected errors with array index prefix, got: %v", errs)
		}
	})
}

func TestRootSliceMarshal(t *testing.T) {
	validator := godantic.NewValidator[[]TUser]()

	t.Run("marshal slice should work", func(t *testing.T) {
		users := []TUser{
			{Name: "Alice", Email: "alice@example.com", Age: 30},
			{Name: "Bob", Email: "bob@example.com", Age: 25},
		}

		jsonData, errs := validator.Marshal(&users)
		if len(errs) != 0 {
			t.Fatalf("got %d errors, want 0: %v", len(errs), errs)
		}

		// Verify we can unmarshal it back
		unmarshaled, errs := validator.Unmarshal(jsonData)
		if len(errs) != 0 {
			t.Fatalf("got %d unmarshal errors: %v", len(errs), errs)
		}
		if len(*unmarshaled) != 2 {
			t.Errorf("got %d users, want 2", len(*unmarshaled))
		}
		if (*unmarshaled)[0].Name != "Alice" {
			t.Errorf("got first name %q, want 'Alice'", (*unmarshaled)[0].Name)
		}
	})

	t.Run("marshal empty slice should work", func(t *testing.T) {
		users := []TUser{}

		jsonData, errs := validator.Marshal(&users)
		if len(errs) != 0 {
			t.Fatalf("got %d errors, want 0: %v", len(errs), errs)
		}

		unmarshaled, errs := validator.Unmarshal(jsonData)
		if len(errs) != 0 {
			t.Fatalf("got %d unmarshal errors: %v", len(errs), errs)
		}
		if len(*unmarshaled) != 0 {
			t.Errorf("got %d users, want 0", len(*unmarshaled))
		}
	})
}
