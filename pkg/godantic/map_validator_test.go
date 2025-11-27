package godantic_test

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// ═══════════════════════════════════════════════════════════════════════════
// ValidateFromStringMap Tests
// For path params, cookies - single value per key
// ═══════════════════════════════════════════════════════════════════════════

func TestValidateFromStringMap(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]string
		wantID   int
		wantSlug string
		wantErr  bool
		errType  godantic.ErrorType
		errField string
	}{
		{
			name:     "valid_all_fields",
			data:     map[string]string{"id": "42", "slug": "hello-world", "active": "true"},
			wantID:   42,
			wantSlug: "hello-world",
			wantErr:  false,
		},
		{
			name:     "valid_int_conversion",
			data:     map[string]string{"id": "123", "slug": "test"},
			wantID:   123,
			wantSlug: "test",
			wantErr:  false,
		},
		{
			name:     "valid_bool_true",
			data:     map[string]string{"id": "1", "slug": "test", "active": "1"},
			wantID:   1,
			wantSlug: "test",
			wantErr:  false,
		},
		{
			name:     "valid_bool_false",
			data:     map[string]string{"id": "1", "slug": "test", "active": "0"},
			wantID:   1,
			wantSlug: "test",
			wantErr:  false,
		},
		{
			name:     "invalid_id_zero_value",
			data:     map[string]string{"id": "0", "slug": "test"},
			wantErr:  true,
			errType:  godantic.ErrorTypeRequired, // 0 is zero value, triggers required error
			errField: "ID",
		},
		{
			name:     "missing_required_slug",
			data:     map[string]string{"id": "1"},
			wantErr:  true,
			errType:  godantic.ErrorTypeRequired,
			errField: "Slug",
		},
		{
			name:     "empty_slug",
			data:     map[string]string{"id": "1", "slug": ""},
			wantErr:  true,
			errType:  godantic.ErrorTypeRequired, // empty string is zero value
			errField: "Slug",
		},
		{
			name:     "unknown_field_ignored",
			data:     map[string]string{"id": "1", "slug": "test", "unknown": "value"},
			wantID:   1,
			wantSlug: "test",
			wantErr:  false,
		},
	}

	validator := godantic.NewValidator[TPathParams]()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, errs := validator.ValidateFromStringMap(tt.data)

			if tt.wantErr {
				if len(errs) == 0 {
					t.Fatal("expected error, got none")
				}
				found := false
				for _, err := range errs {
					if err.Type == tt.errType && (tt.errField == "" || (len(err.Loc) > 0 && err.Loc[len(err.Loc)-1] == tt.errField)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error type %v for field %q, got: %v", tt.errType, tt.errField, errs)
				}
				return
			}

			if len(errs) != 0 {
				t.Fatalf("unexpected errors: %v", errs)
			}
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if result.ID != tt.wantID {
				t.Errorf("ID = %d, want %d", result.ID, tt.wantID)
			}
			if result.Slug != tt.wantSlug {
				t.Errorf("Slug = %q, want %q", result.Slug, tt.wantSlug)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ValidateFromMultiValueMap Tests
// For query params, headers - multiple values per key
// ═══════════════════════════════════════════════════════════════════════════

func TestValidateFromMultiValueMap(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string][]string
		wantPage    int
		wantLimit   int
		wantTags    []string
		wantEnabled bool
		wantScore   float64
		wantErr     bool
		errType     godantic.ErrorType
		errField    string
	}{
		{
			name:      "valid_with_defaults",
			data:      map[string][]string{},
			wantPage:  1,  // default
			wantLimit: 10, // default
			wantErr:   false,
		},
		{
			name:      "valid_override_defaults",
			data:      map[string][]string{"page": {"5"}, "limit": {"25"}},
			wantPage:  5,
			wantLimit: 25,
			wantErr:   false,
		},
		{
			name:     "valid_with_tags_array",
			data:     map[string][]string{"tags": {"go", "rust", "python"}},
			wantPage: 1,
			wantTags: []string{"go", "rust", "python"},
			wantErr:  false,
		},
		{
			name:        "valid_bool_conversion",
			data:        map[string][]string{"enabled": {"true"}},
			wantPage:    1,
			wantEnabled: true,
			wantErr:     false,
		},
		{
			name:      "valid_float_conversion",
			data:      map[string][]string{"score": {"3.14"}},
			wantPage:  1,
			wantScore: 3.14,
			wantErr:   false,
		},
		{
			name:     "invalid_page_below_min",
			data:     map[string][]string{"page": {"-1"}}, // use negative to trigger min constraint
			wantErr:  true,
			errType:  godantic.ErrorTypeConstraint,
			errField: "Page",
		},
		{
			name:     "invalid_limit_above_max",
			data:     map[string][]string{"limit": {"200"}},
			wantErr:  true,
			errType:  godantic.ErrorTypeConstraint,
			errField: "Limit",
		},
		{
			name:      "first_value_used_for_non_array",
			data:      map[string][]string{"page": {"3", "5", "7"}}, // only first is used
			wantPage:  3,
			wantLimit: 10,
			wantErr:   false,
		},
		{
			name:     "empty_values_skipped",
			data:     map[string][]string{"page": {}},
			wantPage: 1, // default used since empty
			wantErr:  false,
		},
	}

	validator := godantic.NewValidator[TQueryParams]()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, errs := validator.ValidateFromMultiValueMap(tt.data)

			if tt.wantErr {
				if len(errs) == 0 {
					t.Fatal("expected error, got none")
				}
				found := false
				for _, err := range errs {
					if err.Type == tt.errType && (tt.errField == "" || (len(err.Loc) > 0 && err.Loc[len(err.Loc)-1] == tt.errField)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error type %v for field %q, got: %v", tt.errType, tt.errField, errs)
				}
				return
			}

			if len(errs) != 0 {
				t.Fatalf("unexpected errors: %v", errs)
			}
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if result.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", result.Page, tt.wantPage)
			}
			if tt.wantLimit != 0 && result.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", result.Limit, tt.wantLimit)
			}
			if tt.wantTags != nil {
				if len(result.Tags) != len(tt.wantTags) {
					t.Errorf("Tags len = %d, want %d", len(result.Tags), len(tt.wantTags))
				}
			}
			if result.Enabled != tt.wantEnabled {
				t.Errorf("Enabled = %v, want %v", result.Enabled, tt.wantEnabled)
			}
			if tt.wantScore != 0 && result.Score != tt.wantScore {
				t.Errorf("Score = %f, want %f", result.Score, tt.wantScore)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Edge Cases
// ═══════════════════════════════════════════════════════════════════════════

func TestValidateFromStringMap_TypeConversions(t *testing.T) {
	type NumericTypes struct {
		Int8Val    int8    `json:"int8_val"`
		Int16Val   int16   `json:"int16_val"`
		Int32Val   int32   `json:"int32_val"`
		Int64Val   int64   `json:"int64_val"`
		Float32Val float32 `json:"float32_val"`
		Float64Val float64 `json:"float64_val"`
	}

	validator := godantic.NewValidator[NumericTypes]()

	result, errs := validator.ValidateFromStringMap(map[string]string{
		"int8_val":    "127",
		"int16_val":   "32767",
		"int32_val":   "2147483647",
		"int64_val":   "9223372036854775807",
		"float32_val": "3.14",
		"float64_val": "2.718281828",
	})

	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if result.Int8Val != 127 {
		t.Errorf("Int8Val = %d, want 127", result.Int8Val)
	}
	if result.Int16Val != 32767 {
		t.Errorf("Int16Val = %d, want 32767", result.Int16Val)
	}
	if result.Int32Val != 2147483647 {
		t.Errorf("Int32Val = %d, want 2147483647", result.Int32Val)
	}
	if result.Int64Val != 9223372036854775807 {
		t.Errorf("Int64Val = %d, want 9223372036854775807", result.Int64Val)
	}
	// Float comparison with tolerance
	if result.Float32Val < 3.13 || result.Float32Val > 3.15 {
		t.Errorf("Float32Val = %f, want ~3.14", result.Float32Val)
	}
	if result.Float64Val < 2.71 || result.Float64Val > 2.72 {
		t.Errorf("Float64Val = %f, want ~2.718", result.Float64Val)
	}
}

func TestValidateFromStringMap_InvalidTypePassthrough(t *testing.T) {
	// When string can't be converted, it's passed as-is and validator handles error
	validator := godantic.NewValidator[TPathParams]()

	_, errs := validator.ValidateFromStringMap(map[string]string{
		"id":   "not-a-number",
		"slug": "test",
	})

	// Should have validation error for invalid type
	if len(errs) == 0 {
		t.Fatal("expected error for invalid int conversion")
	}
}

func TestValidateFromMultiValueMap_EmptySlice(t *testing.T) {
	type WithSlice struct {
		Items []string `json:"items"`
	}

	validator := godantic.NewValidator[WithSlice]()

	// Empty array in multi-value map
	result, errs := validator.ValidateFromMultiValueMap(map[string][]string{
		"items": {},
	})

	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	// Empty slice is skipped, so items will be nil
	if result.Items != nil {
		t.Errorf("Items = %v, want nil", result.Items)
	}
}
