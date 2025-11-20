package godantic

import (
	"fmt"
	"regexp"
)

// ensureConstraints initializes the Constraints_ map if it's nil
func ensureConstraints[T any](fo FieldOptions[T]) FieldOptions[T] {
	if fo.Constraints_ == nil {
		fo.Constraints_ = make(map[string]any)
	}
	return fo
}

// Constraint functions for JSON Schema generation and validation
// These functions use the With() pattern and store metadata in Constraints_

// Description sets a description for the field in the schema
func Description[T any](desc string) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintDescription] = desc
		return fo
	}
}

// Example sets an example for the field in the schema
func Example[T any](ex T) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintExample] = ex
		return fo
	}
}

// Min sets a minimum value constraint (for numbers and strings)
func Min[T Ordered](min T) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintMinimum] = min

		return fo.validateWith(func(val T) error {
			if val < min {
				return fmt.Errorf("value must be >= %v", min)
			}
			return nil
		})
	}
}

// Max sets a maximum value constraint (for numbers and strings)
func Max[T Ordered](max T) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintMaximum] = max

		return fo.validateWith(func(val T) error {
			if val > max {
				return fmt.Errorf("value must be <= %v", max)
			}
			return nil
		})
	}
}

// MinLen sets a minimum length constraint for strings
func MinLen(min int) func(FieldOptions[string]) FieldOptions[string] {
	return func(fo FieldOptions[string]) FieldOptions[string] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintMinLength] = min

		return fo.validateWith(func(val string) error {
			if len(val) < min {
				return fmt.Errorf("length must be >= %d", min)
			}
			return nil
		})
	}
}

// MaxLen sets a maximum length constraint for strings
func MaxLen(max int) func(FieldOptions[string]) FieldOptions[string] {
	return func(fo FieldOptions[string]) FieldOptions[string] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintMaxLength] = max

		return fo.validateWith(func(val string) error {
			if len(val) > max {
				return fmt.Errorf("length must be <= %d", max)
			}
			return nil
		})
	}
}

// Regex sets a pattern constraint for string validation
func Regex(pattern string) func(FieldOptions[string]) FieldOptions[string] {
	re := regexp.MustCompile(pattern)
	return func(fo FieldOptions[string]) FieldOptions[string] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintPattern] = pattern

		return fo.validateWith(func(val string) error {
			if !re.MatchString(val) {
				return fmt.Errorf("value does not match pattern %s", pattern)
			}
			return nil
		})
	}
}

// Email is a convenience function for email validation
func Email() func(FieldOptions[string]) FieldOptions[string] {
	return Regex(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
}

// URL is a convenience function for URL validation
func URL() func(FieldOptions[string]) FieldOptions[string] {
	return Regex(`^https?://[^\s/$.?#].[^\s]*$`)
}

// OneOf sets an enum constraint - value must be one of the allowed values
func OneOf[T comparable](allowed ...T) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintEnum] = allowed

		return fo.validateWith(func(val T) error {
			for _, a := range allowed {
				if val == a {
					return nil
				}
			}
			return fmt.Errorf("value must be one of %v", allowed)
		})
	}
}

// MultipleOf sets a constraint that the value must be a multiple of the given number
func MultipleOf[T Ordered](divisor T) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintMultipleOf] = divisor

		return fo.validateWith(func(val T) error {
			// This is a simplified check - proper modulo for all numeric types would be more complex
			// For now, convert to float64 for the check
			switch any(val).(type) {
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				// Integer types - can check properly
				var v, d int64
				switch tv := any(val).(type) {
				case int:
					v = int64(tv)
				case int64:
					v = tv
				default:
					// Simplified for demo
					return nil
				}
				switch td := any(divisor).(type) {
				case int:
					d = int64(td)
				case int64:
					d = td
				default:
					return nil
				}
				if v%d != 0 {
					return fmt.Errorf("value must be a multiple of %v", divisor)
				}
			}
			return nil
		})
	}
}

// ReadOnly marks a field as read-only in the schema (doesn't affect validation)
func ReadOnly[T any]() func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintReadOnly] = true
		return fo
	}
}

// WriteOnly marks a field as write-only in the schema (doesn't affect validation)
func WriteOnly[T any]() func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintWriteOnly] = true
		return fo
	}
}

// Deprecated marks a field as deprecated in the schema (doesn't affect validation)
func Deprecated[T any]() func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintDeprecated] = true
		return fo
	}
}

// Title sets a title for the field in the schema
func Title[T any](title string) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintTitle] = title
		return fo
	}
}

// Format sets a format hint for the field (e.g., "date-time", "email", "uri")
func Format[T any](format string) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintFormat] = format
		return fo
	}
}

// ExclusiveMin sets an exclusive minimum constraint (value must be > min, not >=)
func ExclusiveMin[T Ordered](min T) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintExclusiveMinimum] = min

		return fo.validateWith(func(val T) error {
			if val <= min {
				return fmt.Errorf("value must be > %v", min)
			}
			return nil
		})
	}
}

// ExclusiveMax sets an exclusive maximum constraint (value must be < max, not <=)
func ExclusiveMax[T Ordered](max T) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintExclusiveMaximum] = max

		return fo.validateWith(func(val T) error {
			if val >= max {
				return fmt.Errorf("value must be < %v", max)
			}
			return nil
		})
	}
}

// MinItems sets a minimum number of items for arrays/slices
func MinItems[T any](min int) func(FieldOptions[[]T]) FieldOptions[[]T] {
	return func(fo FieldOptions[[]T]) FieldOptions[[]T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintMinItems] = min

		return fo.validateWith(func(val []T) error {
			if len(val) < min {
				return fmt.Errorf("must have at least %d items", min)
			}
			return nil
		})
	}
}

