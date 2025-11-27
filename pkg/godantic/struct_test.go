package godantic_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// ═══════════════════════════════════════════════════════════════════════════
// Struct-specific test types (embedded, pointers, non-serializable fields)
// ═══════════════════════════════════════════════════════════════════════════

// TBaseModel tests embedded structs and time.Time validation
type TBaseModel struct {
	ID        int
	CreatedAt time.Time
}

func (b *TBaseModel) FieldID() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Required[int](), godantic.Min(1))
}

func (b *TBaseModel) FieldCreatedAt() godantic.FieldOptions[time.Time] {
	return godantic.Field(
		godantic.Required[time.Time](),
		godantic.Validate(func(t time.Time) error {
			if t.After(time.Now()) {
				return fmt.Errorf("timestamp cannot be in the future")
			}
			if t.Before(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) {
				return fmt.Errorf("timestamp must be after 2000-01-01")
			}
			return nil
		}),
	)
}

// TAddressWithZip extends address with ZipCode for constraint testing
type TAddressWithZip struct {
	Street  string
	City    string
	ZipCode string
}

func (a *TAddressWithZip) FieldStreet() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(3))
}

func (a *TAddressWithZip) FieldCity() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(2))
}

func (a *TAddressWithZip) FieldZipCode() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(5), godantic.MaxLen(10))
}

// TComplexUser tests embedding, pointers, and non-serializable fields
type TComplexUser struct {
	TBaseModel                  // Embedded struct
	Name       string           // Regular field
	Email      *string          // Pointer field
	HomeAddr   TAddressWithZip  // Nested struct
	WorkAddr   *TAddressWithZip // Pointer to struct
	mu         sync.Mutex       // Non-serializable (unexported)
	Ch         chan string      // Channel (exported but non-serializable)
	Age        int              // Regular field
}

func (c *TComplexUser) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(2))
}

func (c *TComplexUser) FieldEmail() godantic.FieldOptions[*string] {
	return godantic.Field(godantic.Required[*string]())
}

func (c *TComplexUser) FieldHomeAddr() godantic.FieldOptions[TAddressWithZip] {
	return godantic.Field(godantic.Required[TAddressWithZip]())
}

func (c *TComplexUser) FieldAge() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Required[int](), godantic.Min(0), godantic.Max(150))
}

func TestEmbeddedStructs(t *testing.T) {
	validator := godantic.NewValidator[TBaseModel]()

	tests := []struct {
		name    string
		base    TBaseModel
		wantErr int
	}{
		{"empty_fails", TBaseModel{ID: 0, CreatedAt: time.Time{}}, 2},
		{"valid_passes", TBaseModel{ID: 42, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, 0},
		{"future_timestamp_fails", TBaseModel{ID: 42, CreatedAt: time.Now().Add(24 * time.Hour)}, 1},
		{"old_timestamp_fails", TBaseModel{ID: 42, CreatedAt: time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC)}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.Validate(&tt.base)
			if len(errs) != tt.wantErr {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErr, len(errs), errs)
			}
		})
	}
}

