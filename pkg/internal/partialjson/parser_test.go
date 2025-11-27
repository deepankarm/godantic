package partialjson_test

import (
	"encoding/json"
	"testing"

	"github.com/deepankarm/godantic/pkg/internal/partialjson"
)

func TestTruncatedString(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"name": "Jo`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should repair to valid JSON
	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	// Should have incomplete field
	if len(result.Incomplete) == 0 {
		t.Error("expected incomplete field")
	}

	// Should track "name" as incomplete
	found := false
	for _, path := range result.Incomplete {
		if len(path) == 1 && path[0] == "name" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['name'], got: %v", result.Incomplete)
	}
}

func TestTruncatedAfterColon(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"name": "John", "age":`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should remove incomplete "age" pair
	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	// Should only have "name"
	if _, ok := m["name"]; !ok {
		t.Error("expected 'name' field")
	}
	if _, ok := m["age"]; ok {
		t.Error("should not have incomplete 'age' field")
	}

	// Should track "age" as incomplete
	found := false
	for _, path := range result.Incomplete {
		if len(path) == 1 && path[0] == "age" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['age'], got: %v", result.Incomplete)
	}
}

func TestTruncatedArray(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"ids": [1, 2,`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should close array
	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	ids, ok := m["ids"].([]any)
	if !ok {
		t.Fatalf("expected array, got: %T", m["ids"])
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 elements, got %d", len(ids))
	}
}

func TestTruncatedNestedObject(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"user": {"name": "John", "address": {"city": "NY`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should close nested objects
	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	// Should track incomplete path
	found := false
	for _, path := range result.Incomplete {
		if len(path) == 3 && path[0] == "user" && path[1] == "address" && path[2] == "city" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['user', 'address', 'city'], got: %v", result.Incomplete)
	}
}

func TestTruncatedDiscriminator(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"type": "do`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should keep partial value
	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["type"] != "do" {
		t.Errorf("expected 'do', got: %v", m["type"])
	}

	// Should track as incomplete
	found := false
	for _, path := range result.Incomplete {
		if len(path) == 1 && path[0] == "type" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['type'], got: %v", result.Incomplete)
	}
}

func TestTruncatedArrayOfObjects(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"tasks": [{"id": 1}, {"id": 2, "title": "Te`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should close structures properly
	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	tasks, ok := m["tasks"].([]any)
	if !ok {
		t.Fatalf("expected array, got: %T", m["tasks"])
	}
	if len(tasks) < 1 {
		t.Error("expected at least one task")
	}

	// Should track incomplete path
	found := false
	for _, path := range result.Incomplete {
		if len(path) >= 2 && path[0] == "tasks" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path starting with ['tasks'], got: %v", result.Incomplete)
	}
}

func TestCompleteJSON(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"name": "John", "age": 30}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Incomplete) != 0 {
		t.Errorf("expected no incomplete fields, got: %v", result.Incomplete)
	}

	if result.TruncatedAt != "complete" {
		t.Errorf("expected 'complete', got: %s", result.TruncatedAt)
	}
}

func TestEmptyInput(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(``))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty object
	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v", err)
	}
}

func TestEscapeSequences(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"text": "Hello\nWor`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should handle escape sequences
	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	// Should track as incomplete
	found := false
	for _, path := range result.Incomplete {
		if len(path) == 1 && path[0] == "text" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['text'], got: %v", result.Incomplete)
	}
}

// Boolean tests
func TestBooleanTrue(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"flag": true}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["flag"] != true {
		t.Errorf("expected true, got: %v", m["flag"])
	}

	if len(result.Incomplete) != 0 {
		t.Errorf("expected no incomplete fields, got: %v", result.Incomplete)
	}
}

func TestBooleanFalse(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"flag": false}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["flag"] != false {
		t.Errorf("expected false, got: %v", m["flag"])
	}
}

func TestBooleanIncompleteTrue(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"flag": tr`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	// Incomplete boolean is repaired to "true" which unmarshals as boolean true
	if m["flag"] != true {
		t.Errorf("expected true, got: %v", m["flag"])
	}

	found := false
	for _, path := range result.Incomplete {
		if len(path) == 1 && path[0] == "flag" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['flag'], got: %v", result.Incomplete)
	}
}

func TestBooleanIncompleteFalse(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"flag": fal`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	// Incomplete boolean is repaired to "false" which unmarshals as boolean false
	if m["flag"] != false {
		t.Errorf("expected false, got: %v", m["flag"])
	}

	found := false
	for _, path := range result.Incomplete {
		if len(path) == 1 && path[0] == "flag" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['flag'], got: %v", result.Incomplete)
	}
}

