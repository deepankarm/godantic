package godantic_test

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// ═══════════════════════════════════════════════════════════════════════════
// Union[any]() Constraint Tests
// Tests for the Union() constraint on `any` typed fields
// ═══════════════════════════════════════════════════════════════════════════

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

func TestUnionConstraint(t *testing.T) {
	validator := godantic.NewValidator[FlexibleConfig]()

	tests := []struct {
		name    string
		config  FlexibleConfig
		wantErr bool
	}{
		{
			name:    "string_value",
			config:  FlexibleConfig{Value: "text"},
			wantErr: false,
		},
		{
			name:    "integer_value",
			config:  FlexibleConfig{Value: 42},
			wantErr: false,
		},
		{
			name:    "object_value",
			config:  FlexibleConfig{Value: map[string]any{"key": "value"}},
			wantErr: false,
		},
		{
			name:    "boolean_invalid",
			config:  FlexibleConfig{Value: true},
			wantErr: true,
		},
		{
			name:    "array_invalid",
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

// ═══════════════════════════════════════════════════════════════════════════
// DiscriminatedUnion() Field Constraint Tests
// Tests for discriminated union at field level (not validator level)
// ═══════════════════════════════════════════════════════════════════════════

type ResponseType interface {
	GetStatus() string
}

type SuccessResponse struct {
	Status string
	Data   map[string]string
}

func (s SuccessResponse) GetStatus() string { return s.Status }

type ErrorResponse struct {
	Status  string
	Message string
	Code    int
}

func (e ErrorResponse) GetStatus() string { return e.Status }

type PendingResponse struct {
	Status   string
	Progress int
}

func (p PendingResponse) GetStatus() string { return p.Status }

type InvalidResponse struct {
	Status string
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
		}),
	)
}

func TestFieldDiscriminatedUnion(t *testing.T) {
	validator := godantic.NewValidator[APIResult]()

	tests := []struct {
		name    string
		result  APIResult
		wantErr bool
	}{
		{
			name: "success_response",
			result: APIResult{
				Response: SuccessResponse{Status: "success", Data: map[string]string{"id": "123"}},
			},
			wantErr: false,
		},
		{
			name: "error_response",
			result: APIResult{
				Response: ErrorResponse{Status: "error", Message: "Not found", Code: 404},
			},
			wantErr: false,
		},
		{
			name: "pending_response",
			result: APIResult{
				Response: PendingResponse{Status: "pending", Progress: 75},
			},
			wantErr: false,
		},
		{
			name: "empty_discriminator",
			result: APIResult{
				Response: ErrorResponse{Status: "", Message: "Bad", Code: 500},
			},
			wantErr: true,
		},
		{
			name: "typo_in_discriminator",
			result: APIResult{
				Response: PendingResponse{Status: "pendin", Progress: 50},
			},
			wantErr: true,
		},
		{
			name: "type_not_in_mapping",
			result: APIResult{
				Response: InvalidResponse{Status: "invalid", Reason: "Bad"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.Validate(&tt.result)
			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", errs, tt.wantErr)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Mixed Constraints Tests (Union + OneOf)
// ═══════════════════════════════════════════════════════════════════════════

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
	)
}

func TestMixedConstraints(t *testing.T) {
	validator := godantic.NewValidator[APIResponse]()

	tests := []struct {
		name     string
		response APIResponse
		wantErr  bool
	}{
		{
			name:     "success_string_payload",
			response: APIResponse{Status: "success", Payload: "completed"},
			wantErr:  false,
		},
		{
			name:     "success_object_payload",
			response: APIResponse{Status: "success", Payload: map[string]any{"id": 1}},
			wantErr:  false,
		},
		{
			name:     "error_array_payload",
			response: APIResponse{Status: "error", Payload: []string{"err1", "err2"}},
			wantErr:  false,
		},
		{
			name:     "invalid_status",
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
