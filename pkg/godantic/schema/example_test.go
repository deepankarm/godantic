package schema_test

import (
	"fmt"
	"strings"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
)

// Types for ExampleGenerator_GenerateFlattened
type exampleSentimentResponse struct {
	Sentiment  string  `json:"sentiment"`
	Confidence float64 `json:"confidence"`
}

func (s *exampleSentimentResponse) FieldSentiment() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.OneOf("positive", "negative", "neutral"),
		godantic.Description[string]("The detected sentiment"),
	)
}

func (s *exampleSentimentResponse) FieldConfidence() godantic.FieldOptions[float64] {
	return godantic.Field(
		godantic.Required[float64](),
		godantic.Min(0.0),
		godantic.Max(1.0),
	)
}

// ExampleGenerator_GenerateFlattened demonstrates generating a flattened
// JSON Schema suitable for LLM APIs (OpenAI, Gemini, Claude) that require
// the root object definition at the top level instead of a $ref.
func ExampleGenerator_GenerateFlattened() {
	g := schema.NewGenerator[exampleSentimentResponse]()
	flatSchema, _ := g.GenerateFlattened()

	// Ready to send to OpenAI/Gemini/Claude structured output API
	schemaType := flatSchema["type"]
	fmt.Println(schemaType)
	// Output: object
}

// Types for ExampleGenerator_GenerateJSON
type exampleUserSchema struct {
	Name string `json:"name"`
}

func (u *exampleUserSchema) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// ExampleGenerator_GenerateJSON demonstrates generating a standard JSON Schema
// as a formatted JSON string, useful for documentation or API specs.
func ExampleGenerator_GenerateJSON() {
	g := schema.NewGenerator[exampleUserSchema]()
	jsonStr, _ := g.GenerateJSON()

	fmt.Println(strings.Contains(jsonStr, `"type"`))
	// Output: true
}
