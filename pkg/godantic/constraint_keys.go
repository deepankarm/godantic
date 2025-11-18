package godantic

// Constraint keys used in the Constraints_ map
// These are used for both validation and JSON Schema generation
const (
	// Schema metadata
	ConstraintDescription = "description"
	ConstraintTitle       = "title"
	ConstraintExample     = "example"
	ConstraintFormat      = "format"
	ConstraintReadOnly    = "readOnly"
	ConstraintWriteOnly   = "writeOnly"
	ConstraintDeprecated  = "deprecated"
	ConstraintDefault     = "default"
	ConstraintConst       = "const"

	// Numeric constraints
	ConstraintMinimum          = "minimum"
	ConstraintMaximum          = "maximum"
	ConstraintExclusiveMinimum = "exclusiveMinimum"
	ConstraintExclusiveMaximum = "exclusiveMaximum"
	ConstraintMultipleOf       = "multipleOf"

	// String constraints
	ConstraintMinLength        = "minLength"
	ConstraintMaxLength        = "maxLength"
	ConstraintPattern          = "pattern"
	ConstraintContentEncoding  = "contentEncoding"
	ConstraintContentMediaType = "contentMediaType"

	// Array constraints
	ConstraintMinItems    = "minItems"
	ConstraintMaxItems    = "maxItems"
	ConstraintUniqueItems = "uniqueItems"

	// Object/Map constraints
	ConstraintMinProperties = "minProperties"
	ConstraintMaxProperties = "maxProperties"

	// Value constraints
	ConstraintEnum = "enum"

	// Union constraints
	ConstraintAnyOf         = "anyOf"
	ConstraintDiscriminator = "discriminator"
)
