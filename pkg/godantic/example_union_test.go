package godantic_test

import (
	"fmt"
	"strings"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Types for ExampleNewValidator_discriminatedUnion
type exampleAnimal interface {
	Sound() string
}

type exampleCat struct {
	Species string `json:"species"`
	Name    string `json:"name"`
}

func (c exampleCat) Sound() string { return "meow" }

func (c *exampleCat) FieldSpecies() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Const("cat"))
}

func (c *exampleCat) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

type exampleDog struct {
	Species string `json:"species"`
	Name    string `json:"name"`
	Breed   string `json:"breed"`
}

func (d exampleDog) Sound() string { return "woof" }

func (d *exampleDog) FieldSpecies() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Const("dog"))
}

func (d *exampleDog) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (d *exampleDog) FieldBreed() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// ExampleNewValidator_discriminatedUnion demonstrates creating a validator
// for an interface type using WithDiscriminator to handle multiple variants.
func ExampleNewValidator_discriminatedUnion() {
	v := godantic.NewValidator[exampleAnimal](
		godantic.WithDiscriminator("species", map[string]any{
			"cat": exampleCat{},
			"dog": exampleDog{},
		}),
	)

	animal, _ := v.Unmarshal([]byte(`{"species": "dog", "name": "Rex", "breed": "Labrador"}`))
	fmt.Printf("Sound: %s\n", (*animal).Sound())
	// Output: Sound: woof
}

// Types for ExampleDiscriminatedUnion_fieldLevel
type exampleTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (t *exampleTextContent) FieldType() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Const("text"))
}

func (t *exampleTextContent) FieldText() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

type exampleImageContent struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func (i *exampleImageContent) FieldType() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Const("image"))
}

func (i *exampleImageContent) FieldURL() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

type exampleMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

func (m *exampleMessage) FieldContent() godantic.FieldOptions[any] {
	return godantic.Field(
		godantic.DiscriminatedUnion[any]("type", map[string]any{
			"text":  exampleTextContent{},
			"image": exampleImageContent{},
		}),
	)
}

// ExampleDiscriminatedUnion_fieldLevel demonstrates using DiscriminatedUnion
// as a field constraint within a struct, allowing polymorphic field values.
func ExampleDiscriminatedUnion_fieldLevel() {
	v := godantic.NewValidator[exampleMessage]()
	msg, _ := v.Unmarshal([]byte(`{
		"role": "user",
		"content": {"type": "text", "text": "Hello"}
	}`))

	contentType := fmt.Sprintf("%T", msg.Content)
	fmt.Printf("Role: %s, Content type: %s\n", msg.Role, contentType[strings.LastIndex(contentType, ".")+1:])
	// Output: Role: user, Content type: exampleTextContent
}

// Types for ExampleValidator_Unmarshal_sliceOfInterfaces
type exampleZoo struct {
	Animals []exampleAnimal `json:"animals"`
}

func (z *exampleZoo) FieldAnimals() godantic.FieldOptions[[]exampleAnimal] {
	return godantic.Field(
		godantic.Required[[]exampleAnimal](),
		godantic.DiscriminatedUnion[[]exampleAnimal]("species", map[string]any{
			"cat": exampleCat{},
			"dog": exampleDog{},
		}),
	)
}

// ExampleValidator_Unmarshal_sliceOfInterfaces demonstrates unmarshaling
// a slice of interface types (discriminated unions) as a struct field.
func ExampleValidator_Unmarshal_sliceOfInterfaces() {
	v := godantic.NewValidator[exampleZoo]()

	zoo, _ := v.Unmarshal([]byte(`{
		"animals": [
			{"species": "cat", "name": "Whiskers"},
			{"species": "dog", "name": "Buddy", "breed": "Golden"}
		]
	}`))

	fmt.Printf("Count: %d, First sound: %s\n", len(zoo.Animals), zoo.Animals[0].Sound())
	// Output: Count: 2, First sound: meow
}
