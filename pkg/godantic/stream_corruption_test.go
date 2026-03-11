package godantic_test

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// Types simulating a structured LLM response with nested fields.
type ReviewResponse struct {
	Status string       `json:"status"`
	Review ReviewDetail `json:"review"`
}

type ReviewDetail struct {
	Items   []ReviewItem `json:"items"`
	Summary string       `json:"summary"`
}

type ReviewItem struct {
	Title   string `json:"title"`
	Comment string `json:"comment"`
}

// TestStreamParser_BufferCorruption_CharByChar feeds JSON character by
// character. Before the fix, partial JSON repair via append() on sub-slices
// overwrites bytes in the shared buffer, corrupting fields that appear
// later in the JSON (e.g., "summary" after "items").
func TestStreamParser_BufferCorruption_CharByChar(t *testing.T) {
	fullJSON := `{"status":"approved","review":{"items":[{"title":"Fix typo in README","comment":"Looks good, minor formatting improvement"}],"summary":"All changes are approved. The typo fix improves documentation clarity."}}`

	parser := godantic.NewStreamParser[ReviewResponse]()
	var lastResult *ReviewResponse
	for i := 0; i < len(fullJSON); i++ {
		result, _, _ := parser.Feed([]byte{fullJSON[i]})
		if result != nil {
			lastResult = result
		}
	}

	if lastResult == nil {
		t.Fatal("parser returned nil after feeding complete JSON")
	}

	if lastResult.Status != "approved" {
		t.Errorf("status = %q, want approved", lastResult.Status)
	}

	if lastResult.Review.Summary == "" {
		t.Error("summary is empty — buffer corruption lost the field")
	}

	if lastResult.Review.Summary != "All changes are approved. The typo fix improves documentation clarity." {
		t.Errorf("summary = %q, want full text", lastResult.Review.Summary)
	}

	if len(lastResult.Review.Items) != 1 {
		t.Errorf("items count = %d, want 1", len(lastResult.Review.Items))
	}
}

// TestStreamParser_BufferCorruption_SmallChunks tests with small chunks
// (3-7 bytes) simulating streaming tokens.
func TestStreamParser_BufferCorruption_SmallChunks(t *testing.T) {
	fullJSON := `{"status":"completed","review":{"items":[{"title":"Add unit tests","comment":"Good coverage of edge cases"}],"summary":"The PR adds comprehensive test coverage for the new feature."}}`

	chunkSizes := []int{3, 5, 7, 4, 6}
	parser := godantic.NewStreamParser[ReviewResponse]()
	var lastResult *ReviewResponse
	pos := 0
	chunkIdx := 0
	for pos < len(fullJSON) {
		size := chunkSizes[chunkIdx%len(chunkSizes)]
		end := pos + size
		if end > len(fullJSON) {
			end = len(fullJSON)
		}
		result, _, _ := parser.Feed([]byte(fullJSON[pos:end]))
		if result != nil {
			lastResult = result
		}
		pos = end
		chunkIdx++
	}

	if lastResult == nil {
		t.Fatal("parser returned nil")
	}

	if lastResult.Review.Summary == "" {
		t.Error("summary is empty — buffer corruption lost the field")
	}

	expected := "The PR adds comprehensive test coverage for the new feature."
	if lastResult.Review.Summary != expected {
		t.Errorf("summary = %q, want %q", lastResult.Review.Summary, expected)
	}
}
