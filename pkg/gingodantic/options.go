package gingodantic

import (
	"reflect"

	"github.com/deepankarm/godantic/pkg/godantic"
)

// SchemaOption configures an endpoint schema
type SchemaOption func(*EndpointSpec)

// WithSummary sets the endpoint summary
func WithSummary(s string) SchemaOption {
	return func(spec *EndpointSpec) {
		spec.Summary = s
	}
}

// WithDescription sets the endpoint description
func WithDescription(d string) SchemaOption {
	return func(spec *EndpointSpec) {
		spec.Description = d
	}
}

// WithTags adds tags to the endpoint
func WithTags(tags ...string) SchemaOption {
	return func(spec *EndpointSpec) {
		spec.Tags = append(spec.Tags, tags...)
	}
}

// WithRequest specifies the request body type and creates a validator for it
func WithRequest[T any]() SchemaOption {
	var zero T
	validator := godantic.NewValidator[T]()

	return func(spec *EndpointSpec) {
		spec.RequestType = reflect.TypeOf(zero)
		spec.validators.request = func(data []byte) (any, godantic.ValidationErrors) {
			obj, errs := validator.Marshal(data)
			if errs != nil {
				return nil, errs
			}
			return obj, nil
		}
	}
}

// WithResponse specifies a response type with status code
func WithResponse[T any](statusCode int, description ...string) SchemaOption {
	var zero T
	desc := ""
	if len(description) > 0 {
		desc = description[0]
	}
	return func(spec *EndpointSpec) {
		if spec.Responses == nil {
			spec.Responses = make(map[int]ResponseSpec)
		}
		spec.Responses[statusCode] = ResponseSpec{
			Type:        reflect.TypeOf(zero),
			Description: desc,
		}
	}
}

// WithSkipValidation disables godantic validation for this endpoint
// By default, validation is enabled when a Request type is specified
func WithSkipValidation() SchemaOption {
	return func(spec *EndpointSpec) {
		spec.SkipValidation = true
	}
}

// WithDeprecated marks the endpoint as deprecated
func WithDeprecated() SchemaOption {
	return func(spec *EndpointSpec) {
		spec.Deprecated = true
	}
}

// WithRequestExamples adds examples for the request body
func WithRequestExamples(examples map[string]any) SchemaOption {
	return func(spec *EndpointSpec) {
		spec.RequestExamples = examples
	}
}

// WithResponseExamples adds examples for a specific response status code
func WithResponseExamples(statusCode int, examples map[string]any) SchemaOption {
	return func(spec *EndpointSpec) {
		if spec.Responses == nil {
			spec.Responses = make(map[int]ResponseSpec)
		}
		resp := spec.Responses[statusCode]
		resp.Examples = examples
		spec.Responses[statusCode] = resp
	}
}

// WithPathParams specifies path parameter types and creates a validator for them
func WithPathParams[T any]() SchemaOption {
	var zero T
	validator := godantic.NewValidator[T]()

	return func(spec *EndpointSpec) {
		spec.ParamTypes.Path = reflect.TypeOf(zero)
		spec.validators.path = func(pathParams map[string]string) (any, godantic.ValidationErrors) {
			return validator.ValidateFromStringMap(pathParams)
		}
	}
}

// WithHeaderParams specifies header parameter types and creates a validator for them
func WithHeaderParams[T any]() SchemaOption {
	var zero T
	validator := godantic.NewValidator[T]()

	return func(spec *EndpointSpec) {
		spec.ParamTypes.Header = reflect.TypeOf(zero)
		spec.validators.header = func(headerParams map[string][]string) (any, godantic.ValidationErrors) {
			return validator.ValidateFromMultiValueMap(headerParams)
		}
	}
}

// WithCookieParams specifies cookie parameter types and creates a validator for them
func WithCookieParams[T any]() SchemaOption {
	var zero T
	validator := godantic.NewValidator[T]()

	return func(spec *EndpointSpec) {
		spec.ParamTypes.Cookie = reflect.TypeOf(zero)
		spec.validators.cookie = func(cookieParams map[string]string) (any, godantic.ValidationErrors) {
			return validator.ValidateFromStringMap(cookieParams)
		}
	}
}

// WithQueryParams specifies the query parameter type and creates a validator for it
func WithQueryParams[T any]() SchemaOption {
	var zero T
	validator := godantic.NewValidator[T]()

	return func(spec *EndpointSpec) {
		spec.ParamTypes.Query = reflect.TypeOf(zero)
		spec.validators.query = func(queryParams map[string][]string) (any, godantic.ValidationErrors) {
			return validator.ValidateFromMultiValueMap(queryParams)
		}
	}
}
