package godantic_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Test embedded structs
type BaseModel struct {
	ID        int
	CreatedAt time.Time
}

func (b *BaseModel) FieldID() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.Min(1),
	)
}

func (b *BaseModel) FieldCreatedAt() godantic.FieldOptions[time.Time] {
	return godantic.Field(
		godantic.Required[time.Time](),
		godantic.Validate(func(t time.Time) error {
			// Validate that timestamp is not in the future
			if t.After(time.Now()) {
				return fmt.Errorf("timestamp cannot be in the future")
			}
			// Validate that timestamp is not too old (e.g., before 2000)
			if t.Before(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) {
				return fmt.Errorf("timestamp must be after 2000-01-01")
			}
			return nil
		}),
	)
}

// Nested struct
type Address struct {
	Street  string
	City    string
	ZipCode string
}

func (a *Address) FieldStreet() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(3),
	)
}

func (a *Address) FieldCity() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(2),
	)
}

func (a *Address) FieldZipCode() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(5),
		godantic.MaxLen(10),
	)
}

// Complex struct with embedding, nested structs, pointers, and non-serializable fields
type ComplexUser struct {
	BaseModel             // Embedded struct
	Name      string      // Regular field
	Email     *string     // Pointer field
	HomeAddr  Address     // Nested struct
	WorkAddr  *Address    // Pointer to struct
	mu        sync.Mutex  // Non-serializable (unexported)
	Ch        chan string // Channel (exported but non-serializable)
	Age       int         // Regular field
}

func (c *ComplexUser) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(2),
	)
}

func (c *ComplexUser) FieldEmail() godantic.FieldOptions[*string] {
	return godantic.Field(godantic.Required[*string]())
}

func (c *ComplexUser) FieldHomeAddr() godantic.FieldOptions[Address] {
	return godantic.Field(godantic.Required[Address]())
}

func (c *ComplexUser) FieldAge() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.Min(0),
		godantic.Max(150),
	)
}

