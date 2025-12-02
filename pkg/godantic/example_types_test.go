package godantic_test

import (
	"fmt"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Types for ExampleValidator_Unmarshal_nestedStruct
type exampleAddress struct {
	City string `json:"city"`
}

func (a *exampleAddress) FieldCity() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

type exampleTimestamps struct {
	CreatedAt string `json:"created_at"`
}

type exampleUserNested struct {
	exampleTimestamps                // Embedded
	Name              string         `json:"name"`
	Address           exampleAddress `json:"address"` // Nested
}

func (u *exampleUserNested) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// ExampleValidator_Unmarshal_nestedStruct demonstrates handling nested
// and embedded structs with validation.
func ExampleValidator_Unmarshal_nestedStruct() {
	v := godantic.NewValidator[exampleUserNested]()
	user, _ := v.Unmarshal([]byte(`{
		"name": "Alice",
		"address": {"city": "NYC"},
		"created_at": "2024-01-01"
	}`))

	fmt.Printf("Name: %s, City: %s\n", user.Name, user.Address.City)
	// Output: Name: Alice, City: NYC
}

// Types for ExampleValidator_Unmarshal_pointerFields
type exampleUserPointers struct {
	Name    *string         `json:"name"`
	Age     *int            `json:"age"`
	Address *exampleAddress `json:"address"`
}

func (u *exampleUserPointers) FieldName() godantic.FieldOptions[*string] {
	return godantic.Field(godantic.Required[*string]())
}

func (u *exampleUserPointers) FieldAge() godantic.FieldOptions[*int] {
	return godantic.Field[*int]() // Optional field, no constraints needed
}

// ExampleValidator_Unmarshal_pointerFields demonstrates handling pointer
// fields, including optional fields and nil checks.
func ExampleValidator_Unmarshal_pointerFields() {
	v := godantic.NewValidator[exampleUserPointers]()
	user, _ := v.Unmarshal([]byte(`{
		"name": "Bob",
		"age": 30,
		"address": {"city": "SF"}
	}`))

	fmt.Printf("Name: %s, Age: %d, City: %s\n", *user.Name, *user.Age, user.Address.City)
	// Output: Name: Bob, Age: 30, City: SF
}

// Types for ExampleValidator_Unmarshal_slices
type exampleItem struct {
	Name string `json:"name"`
}

func (i *exampleItem) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

type exampleUserWithSlice struct {
	Name  string        `json:"name"`
	Items []exampleItem `json:"items"`
}

func (u *exampleUserWithSlice) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (u *exampleUserWithSlice) FieldItems() godantic.FieldOptions[[]exampleItem] {
	return godantic.Field(godantic.Required[[]exampleItem]())
}

// ExampleValidator_Unmarshal_slices demonstrates handling slices in structs
// and root-level slices of structs.
func ExampleValidator_Unmarshal_slices() {
	// Slice in struct
	v1 := godantic.NewValidator[exampleUserWithSlice]()
	user, _ := v1.Unmarshal([]byte(`{
		"name": "Charlie",
		"items": [{"name": "A"}, {"name": "B"}]
	}`))
	fmt.Printf("User: %s, Items: %d\n", user.Name, len(user.Items))

	// Root-level slice
	v2 := godantic.NewValidator[[]exampleItem]()
	items, _ := v2.Unmarshal([]byte(`[{"name": "A"}, {"name": "B"}]`))
	fmt.Printf("Root slice length: %d\n", len(*items))

	// Output:
	// User: Charlie, Items: 2
	// Root slice length: 2
}

// Types for ExampleValidator_Unmarshal_maps
type exampleUserWithMap struct {
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
	Scores   map[string]int    `json:"scores"`
}

func (u *exampleUserWithMap) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// ExampleValidator_Unmarshal_maps demonstrates handling map fields
// with string keys and various value types.
func ExampleValidator_Unmarshal_maps() {
	v := godantic.NewValidator[exampleUserWithMap]()
	user, _ := v.Unmarshal([]byte(`{
		"name": "David",
		"metadata": {"role": "admin", "dept": "eng"},
		"scores": {"math": 95, "science": 88}
	}`))

	fmt.Printf("Name: %s, Role: %s, Math Score: %d\n",
		user.Name, user.Metadata["role"], user.Scores["math"])
	// Output: Name: David, Role: admin, Math Score: 95
}
