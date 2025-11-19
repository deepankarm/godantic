package godantic_test

import (
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
