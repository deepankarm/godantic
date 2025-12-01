package godantic

import (
	"reflect"
	"testing"

	"github.com/deepankarm/godantic/pkg/internal/walk"
)

// ═══════════════════════════════════════════════════════════════════════════
// structPathToJSONPath Tests
// ═══════════════════════════════════════════════════════════════════════════

type testUser struct {
	Name    string `json:"name"`
	Age     int    `json:"age"`
	Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	} `json:"address"`
	Tags []string `json:"tags"`
}

type testUserSnakeCase struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type testWithPointer struct {
	Profile *struct {
		Bio string `json:"bio"`
	} `json:"profile"`
}

func TestStructPathToJSONPath(t *testing.T) {
	tests := []struct {
		name       string
		structPath []string
		typ        reflect.Type
		want       string
	}{
		{
			name:       "empty",
			structPath: []string{},
			typ:        reflect.TypeOf(testUser{}),
			want:       "",
		},
		{
			name:       "single_field",
			structPath: []string{"Name"},
			typ:        reflect.TypeOf(testUser{}),
			want:       "name",
		},
		{
			name:       "nested_field",
			structPath: []string{"Address", "City"},
			typ:        reflect.TypeOf(testUser{}),
			want:       "address.city",
		},
		{
			name:       "snake_case_tags",
			structPath: []string{"FirstName"},
			typ:        reflect.TypeOf(testUserSnakeCase{}),
			want:       "first_name",
		},
		{
			name:       "array_index",
			structPath: []string{"Tags", "[0]"},
			typ:        reflect.TypeOf(testUser{}),
			want:       "tags[0]",
		},
		{
			name:       "pointer_type",
			structPath: []string{"Profile", "Bio"},
			typ:        reflect.TypeOf(testWithPointer{}),
			want:       "profile.bio",
		},
		{
			name:       "pointer_to_struct",
			structPath: []string{"Profile"},
			typ:        reflect.TypeOf(&testWithPointer{}),
			want:       "profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := structPathToJSONPath(tt.structPath, tt.typ)
			if got != tt.want {
				t.Errorf("structPathToJSONPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// prefixErrors Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestPrefixErrors(t *testing.T) {
	t.Run("prefix single error", func(t *testing.T) {
		errs := ValidationErrors{
			{Loc: []string{"Name"}, Message: "required field", Type: "required"},
		}

		result := prefixErrors(errs, "[0]")
		if len(result) != 1 {
			t.Fatalf("expected 1 error, got %d", len(result))
		}
		if len(result[0].Loc) != 2 {
			t.Fatalf("expected 2 path segments, got %d", len(result[0].Loc))
		}
		if result[0].Loc[0] != "[0]" {
			t.Errorf("expected first segment '[0]', got '%s'", result[0].Loc[0])
		}
		if result[0].Loc[1] != "Name" {
			t.Errorf("expected second segment 'Name', got '%s'", result[0].Loc[1])
		}
		if result[0].Message != "required field" {
			t.Errorf("message should be preserved, got '%s'", result[0].Message)
		}
	})

	t.Run("prefix multiple errors", func(t *testing.T) {
		errs := ValidationErrors{
			{Loc: []string{"Name"}, Message: "required", Type: "required"},
			{Loc: []string{"Age"}, Message: "min", Type: "constraint"},
			{Loc: []string{"Email"}, Message: "invalid", Type: "validation"},
		}

		result := prefixErrors(errs, "[2]")
		if len(result) != 3 {
			t.Fatalf("expected 3 errors, got %d", len(result))
		}
		for i, err := range result {
			if err.Loc[0] != "[2]" {
				t.Errorf("error %d: expected prefix '[2]', got '%s'", i, err.Loc[0])
			}
		}
	})

	t.Run("prefix nested path", func(t *testing.T) {
		errs := ValidationErrors{
			{Loc: []string{"Address", "City"}, Message: "required", Type: "required"},
		}

		result := prefixErrors(errs, "[1]")
		if len(result[0].Loc) != 3 {
			t.Fatalf("expected 3 path segments, got %d: %v", len(result[0].Loc), result[0].Loc)
		}
		if result[0].Loc[0] != "[1]" || result[0].Loc[1] != "Address" || result[0].Loc[2] != "City" {
			t.Errorf("unexpected path: %v", result[0].Loc)
		}
	})

	t.Run("prefix empty error list", func(t *testing.T) {
		errs := ValidationErrors{}
		result := prefixErrors(errs, "[0]")
		if len(result) != 0 {
			t.Errorf("expected 0 errors, got %d", len(result))
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// filterIncompleteFieldErrors Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestFilterIncompleteFieldErrors(t *testing.T) {
	typ := reflect.TypeOf(testUser{})

	t.Run("no_incomplete_paths", func(t *testing.T) {
		errs := []walk.ValidationError{
			{Loc: []string{"Name"}, Message: "required", Type: "required"},
			{Loc: []string{"Age"}, Message: "min", Type: "constraint"},
		}

		result := filterIncompleteFieldErrors(errs, nil, typ)
		if len(result) != 2 {
			t.Errorf("expected 2 errors, got %d", len(result))
		}
	})

	t.Run("filter_incomplete_field", func(t *testing.T) {
		errs := []walk.ValidationError{
			{Loc: []string{"Name"}, Message: "required", Type: "required"},
			{Loc: []string{"Age"}, Message: "min", Type: "constraint"},
		}
		incompletePaths := [][]string{{"name"}} // name is incomplete

		result := filterIncompleteFieldErrors(errs, incompletePaths, typ)
		if len(result) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(result), result)
		}
		if result[0].Loc[0] != "Age" {
			t.Errorf("expected Age error, got %v", result[0].Loc)
		}
	})

	t.Run("filter_nested_incomplete", func(t *testing.T) {
		errs := []walk.ValidationError{
			{Loc: []string{"Address", "City"}, Message: "required", Type: "required"},
			{Loc: []string{"Name"}, Message: "required", Type: "required"},
		}
		incompletePaths := [][]string{{"address", "city"}} // address.city is incomplete

		result := filterIncompleteFieldErrors(errs, incompletePaths, typ)
		if len(result) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(result), result)
		}
		if result[0].Loc[0] != "Name" {
			t.Errorf("expected Name error, got %v", result[0].Loc)
		}
	})

	t.Run("filter_parent_incomplete", func(t *testing.T) {
		errs := []walk.ValidationError{
			{Loc: []string{"Address", "City"}, Message: "required", Type: "required"},
			{Loc: []string{"Address", "Street"}, Message: "required", Type: "required"},
		}
		incompletePaths := [][]string{{"address"}} // whole address is incomplete

		result := filterIncompleteFieldErrors(errs, incompletePaths, typ)
		if len(result) != 0 {
			t.Errorf("expected 0 errors (parent incomplete), got %d: %v", len(result), result)
		}
	})

	t.Run("filter_array_element", func(t *testing.T) {
		errs := []walk.ValidationError{
			{Loc: []string{"Tags", "[0]"}, Message: "min", Type: "constraint"},
		}
		incompletePaths := [][]string{{"tags", "[0]"}}

		result := filterIncompleteFieldErrors(errs, incompletePaths, typ)
		if len(result) != 0 {
			t.Errorf("expected 0 errors (array element incomplete), got %d", len(result))
		}
	})

	t.Run("preserve_complete_sibling", func(t *testing.T) {
		errs := []walk.ValidationError{
			{Loc: []string{"Tags", "[0]"}, Message: "min", Type: "constraint"},
			{Loc: []string{"Tags", "[1]"}, Message: "min", Type: "constraint"},
		}
		incompletePaths := [][]string{{"tags", "[0]"}} // only [0] is incomplete

		result := filterIncompleteFieldErrors(errs, incompletePaths, typ)
		if len(result) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(result), result)
		}
	})
}