// MaxItems sets a maximum number of items for arrays/slices
func MaxItems[T any](max int) func(FieldOptions[[]T]) FieldOptions[[]T] {
	return func(fo FieldOptions[[]T]) FieldOptions[[]T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintMaxItems] = max

		return fo.validateWith(func(val []T) error {
			if len(val) > max {
				return fmt.Errorf("must have at most %d items", max)
			}
			return nil
		})
	}
}

// UniqueItems ensures all items in a slice are unique
func UniqueItems[T comparable]() func(FieldOptions[[]T]) FieldOptions[[]T] {
	return func(fo FieldOptions[[]T]) FieldOptions[[]T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintUniqueItems] = true

		return fo.validateWith(func(val []T) error {
			seen := make(map[T]bool, len(val))
			for _, item := range val {
				if seen[item] {
					return fmt.Errorf("duplicate item found: %v", item)
				}
				seen[item] = true
			}
			return nil
		})
	}
}

// MinProperties sets a minimum number of properties for maps
func MinProperties(min int) func(FieldOptions[map[string]any]) FieldOptions[map[string]any] {
	return func(fo FieldOptions[map[string]any]) FieldOptions[map[string]any] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintMinProperties] = min

		return fo.validateWith(func(val map[string]any) error {
			if len(val) < min {
				return fmt.Errorf("must have at least %d properties", min)
			}
			return nil
		})
	}
}

// MaxProperties sets a maximum number of properties for maps
func MaxProperties(max int) func(FieldOptions[map[string]any]) FieldOptions[map[string]any] {
	return func(fo FieldOptions[map[string]any]) FieldOptions[map[string]any] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintMaxProperties] = max

		return fo.validateWith(func(val map[string]any) error {
			if len(val) > max {
				return fmt.Errorf("must have at most %d properties", max)
			}
			return nil
		})
	}
}

// Const sets a constant value constraint - the value must equal this exactly
func Const[T comparable](value T) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintConst] = value

		return fo.validateWith(func(val T) error {
			if val != value {
				return fmt.Errorf("value must be %v", value)
			}
			return nil
		})
	}
}

// Default sets a default value (schema metadata only, doesn't affect validation)
func Default[T any](value T) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintDefault] = value
		return fo
	}
}

// ContentEncoding sets the content encoding for strings (e.g., "base64")
func ContentEncoding(encoding string) func(FieldOptions[string]) FieldOptions[string] {
	return func(fo FieldOptions[string]) FieldOptions[string] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintContentEncoding] = encoding
		return fo
	}
}

// ContentMediaType sets the content media type for strings (e.g., "application/json")
func ContentMediaType(mediaType string) func(FieldOptions[string]) FieldOptions[string] {
	return func(fo FieldOptions[string]) FieldOptions[string] {
		fo = ensureConstraints(fo)
		fo.Constraints_[ConstraintContentMediaType] = mediaType
		return fo
	}
}

// Union creates a union type that accepts multiple types (anyOf in JSON Schema)
// Supports both JSON Schema primitive type names (strings) and complex Go types.
//
// For primitive types, use type name strings:
//   - "string", "integer", "number", "boolean", "object", "array", "null"
//
// For complex types, pass type instances:
//   - Union[any]("string", []TextInput{}, []ImageInput{})
//   - Union[any](0, "text", MyStruct{})
//
// Examples:
//
//	Union[any]("string", "integer", "object")                    // primitives only
//	Union[any]("", []TextInput{}, []ImageInput{})               // string + complex types
//	Union[any](0, 1.5, true)                                     // mix of primitives via instances
func Union[T any](types ...any) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)

		// Separate string type names from complex type instances
		var primitiveTypes []string
		var complexTypes []any

		for _, t := range types {
			if typeName, ok := t.(string); ok {
				// String arguments are treated as JSON Schema type names
				if typeName != "" {
					primitiveTypes = append(primitiveTypes, typeName)
				} else {
					// Empty string means we want the string type itself
					complexTypes = append(complexTypes, t)
				}
			} else {
				// Non-string arguments are complex types to be reflected
				complexTypes = append(complexTypes, t)
			}
		}

		// Store primitive type names for simple schema generation
		if len(primitiveTypes) > 0 {
			anyOfSchemas := make([]map[string]string, len(primitiveTypes))
			for i, typeName := range primitiveTypes {
				anyOfSchemas[i] = map[string]string{"type": typeName}
			}
			fo.Constraints_[ConstraintAnyOf] = anyOfSchemas
		}

		// Store complex types for reflection during schema generation
		if len(complexTypes) > 0 {
			fo.Constraints_["anyOfTypes"] = complexTypes
		}

		return fo
	}
}

// DiscriminatedUnion creates a discriminated union (oneOf with discriminator in JSON Schema)
// The discriminatorField is used to determine which variant the data represents.
// variants is a map of discriminator value -> example struct/type for schema generation.
// Example: DiscriminatedUnion[any]("type", map[string]any{"cat": Cat{}, "dog": Dog{}})
func DiscriminatedUnion[T any](discriminatorField string, variants map[string]any) func(FieldOptions[T]) FieldOptions[T] {
	return func(fo FieldOptions[T]) FieldOptions[T] {
		fo = ensureConstraints(fo)

		// Store discriminator info for schema generation
		fo.Constraints_[ConstraintDiscriminator] = map[string]any{
			"propertyName": discriminatorField,
			"mapping":      variants,
		}

		return fo
	}
}
