// Example: Streaming Partial JSON with godantic
//
// Demonstrates:
// 1. Automatic JSON Schema generation for LLM structured output
// 2. StreamParser.Feed() for real-time partial JSON parsing
// 3. Defaults applied automatically as data streams in
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
	"google.golang.org/genai"
)

type ProjectPriority string

const (
	ProjectPriorityLow      ProjectPriority = "low"
	ProjectPriorityMedium   ProjectPriority = "medium"
	ProjectPriorityHigh     ProjectPriority = "high"
	ProjectPriorityCritical ProjectPriority = "critical"
)

func (p ProjectPriority) FieldPriority() godantic.FieldOptions[ProjectPriority] {
	return godantic.Field(
		godantic.Required[ProjectPriority](),
		godantic.Default(ProjectPriorityMedium),
		godantic.OneOf(ProjectPriorityLow, ProjectPriorityMedium, ProjectPriorityHigh, ProjectPriorityCritical),
	)
}

// ProjectAnalysis - analysis result from LLM
type ProjectAnalysis struct {
	ProjectName string          `json:"project_name"`
	Summary     string          `json:"summary"` // Long field - perfect for streaming
	KeyInsights []string        `json:"key_insights"`
	Confidence  float64         `json:"confidence"` // Default: 0.85
	Priority    ProjectPriority `json:"priority"`   // Default: "medium"
}

func (p *ProjectAnalysis) FieldProjectName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (p *ProjectAnalysis) FieldSummary() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(50),
		godantic.Description[string]("Comprehensive project summary (300+ words)"),
	)
}

func (p *ProjectAnalysis) FieldKeyInsights() godantic.FieldOptions[[]string] {
	return godantic.Field(
		godantic.Description[[]string]("Key takeaways from the analysis. Should be 2-3 key takeaways."),
	)
}

func (p *ProjectAnalysis) FieldConfidence() godantic.FieldOptions[float64] {
	return godantic.Field(
		godantic.Default(0.85),
		godantic.Min(0.0),
		godantic.Max(1.0),
		godantic.Description[float64]("Confidence score for this analysis"),
	)
}

// StreamStats holds metrics about the streaming session
type StreamStats struct {
	JSONCount       int
	TimeToFirstJSON time.Duration
}

// Stream processor using godantic.StreamParser
func processStream(chunks <-chan string, startTime time.Time, stats *StreamStats, wg *sync.WaitGroup) {
	defer wg.Done()

	parser := godantic.NewStreamParser[ProjectAnalysis]()
	var firstJSONTime time.Time

	for chunk := range chunks {
		result, state, _ := parser.Feed([]byte(chunk))

		if result == nil {
			continue
		}

		stats.JSONCount++
		if firstJSONTime.IsZero() {
			firstJSONTime = time.Now()
			stats.TimeToFirstJSON = firstJSONTime.Sub(startTime)
		}

		// Clear screen and show current state
		fmt.Print("\033[2J\033[H")
		fmt.Println("╔════════════════════════════════════════════════════════════════════╗")
		fmt.Println("║  godantic.StreamParser - Watch JSON build in real-time!           ║")
		fmt.Println("╚════════════════════════════════════════════════════════════════════╝")
		fmt.Println()

		if state.IsComplete {
			fmt.Println("✅ COMPLETE - All fields received and validated")
		} else {
			waiting := state.WaitingFor()
			if len(waiting) > 0 {
				fmt.Printf("⏳ STREAMING - Waiting for: %s\n", waiting[0])
			}
		}
		fmt.Println()

		// Pretty print the current result
		prettyJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(prettyJSON))

		if state.IsComplete {
			fmt.Println()
			fmt.Println("────────────────────────────────────────────────────────────────────")
			fmt.Println("✨ Defaults applied:")
			if result.Confidence == 0.85 {
				fmt.Println("  • confidence = 0.85 (default)")
			}
			if result.Priority == ProjectPriorityMedium {
				fmt.Println("  • priority = \"medium\" (default)")
			}
		}
	}
}

func main() {
	// Generate JSON Schema
	schemaGen := schema.NewGenerator[ProjectAnalysis]()
	jsonSchema, err := schemaGen.GenerateFlattened()
	if err != nil {
		log.Fatalf("Failed to generate schema: %v", err)
	}

	fmt.Println("╔════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║  godantic: Automatic JSON Schema Generation                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	projectUpdate := `
Project: Mobile App Redesign Q1 2025

Timeline and Status:
- Started on Jan 15, 2025 by Sarah Chen (Product Manager)
- Initial research and requirements gathering completed in 2 weeks
- Design phase progressing well - wireframes done, high-fidelity mockups 70% complete
- Currently on milestone 2 of 5 total milestones
- Development team blocked waiting for final API specifications
- Target completion: March 30, 2025 (estimated 45 days total duration)

Key Metrics:
- User research sessions: 12 completed
- Design iterations: 3
- Stakeholder approvals: 4/5 received

Risk: API spec delay could impact timeline by 1-2 weeks if not resolved soon.
`

	ctx := context.Background()
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}

	prompt := fmt.Sprintf(`Analyze this project update. Provide:
- project_name: Short project name
- summary: Comprehensive summary (100+ words covering all aspects)
- key_insights: Array of 4-5 key takeaways
- confidence: Your confidence in this analysis (0.0-1.0)
- priority: Project priority (low/medium/high/critical)

Project Update:
%s`, projectUpdate)

	chunks := make(chan string, 100)
	var wg sync.WaitGroup
	stats := &StreamStats{}

	fmt.Println("🚀 Streaming from Gemini with godantic-generated schema...")
	fmt.Println()
	startTime := time.Now()

	wg.Add(1)
	go processStream(chunks, startTime, stats, &wg)

	iter := client.Models.GenerateContentStream(
		ctx,
		"gemini-2.5-flash",
		genai.Text(prompt),
		&genai.GenerateContentConfig{
			ResponseMIMEType:   "application/json",
			ResponseJsonSchema: jsonSchema,
		},
	)

	var ttft time.Duration
	for resp, err := range iter {
		if err != nil {
			log.Printf("Stream error: %v", err)
			break
		}
		for _, cand := range resp.Candidates {
			for _, part := range cand.Content.Parts {
				if part.Text != "" {
					if ttft == 0 {
						ttft = time.Since(startTime)
					}
					chunks <- part.Text
				}
			}
		}
	}
	close(chunks)
	wg.Wait()

	totalTime := time.Since(startTime)
	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════════════")
	fmt.Println("📈 Performance Metrics:")
	fmt.Printf("  • TTFT (Time To First Token):  %v\n", ttft.Round(time.Millisecond))
	fmt.Printf("  • Time to first JSON parse:    %v\n", stats.TimeToFirstJSON.Round(time.Millisecond))
	fmt.Printf("  • Total JSON updates shown:    %d\n", stats.JSONCount)
	fmt.Printf("  • Total time:                  %v\n", totalTime.Round(time.Millisecond))
}
