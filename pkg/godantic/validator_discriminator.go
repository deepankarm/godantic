package godantic

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/deepankarm/godantic/pkg/internal/reflectutil"
)

// validateDiscriminatedUnion handles validation for discriminated union types (interfaces)
func (v *Validator[T]) validateDiscriminatedUnion(data []byte, cfg *discriminatorConfig) (*T, ValidationErrors) {
	instance, errs := newUnionFromJSON[T](data, cfg)
	if errs != nil {
		return nil, errs
	}

	// Use Walker for unmarshal + defaults + validation (single traversal)
	if walkErrs := walkParse(instance.ptr, data); len(walkErrs) > 0 {
		for _, e := range walkErrs {
			if e.Type == ErrorTypeJSONDecode {
				return nil, walkErrs
			}
		}
		result := instance.Result()
		return &result, walkErrs
	}

	result := instance.Result()
	return &result, nil
}

// unmarshalDiscriminatedUnion handles unmarshaling (struct → JSON) for discriminated unions
func (v *Validator[T]) unmarshalDiscriminatedUnion(obj *T, cfg *discriminatorConfig) ([]byte, ValidationErrors) {
	instance, errs := newUnionFromStruct[T](obj, cfg)
	if errs != nil {
		return nil, errs
	}

	if err := walkDefaults(instance.ptr); err != nil {
		return nil, ValidationErrors{{Message: fmt.Sprintf("apply defaults failed: %v", err), Type: ErrorTypeInternal}}
	}
	if errs := walkValidate(instance.ptr); len(errs) > 0 {
		return nil, errs
	}

	data, err := json.Marshal(instance.ptr.Interface())
	if err != nil {
		return nil, ValidationErrors{{Message: fmt.Sprintf("json marshal failed: %v", err), Type: ErrorTypeJSONEncode}}
	}
	return data, nil
}

// unionInstance encapsulates discriminated union processing state
type unionInstance[T any] struct {
	ptr          reflect.Value
	concreteType reflect.Type
}

// Result converts the internal value to the target interface type T.
func (u *unionInstance[T]) Result() T {
	return reflectutil.ConvertToInterfaceType[T](u.ptr, u.concreteType)
}

// newUnionFromJSON creates a union instance by peeking at JSON to find discriminator.
func newUnionFromJSON[T any](data []byte, cfg *discriminatorConfig) (*unionInstance[T], ValidationErrors) {
	var peek map[string]any
	if err := json.Unmarshal(data, &peek); err != nil {
		return nil, ValidationErrors{{Message: fmt.Sprintf("json unmarshal failed: %v", err), Type: ErrorTypeJSONDecode}}
	}

	discValue, ok := peek[cfg.field]
	if !ok {
		return nil, ValidationErrors{{Loc: []string{cfg.field}, Message: fmt.Sprintf("discriminator field '%s' not found", cfg.field), Type: ErrorTypeDiscriminatorMissing}}
	}

	concreteType, validationErr := cfg.lookupConcreteType(fmt.Sprintf("%v", discValue))
	if validationErr != nil {
		return nil, ValidationErrors{*validationErr}
	}

	elemType := reflectutil.UnwrapPointer(concreteType)
	return &unionInstance[T]{ptr: reflect.New(elemType), concreteType: concreteType}, nil
}

// newUnionFromStruct creates a union instance from an existing struct value.
func newUnionFromStruct[T any](obj *T, cfg *discriminatorConfig) (*unionInstance[T], ValidationErrors) {
	concreteValue, errs := unwrapToConcreteValue(reflect.ValueOf(obj))
	if errs != nil {
		return nil, errs
	}

	concreteType := concreteValue.Type()
	discField := reflectutil.FieldByJSONName(concreteValue, concreteType, cfg.field)
	if !discField.IsValid() {
		return nil, ValidationErrors{{Loc: []string{cfg.field}, Message: fmt.Sprintf("discriminator field '%s' not found in type %s", cfg.field, concreteType.Name()), Type: ErrorTypeDiscriminatorMissing}}
	}

	expectedType, validationErr := cfg.lookupConcreteType(fmt.Sprintf("%v", discField.Interface()))
	if validationErr != nil {
		return nil, ValidationErrors{*validationErr}
	}

	if err := checkTypeMatch(concreteType, expectedType); err != nil {
		return nil, err
	}

	structValue := concreteValue
	if concreteValue.Kind() == reflect.Pointer {
		structValue = concreteValue.Elem()
	}
	elemType := reflectutil.UnwrapPointer(concreteType)

	return &unionInstance[T]{ptr: reflectutil.MakeAddressable(structValue, elemType), concreteType: concreteType}, nil
}

// unwrapToConcreteValue unwraps pointer/interface to get the concrete struct value.
func unwrapToConcreteValue(v reflect.Value) (reflect.Value, ValidationErrors) {
	if v.Kind() != reflect.Pointer {
		return reflect.Value{}, ValidationErrors{{Message: "obj must be a pointer", Type: ErrorTypeInternal}}
	}

	v = v.Elem()
	if !v.IsValid() || v.IsZero() {
		return reflect.Value{}, ValidationErrors{{Message: "obj is nil or zero value", Type: ErrorTypeInternal}}
	}

	if v.Kind() == reflect.Interface {
		v = v.Elem()
		if !v.IsValid() {
			return reflect.Value{}, ValidationErrors{{Message: "interface value is nil", Type: ErrorTypeInternal}}
		}
	}
	return v, nil
}

// checkTypeMatch validates the concrete type matches the expected type from discriminator.
func checkTypeMatch(concreteType, expectedType reflect.Type) ValidationErrors {
	expectedElem := reflectutil.UnwrapPointer(expectedType)
	if concreteType != expectedElem && concreteType != expectedType {
		return ValidationErrors{{Message: fmt.Sprintf("type mismatch: expected %s, got %s", expectedType.Name(), concreteType.Name()), Type: ErrorTypeMismatch}}
	}
	return nil
}
