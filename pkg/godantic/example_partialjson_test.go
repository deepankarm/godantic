package godantic_test

import (
	"fmt"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Types for ExampleStreamParser_Feed
type exampleStreamTask struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func (t *exampleStreamTask) FieldTitle() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

// ExampleStreamParser_Feed demonstrates parsing incomplete JSON as it
// streams from LLM APIs, showing how to track completion state and
// which fields are still incomplete.
func ExampleStreamParser_Feed() {
	parser := godantic.NewStreamParser[exampleStreamTask]()

	// Simulate LLM streaming chunks
	chunks := []string{
		`{"title": "Lear`,
		`n Go", "do`,
		`ne": tr`,
		`ue}`,
	}

	var result *exampleStreamTask
	var state *godantic.PartialState

	for _, chunk := range chunks {
		result, state, _ = parser.Feed([]byte(chunk))
		if !state.IsComplete {
			waiting := state.WaitingFor()
			if len(waiting) > 0 {
				fmt.Printf("Waiting for: %v\n", waiting)
			}
		}
	}

	fmt.Printf("Complete: %v, Title: %s\n", state.IsComplete, result.Title)
	// Output:
	// Waiting for: [title]
	// Waiting for: [done]
	// Complete: true, Title: Learn Go
}