// Null tests
func TestNullComplete(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"value": null}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["value"] != nil {
		t.Errorf("expected nil, got: %v", m["value"])
	}

	if len(result.Incomplete) != 0 {
		t.Errorf("expected no incomplete fields, got: %v", result.Incomplete)
	}
}

func TestNullIncomplete(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"value": nu`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	// Incomplete null is repaired to "null" which unmarshals as nil
	if m["value"] != nil {
		t.Errorf("expected nil, got: %v", m["value"])
	}

	found := false
	for _, path := range result.Incomplete {
		if len(path) == 1 && path[0] == "value" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['value'], got: %v", result.Incomplete)
	}
}

// Number tests
func TestNumberInteger(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"age": 42}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["age"] != float64(42) {
		t.Errorf("expected 42, got: %v", m["age"])
	}
}

func TestNumberNegative(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"temp": -10}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["temp"] != float64(-10) {
		t.Errorf("expected -10, got: %v", m["temp"])
	}
}

func TestNumberDecimal(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"pi": 3.14159}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["pi"] != 3.14159 {
		t.Errorf("expected 3.14159, got: %v", m["pi"])
	}
}

func TestNumberExponent(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"big": 1e10}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["big"] != 1e10 {
		t.Errorf("expected 1e10, got: %v", m["big"])
	}
}

func TestNumberExponentNegative(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"small": 1e-5}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["small"] != 1e-5 {
		t.Errorf("expected 1e-5, got: %v", m["small"])
	}
}

func TestNumberIncompleteAfterMinus(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"temp": -`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["temp"] != float64(0) {
		t.Errorf("expected 0, got: %v", m["temp"])
	}

	found := false
	for _, path := range result.Incomplete {
		if len(path) == 1 && path[0] == "temp" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['temp'], got: %v", result.Incomplete)
	}
}

func TestNumberIncompleteAfterDecimal(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"pi": 3.`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["pi"] != float64(3) {
		t.Errorf("expected 3, got: %v", m["pi"])
	}

	found := false
	for _, path := range result.Incomplete {
		if len(path) == 1 && path[0] == "pi" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['pi'], got: %v", result.Incomplete)
	}
}

func TestNumberIncompleteAfterExponent(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"big": 1e`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["big"] != float64(1) {
		t.Errorf("expected 1, got: %v", m["big"])
	}

	found := false
	for _, path := range result.Incomplete {
		if len(path) == 1 && path[0] == "big" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['big'], got: %v", result.Incomplete)
	}
}

func TestNumberIncompleteAfterExponentSign(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"big": 1e-`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["big"] != float64(1) {
		t.Errorf("expected 1, got: %v", m["big"])
	}
}

// String escape sequence tests
func TestStringUnicodeEscape(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"text": "\u0041"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["text"] != "A" {
		t.Errorf("expected 'A', got: %v", m["text"])
	}
}

func TestStringUnicodeEscapeIncomplete(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"text": "\u004`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	// Should have the text with escaped backslash (incomplete unicode escape is removed)
	if m["text"] != "\\" {
		t.Errorf("expected '\\\\', got: %q", m["text"])
	}

	found := false
	for _, path := range result.Incomplete {
		if len(path) == 1 && path[0] == "text" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['text'], got: %v", result.Incomplete)
	}
}

func TestStringEscapeSequences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"newline", `{"text": "Hello\nWorld"}`, "Hello\nWorld"},
		{"tab", `{"text": "Hello\tWorld"}`, "Hello\tWorld"},
		{"quote", `{"text": "Say \"hello\""}`, "Say \"hello\""},
		{"backslash", `{"text": "C:\\path"}`, "C:\\path"},
		{"carriage return", `{"text": "Hello\rWorld"}`, "Hello\rWorld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := partialjson.NewParser(false)
			result, err := parser.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var m map[string]any
			if err := json.Unmarshal(result.Repaired, &m); err != nil {
				t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
			}

			if m["text"] != tt.expected {
				t.Errorf("expected %q, got: %q", tt.expected, m["text"])
			}
		})
	}
}

