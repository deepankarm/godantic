package godantic_bench

import (
	"encoding/json"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// ============================================================================
// Benchmark Fixtures
// ============================================================================

type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	InStock     bool    `json:"in_stock"`
	Description string  `json:"description"`
}

func (p *Product) FieldID() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Required[int](), godantic.Min(1))
}

func (p *Product) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(1), godantic.MaxLen(200))
}

func (p *Product) FieldPrice() godantic.FieldOptions[float64] {
	return godantic.Field(godantic.Required[float64](), godantic.Min(0.0))
}

func (p *Product) FieldInStock() godantic.FieldOptions[bool] {
	return godantic.Field(godantic.Default(true))
}

func (p *Product) FieldDescription() godantic.FieldOptions[string] {
	return godantic.Field(godantic.MaxLen(1000))
}

// ============================================================================
// Benchmarks: Unmarshal (JSON → Struct)
// ============================================================================

func BenchmarkUnmarshal_Simple(b *testing.B) {
	validator := godantic.NewValidator[Product]()
	data := []byte(`{"id":1,"name":"Widget","price":19.99,"in_stock":true,"description":"A useful widget"}`)

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, errs := validator.Unmarshal(data)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

func BenchmarkUnmarshal_WithDefaults(b *testing.B) {
	validator := godantic.NewValidator[Product]()
	data := []byte(`{"id":1,"name":"Widget","price":19.99}`) // in_stock missing, should use default

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, errs := validator.Unmarshal(data)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

func BenchmarkUnmarshal_Medium(b *testing.B) {
	validator := godantic.NewValidator[MediumUser]()
	data := []byte(`{
		"id":1,
		"username":"johndoe",
		"email":"john@example.com",
		"first_name":"John",
		"last_name":"Doe",
		"phone_number":"+12345678901",
		"age":30,
		"is_active":true,
		"roles":["user","admin"],
		"address":{
			"street":"123 Main St",
			"city":"New York",
			"state":"NY",
			"zip":"10001",
			"country":"US"
		},
		"bio":"Software engineer"
	}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, errs := validator.Unmarshal(data)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

// Comparison: godantic.Unmarshal vs json.Unmarshal
func BenchmarkUnmarshal_Godantic(b *testing.B) {
	validator := godantic.NewValidator[Product]()
	data := []byte(`{"id":1,"name":"Widget","price":19.99,"in_stock":true,"description":"A useful widget"}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, errs := validator.Unmarshal(data)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

func BenchmarkUnmarshal_StdJSON(b *testing.B) {
	data := []byte(`{"id":1,"name":"Widget","price":19.99,"in_stock":true,"description":"A useful widget"}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var product Product
		if err := json.Unmarshal(data, &product); err != nil {
			b.Fatalf("unmarshal failed: %v", err)
		}
	}
}

// ============================================================================
// Benchmarks: Marshal (Struct → JSON)
// ============================================================================

func BenchmarkMarshal_Simple(b *testing.B) {
	validator := godantic.NewValidator[Product]()
	product := Product{
		ID:          1,
		Name:        "Widget",
		Price:       19.99,
		InStock:     true,
		Description: "A useful widget",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, errs := validator.Marshal(&product)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

func BenchmarkMarshal_WithDefaults(b *testing.B) {
	validator := godantic.NewValidator[Product]()
	product := Product{
		ID:    1,
		Name:  "Widget",
		Price: 19.99,
		// InStock not set, should apply default
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, errs := validator.Marshal(&product)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

func BenchmarkMarshal_Medium(b *testing.B) {
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

	for i := 0; i < b.N; i++ {
		_, err := validator.Marshal(&user)
		if err != nil {
			b.Fatalf("unmarshal failed: %v", err)
		}
	}
}

// Comparison: godantic.Marshal vs json.Marshal
func BenchmarkMarshal_Godantic(b *testing.B) {
	validator := godantic.NewValidator[Product]()
	product := Product{
		ID:          1,
		Name:        "Widget",
		Price:       19.99,
		InStock:     true,
		Description: "A useful widget",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, errs := validator.Marshal(&product)
		if len(errs) != 0 {
			b.Fatalf("unexpected validation errors: %v", errs)
		}
	}
}

func BenchmarkMarshal_StdJSON(b *testing.B) {
	product := Product{
		ID:          1,
		Name:        "Widget",
		Price:       19.99,
		InStock:     true,
		Description: "A useful widget",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(&product)
		if err != nil {
			b.Fatalf("marshal failed: %v", err)
		}
	}
}

// ============================================================================
// Benchmarks: ApplyDefaults
// ============================================================================

func BenchmarkApplyDefaults_Simple(b *testing.B) {
	validator := godantic.NewValidator[Product]()
	product := Product{
		ID:    1,
		Name:  "Widget",
		Price: 19.99,
		// InStock missing, should apply default
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := validator.ApplyDefaults(&product)
		if err != nil {
			b.Fatalf("apply defaults failed: %v", err)
		}
	}
}
