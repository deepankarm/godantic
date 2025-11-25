package godantic_test

import (
	"encoding/json"
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

type FlexibleConfig struct {
	Value any
}

func (c *FlexibleConfig) FieldValue() godantic.FieldOptions[any] {
	return godantic.Field(
		godantic.Required[any](),
		godantic.Union[any]("string", "integer", "object"),
		godantic.Description[any]("Can be a string, number, or configuration object"),
	)
}

func TestUnion(t *testing.T) {
	validator := godantic.NewValidator[FlexibleConfig]()

	tests := []struct {
		name    string
		config  FlexibleConfig
		wantErr bool
	}{
		{
			name:    "string value should pass",
			config:  FlexibleConfig{Value: "text"},
			wantErr: false,
		},
		{
			name:    "integer value should pass",
			config:  FlexibleConfig{Value: 42},
			wantErr: false,
		},
		{
			name:    "object value should pass",
			config:  FlexibleConfig{Value: map[string]any{"key": "value"}},
			wantErr: false,
		},
		{
			name:    "boolean value should fail",
			config:  FlexibleConfig{Value: true},
			wantErr: true,
		},
		{
			name:    "array value should fail",
			config:  FlexibleConfig{Value: []string{"a", "b"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.Validate(&tt.config)
			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", errs, tt.wantErr)
			}
		})
	}
}

// Test discriminated unions with real-world API response example
type ResponseType interface {
	GetStatus() string
}

type SuccessResponse struct {
	Status string // "success"
	Data   map[string]string
}

func (s SuccessResponse) GetStatus() string { return s.Status }

type ErrorResponse struct {
	Status  string // "error"
	Message string
	Code    int
}

func (e ErrorResponse) GetStatus() string { return e.Status }

type PendingResponse struct {
	Status   string // "pending"
	Progress int
}

func (p PendingResponse) GetStatus() string { return p.Status }

type InvalidResponse struct {
	Status string // "invalid"
	Reason string
}

func (i InvalidResponse) GetStatus() string { return i.Status }

type APIResult struct {
	Response ResponseType
}

func (a *APIResult) FieldResponse() godantic.FieldOptions[ResponseType] {
	return godantic.Field(
		godantic.Required[ResponseType](),
		godantic.DiscriminatedUnion[ResponseType]("Status", map[string]any{
			"success": SuccessResponse{},
			"error":   ErrorResponse{},
			"pending": PendingResponse{},
			// invalid response is not in the discriminator mapping
		}),
		godantic.Description[ResponseType]("API response - structure depends on Status field"),
	)
}

func TestDiscriminatedUnionSemantics(t *testing.T) {
	validator := godantic.NewValidator[APIResult]()

	tests := []struct {
		name    string
		result  APIResult
		wantErr bool
	}{
		{
			name: "success response with status='success'",
			result: APIResult{
				Response: SuccessResponse{
					Status: "success",
					Data:   map[string]string{"user_id": "123"},
				},
			},
			wantErr: false,
		},
		{
			name: "error response with status='error'",
			result: APIResult{
				Response: ErrorResponse{
					Status:  "error",
					Message: "Not found",
					Code:    404,
				},
			},
			wantErr: false,
		},
		{
			name: "pending response with status='pending'",
			result: APIResult{
				Response: PendingResponse{
					Status:   "pending",
					Progress: 75,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid discriminator: empty string",
			result: APIResult{
				Response: ErrorResponse{
					Status:  "", // Empty
					Message: "Something failed",
					Code:    500,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid discriminator: typo in value",
			result: APIResult{
				Response: PendingResponse{
					Status:   "pendin", // Typo
					Progress: 50,
				},
			},
			wantErr: true,
		},
		{
			name: "type implements interface but not in discriminator mapping",
			result: APIResult{
				Response: InvalidResponse{
					Status: "invalid",
					Reason: "Something went wrong",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.Validate(&tt.result)
			hasErr := len(errs) > 0

			if hasErr != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", errs, tt.wantErr)
			}
		})
	}
}

// Test mixing union types with other constraints (OneOf + Union)
type APIResponse struct {
	Status  string
	Payload any
}

func (r *APIResponse) FieldStatus() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.OneOf("success", "error", "pending"),
	)
}

func (r *APIResponse) FieldPayload() godantic.FieldOptions[any] {
	return godantic.Field(
		godantic.Required[any](),
		godantic.Union[any]("string", "object", "array"),
		godantic.Description[any]("Response payload can be a string, object, or array"),
	)
}

func TestComplexUnion(t *testing.T) {
	validator := godantic.NewValidator[APIResponse]()

	tests := []struct {
		name     string
		response APIResponse
		wantErr  bool
	}{
		{
			name:     "success with string payload",
			response: APIResponse{Status: "success", Payload: "completed"},
			wantErr:  false,
		},
		{
			name:     "success with object payload",
			response: APIResponse{Status: "success", Payload: map[string]any{"id": 1, "name": "test"}},
			wantErr:  false,
		},
		{
			name:     "error with array payload",
			response: APIResponse{Status: "error", Payload: []string{"error1", "error2"}},
			wantErr:  false,
		},
		{
			name:     "invalid status should fail",
			response: APIResponse{Status: "invalid", Payload: "data"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.Validate(&tt.response)
			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", errs, tt.wantErr)
			}
		})
	}
}

// Test slices of discriminated unions
type ContentBlock interface {
	GetBlockType() string
}

type TextContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (t TextContentBlock) GetBlockType() string { return t.Type }

type ImageContentBlock struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func (i ImageContentBlock) GetBlockType() string { return i.Type }

type CodeContentBlock struct {
	Type     string `json:"type"`
	Language string `json:"language"`
	Code     string `json:"code"`
}

func (c CodeContentBlock) GetBlockType() string { return c.Type }

type RichDocument struct {
	Title  string
	Blocks []ContentBlock
}

func (d *RichDocument) FieldTitle() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string]())
}

