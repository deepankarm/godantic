package godantic

import (
	"reflect"
)

// fieldScanner provides common reflection utilities for scanning Field{Name}() methods
type fieldScanner struct{}

// scanFieldOptionsFromType scans a type for Field{Name}() methods and returns field options
// This is the core shared logic used by both Validator and schema generation
func (fs *fieldScanner) scanFieldOptionsFromType(typ reflect.Type) map[string]*fieldOptionHolder {
	fieldOptions := make(map[string]*fieldOptionHolder)

	// First, look for Field{Name}() methods on the parent struct
	ptrType := reflect.PointerTo(typ)
	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		if len(method.Name) > 5 && method.Name[:5] == "Field" {
			fieldName := method.Name[5:]
			// Call method to get options
			zeroPtr := reflect.New(typ)
			result := method.Func.Call([]reflect.Value{zeroPtr})
			if len(result) > 0 {
				holder := fs.extractFieldOptions(result[0])
				fieldOptions[fieldName] = holder
			}
		}
	}

	// Second, check each struct field for type-level validation
	// Only if parent struct didn't define Field{Name}() method
	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		fieldName := structField.Name

		// Skip if parent struct already defined validation for this field
		if _, exists := fieldOptions[fieldName]; exists {
			continue
		}

		// Check if the field's type has Field{TypeName}() method
		fieldType := structField.Type
		typeName := fieldType.Name()
		if typeName == "" {
			continue // Skip anonymous types
		}

		methodName := "Field" + typeName

		// Try pointer receiver first
		ptrFieldType := reflect.PointerTo(fieldType)
		method, found := ptrFieldType.MethodByName(methodName)
		if !found {
			// Try value receiver
			method, found = fieldType.MethodByName(methodName)
		}

		if found {
			// Create a zero value instance of the field type
			var fieldInstance reflect.Value
			if method.Type.In(0).Kind() == reflect.Pointer {
				fieldInstance = reflect.New(fieldType)
			} else {
				fieldInstance = reflect.Zero(fieldType)
			}

			// Call the type's Field{TypeName}() method
			result := method.Func.Call([]reflect.Value{fieldInstance})
			if len(result) > 0 {
				holder := fs.extractFieldOptions(result[0])
				fieldOptions[fieldName] = holder
			}
		}
	}

	return fieldOptions
}

// extractFieldOptions extracts validation info from FieldOptions[T] using reflection
func (fs *fieldScanner) extractFieldOptions(optsValue reflect.Value) *fieldOptionHolder {
	holder := &fieldOptionHolder{
		required:    optsValue.FieldByName("Required_").Bool(),
		validators:  []func(any) error{},
		constraints: make(map[string]any),
	}

	// Extract constraints map
	constraintsField := optsValue.FieldByName("Constraints_")
	if constraintsField.IsValid() && !constraintsField.IsNil() {
		// Copy constraints map
		iter := constraintsField.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			value := iter.Value().Interface()
			holder.constraints[key] = value
		}
	}

	// Extract validators using reflection
	validatorsField := optsValue.FieldByName("Validators_")
	if validatorsField.IsValid() && validatorsField.Len() > 0 {
		for j := 0; j < validatorsField.Len(); j++ {
			validatorFunc := validatorsField.Index(j)
			// Wrap the typed validator in a type-erased one
			holder.validators = append(holder.validators, func(val any) error {
				// Call the validator using reflection
				results := validatorFunc.Call([]reflect.Value{reflect.ValueOf(val)})
				if len(results) > 0 && !results[0].IsNil() {
					return results[0].Interface().(error)
				}
				return nil
			})
		}
	}

	return holder
}

