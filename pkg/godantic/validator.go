package godantic

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/deepankarm/godantic/pkg/internal/errors"
)

// Re-exported types from internal/errors for public API.
type (
	ValidationError  = errors.ValidationError
	ValidationErrors = errors.ValidationErrors
	ErrorType        = errors.ErrorType
)

// Error type constants - re-exported for public API.
// Usage: err.Type == godantic.ErrorTypeRequired
const (
	ErrorTypeRequired             = errors.ErrorTypeRequired
	ErrorTypeConstraint           = errors.ErrorTypeConstraint
	ErrorTypeInternal             = errors.ErrorTypeInternal
	ErrorTypeJSONDecode           = errors.ErrorTypeJSONDecode
	ErrorTypeJSONEncode           = errors.ErrorTypeJSONEncode
	ErrorTypeHookError            = errors.ErrorTypeHookError
	ErrorTypeDiscriminatorMissing = errors.ErrorTypeDiscriminatorMissing
	ErrorTypeDiscriminatorInvalid = errors.ErrorTypeDiscriminatorInvalid
	ErrorTypeMismatch             = errors.ErrorTypeMismatch
	ErrorTypeMarshalError         = errors.ErrorTypeMarshalError
)

// Ordered is a constraint for types that support comparison
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 | ~string
}

// FieldOptions defines validation rules and metadata
type FieldOptions[T any] struct {
	Required_    bool
	Validators_  []func(T) error
	Constraints_ map[string]any // For schema generation (description, example, min, max, minLength, etc.)
}

func (fo FieldOptions[T]) validateWith(fn func(T) error) FieldOptions[T] {
	fo.Validators_ = append(fo.Validators_, fn)
	return fo
}

// Field creates a FieldOptions with multiple constraints applied
func Field[T any](fns ...func(FieldOptions[T]) FieldOptions[T]) FieldOptions[T] {
	fo := FieldOptions[T]{}
	for _, fn := range fns {
		fo = fn(fo)
	}
	return fo
}

// Required marks a field as required (can be used with Field)
func Required[T any]() func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo.Required_ = true
		return fo
	}
}

// Validate adds a custom validator function (can be used with Field)
func Validate[T any](fn func(T) error) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo.Validators_ = append(fo.Validators_, fn)
		return fo
	}
}

// fieldOptionHolder holds field options with type erasure
type fieldOptionHolder struct {
	required    bool
	validators  []func(any) error
	constraints map[string]any // Includes description, example, and all schema metadata
}

// Required returns whether the field is required
func (foh *fieldOptionHolder) Required() bool {
	return foh.required
}

// Constraints returns the constraints map
func (foh *fieldOptionHolder) Constraints() map[string]any {
	return foh.constraints
}

// Validator validates structs or discriminated union interfaces
type Validator[T any] struct {
	fieldOptions map[string]*fieldOptionHolder
	config       validatorConfig
}

// NewValidator creates a new validator for type T.
// For concrete structs, use without options: NewValidator[MyStruct]()
// For discriminated unions (interfaces), use with WithDiscriminator option:
//
//	NewValidator[MyInterface](WithDiscriminator("type", map[string]any{...}))
func NewValidator[T any](opts ...ValidatorOption) *Validator[T] {
	v := &Validator[T]{
		fieldOptions: make(map[string]*fieldOptionHolder),
	}

	// Apply options
	for _, opt := range opts {
		opt.apply(&v.config)
	}

	// Only scan field options if this is a concrete struct (not a discriminated union interface)
	if v.config.discriminator == nil {
		v.scanFieldOptions()
	}

	return v
}

func (v *Validator[T]) scanFieldOptions() {
	var zero T
	typ := reflect.TypeOf(zero)
	v.fieldOptions = scanner.scanFieldOptionsFromType(typ)
}

func (v *Validator[T]) Validate(obj *T) ValidationErrors {
	objPtr := reflect.ValueOf(obj)
	return walkValidate(objPtr)
}

// ApplyDefaults applies default values to zero-valued fields that have defaults defined.
// This should be called after JSON unmarshaling to set defaults for missing fields.
// Returns an error if reflection fails.
func (v *Validator[T]) ApplyDefaults(obj *T) error {
	objPtr := reflect.ValueOf(obj)
	return walkDefaults(objPtr)
}

