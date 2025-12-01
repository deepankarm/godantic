package valid

import "github.com/deepankarm/godantic/pkg/godantic"

// Minimal valid cases - just enough to verify no false positives.
// Full patterns are already tested in pkg/godantic/testdata_test.go

// ───────────────────────────────────────────────────────────────────────────
// Basic struct
// ───────────────────────────────────────────────────────────────────────────

type User struct {
	Name  string
	Email string
	Age   int
}

func (u *User) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (u *User) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (u *User) FieldAge() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Required[int]())
}

// ───────────────────────────────────────────────────────────────────────────
// Complex types: slice, map, nested struct, pointer
// ───────────────────────────────────────────────────────────────────────────

type Address struct {
	City string
}

type Order struct {
	Items    []string
	Metadata map[string]int
	Shipping Address
	Billing  *Address
}

func (o *Order) FieldItems() godantic.FieldOptions[[]string] {
	return godantic.Field(godantic.Required[[]string]())
}

func (o *Order) FieldMetadata() godantic.FieldOptions[map[string]int] {
	return godantic.Field(godantic.Required[map[string]int]())
}

func (o *Order) FieldShipping() godantic.FieldOptions[Address] {
	return godantic.Field(godantic.Required[Address]())
}

func (o *Order) FieldBilling() godantic.FieldOptions[*Address] {
	return godantic.Field[*Address]()
}

// ───────────────────────────────────────────────────────────────────────────
// Non-Field methods - should be ignored
// ───────────────────────────────────────────────────────────────────────────

func (u *User) GetName() string { // Doesn't start with "Field", ignored
	return u.Name
}

// ═══════════════════════════════════════════════════════════════════════════
// EMBEDDED STRUCT TEST CASES - Not covered in testdata_test.go
// ═══════════════════════════════════════════════════════════════════════════

// ───────────────────────────────────────────────────────────────────────────
// Simple embedded struct
// ───────────────────────────────────────────────────────────────────────────

type Base struct {
	ID int
}

type Extended struct {
	Base
	Name string
}

func (e *Extended) FieldID() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Required[int]())
}

func (e *Extended) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// ───────────────────────────────────────────────────────────────────────────
// Deeply nested embedded (through pointer)
// ───────────────────────────────────────────────────────────────────────────

type DeepBase struct {
	DeepID int
}

type MiddleLayer struct {
	*DeepBase
	MiddleName string
}

type TopLevel struct {
	MiddleLayer
	TopName string
}

func (t *TopLevel) FieldDeepID() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Required[int]())
}

func (t *TopLevel) FieldMiddleName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (t *TopLevel) FieldTopName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// Test that nolint comments work correctly

type TestStruct struct {
	Name string
}

func (t *TestStruct) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// nolint:godanticlint
// FieldOptions is a getter method, not a validation method
func (t *TestStruct) FieldOptions() map[string]any {
	return nil
}

// nolint:godanticlint
func (t *TestStruct) FieldSomething() string {
	return ""
}