func (d *RichDocument) FieldBlocks() godantic.FieldOptions[[]ContentBlock] {
	return godantic.Field(
		godantic.Required[[]ContentBlock](),
		godantic.DiscriminatedUnion[[]ContentBlock]("type", map[string]any{
			"text":  TextContentBlock{},
			"image": ImageContentBlock{},
			"code":  CodeContentBlock{},
		}),
	)
}

func TestSliceOfDiscriminatedUnions(t *testing.T) {
	validator := godantic.NewValidator[RichDocument]()

	t.Run("validate struct with slice of discriminated unions", func(t *testing.T) {
		doc := RichDocument{
			Title: "Test Doc",
			Blocks: []ContentBlock{
				TextContentBlock{Type: "text", Text: "Hello"},
				ImageContentBlock{Type: "image", URL: "http://example.com/img.png"},
				CodeContentBlock{Type: "code", Language: "go", Code: "fmt.Println()"},
			},
		}
		errs := validator.Validate(&doc)
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("marshal JSON → struct with slice of discriminated unions", func(t *testing.T) {
		jsonStr := `{
			"Title": "Test Doc",
			"Blocks": [
				{"type": "text", "text": "Hello"},
				{"type": "image", "url": "http://example.com/img.png"},
				{"type": "code", "language": "go", "code": "fmt.Println()"}
			]
		}`

		doc, errs := validator.Marshal([]byte(jsonStr))
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}

		if doc.Title != "Test Doc" {
			t.Errorf("expected title 'Test Doc', got %s", doc.Title)
		}

		if len(doc.Blocks) != 3 {
			t.Fatalf("expected 3 content blocks, got %d", len(doc.Blocks))
		}

		// Verify each block type
		if _, ok := doc.Blocks[0].(TextContentBlock); !ok {
			t.Errorf("expected first block to be TextContentBlock, got %T", doc.Blocks[0])
		}
		if _, ok := doc.Blocks[1].(ImageContentBlock); !ok {
			t.Errorf("expected second block to be ImageContentBlock, got %T", doc.Blocks[1])
		}
		if _, ok := doc.Blocks[2].(CodeContentBlock); !ok {
			t.Errorf("expected third block to be CodeContentBlock, got %T", doc.Blocks[2])
		}
	})

	t.Run("unmarshal struct → JSON with slice of discriminated unions", func(t *testing.T) {
		doc := RichDocument{
			Title: "Round Trip",
			Blocks: []ContentBlock{
				TextContentBlock{Type: "text", Text: "World"},
				CodeContentBlock{Type: "code", Language: "python", Code: "print('hi')"},
			},
		}

		jsonData, errs := validator.Unmarshal(&doc)
		if len(errs) != 0 {
			t.Fatalf("expected no errors, got %v", errs)
		}

		// Verify JSON contains correct structure
		var result map[string]any
		if err := json.Unmarshal(jsonData, &result); err != nil {
			t.Fatalf("failed to parse result JSON: %v", err)
		}

		if result["Title"] != "Round Trip" {
			t.Errorf("expected title 'Round Trip', got %v", result["Title"])
		}

		blocks, ok := result["Blocks"].([]any)
		if !ok {
			t.Fatalf("expected Blocks to be array, got %T", result["Blocks"])
		}

		if len(blocks) != 2 {
			t.Fatalf("expected 2 blocks, got %d", len(blocks))
		}

		// Verify first block is text
		block0 := blocks[0].(map[string]any)
		if block0["type"] != "text" || block0["text"] != "World" {
			t.Errorf("first block mismatch: %v", block0)
		}

		// Verify second block is code
		block1 := blocks[1].(map[string]any)
		if block1["type"] != "code" || block1["language"] != "python" {
			t.Errorf("second block mismatch: %v", block1)
		}
	})
}
