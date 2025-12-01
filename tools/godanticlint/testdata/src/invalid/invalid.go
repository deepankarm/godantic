package invalid

import "github.com/deepankarm/godantic/pkg/godantic"

// ═══════════════════════════════════════════════════════════════════════════
// INVALID TEST CASES - Field name typos and type mismatches
// ═══════════════════════════════════════════════════════════════════════════

// ───────────────────────────────────────────────────────────────────────────
// Field name typos
// ───────────────────────────────────────────────────────────────────────────

type User struct {
	Email    string
	Username string
}

func (u *User) FieldEmal() godantic.FieldOptions[string] { // want "method FieldEmal\\(\\) does not correspond to any field on User"
	return godantic.Field(godantic.Required[string]())
}

func (u *User) FieldEmails() godantic.FieldOptions[string] { // want "method FieldEmails\\(\\) does not correspond to any field on User"
	return godantic.Field(godantic.Required[string]())
}

func (u *User) FieldUsernam() godantic.FieldOptions[string] { // want "method FieldUsernam\\(\\) does not correspond to any field on User"
	return godantic.Field(godantic.Required[string]())
}

// ───────────────────────────────────────────────────────────────────────────
// Non-existent fields
// ───────────────────────────────────────────────────────────────────────────

type Product struct {
	Name  string
	Price float64
}

func (p *Product) FieldNonExistent() godantic.FieldOptions[string] { // want "method FieldNonExistent\\(\\) does not correspond to any field on Product"
	return godantic.Field(godantic.Required[string]())
}

// ───────────────────────────────────────────────────────────────────────────
// Type mismatches - basic types
// ───────────────────────────────────────────────────────────────────────────

func (u *User) FieldEmail() godantic.FieldOptions[int] { // want "method FieldEmail\\(\\) returns FieldOptions\\[int\\] but field Email has type string"
	return godantic.Field(godantic.Required[int]())
}

func (u *User) FieldUsername() godantic.FieldOptions[bool] { // want "method FieldUsername\\(\\) returns FieldOptions\\[bool\\] but field Username has type string"
	return godantic.Field(godantic.Required[bool]())
}

// ───────────────────────────────────────────────────────────────────────────
// Type mismatches - slices
// ───────────────────────────────────────────────────────────────────────────

type Order struct {
	Items []string
	Tags  []int
}

func (o *Order) FieldItems() godantic.FieldOptions[[]int] { // want "method FieldItems\\(\\) returns FieldOptions\\[\\[\\]int\\] but field Items has type \\[\\]string"
	return godantic.Field(godantic.Required[[]int]())
}

func (o *Order) FieldTags() godantic.FieldOptions[[]string] { // want "method FieldTags\\(\\) returns FieldOptions\\[\\[\\]string\\] but field Tags has type \\[\\]int"
	return godantic.Field(godantic.Required[[]string]())
}

// ───────────────────────────────────────────────────────────────────────────
// Type mismatches - maps
// ───────────────────────────────────────────────────────────────────────────

type Config struct {
	Settings map[string]string
}

func (c *Config) FieldSettings() godantic.FieldOptions[map[string]int] { // want "method FieldSettings\\(\\) returns FieldOptions\\[map\\[string\\]int\\] but field Settings has type map\\[string\\]string"
	return godantic.Field(godantic.Required[map[string]int]())
}

// ───────────────────────────────────────────────────────────────────────────
// Type mismatches - pointer vs value
// ───────────────────────────────────────────────────────────────────────────

type UserWithPointers struct {
	Name *string
	Age  int
}

func (u *UserWithPointers) FieldName() godantic.FieldOptions[string] { // want "method FieldName\\(\\) returns FieldOptions\\[string\\] but field Name has type \\*string \\(pointer mismatch\\)"
	return godantic.Field(godantic.Required[string]())
}

func (u *UserWithPointers) FieldAge() godantic.FieldOptions[*int] { // want "method FieldAge\\(\\) returns FieldOptions\\[\\*int\\] but field Age has type int \\(pointer mismatch\\)"
	return godantic.Field(godantic.Required[*int]())
}

// ───────────────────────────────────────────────────────────────────────────
// Wrong return type - not FieldOptions at all
// ───────────────────────────────────────────────────────────────────────────

func (p *Product) FieldName() string { // want "method FieldName\\(\\) must return FieldOptions\\[T\\], got string"
	return ""
}

func (p *Product) FieldPrice() float64 { // want "method FieldPrice\\(\\) must return FieldOptions\\[T\\], got float64"
	return 0
}
