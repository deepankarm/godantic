// Example: OpenAI Structured Outputs with Godantic
//
// Demonstrates extracting structured meeting data from unstructured text using
// OpenAI's structured outputs API with Godantic for schema generation and validation.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type ActionItem struct {
	Task     string `json:"task"`
	Assignee string `json:"assignee"`
	Deadline string `json:"deadline"`
}

func (a *ActionItem) FieldTask() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(3),
		godantic.Description[string]("Task description"),
	)
}

func (a *ActionItem) FieldAssignee() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(1),
		godantic.Description[string]("Person assigned to the task"),
	)
}

func (a *ActionItem) FieldDeadline() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(1),
		godantic.Description[string]("Task deadline"),
	)
}

type MeetingSummary struct {
	Title       string       `json:"title"`
	Date        string       `json:"date"`
	Attendees   []string     `json:"attendees"`
	Summary     string       `json:"summary"`
	ActionItems []ActionItem `json:"action_items"`
}

func (m *MeetingSummary) FieldTitle() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(3),
		godantic.Description[string]("Meeting title"),
	)
}

func (m *MeetingSummary) FieldDate() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Description[string]("Meeting date (YYYY-MM-DD)"),
	)
}

func (m *MeetingSummary) FieldAttendees() godantic.FieldOptions[[]string] {
	return godantic.Field(
		godantic.Required[[]string](),
		godantic.MinItems[string](1),
		godantic.Description[[]string]("List of attendees"),
	)
}

func (m *MeetingSummary) FieldSummary() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(10),
		godantic.Description[string]("Brief meeting summary"),
	)
}

func (m *MeetingSummary) FieldActionItems() godantic.FieldOptions[[]ActionItem] {
	return godantic.Field(
		godantic.Required[[]ActionItem](),
		godantic.MinItems[ActionItem](0),
		godantic.Description[[]ActionItem]("Action items from the meeting"),
	)
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Generate flattened schema for LLM API (no $ref at root)
	schemaGen := schema.NewGenerator[MeetingSummary]()
	flatSchema, err := schemaGen.GenerateFlattened()
	if err != nil {
		log.Fatalf("Failed to generate schema: %v", err)
	}

	meetingTranscript := `Team standup on November 19, 2025. Present: Alice, Bob, and Charlie.
Alice completed the authentication module and it's ready for review.
Bob is working on the API documentation and expects to finish by end of week.
Charlie reported a critical bug in the payment processing - needs immediate attention.
Action items:
1. Bob will fix the payment bug by tomorrow (Nov 20)
2. Alice to review Bob's API documentation by Friday (Nov 22)
3. Charlie to investigate database performance issues by end of week`

	client := openai.NewClient(option.WithAPIKey(apiKey))

	completion, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("Extract structured meeting information from the transcript."),
			openai.UserMessage(meetingTranscript),
		},
		Model: openai.ChatModelGPT4o2024_08_06,
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:        "meeting_summary",
					Description: openai.String("Structured meeting summary with action items"),
					Schema:      flatSchema,
					Strict:      openai.Bool(true),
				},
			},
		},
	})

	if err != nil {
		log.Fatalf("OpenAI API error: %v", err)
	}

	validator := godantic.NewValidator[MeetingSummary]()
	meeting, errs := validator.ValidateJSON([]byte(completion.Choices[0].Message.Content))
	if len(errs) > 0 {
		fmt.Println("Validation errors:")
		for _, e := range errs {
			fmt.Printf("  %v: %s\n", e.Loc, e.Message)
		}
		os.Exit(1)
	}

	// Display results
	fmt.Printf("Meeting: %s (%s)\n", meeting.Title, meeting.Date)
	fmt.Printf("Attendees: %s\n", strings.Join(meeting.Attendees, ", "))
	fmt.Printf("Summary: %s\n\n", meeting.Summary)
	fmt.Printf("Action Items:\n")
	for i, item := range meeting.ActionItems {
		fmt.Printf("  %d. %s (assigned: %s, deadline: %s)\n", i+1, item.Task, item.Assignee, item.Deadline)
	}

	/*
		Meeting: Team Standup (2025-11-19)
		Attendees: Alice, Bob, Charlie
		Summary: The team discussed the progress on current tasks. Alice has completed the authentication module, Bob is finalizing the API documentation, and Charlie identified a critical bug in payment processing that requires urgent resolution.

		Action Items:
		  1. Fix the payment bug. (assigned: Bob, deadline: 2025-11-20)
		  2. Review Bob's API documentation. (assigned: Alice, deadline: 2025-11-22)
		  3. Investigate database performance issues. (assigned: Charlie, deadline: 2025-11-22)
	*/
}
