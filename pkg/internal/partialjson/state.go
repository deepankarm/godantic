package partialjson

// ParseResult contains the repaired JSON and metadata about what's incomplete.
type ParseResult struct {
	// Repaired is valid JSON with incomplete parts closed/removed
	Repaired []byte

	// Incomplete tracks which JSON paths are truncated
	// e.g., ["tasks", "[1]", "title"] means tasks[1].title was cut off
	Incomplete [][]string

	// TruncatedAt indicates where the input was cut off
	// "string" | "array" | "object" | "key" | "value" | "complete"
	TruncatedAt string
}

// TruncationInfo describes how a field was truncated.
type TruncationInfo struct {
	Path        []string // JSON path, e.g., ["user", "name"]
	TruncatedAt string   // "string", "array", "object", "key", "value"
	RawValue    string   // The partial raw value (for debugging)
}