func TestStringStrictMode(t *testing.T) {
	parser := partialjson.NewParser(true) // strict mode
	// Use literal newline character (not escape sequence)
	result, err := parser.Parse([]byte("{\"text\": \"Hello\nWorld\"}"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// In strict mode, literal newline should mark string as incomplete
	found := false
	for _, path := range result.Incomplete {
		if len(path) == 1 && path[0] == "text" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['text'] in strict mode, got: %v", result.Incomplete)
	}
}

func TestStringIncompleteEscape(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"text": "Hello\`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	// Should have the text with escaped backslash
	if m["text"] != "Hello\\" {
		t.Errorf("expected 'Hello\\\\', got: %q", m["text"])
	}

	found := false
	for _, path := range result.Incomplete {
		if len(path) == 1 && path[0] == "text" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['text'], got: %v", result.Incomplete)
	}
}

// Array edge cases
func TestEmptyArray(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"items": []}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	items, ok := m["items"].([]any)
	if !ok {
		t.Fatalf("expected array, got: %T", m["items"])
	}
	if len(items) != 0 {
		t.Errorf("expected empty array, got: %v", items)
	}
}

func TestArrayWithMixedTypes(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"data": [1, "two", true, null, {"nested": "value"}]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	data, ok := m["data"].([]any)
	if !ok {
		t.Fatalf("expected array, got: %T", m["data"])
	}
	if len(data) != 5 {
		t.Errorf("expected 5 elements, got: %d", len(data))
	}
}

func TestArrayIncompleteAfterComma(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"items": [1, 2,`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	items, ok := m["items"].([]any)
	if !ok {
		t.Fatalf("expected array, got: %T", m["items"])
	}
	if len(items) != 2 {
		t.Errorf("expected 2 elements, got: %d", len(items))
	}
}

func TestArrayNested(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"matrix": [[1, 2], [3, 4]]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	matrix, ok := m["matrix"].([]any)
	if !ok {
		t.Fatalf("expected array, got: %T", m["matrix"])
	}
	if len(matrix) != 2 {
		t.Errorf("expected 2 rows, got: %d", len(matrix))
	}
}

func TestArrayIncompleteNested(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"matrix": [[1, 2], [3,`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	matrix, ok := m["matrix"].([]any)
	if !ok {
		t.Fatalf("expected array, got: %T", m["matrix"])
	}
	if len(matrix) < 1 {
		t.Error("expected at least one row")
	}
}

// Object edge cases
func TestEmptyObject(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if len(m) != 0 {
		t.Errorf("expected empty object, got: %v", m)
	}
}

func TestObjectIncompleteKey(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"name": "John", "ag`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if _, ok := m["age"]; ok {
		t.Error("should not have incomplete 'age' field")
	}
	if m["name"] != "John" {
		t.Errorf("expected 'John', got: %v", m["name"])
	}
}

func TestObjectIncompleteAfterComma(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"name": "John",`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["name"] != "John" {
		t.Errorf("expected 'John', got: %v", m["name"])
	}
}

func TestObjectIncompleteAfterColon(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"name":`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if _, ok := m["name"]; ok {
		t.Error("should not have incomplete 'name' field")
	}
}

// Root-level value tests
func TestRootArray(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`[1, 2, 3]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var arr []any
	if err := json.Unmarshal(result.Repaired, &arr); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if len(arr) != 3 {
		t.Errorf("expected 3 elements, got: %d", len(arr))
	}
}

func TestRootString(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`"hello"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var str string
	if err := json.Unmarshal(result.Repaired, &str); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if str != "hello" {
		t.Errorf("expected 'hello', got: %q", str)
	}
}

func TestRootNumber(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`42`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var num float64
	if err := json.Unmarshal(result.Repaired, &num); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if num != 42 {
		t.Errorf("expected 42, got: %v", num)
	}
}

func TestRootBoolean(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`true`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var b bool
	if err := json.Unmarshal(result.Repaired, &b); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if b != true {
		t.Errorf("expected true, got: %v", b)
	}
}

func TestRootNull(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`null`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var v any
	if err := json.Unmarshal(result.Repaired, &v); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if v != nil {
		t.Errorf("expected nil, got: %v", v)
	}
}

// Whitespace tests
func TestWhitespaceOnly(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`   `))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}
}

func TestWhitespaceAroundValues(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`   {"name": "John"}   `))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(result.Repaired, &m); err != nil {
		t.Fatalf("repaired JSON is invalid: %v, got: %s", err, string(result.Repaired))
	}

	if m["name"] != "John" {
		t.Errorf("expected 'John', got: %v", m["name"])
	}
}

// Path tracking tests
func TestPathTrackingNested(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"a": {"b": {"c": "incomplete`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, path := range result.Incomplete {
		if len(path) == 3 && path[0] == "a" && path[1] == "b" && path[2] == "c" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['a', 'b', 'c'], got: %v", result.Incomplete)
	}
}

func TestPathTrackingArrayIndex(t *testing.T) {
	parser := partialjson.NewParser(false)
	result, err := parser.Parse([]byte(`{"items": [{"id": 1}, {"id": 2, "name": "incomplete`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, path := range result.Incomplete {
		if len(path) == 3 && path[0] == "items" && path[1] == "[1]" && path[2] == "name" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected incomplete path ['items', '[1]', 'name'], got: %v", result.Incomplete)
	}
}
