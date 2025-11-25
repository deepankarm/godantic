package godantic

import (
	"encoding/json"
	"reflect"
)

// Hook method names
const (
	hookBeforeValidate  = "BeforeValidate"
	hookAfterValidate   = "AfterValidate"
	hookBeforeSerialize = "BeforeSerialize"
	hookAfterSerialize  = "AfterSerialize"
)

// callNoArgHook calls a hook method with no arguments that returns error
func callNoArgHook(obj any, methodName string) error {
	method := reflect.ValueOf(obj).MethodByName(methodName)
	if !method.IsValid() {
		return nil
	}

	results := method.Call([]reflect.Value{})
	if len(results) > 0 && !results[0].IsNil() {
		if err, ok := results[0].Interface().(error); ok {
			return err
		}
	}
	return nil
}

// callAfterValidateHook calls AfterValidate if the type implements it
func callAfterValidateHook[T any](obj *T) error {
	return callNoArgHook(obj, hookAfterValidate)
}

// callBeforeSerializeHook calls BeforeSerialize if the type implements it
func callBeforeSerializeHook[T any](obj *T) error {
	return callNoArgHook(obj, hookBeforeSerialize)
}

// callAfterSerializeHook calls AfterSerialize if the type implements it
func callAfterSerializeHook[T any](data []byte) ([]byte, error) {
	var obj T
	method := reflect.ValueOf(&obj).MethodByName(hookAfterSerialize)
	if !method.IsValid() {
		return data, nil
	}

	results := method.Call([]reflect.Value{reflect.ValueOf(data)})
	if len(results) < 2 {
		return data, nil
	}

	// First result is []byte, second is error
	if !results[1].IsNil() {
		if err, ok := results[1].Interface().(error); ok {
			return nil, err
		}
	}

	if resultData, ok := results[0].Interface().([]byte); ok {
		return resultData, nil
	}

	return data, nil
}

// applyBeforeValidateHook applies BeforeValidate hook and converts result to requested type
func applyBeforeValidateHook[R any](valPtr reflect.Value, data []byte) (R, ValidationErrors) {
	var zero R

	// Check if hook exists
	ptrType := valPtr.Type()
	method, hasHook := ptrType.MethodByName(hookBeforeValidate)

	if !hasHook {
		// No hook - parse directly to requested type
		return parseToType[R](data)
	}

	// Parse to map[string]any for modification
	var rawDataAny map[string]any
	if err := json.Unmarshal(data, &rawDataAny); err != nil {
		return zero, ValidationErrors{{Loc: []string{}, Message: "failed to parse JSON: " + err.Error(), Type: ErrorTypeJSONDecode}}
	}

	// Call the hook
	results := method.Func.Call([]reflect.Value{valPtr, reflect.ValueOf(rawDataAny)})
	if len(results) > 0 && !results[0].IsNil() {
		if err, ok := results[0].Interface().(error); ok {
			return zero, ValidationErrors{{Loc: []string{}, Message: "BeforeValidate hook failed: " + err.Error(), Type: ErrorTypeHookError}}
		}
	}

	// Convert modified map to requested type
	return convertMapToType[R](rawDataAny)
}

// parseToType parses JSON bytes to the requested type (no hook path)
func parseToType[R any](data []byte) (R, ValidationErrors) {
	var zero R
	var result any

	switch any(zero).(type) {
	case map[string]json.RawMessage:
		var rawData map[string]json.RawMessage
		if err := json.Unmarshal(data, &rawData); err != nil {
			return zero, ValidationErrors{{Loc: []string{}, Message: "failed to parse JSON: " + err.Error(), Type: ErrorTypeJSONDecode}}
		}
		result = rawData
	case []byte:
		result = data
	default:
		return zero, ValidationErrors{{Loc: []string{}, Message: "unsupported return type for BeforeValidate hook", Type: ErrorTypeInternal}}
	}

	return result.(R), nil
}

// convertMapToType converts map[string]any to the requested type (with hook path)
func convertMapToType[R any](rawDataAny map[string]any) (R, ValidationErrors) {
	var zero R
	var result any

	switch any(zero).(type) {
	case map[string]json.RawMessage:
		// Convert map[string]any to map[string]json.RawMessage field-by-field
		rawData := make(map[string]json.RawMessage, len(rawDataAny))
		for key, value := range rawDataAny {
			valueBytes, err := json.Marshal(value)
			if err != nil {
				return zero, ValidationErrors{{Loc: []string{key}, Message: "failed to marshal field: " + err.Error(), Type: ErrorTypeJSONEncode}}
			}
			rawData[key] = valueBytes
		}
		result = rawData
	case []byte:
		// Marshal entire map to JSON bytes
		modifiedData, err := json.Marshal(rawDataAny)
		if err != nil {
			return zero, ValidationErrors{{Loc: []string{}, Message: "failed to marshal modified data: " + err.Error(), Type: ErrorTypeJSONEncode}}
		}
		result = modifiedData
	default:
		return zero, ValidationErrors{{Loc: []string{}, Message: "unsupported return type for BeforeValidate hook", Type: ErrorTypeInternal}}
	}

	return result.(R), nil
}