func TestEmbeddedStructs(t *testing.T) {
	validator := godantic.NewValidator[BaseModel]()

	t.Run("empty embedded struct should fail", func(t *testing.T) {
		base := BaseModel{ID: 0, CreatedAt: time.Time{}} // Zero time
		errs := validator.Validate(&base)
		if len(errs) != 2 {
			t.Errorf("expected 2 errors, got %d", len(errs))
		}
	})

	t.Run("valid embedded struct should pass", func(t *testing.T) {
		base := BaseModel{ID: 42, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
		errs := validator.Validate(&base)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("future timestamp should fail", func(t *testing.T) {
		futureTime := time.Now().Add(24 * time.Hour)
		base := BaseModel{ID: 42, CreatedAt: futureTime}
		errs := validator.Validate(&base)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
	})

	t.Run("timestamp before 2000 should fail", func(t *testing.T) {
		oldTime := time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC)
		base := BaseModel{ID: 42, CreatedAt: oldTime}
		errs := validator.Validate(&base)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
	})
}

func TestNestedStructs(t *testing.T) {
	validator := godantic.NewValidator[Address]()

	t.Run("invalid nested struct should fail", func(t *testing.T) {
		addr := Address{Street: "A", City: "B", ZipCode: "123"}
		errs := validator.Validate(&addr)
		if len(errs) != 3 {
			t.Errorf("expected 3 errors, got %d", len(errs))
		}
		// Check that we got the expected errors (order doesn't matter)
		errorMsgs := make(map[string]bool)
		for _, err := range errs {
			errorMsgs[err.Error()] = true
		}
		expectedErrors := []string{
			"City: length must be >= 2",
			"Street: length must be >= 3",
			"ZipCode: length must be >= 5",
		}
		for _, expected := range expectedErrors {
			if !errorMsgs[expected] {
				t.Errorf("expected error '%s' not found in: %v", expected, errs)
			}
		}
	})

	t.Run("valid nested struct should pass", func(t *testing.T) {
		addr := Address{Street: "123 Main St", City: "NYC", ZipCode: "10001"}
		errs := validator.Validate(&addr)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})
}

func TestComplexStructWithEmbedding(t *testing.T) {
	validator := godantic.NewValidator[ComplexUser]()
	email := "test@example.com"

	t.Run("valid complex struct should pass", func(t *testing.T) {
		complex := ComplexUser{
			BaseModel: BaseModel{ID: 1, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			Name:      "John Doe",
			Email:     &email,
			HomeAddr:  Address{Street: "123 Main St", City: "NYC", ZipCode: "10001"},
			Age:       30,
			Ch:        make(chan string), // Non-serializable field
		}

		errs := validator.Validate(&complex)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("nil pointer in required field should fail", func(t *testing.T) {
		complex := ComplexUser{
			BaseModel: BaseModel{ID: 1, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			Name:      "Jane Doe",
			Email:     nil, // Required but nil
			HomeAddr:  Address{Street: "123 Main St", City: "NYC", ZipCode: "10001"},
			Age:       25,
		}

		errs := validator.Validate(&complex)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Email: required field" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("nil pointer to optional struct should pass", func(t *testing.T) {
		complex := ComplexUser{
			BaseModel: BaseModel{ID: 1, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			Name:      "Bob Smith",
			Email:     &email,
			HomeAddr:  Address{Street: "123 Main St", City: "NYC", ZipCode: "10001"},
			WorkAddr:  nil, // Optional pointer, nil is ok
			Age:       35,
		}

		errs := validator.Validate(&complex)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("non-nil pointer to nested struct should pass", func(t *testing.T) {
		workAddr := Address{Street: "456 Corp Blvd", City: "LA", ZipCode: "90001"}
		complex := ComplexUser{
			BaseModel: BaseModel{ID: 1, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			Name:      "Alice Johnson",
			Email:     &email,
			HomeAddr:  Address{Street: "123 Main St", City: "NYC", ZipCode: "10001"},
			WorkAddr:  &workAddr,
			Age:       28,
		}

		errs := validator.Validate(&complex)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("invalid age in complex struct should fail", func(t *testing.T) {
		complex := ComplexUser{
			BaseModel: BaseModel{ID: 1, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			Name:      "Old Person",
			Email:     &email,
			HomeAddr:  Address{Street: "123 Main St", City: "NYC", ZipCode: "10001"},
			Age:       200, // Invalid: > 150
		}

		errs := validator.Validate(&complex)
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
		if errs[0].Error() != "Age: value must be <= 150" {
			t.Errorf("unexpected error: %v", errs[0])
		}
	})

	t.Run("multiple validation failures", func(t *testing.T) {
		complex := ComplexUser{
			BaseModel: BaseModel{ID: 0, CreatedAt: time.Time{}}, // Both invalid
			Name:      "X",                                      // Too short
			Email:     nil,                                      // Required but nil
			HomeAddr:  Address{},                                // All fields invalid
			Age:       -5,                                       // Invalid: < 0
		}

		errs := validator.Validate(&complex)
		if len(errs) < 5 {
			t.Errorf("expected at least 5 errors, got %d: %v", len(errs), errs)
		}
	})
}

func TestNonSerializableFields(t *testing.T) {
	validator := godantic.NewValidator[ComplexUser]()
	email := "test@example.com"

	t.Run("struct with mutex and channel should validate fine", func(t *testing.T) {
		complex := ComplexUser{
			BaseModel: BaseModel{ID: 1, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			Name:      "Test User",
			Email:     &email,
			HomeAddr:  Address{Street: "123 Main St", City: "NYC", ZipCode: "10001"},
			Age:       25,
			mu:        sync.Mutex{},      // Unexported, should be ignored
			Ch:        make(chan string), // Should be ignored
		}

		// Should not panic
		errs := validator.Validate(&complex)
		if len(errs) != 0 {
			t.Errorf("expected no errors with non-serializable fields, got %d: %v", len(errs), errs)
		}
	})
}

func TestPointerFields(t *testing.T) {
	validator := godantic.NewValidator[ComplexUser]()

	t.Run("required pointer field nil should fail", func(t *testing.T) {
		complex := ComplexUser{
			BaseModel: BaseModel{ID: 1, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			Name:      "Test",
			Email:     nil,
			HomeAddr:  Address{Street: "123 Main St", City: "NYC", ZipCode: "10001"},
			Age:       25,
		}

		errs := validator.Validate(&complex)
		hasEmailError := false
		for _, err := range errs {
			if err.Error() == "Email: required field" {
				hasEmailError = true
				break
			}
		}
		if !hasEmailError {
			t.Error("expected Email required field error")
		}
	})

	t.Run("required pointer field non-nil should pass", func(t *testing.T) {
		email := "test@example.com"
		complex := ComplexUser{
			BaseModel: BaseModel{ID: 1, CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			Name:      "Test",
			Email:     &email,
			HomeAddr:  Address{Street: "123 Main St", City: "NYC", ZipCode: "10001"},
			Age:       25,
		}

		errs := validator.Validate(&complex)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})
}
