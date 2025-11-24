package godantic_bench

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// ============================================================================
// Benchmark Fixtures
// ============================================================================

// Simple struct (3-5 fields) - baseline
type SimpleUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func (u *SimpleUser) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(1),
		godantic.MaxLen(100),
	)
}

func (u *SimpleUser) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Email(),
	)
}

func (u *SimpleUser) FieldAge() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Min(0),
		godantic.Max(150),
	)
}

// Medium struct (10-20 fields, nested) - typical API
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
}

func (a *Address) FieldStreet() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(1))
}

func (a *Address) FieldCity() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(1))
}

func (a *Address) FieldState() godantic.FieldOptions[string] {
	return godantic.Field(godantic.MinLen(2), godantic.MaxLen(2))
}

func (a *Address) FieldZip() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (a *Address) FieldCountry() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(2), godantic.MaxLen(2))
}

type MediumUser struct {
	ID          int      `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	PhoneNumber string   `json:"phone_number"`
	Age         int      `json:"age"`
	IsActive    bool     `json:"is_active"`
	Roles       []string `json:"roles"`
	Address     Address  `json:"address"`
	Bio         string   `json:"bio"`
}

func (u *MediumUser) FieldID() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Required[int](), godantic.Min(1))
}

func (u *MediumUser) FieldUsername() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(3), godantic.MaxLen(20))
}

func (u *MediumUser) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.Email())
}

func (u *MediumUser) FieldFirstName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(1), godantic.MaxLen(50))
}

func (u *MediumUser) FieldLastName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(1), godantic.MaxLen(50))
}

func (u *MediumUser) FieldPhoneNumber() godantic.FieldOptions[string] {
	return godantic.Field(godantic.MinLen(0))
}

func (u *MediumUser) FieldAge() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Min(0), godantic.Max(150))
}

func (u *MediumUser) FieldRoles() godantic.FieldOptions[[]string] {
	return godantic.Field(godantic.Required[[]string]())
}

func (u *MediumUser) FieldAddress() godantic.FieldOptions[Address] {
	return godantic.Field(godantic.Required[Address]())
}

func (u *MediumUser) FieldBio() godantic.FieldOptions[string] {
	return godantic.Field(godantic.MaxLen(500))
}

// ============================================================================
// Benchmarks: Validation
// ============================================================================

func BenchmarkValidate_Simple(b *testing.B) {
	validator := godantic.NewValidator[SimpleUser]()
	user := SimpleUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		errs := validator.Validate(&user)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

func BenchmarkValidate_Medium(b *testing.B) {
	validator := godantic.NewValidator[MediumUser]()
	user := MediumUser{
		ID:          1,
		Username:    "johndoe",
		Email:       "john@example.com",
		FirstName:   "John",
		LastName:    "Doe",
		PhoneNumber: "+12345678901",
		Age:         30,
		IsActive:    true,
		Roles:       []string{"user", "admin"},
		Address: Address{
			Street:  "123 Main St",
			City:    "New York",
			State:   "NY",
			Zip:     "10001",
			Country: "US",
		},
		Bio: "Software engineer",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		errs := validator.Validate(&user)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

// ============================================================================
// Benchmarks: Constraint Types
// ============================================================================

func BenchmarkConstraint_Email(b *testing.B) {
	validator := godantic.NewValidator[SimpleUser]()
	user := SimpleUser{
		Name:  "John Doe",
		Email: "john.doe+test@example.com",
		Age:   30,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		errs := validator.Validate(&user)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

func BenchmarkConstraint_Pattern(b *testing.B) {
	validator := godantic.NewValidator[MediumUser]()
	user := MediumUser{
		ID:          1,
		Username:    "johndoe",
		Email:       "john@example.com",
		FirstName:   "John",
		LastName:    "Doe",
		PhoneNumber: "+12345678901",
		Roles:       []string{"user"},
		Address: Address{
			Street:  "123 Main St",
			City:    "New York",
			State:   "NY",
			Zip:     "10001-1234",
			Country: "US",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		errs := validator.Validate(&user)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

func BenchmarkConstraint_MinMaxLen(b *testing.B) {
	validator := godantic.NewValidator[SimpleUser]()
	user := SimpleUser{
		Name:  "John Doe with a longer name",
		Email: "john@example.com",
		Age:   30,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		errs := validator.Validate(&user)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

func BenchmarkConstraint_MinMaxValue(b *testing.B) {
	validator := godantic.NewValidator[SimpleUser]()
	user := SimpleUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   75,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		errs := validator.Validate(&user)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

// ============================================================================
// Benchmarks: Validator Creation
// ============================================================================

func BenchmarkValidatorCreation_Simple(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = godantic.NewValidator[SimpleUser]()
	}
}

func BenchmarkValidatorCreation_Medium(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = godantic.NewValidator[MediumUser]()
	}
}

// ============================================================================
// Benchmarks: Validator Reuse vs Recreation
// ============================================================================

func BenchmarkValidate_ReuseValidator(b *testing.B) {
	validator := godantic.NewValidator[SimpleUser]()
	user := SimpleUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		errs := validator.Validate(&user)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

func BenchmarkValidate_RecreateValidator(b *testing.B) {
	user := SimpleUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		validator := godantic.NewValidator[SimpleUser]()
		errs := validator.Validate(&user)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}
