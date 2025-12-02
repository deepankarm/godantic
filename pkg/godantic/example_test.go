package godantic_test

import (
	"fmt"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Types for ExampleValidator_Unmarshal
type exampleTaskUnmarshal struct {
	Title    string `json:"title"`
	Priority string `json:"priority"`
	Score    int    `json:"score"`
}

func (t *exampleTaskUnmarshal) FieldTitle() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(1),
		godantic.Description[string]("Task title"),
	)
}

func (t *exampleTaskUnmarshal) FieldPriority() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.OneOf("low", "medium", "high"),
	)
}

func (t *exampleTaskUnmarshal) FieldScore() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Min(0),
		godantic.Max(100),
		godantic.Default(50),
	)
}

// ExampleValidator_Unmarshal demonstrates unmarshaling JSON into a struct
// with validation, constraints, and default values applied.
func ExampleValidator_Unmarshal() {
	v := godantic.NewValidator[exampleTaskUnmarshal]()
	task, errs := v.Unmarshal([]byte(`{"title": "Review PR", "priority": "high"}`))

	fmt.Printf("Title: %s, Priority: %s, Score: %d, Errors: %d\n",
		task.Title, task.Priority, task.Score, len(errs))
	// Output: Title: Review PR, Priority: high, Score: 50, Errors: 0
}

// Types for ExampleValidator_Unmarshal_validationErrors
type exampleTaskError struct {
	Priority string `json:"priority"`
}

func (t *exampleTaskError) FieldPriority() godantic.FieldOptions[string] {
	return godantic.Field(godantic.OneOf("low", "medium", "high"))
}

// ExampleValidator_Unmarshal_validationErrors demonstrates how validation
// errors are returned when input data fails validation constraints.
func ExampleValidator_Unmarshal_validationErrors() {
	v := godantic.NewValidator[exampleTaskError]()
	_, errs := v.Unmarshal([]byte(`{"priority": "urgent"}`))

	fmt.Printf("Error: %s at %v\n", errs[0].Message, errs[0].Loc)
	// Output: Error: value must be one of [low medium high] at [Priority]
}

// Types for ExampleValidator_Marshal
type exampleTaskMarshal struct {
	Title    string `json:"title"`
	Priority string `json:"priority"`
	Score    int    `json:"score"`
}

func (t *exampleTaskMarshal) FieldTitle() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (t *exampleTaskMarshal) FieldPriority() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.OneOf("low", "medium", "high"),
	)
}

func (t *exampleTaskMarshal) FieldScore() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Default(50),
		godantic.Min(0),
		godantic.Max(100),
	)
}

// ExampleValidator_Marshal demonstrates marshaling a struct to JSON
// with validation and default values applied.
func ExampleValidator_Marshal() {
	v := godantic.NewValidator[exampleTaskMarshal]()
	task := &exampleTaskMarshal{
		Title:    "Complete docs",
		Priority: "high",
		// Score is zero value, will get default
	}

	jsonBytes, errs := v.Marshal(task)
	if len(errs) > 0 {
		fmt.Printf("Validation errors: %d\n", len(errs))
		return
	}

	fmt.Println(string(jsonBytes))
	// Output: {"title":"Complete docs","priority":"high","score":50}
}