func TestNestedStructs(t *testing.T) {
	validator := godantic.NewValidator[TAddressWithZip]()

	t.Run("invalid_fails", func(t *testing.T) {
		addr := TAddressWithZip{Street: "A", City: "B", ZipCode: "123"}
		errs := validator.Validate(&addr)
		if len(errs) != 3 {
			t.Errorf("expected 3 errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("valid_passes", func(t *testing.T) {
		addr := TAddressWithZip{Street: "123 Main St", City: "NYC", ZipCode: "10001"}
		errs := validator.Validate(&addr)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})
}

func TestComplexStructWithEmbedding(t *testing.T) {
	validator := godantic.NewValidator[TComplexUser]()
	email := "test@example.com"
	validAddr := TAddressWithZip{Street: "123 Main St", City: "NYC", ZipCode: "10001"}
	validTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("valid_passes", func(t *testing.T) {
		c := TComplexUser{
			TBaseModel: TBaseModel{ID: 1, CreatedAt: validTime},
			Name:       "John Doe", Email: &email, HomeAddr: validAddr, Age: 30,
			Ch: make(chan string),
		}
		errs := validator.Validate(&c)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("nil_required_pointer_fails", func(t *testing.T) {
		c := TComplexUser{
			TBaseModel: TBaseModel{ID: 1, CreatedAt: validTime},
			Name:       "Jane", Email: nil, HomeAddr: validAddr, Age: 25,
		}
		errs := validator.Validate(&c)
		if len(errs) != 1 || errs[0].Error() != "Email: required field" {
			t.Errorf("expected Email required error, got: %v", errs)
		}
	})

	t.Run("nil_optional_pointer_passes", func(t *testing.T) {
		c := TComplexUser{
			TBaseModel: TBaseModel{ID: 1, CreatedAt: validTime},
			Name:       "Bob", Email: &email, HomeAddr: validAddr, WorkAddr: nil, Age: 35,
		}
		errs := validator.Validate(&c)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("non_nil_pointer_struct_passes", func(t *testing.T) {
		workAddr := TAddressWithZip{Street: "456 Corp", City: "LA", ZipCode: "90001"}
		c := TComplexUser{
			TBaseModel: TBaseModel{ID: 1, CreatedAt: validTime},
			Name:       "Alice", Email: &email, HomeAddr: validAddr, WorkAddr: &workAddr, Age: 28,
		}
		errs := validator.Validate(&c)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("invalid_age_fails", func(t *testing.T) {
		c := TComplexUser{
			TBaseModel: TBaseModel{ID: 1, CreatedAt: validTime},
			Name:       "Old", Email: &email, HomeAddr: validAddr, Age: 200,
		}
		errs := validator.Validate(&c)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d: %v", len(errs), errs)
		}
	})

	t.Run("multiple_failures", func(t *testing.T) {
		c := TComplexUser{
			TBaseModel: TBaseModel{ID: 0, CreatedAt: time.Time{}},
			Name:       "X", Email: nil, HomeAddr: TAddressWithZip{}, Age: -5,
		}
		errs := validator.Validate(&c)
		if len(errs) < 5 {
			t.Errorf("expected >= 5 errors, got %d: %v", len(errs), errs)
		}
	})
}

func TestNonSerializableFields(t *testing.T) {
	validator := godantic.NewValidator[TComplexUser]()
	email := "test@example.com"
	validTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	validAddr := TAddressWithZip{Street: "123 Main St", City: "NYC", ZipCode: "10001"}

	c := TComplexUser{
		TBaseModel: TBaseModel{ID: 1, CreatedAt: validTime},
		Name:       "Test", Email: &email, HomeAddr: validAddr, Age: 25,
		mu: sync.Mutex{}, Ch: make(chan string),
	}

	// Should not panic
	errs := validator.Validate(&c)
	if len(errs) != 0 {
		t.Errorf("expected no errors with non-serializable fields, got %d: %v", len(errs), errs)
	}
}

func TestPointerFields(t *testing.T) {
	validator := godantic.NewValidator[TComplexUser]()
	email := "test@example.com"
	validTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	validAddr := TAddressWithZip{Street: "123 Main St", City: "NYC", ZipCode: "10001"}

	t.Run("nil_required_fails", func(t *testing.T) {
		c := TComplexUser{TBaseModel: TBaseModel{ID: 1, CreatedAt: validTime}, Name: "Test", Email: nil, HomeAddr: validAddr, Age: 25}
		errs := validator.Validate(&c)
		found := false
		for _, err := range errs {
			if err.Error() == "Email: required field" {
				found = true
			}
		}
		if !found {
			t.Error("expected Email required error")
		}
	})

	t.Run("non_nil_passes", func(t *testing.T) {
		c := TComplexUser{TBaseModel: TBaseModel{ID: 1, CreatedAt: validTime}, Name: "Test", Email: &email, HomeAddr: validAddr, Age: 25}
		errs := validator.Validate(&c)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})
}

func TestNestedRequiredFieldErrors(t *testing.T) {
	validator := godantic.NewValidator[TComplexUser]()
	email := "test@example.com"
	validTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	c := TComplexUser{
		TBaseModel: TBaseModel{ID: 1, CreatedAt: validTime},
		Name:       "Test", Email: &email, HomeAddr: TAddressWithZip{}, Age: 25,
	}

	errs := validator.Validate(&c)
	if len(errs) < 3 {
		t.Fatalf("expected >= 3 errors for empty Address, got %d: %v", len(errs), errs)
	}

	// Check errors show full path: HomeAddr.Street, etc.
	var foundStreet, foundCity, foundZip bool
	for _, err := range errs {
		if len(err.Loc) == 2 && err.Loc[0] == "HomeAddr" {
			switch err.Loc[1] {
			case "Street":
				foundStreet = true
			case "City":
				foundCity = true
			case "ZipCode":
				foundZip = true
			}
		}
	}

	if !foundStreet || !foundCity || !foundZip {
		t.Errorf("expected HomeAddr.Street/City/ZipCode errors, got: %v", errs)
	}
}