// validateNestedStruct validates a nested struct field using reflection
// This is shared between Validator and reflectValidator
func (fs *fieldScanner) validateNestedStruct(field reflect.Value, parentPath []string, fieldOptions map[string]*fieldOptionHolder) []ValidationError {
	// Get the type of the nested struct
	fieldType := field.Type()

	// Check if the nested struct has Field methods (has validation)
	ptrType := reflect.PointerTo(fieldType)
	hasValidation := false
	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		if len(method.Name) > 5 && method.Name[:5] == "Field" {
			hasValidation = true
			break
		}
	}

	if !hasValidation {
		return nil
	}

	// For nested validation, we need to scan the nested struct's Field methods
	// and validate recursively
	var errs []ValidationError
	ptrVal := field.Addr()

	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		if len(method.Name) <= 5 || method.Name[:5] != "Field" {
			continue
		}

		nestedFieldName := method.Name[5:]
		nestedField := field.FieldByName(nestedFieldName)
		if !nestedField.IsValid() {
			continue
		}

		// Get field options by calling the Field method
		result := method.Func.Call([]reflect.Value{ptrVal})
		if len(result) == 0 {
			continue
		}

		// Extract validation info from the result
		nestedFieldPath := append(append([]string{}, parentPath...), nestedFieldName)
		resultType := result[0].Type()

		if resultType.Kind() != reflect.Struct {
			continue
		}

		// Extract Required_
		requiredField := result[0].FieldByName("Required_")
		isRequired := requiredField.IsValid() && requiredField.Kind() == reflect.Bool && requiredField.Bool()

		// Extract Constraints_ to check for defaults
		constraintsField := result[0].FieldByName("Constraints_")
		hasDefault := false
		if constraintsField.IsValid() && constraintsField.Kind() == reflect.Map {
			defaultVal := constraintsField.MapIndex(reflect.ValueOf(ConstraintDefault))
			hasDefault = defaultVal.IsValid()
		}

		// Check if field is zero and required
		if nestedField.IsZero() {
			if isRequired && !hasDefault {
				errs = append(errs, ValidationError{
					Loc:     nestedFieldPath,
					Message: "required field",
					Type:    "required",
				})
				continue
			}
			// Skip validation for optional zero fields
			continue
		}

		// Extract and run validators
		validatorsField := result[0].FieldByName("Validators_")
		if validatorsField.IsValid() && validatorsField.Kind() == reflect.Slice {
			fieldValue := nestedField.Interface()
			for j := 0; j < validatorsField.Len(); j++ {
				validator := validatorsField.Index(j)
				if validator.Kind() == reflect.Func {
					// Call the validator function
					validatorResults := validator.Call([]reflect.Value{reflect.ValueOf(fieldValue)})
					if len(validatorResults) > 0 && !validatorResults[0].IsNil() {
						// Validator returned an error
						if err, ok := validatorResults[0].Interface().(error); ok {
							errs = append(errs, ValidationError{
								Loc:     nestedFieldPath,
								Message: err.Error(),
								Type:    "constraint",
							})
						}
					}
				}
			}
		}

		// Recursively validate nested structs
		if nestedField.Kind() == reflect.Struct && !isBasicType(nestedField.Type()) {
			nestedErrs := fs.validateNestedStruct(nestedField, nestedFieldPath, fieldOptions)
			errs = append(errs, nestedErrs...)
		}

		// Validate pointer to nested struct
		if nestedField.Kind() == reflect.Ptr && !nestedField.IsNil() && nestedField.Elem().Kind() == reflect.Struct {
			nestedErrs := fs.validateNestedStruct(nestedField.Elem(), nestedFieldPath, fieldOptions)
			errs = append(errs, nestedErrs...)
		}
	}
	return errs
}

// applyDefaultsToStruct applies default values to struct fields using reflection
func (fs *fieldScanner) applyDefaultsToStruct(objPtr reflect.Value, fieldOptions map[string]*fieldOptionHolder) error {
	val := objPtr.Elem()

	for fieldName, opts := range fieldOptions {
		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}

		// Only apply default if field is zero value
		if !field.IsZero() {
			continue
		}

		// Check if field can be set
		if !field.CanSet() {
			continue
		}

		// Get default from constraints
		defaultVal, ok := opts.constraints[ConstraintDefault]
		if !ok {
			continue
		}

		// Set the default value
		defaultReflect := reflect.ValueOf(defaultVal)
		if defaultReflect.Type().AssignableTo(field.Type()) {
			field.Set(defaultReflect)
		}
	}

	return nil
}

// Global scanner instance for use across the package
var scanner = &fieldScanner{}

// FieldOptionInfo provides field validation info for schema generation
type FieldOptionInfo struct {
	Required    bool
	Constraints map[string]any
}

