package godantic_bench

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

// ============================================================================
// Benchmark Fixtures
// ============================================================================

// Simple struct
type SimpleModel struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func (m *SimpleModel) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(1),
		godantic.MaxLen(100),
		godantic.Description[string]("User's full name"),
	)
}

func (m *SimpleModel) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Email(),
		godantic.Description[string]("User's email address"),
	)
}

func (m *SimpleModel) FieldAge() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Min(0),
		godantic.Max(150),
		godantic.Description[int]("User's age in years"),
	)
}

// Medium struct with nesting
type SchemaAddress struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
}

func (a *SchemaAddress) FieldStreet() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(1))
}

func (a *SchemaAddress) FieldCity() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(1))
}

func (a *SchemaAddress) FieldState() godantic.FieldOptions[string] {
	return godantic.Field(godantic.MinLen(2), godantic.MaxLen(2))
}

func (a *SchemaAddress) FieldZip() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (a *SchemaAddress) FieldCountry() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(2), godantic.MaxLen(2))
}

type MediumModel struct {
	ID       int           `json:"id"`
	Username string        `json:"username"`
	Email    string        `json:"email"`
	Roles    []string      `json:"roles"`
	Address  SchemaAddress `json:"address"`
}

func (m *MediumModel) FieldID() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Required[int](), godantic.Min(1))
}

func (m *MediumModel) FieldUsername() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(3), godantic.MaxLen(20))
}

func (m *MediumModel) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.Email())
}

func (m *MediumModel) FieldRoles() godantic.FieldOptions[[]string] {
	return godantic.Field(godantic.Required[[]string]())
}

func (m *MediumModel) FieldAddress() godantic.FieldOptions[SchemaAddress] {
	return godantic.Field(godantic.Required[SchemaAddress]())
}

// Discriminated union
type SchemaAnimal interface {
	IsAnimal()
}

type SchemaDog struct {
	Species string `json:"species"`
	Breed   string `json:"breed"`
}

func (SchemaDog) IsAnimal() {}

type SchemaCat struct {
	Species string `json:"species"`
	Lives   int    `json:"lives"`
}

func (SchemaCat) IsAnimal() {}

type SchemaPetOwner struct {
	Name string       `json:"name"`
	Pet  SchemaAnimal `json:"pet"`
}

func (p *SchemaPetOwner) FieldPet() godantic.FieldOptions[SchemaAnimal] {
	return godantic.Field(
		godantic.Required[SchemaAnimal](),
		godantic.DiscriminatedUnion[SchemaAnimal](
			"species",
			map[string]any{
				"dog": SchemaDog{},
				"cat": SchemaCat{},
			},
		),
	)
}

// ============================================================================
// Benchmarks: Schema Generation
// ============================================================================

func BenchmarkSchemaGeneration_Simple(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		gen := schema.NewGenerator[SimpleModel]()
		_, err := gen.Generate()
		if err != nil {
			b.Fatalf("schema generation failed: %v", err)
		}
	}
}

func BenchmarkSchemaGeneration_Medium(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		gen := schema.NewGenerator[MediumModel]()
		_, err := gen.Generate()
		if err != nil {
			b.Fatalf("schema generation failed: %v", err)
		}
	}
}

func BenchmarkSchemaGeneration_DiscriminatedUnion(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		gen := schema.NewGenerator[SchemaPetOwner]()
		_, err := gen.Generate()
		if err != nil {
			b.Fatalf("schema generation failed: %v", err)
		}
	}
}

// ============================================================================
// Benchmarks: Generator Reuse vs Recreation
// ============================================================================

func BenchmarkSchemaGeneration_ReuseGenerator(b *testing.B) {
	gen := schema.NewGenerator[SimpleModel]()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := gen.Generate()
		if err != nil {
			b.Fatalf("schema generation failed: %v", err)
		}
	}
}

func BenchmarkSchemaGeneration_RecreateGenerator(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		gen := schema.NewGenerator[SimpleModel]()
		_, err := gen.Generate()
		if err != nil {
			b.Fatalf("schema generation failed: %v", err)
		}
	}
}

// ============================================================================
// Benchmarks: Different Options
// ============================================================================

func BenchmarkSchemaGeneration_WithAutoTitles(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		gen := schema.NewGenerator[SimpleModel]().WithAutoTitles(true)
		_, err := gen.Generate()
		if err != nil {
			b.Fatalf("schema generation failed: %v", err)
		}
	}
}

func BenchmarkSchemaGeneration_WithoutAutoTitles(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		gen := schema.NewGenerator[SimpleModel]().WithAutoTitles(false)
		_, err := gen.Generate()
		if err != nil {
			b.Fatalf("schema generation failed: %v", err)
		}
	}
}
