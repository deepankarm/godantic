package godantic

import "reflect"

// ValidatorOption configures a Validator with additional capabilities
type ValidatorOption interface {
	apply(*validatorConfig)
}

// validatorConfig holds configuration for a Validator
type validatorConfig struct {
	discriminator *discriminatorConfig
}

// discriminatorConfig holds configuration for discriminated union validation
type discriminatorConfig struct {
	field    string                  // The discriminator field name (e.g., "event", "type")
	variants map[string]reflect.Type // Map of discriminator value -> concrete type
}

// WithDiscriminator configures a validator to handle discriminated unions (interfaces).
// The field parameter is the name of the discriminator field.
// The variants map specifies which concrete type to use for each discriminator value.
//
// Example:
//
//	validator := godantic.NewValidator[Animal](
//	    godantic.WithDiscriminator("species", map[string]any{
//	        "cat":  Cat{},
//	        "dog":  Dog{},
//	        "bird": Bird{},
//	    }),
//	)
func WithDiscriminator(field string, variants map[string]any) ValidatorOption {
	return &discriminatorOption{
		field:    field,
		variants: variants,
	}
}

type discriminatorOption struct {
	field    string
	variants map[string]any
}

func (d *discriminatorOption) apply(cfg *validatorConfig) {
	// Convert variants map to map[string]reflect.Type
	typeMap := make(map[string]reflect.Type, len(d.variants))
	for key, val := range d.variants {
		typeMap[key] = reflect.TypeOf(val)
	}

	cfg.discriminator = &discriminatorConfig{
		field:    d.field,
		variants: typeMap,
	}
}

// WithDiscriminatorTyped is a type-safe variant that accepts typed discriminator keys
// This is useful when the discriminator is an enum type rather than a string.
//
// Example:
//
//	validator := godantic.NewValidator[ClientMessage](
//	    godantic.WithDiscriminatorTyped("event", map[ClientEvent]any{
//	        ClientEventConnect: ConnectClientMessage{},
//	        ClientEventNewQuery: NewQueryClientMessage{},
//	    }),
//	)
func WithDiscriminatorTyped[K ~string](field string, variants map[K]any) ValidatorOption {
	// Convert typed map to string map
	stringVariants := make(map[string]any, len(variants))
	for key, val := range variants {
		stringVariants[string(key)] = val
	}
	return WithDiscriminator(field, stringVariants)
}