// ScanTypeFieldOptions scans a reflect.Type for Field{Name}() methods
// Returns a map of field name -> field option info
// This is the public API for schema generation and other external uses
func ScanTypeFieldOptions(t reflect.Type) map[string]FieldOptionInfo {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	internalOpts := scanner.scanFieldOptionsFromType(t)

	// Convert to public API
	result := make(map[string]FieldOptionInfo, len(internalOpts))
	for fieldName, holder := range internalOpts {
		result[fieldName] = FieldOptionInfo{
			Required:    holder.required,
			Constraints: holder.constraints,
		}
	}

	return result
}

// CollectStructTypes recursively collects all struct types from a type
// This is useful for schema generation and type analysis
func CollectStructTypes(t reflect.Type, types map[string]reflect.Type) {
	if t == nil {
		return
	}

	// Handle pointers
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Only process structs
	if t.Kind() != reflect.Struct {
		return
	}

	// Add this type to the map
	if t.Name() != "" {
		types[t.Name()] = t
	}

	// Recursively process all struct fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldType := field.Type

		// Handle slices/arrays
		if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
			fieldType = fieldType.Elem()
		}

		// Handle pointers
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}

		// Recursively collect struct types
		if fieldType.Kind() == reflect.Struct {
			CollectStructTypes(fieldType, types)
		}
	}
}

// GetJSONSchemaType returns the JSON Schema type string for a Go type
func GetJSONSchemaType(t reflect.Type) string {
	if t == nil {
		return ""
	}

	// Handle pointers
	if t.Kind() == reflect.Pointer {
		return GetJSONSchemaType(t.Elem())
	}

	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	case reflect.Interface:
		// Empty interface is any, which could be considered object or any
		return "object"
	}
	return ""
}

// validateFieldsWithReflection validates struct fields using field options and reflection
// This is the core validation loop used by both Validator and discriminated union validation
func validateFieldsWithReflection(
	objPtr reflect.Value,
	fieldOptions map[string]*fieldOptionHolder,
	path []string,
	validateUnions func(value any, constraints map[string]any, path []string) *ValidationError,
) []ValidationError {
	var errs []ValidationError
	val := objPtr.Elem()

	for fieldName, opts := range fieldOptions {
		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}

		currentPath := append(append([]string{}, path...), fieldName)
		value := field.Interface()

		// Check required fields
		if opts.required && field.IsZero() {
			if _, hasDefault := opts.constraints[ConstraintDefault]; !hasDefault {
				errs = append(errs, ValidationError{
					Loc:     currentPath,
					Message: "required field",
					Type:    "required",
				})
				continue
			}
		}

		// Skip validation for zero values
		if field.IsZero() {
			continue
		}

		// Run validators
		for _, validator := range opts.validators {
			if err := validator(value); err != nil {
				errs = append(errs, ValidationError{
					Loc:     currentPath,
					Message: err.Error(),
					Type:    "constraint",
				})
			}
		}

		// Validate union constraints (if validator function provided)
		if validateUnions != nil {
			if unionErr := validateUnions(value, opts.constraints, currentPath); unionErr != nil {
				errs = append(errs, *unionErr)
			}
		}

		// Recursively validate nested structs
		if field.Kind() == reflect.Struct && !isBasicType(field.Type()) {
			nestedErrs := scanner.validateNestedStruct(field, currentPath, fieldOptions)
			errs = append(errs, nestedErrs...)
		}

		// Validate pointer to struct
		if field.Kind() == reflect.Pointer && !field.IsNil() && field.Elem().Kind() == reflect.Struct {
			nestedErrs := scanner.validateNestedStruct(field.Elem(), currentPath, fieldOptions)
			errs = append(errs, nestedErrs...)
		}
	}

	return errs
}

// isBasicType checks if a type is a basic Go type (not a custom struct that needs validation)
func isBasicType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String,
		reflect.Slice, reflect.Map, reflect.Array:
		return true
	}

	// Check for time.Time and other standard library types we don't want to recurse into
	if t.PkgPath() == "time" || t.PkgPath() == "sync" {
		return true
	}

	return false
}
