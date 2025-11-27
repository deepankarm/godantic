package godantic

import "sync"

// StreamParser provides stateful parsing for streaming JSON chunks.
// Designed for LLM streaming APIs (Anthropic, OpenAI, etc.)
type StreamParser[T any] struct {
	validator *Validator[T]
	buffer    []byte
	mu        sync.Mutex
}

// NewStreamParser creates a parser for streaming JSON.
// For discriminated unions, use NewStreamParserWithValidator instead.
func NewStreamParser[T any]() *StreamParser[T] {
	return &StreamParser[T]{
		validator: NewValidator[T](),
		buffer:    make([]byte, 0, 1024),
	}
}

// NewStreamParserWithValidator creates a parser with a custom validator.
// Use this for discriminated unions or when you need custom validator options.
func NewStreamParserWithValidator[T any](validator *Validator[T]) *StreamParser[T] {
	return &StreamParser[T]{
		validator: validator,
		buffer:    make([]byte, 0, 1024),
	}
}

// Feed adds a new chunk of JSON data and returns the current state.
// The buffer accumulates chunks, so call Feed() for each delta.
//
// Example:
//
//	parser := godantic.NewStreamParser[ToolCall]()
//
//	// Feed chunks as they arrive
//	result, state, errs := parser.Feed([]byte(`{"type": "sear`))
//	result, state, errs = parser.Feed([]byte(`ch", "query": "go`))
//	result, state, errs = parser.Feed([]byte(`lang"}`))
//
//	if state.IsComplete {
//	    // Full JSON received
//	}
func (sp *StreamParser[T]) Feed(chunk []byte) (*T, *PartialState, ValidationErrors) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.buffer = append(sp.buffer, chunk...)
	return sp.validator.MarshalPartial(sp.buffer)
}

// Reset clears the buffer and starts fresh.
func (sp *StreamParser[T]) Reset() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.buffer = sp.buffer[:0]
}

// Buffer returns the current accumulated buffer.
func (sp *StreamParser[T]) Buffer() []byte {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.buffer
}
