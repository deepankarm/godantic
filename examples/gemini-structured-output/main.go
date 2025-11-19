// Example: Google Gemini Structured Outputs with Godantic
//
// Demonstrates extracting structured task data with enums, date-time fields,
// and union types (anyOf) using Gemini's JSON mode with Godantic.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
	"google.golang.org/genai"
)

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusBlocked    Status = "blocked"
)

func (s Status) FieldStatus() godantic.FieldOptions[Status] {
	return godantic.Field(
		godantic.Required[Status](),
		godantic.OneOf(StatusTodo, StatusInProgress, StatusDone, StatusBlocked),
	)
}

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

func (p Priority) FieldPriority() godantic.FieldOptions[Priority] {
	return godantic.Field(
		godantic.Required[Priority](),
		godantic.OneOf(PriorityLow, PriorityMedium, PriorityHigh),
	)
}

type DetailedEstimate struct {
	Hours   int    `json:"hours"`
	Minutes int    `json:"minutes"`
	Notes   string `json:"notes"`
}

func (d *DetailedEstimate) FieldHours() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Required[int](), godantic.Min(0))
}

func (d *DetailedEstimate) FieldMinutes() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Required[int](), godantic.Min(0), godantic.Max(59))
}

func (d *DetailedEstimate) FieldNotes() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

type Task struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      Status   `json:"status"`
	Priority    Priority `json:"priority"`
	Assignee    *string  `json:"assignee"` // Nullable string
	DueDate     *string  `json:"due_date"` // Nullable date-time
	Estimate    any      `json:"estimate"` // Union: string OR DetailedEstimate
	Tags        []string `json:"tags"`
}

func (t *Task) FieldTitle() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(3),
		godantic.Description[string]("Task title"),
	)
}

func (t *Task) FieldDescription() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(10),
		godantic.Description[string]("Detailed task description"),
	)
}

func (t *Task) FieldStatus() godantic.FieldOptions[Status] {
	return godantic.Field(
		godantic.Required[Status](),
		godantic.Description[Status]("Current task status"),
	)
}

func (t *Task) FieldPriority() godantic.FieldOptions[Priority] {
	return godantic.Field(
		godantic.Required[Priority](),
		godantic.Description[Priority]("Task priority level"),
	)
}

func (t *Task) FieldAssignee() godantic.FieldOptions[*string] {
	return godantic.Field(
		godantic.Description[*string]("Person assigned to the task (optional)"),
	)
}

func (t *Task) FieldDueDate() godantic.FieldOptions[*string] {
	return godantic.Field(
		godantic.Format[*string]("date-time"),
		godantic.Description[*string]("Task deadline in ISO 8601 format (optional)"),
	)
}

func (t *Task) FieldEstimate() godantic.FieldOptions[any] {
	return godantic.Field(
		godantic.Required[any](),
		godantic.Union[any]("string", DetailedEstimate{}),
		godantic.Description[any]("Time estimate: simple string or detailed breakdown"),
	)
}

func (t *Task) FieldTags() godantic.FieldOptions[[]string] {
	return godantic.Field(
		godantic.Required[[]string](),
		godantic.MinItems[string](0),
		godantic.Description[[]string]("Task tags for categorization"),
	)
}

type TaskList struct {
	Tasks []Task `json:"tasks"`
}

func (tl *TaskList) FieldTasks() godantic.FieldOptions[[]Task] {
	return godantic.Field(
		godantic.Required[[]Task](),
		godantic.MinItems[Task](1),
		godantic.Description[[]Task]("List of extracted tasks"),
	)
}

func main() {
	// Generate flattened schema for Gemini API
	schemaGen := schema.NewGenerator[TaskList]()
	jsonSchema, err := schemaGen.GenerateFlattened()
	if err != nil {
		log.Fatalf("Failed to generate schema: %v", err)
	}

	projectDescription := `Our team is working on a new mobile app launch. 
Sarah needs to finalize the UI designs by next Friday (2025-12-29T17:00:00Z) - this is high priority and should take about 8 hours.
The backend API integration is currently in progress and assigned to Mike - estimate is 3 hours of coding plus 30 minutes of testing.
We also need someone to write the user documentation, but that's lower priority and can wait - roughly 2 days of work.
There's a critical bug in the login flow that's blocking the QA team - needs immediate attention, probably 1-2 hours to fix.`

	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}

	result, err := client.Models.GenerateContent(
		ctx,
		"gemini-2.5-pro",
		genai.Text(projectDescription),
		&genai.GenerateContentConfig{
			ResponseMIMEType:   "application/json",
			ResponseJsonSchema: jsonSchema,
		},
	)
	if err != nil {
		log.Fatalf("Gemini API error: %v", err)
	}

	validator := godantic.NewValidator[TaskList]()
	taskList, errs := validator.ValidateJSON([]byte(result.Text()))
	if len(errs) > 0 {
		fmt.Println("Validation errors:")
		for _, e := range errs {
			fmt.Printf("  %v: %s\n", e.Loc, e.Message)
		}
		os.Exit(1)
	}

	// Display results
	fmt.Printf("Extracted %d tasks:\n\n", len(taskList.Tasks))
	for i, task := range taskList.Tasks {
		fmt.Printf("%d. %s\n", i+1, task.Title)
		fmt.Printf("   Status: %s | Priority: %s\n", task.Status, task.Priority)
		if task.Assignee != nil {
			fmt.Printf("   Assigned to: %s\n", *task.Assignee)
		}
		if task.DueDate != nil {
			fmt.Printf("   Due: %s\n", *task.DueDate)
		}

		// Handle union type: estimate can be string or DetailedEstimate
		switch est := task.Estimate.(type) {
		case string:
			fmt.Printf("   Estimate: %s\n", est)
		case map[string]any:
			// When unmarshaled from JSON, structured objects become maps
			fmt.Printf("   Estimate: %vh %vm (%v)\n", est["hours"], est["minutes"], est["notes"])
		}

		if len(task.Tags) > 0 {
			fmt.Printf("   Tags: %v\n", task.Tags)
		}
		fmt.Println()
	}
}
