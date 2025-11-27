package godantic

import (
	"reflect"
	"strings"

	"github.com/deepankarm/godantic/pkg/internal/partialjson"
)

// PartialState tracks the completeness of a parsed struct.
type PartialState struct {
	// IsComplete is true if the entire struct was fully parsed
	IsComplete bool

	// IncompleteFields lists fields that were truncated
	IncompleteFields []IncompleteField
}

// IncompleteField describes a single incomplete field.
type IncompleteField struct {
	// Path to the field as JSON path, e.g., ["user", "name"]
	// These are JSON field names (lowercase, snake_case)
	Path []string

	// JSONPath as string, e.g., "user.name"
	JSONPath string

	// Reason describes why it's incomplete
	// "string" | "array" | "object" | "key" | "value" | "complete"
	Reason string
}

// IsFieldComplete checks if a specific field path is complete.
// Path should be JSON field names, e.g., ["user", "name"]
func (ps *PartialState) IsFieldComplete(path ...string) bool {
	if ps.IsComplete {
		return true
	}

	pathStr := strings.Join(path, ".")
	for _, incomplete := range ps.IncompleteFields {
		if incomplete.JSONPath == pathStr {
			return false
		}
	}
	return true
}

// WaitingFor returns JSON paths of fields that are incomplete.
// Useful for UI: "Waiting for: user.name, user.email..."
func (ps *PartialState) WaitingFor() []string {
	if ps.IsComplete {
		return nil
	}

	result := make([]string, 0, len(ps.IncompleteFields))
	for _, field := range ps.IncompleteFields {
		result = append(result, field.JSONPath)
	}
	return result
}

// MergeIncompleteFields adds additional incomplete fields to the state.
func (ps *PartialState) MergeIncompleteFields(paths [][]string, reason string) {
	for _, path := range paths {
		ps.IncompleteFields = append(ps.IncompleteFields, IncompleteField{
			Path:     path,
			JSONPath: partialjson.JoinPath(path),
			Reason:   reason,
		})
	}
	if len(ps.IncompleteFields) > 0 {
		ps.IsComplete = false
	}
}

// buildPartialStateFromPaths converts parser incomplete paths to PartialState.
func buildPartialStateFromPaths(incompletePaths [][]string, truncatedAt string) *PartialState {
	partialState := &PartialState{
		IsComplete:       len(incompletePaths) == 0,
		IncompleteFields: make([]IncompleteField, 0, len(incompletePaths)),
	}

	// Use TruncatedAt from parser for root-level truncation
	reason := truncatedAt
	if reason == "" || reason == "complete" {
		reason = "incomplete"
	}

	for _, path := range incompletePaths {
		partialState.IncompleteFields = append(partialState.IncompleteFields, IncompleteField{
			Path:     path,
			JSONPath: partialjson.JoinPath(path),
			Reason:   reason,
		})
	}

	return partialState
}

// PartialUnmarshalResult contains the unmarshaled struct and incomplete field information.
type PartialUnmarshalResult struct {
	// Value is the unmarshaled struct value
	Value reflect.Value

	// IncompletePaths tracks which JSON paths were incomplete
	// These are JSON paths (lowercase, snake_case), e.g., ["user", "name"]
	IncompletePaths [][]string

	// TruncatedAt indicates where the root was truncated
	// "string" | "array" | "object" | "key" | "value" | "complete"
	TruncatedAt string

	// Errors from unmarshaling (not validation errors)
	Errors []ValidationError
}