// Unmarshal unmarshals JSON data, applies defaults, and validates.
// This is a convenience method that combines the three common steps:
// 1. Unmarshal JSON into the struct (or route to correct type for discriminated unions)
// 2. Apply default values to zero-valued fields
// 3. Validate the struct
// Returns the populated struct and any validation errors.
func (v *Validator[T]) Unmarshal(data []byte) (*T, ValidationErrors) {
	// Check if this is a discriminated union validator
	if v.config.discriminator != nil {
		return v.validateDiscriminatedUnion(data, v.config.discriminator)
	}

	var obj T
	objPtr := reflect.New(reflect.TypeOf(obj))

	// BeforeValidate hook: transform JSON before unmarshaling
	data, hookErrs := applyBeforeValidateHook[[]byte](objPtr, data)
	if hookErrs != nil {
		return nil, hookErrs
	}

	// Use the tree walker for unmarshal + defaults + validation
	errs := walkParse(objPtr, data)

	// Return nil on JSON decode errors (before we have a valid struct)
	for _, e := range errs {
		if e.Type == "json_decode" {
			return nil, errs
		}
	}

	obj = objPtr.Elem().Interface().(T)

	if len(errs) > 0 {
		return &obj, errs
	}

	// AfterValidate hook: transform struct after validation
	if err := callAfterValidateHook(&obj); err != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("AfterValidate hook failed: %v", err),
			Type:    ErrorTypeHookError,
		}}
	}

	return &obj, nil
}

// Marshal validates the struct, applies defaults, and marshals to JSON.
// This is a convenience method that combines the three common steps:
// 1. Validate the struct
// 2. Apply default values to zero-valued fields
// 3. Marshal the struct to JSON
// Returns the JSON bytes and any validation errors.
func (v *Validator[T]) Marshal(obj *T) ([]byte, ValidationErrors) {
	// Check if this is a discriminated union validator
	if v.config.discriminator != nil {
		return v.marshalDiscriminatedUnion(obj, v.config.discriminator)
	}

	// BeforeSerialize hook: transform struct before validation
	if err := callBeforeSerializeHook(obj); err != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("BeforeSerialize hook failed: %v", err),
			Type:    ErrorTypeHookError,
		}}
	}

	// Validate struct
	errs := v.Validate(obj)
	if len(errs) > 0 {
		return nil, errs
	}

	// Apply defaults to ensure all default values are set
	if err := v.ApplyDefaults(obj); err != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("apply defaults failed: %v", err),
			Type:    ErrorTypeInternal,
		}}
	}

	// Marshal to JSON
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("json marshal failed: %v", err),
			Type:    ErrorTypeJSONEncode,
		}}
	}

	// AfterSerialize hook: transform JSON after marshaling
	data, err = callAfterSerializeHook[T](data)
	if err != nil {
		return nil, ValidationErrors{{
			Loc:     []string{},
			Message: fmt.Sprintf("AfterSerialize hook failed: %v", err),
			Type:    ErrorTypeHookError,
		}}
	}

	return data, nil
}

// FieldOptions returns the field options map (for schema generation)
func (v *Validator[T]) FieldOptions() map[string]any {
	result := make(map[string]any, len(v.fieldOptions))
	for k, v := range v.fieldOptions {
		result[k] = v
	}
	return result
}

// UnmarshalPartial parses potentially incomplete JSON into a struct.
// Returns the partially populated struct, its completion state, and any errors.
//
// Unlike Unmarshal(), this method:
// - Does NOT fail on incomplete JSON
// - Tracks which fields are incomplete
// - Skips validation for incomplete fields
//
// Example (LLM streaming):
//
//	validator := godantic.NewValidator[ToolCall]()
//	result, state, errs := validator.UnmarshalPartial([]byte(`{"type": "sear`))
//	if state.IsComplete {
//	    // Full JSON received - use result
//	}
func (v *Validator[T]) UnmarshalPartial(data []byte) (*T, *PartialState, ValidationErrors) {
	if v.config.discriminator != nil {
		return v.unmarshalPartialDiscriminatedUnion(data, v.config.discriminator)
	}

	// Parse and repair the incomplete JSON first
	parseResult, parseErrs := parsePartialJSON(data)
	if parseErrs != nil {
		return nil, &PartialState{IsComplete: false}, parseErrs
	}

	var obj T
	objPtr := reflect.New(reflect.TypeOf(obj))

	return unmarshalPartialCommon[T](objPtr, parseResult)
}
