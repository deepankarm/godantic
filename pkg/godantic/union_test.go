package godantic_test

import (
	"testing"

	"github.com/deepankarm/godantic/pkg/godantic"
)

type FlexibleConfig struct {
	// Value can be string, integer, or object
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

type Cat struct {
	Type string
	Meow string
}

type Dog struct {
	Type string
	Bark string
}

type Bird struct {
	Type  string
	Chirp string
}

type AnimalResponse struct {
	Animal any
}

func (a *AnimalResponse) FieldAnimal() godantic.FieldOptions[any] {
	return godantic.Field(
		godantic.Required[any](),
		godantic.DiscriminatedUnion[any]("Type", map[string]any{
			"cat":  Cat{},
			"dog":  Dog{},
			"bird": Bird{},
		}),
		godantic.Description[any]("The animal can be a cat, dog, or bird"),
	)
}

func TestDiscriminatedUnion(t *testing.T) {
	validator := godantic.NewValidator[AnimalResponse]()

	tests := []struct {
		name    string
		animal  AnimalResponse
		wantErr bool
	}{
		{
			name:    "cat should pass",
			animal:  AnimalResponse{Animal: Cat{Type: "cat", Meow: "meow"}},
			wantErr: false,
		},
		{
			name:    "dog should pass",
			animal:  AnimalResponse{Animal: Dog{Type: "dog", Bark: "woof"}},
			wantErr: false,
		},
		{
			name:    "bird should pass",
			animal:  AnimalResponse{Animal: Bird{Type: "bird", Chirp: "tweet"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.Validate(&tt.animal)
			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", errs, tt.wantErr)
			}
		})
	}
}

// Test complex union scenario - mixing union types with other constraints

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
