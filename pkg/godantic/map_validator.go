package godantic

import (
	"encoding/json"
	"reflect"
	"strings"
)

// ValidateFromStringMap validates data from a map[string]string (for path params, cookies)
// Converts string values to appropriate Go types based on struct field types
func (v *Validator[T]) ValidateFromStringMap(data map[string]string) (*T, ValidationErrors) {
	var zero T
	typ := reflect.TypeOf(zero)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	// Map JSON field names to struct field types
	fieldTypes := make(map[string]reflect.Type)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			fieldName := strings.Split(jsonTag, ",")[0]
			fieldTypes[fieldName] = field.Type
		}
	}

	// Convert string values to appropriate types
	dataMap := make(map[string]any)
	for key, value := range data {
		fieldType, ok := fieldTypes[key]
		if !ok {
			// Unknown field, pass as string
			dataMap[key] = value
			continue
		}

		// Convert based on field type
		dataMap[key] = convertStringToType(value, fieldType)
	}

	// Marshal to JSON and validate
	jsonData, err := json.Marshal(dataMap)
	if err != nil {
		return nil, ValidationErrors{
			{
				Loc:     []string{},
				Message: "failed to marshal data: " + err.Error(),
				Type:    ErrorTypeMarshalError,
			},
		}
	}

	return v.Unmarshal(jsonData)
}

// ValidateFromMultiValueMap validates data from a map[string][]string (for query params, headers)
// Converts string values to appropriate Go types based on struct field types
func (v *Validator[T]) ValidateFromMultiValueMap(data map[string][]string) (*T, ValidationErrors) {
	var zero T
	typ := reflect.TypeOf(zero)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	// Map JSON field names to struct field types
	fieldTypes := make(map[string]reflect.Type)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			fieldName := strings.Split(jsonTag, ",")[0]
			fieldTypes[fieldName] = field.Type
		}
	}

	// Convert multi-value string data to appropriate types
	dataMap := make(map[string]any)
	for key, values := range data {
		if len(values) == 0 {
			continue
		}

		fieldType, ok := fieldTypes[key]
		if !ok {
			// Unknown field, use first value as string
			dataMap[key] = values[0]
			continue
		}

		// For array/slice types, use all values
		if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
			dataMap[key] = values
		} else {
			// For non-array types, use first value
			dataMap[key] = convertStringToType(values[0], fieldType)
		}
	}

	// Marshal to JSON and validate
	jsonData, err := json.Marshal(dataMap)
	if err != nil {
		return nil, ValidationErrors{
			{
				Loc:     []string{},
				Message: "failed to marshal data: " + err.Error(),
				Type:    ErrorTypeMarshalError,
			},
		}
	}

	return v.Unmarshal(jsonData)
}

// convertStringToType converts a string value to the appropriate Go type
func convertStringToType(value string, fieldType reflect.Type) any {
	switch fieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intVal, err := json.Number(value).Int64(); err == nil {
			return intVal
		}
		return value // Let validator handle the error

	case reflect.Float32, reflect.Float64:
		if floatVal, err := json.Number(value).Float64(); err == nil {
			return floatVal
		}
		return value

	case reflect.Bool:
		switch value {
		case "true", "1":
			return true
		case "false", "0":
			return false
		}
		return value

	default:
		return value
	}
}
