package gingodantic

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/deepankarm/godantic/pkg/godantic/schema"
	"github.com/deepankarm/godantic/pkg/internal/reflectutil"
	"github.com/gin-gonic/gin"
)

// API holds the OpenAPI specification
type API struct {
	mu        sync.RWMutex
	endpoints map[string]*EndpointSpec // key: "METHOD /path"
	info      APIInfo
}

type APIInfo struct {
	Title       string
	Version     string
	Description string
}

type EndpointSpec struct {
	Method         string
	Path           string
	Summary        string
	Description    string
	Tags           []string
	Deprecated     bool
	SkipValidation bool

	// Type information for schema generation
	RequestType     reflect.Type
	ParamTypes      ParamTypes
	Responses       map[int]ResponseSpec
	RequestExamples map[string]any

	// Internal validation functions
	validators validators
}

// ParamTypes holds all parameter type information
type ParamTypes struct {
	Query  reflect.Type
	Path   reflect.Type
	Header reflect.Type
	Cookie reflect.Type
}

// validators holds validation functions for different parameter types
type validators struct {
	request func([]byte) (any, godantic.ValidationErrors)
	query   func(map[string][]string) (any, godantic.ValidationErrors)
	path    func(map[string]string) (any, godantic.ValidationErrors)
	header  func(map[string][]string) (any, godantic.ValidationErrors)
	cookie  func(map[string]string) (any, godantic.ValidationErrors)
}

type ResponseSpec struct {
	Type        reflect.Type
	Description string
	Examples    map[string]any // key: example name
}

// New creates a new API instance
func New(title, version string) *API {
	return &API{
		endpoints: make(map[string]*EndpointSpec),
		info: APIInfo{
			Title:   title,
			Version: version,
		},
	}
}

// OpenAPISchema creates a middleware that registers endpoint schema and optionally validates
func (api *API) OpenAPISchema(method, path string, opts ...SchemaOption) gin.HandlerFunc {
	spec := &EndpointSpec{
		Method:    method,
		Path:      path,
		Responses: make(map[int]ResponseSpec),
	}

	for _, opt := range opts {
		opt(spec)
	}

	// Register the schema
	key := method + " " + path
	api.mu.Lock()
	api.endpoints[key] = spec
	api.mu.Unlock()

	// Return middleware that validates all parameters
	return func(c *gin.Context) {
		if spec.SkipValidation {
			c.Next()
			return
		}

		// Validate path parameters
		if spec.validators.path != nil {
			pathParams := make(map[string]string)
			for _, param := range c.Params {
				pathParams[param.Key] = param.Value
			}
			validated, errs := spec.validators.path(pathParams)
			if !validateAndStore(c, "validated_path", validated, errs) {
				return
			}
		}

		// Validate header parameters
		if spec.validators.header != nil {
			validated, errs := spec.validators.header(c.Request.Header)
			if !validateAndStore(c, "validated_headers", validated, errs) {
				return
			}
		}

		// Validate cookie parameters
		if spec.validators.cookie != nil {
			cookieParams := make(map[string]string)
			for _, cookie := range c.Request.Cookies() {
				cookieParams[cookie.Name] = cookie.Value
			}
			validated, errs := spec.validators.cookie(cookieParams)
			if !validateAndStore(c, "validated_cookies", validated, errs) {
				return
			}
		}

		// Validate query parameters
		if spec.validators.query != nil {
			validated, errs := spec.validators.query(c.Request.URL.Query())
			if !validateAndStore(c, "validated_query", validated, errs) {
				return
			}
		}

		// Validate request body
		if spec.validators.request != nil {
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
				c.Abort()
				return
			}
			validated, errs := spec.validators.request(body)
			if !validateAndStore(c, "validated_request", validated, errs) {
				return
			}
		}

		c.Next()
	}
}

// validateAndStore is a helper that validates data and stores it in context
// Returns false if validation failed (and has already sent error response)
func validateAndStore(c *gin.Context, contextKey string, validated any, validationErrs godantic.ValidationErrors) bool {
	if validationErrs != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation failed",
			"details": validationErrs,
		})
		c.Abort()
		return false
	}
	c.Set(contextKey, validated)
	return true
}

// GetValidated retrieves validated request data from context
// Use this in your handlers to get the validated and unmarshaled request
func GetValidated[T any](c *gin.Context) (*T, bool) {
	val, exists := c.Get("validated_request")
	if !exists {
		return nil, false
	}
	typed, ok := val.(*T)
	return typed, ok
}

// GetValidatedQuery retrieves validated query parameters from context
// Use this in your handlers to get the validated query params
func GetValidatedQuery[T any](c *gin.Context) (*T, bool) {
	val, exists := c.Get("validated_query")
	if !exists {
		return nil, false
	}
	typed, ok := val.(*T)
	return typed, ok
}

// GetValidatedPath retrieves validated path parameters from context
// Use this in your handlers to get the validated path params
func GetValidatedPath[T any](c *gin.Context) (*T, bool) {
	val, exists := c.Get("validated_path")
	if !exists {
		return nil, false
	}
	typed, ok := val.(*T)
	return typed, ok
}

// GetValidatedHeaders retrieves validated header parameters from context
// Use this in your handlers to get the validated headers
func GetValidatedHeaders[T any](c *gin.Context) (*T, bool) {
	val, exists := c.Get("validated_headers")
	if !exists {
		return nil, false
	}
	typed, ok := val.(*T)
	return typed, ok
}

// GetValidatedCookies retrieves validated cookie parameters from context
// Use this in your handlers to get the validated cookies
func GetValidatedCookies[T any](c *gin.Context) (*T, bool) {
	val, exists := c.Get("validated_cookies")
	if !exists {
		return nil, false
	}
	typed, ok := val.(*T)
	return typed, ok
}

// OpenAPIHandler returns a handler that serves the OpenAPI spec
func (api *API) OpenAPIHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		spec := api.GenerateOpenAPI()
		c.JSON(http.StatusOK, spec)
	}
}

// GenerateOpenAPI generates the OpenAPI 3.0 specification
func (api *API) GenerateOpenAPI() map[string]any {
	api.mu.RLock()
	defer api.mu.RUnlock()

	paths := make(map[string]any)
	components := map[string]any{
		"schemas": make(map[string]any),
	}

	for _, endpoint := range api.endpoints {
		openAPIPath := ConvertGinPathToOpenAPI(endpoint.Path)

		pathItem := paths[openAPIPath]
		if pathItem == nil {
			pathItem = make(map[string]any)
			paths[openAPIPath] = pathItem
		}

		operation := api.buildOperation(endpoint, openAPIPath, components)
		method := strings.ToLower(endpoint.Method)
		pathItem.(map[string]any)[method] = operation
	}

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       api.info.Title,
			"version":     api.info.Version,
			"description": api.info.Description,
		},
		"paths":      paths,
		"components": components,
	}
}

// buildOperation creates an OpenAPI operation object for an endpoint
func (api *API) buildOperation(endpoint *EndpointSpec, openAPIPath string, components map[string]any) map[string]any {
	operation := make(map[string]any)

	// Add basic metadata
	if endpoint.Summary != "" {
		operation["summary"] = endpoint.Summary
	}
	if endpoint.Description != "" {
		operation["description"] = endpoint.Description
	}
	if len(endpoint.Tags) > 0 {
		operation["tags"] = endpoint.Tags
	}
	if endpoint.Deprecated {
		operation["deprecated"] = true
	}

	// Add parameters
	if params := api.collectParameters(endpoint, openAPIPath); len(params) > 0 {
		operation["parameters"] = params
	}

	// Add request body
	if requestBody := api.buildRequestBody(endpoint, components); requestBody != nil {
		operation["requestBody"] = requestBody
	}

	// Add responses
	operation["responses"] = api.buildResponses(endpoint, components)

	return operation
}

// collectParameters gathers all parameters (path, header, cookie, query) for an endpoint
func (api *API) collectParameters(endpoint *EndpointSpec, openAPIPath string) []any {
	parameters := make([]any, 0)

	// Add path parameters
	pathParamNames := ExtractPathParameters(openAPIPath)
	if endpoint.ParamTypes.Path != nil {
		pathParams := extractParametersFromType(endpoint.ParamTypes.Path, "path", pathParamNames)
		parameters = append(parameters, pathParams...)
	} else {
		for _, paramName := range pathParamNames {
			parameters = append(parameters, map[string]any{
				"name":        paramName,
				"in":          "path",
				"required":    true,
				"schema":      map[string]any{"type": "string"},
				"description": paramName + " parameter",
			})
		}
	}

	// Add header, cookie, and query parameters
	if endpoint.ParamTypes.Header != nil {
		parameters = append(parameters, extractParametersFromType(endpoint.ParamTypes.Header, "header", nil)...)
	}
	if endpoint.ParamTypes.Cookie != nil {
		parameters = append(parameters, extractParametersFromType(endpoint.ParamTypes.Cookie, "cookie", nil)...)
	}
	if endpoint.ParamTypes.Query != nil {
		parameters = append(parameters, extractParametersFromType(endpoint.ParamTypes.Query, "query", nil)...)
	}

	return parameters
}

// buildRequestBody creates the request body object for an endpoint
func (api *API) buildRequestBody(endpoint *EndpointSpec, components map[string]any) map[string]any {
	if endpoint.RequestType == nil {
		return nil
	}

	flattenedSchema, err := generateSchemaFromType(endpoint.RequestType)
	if err != nil {
		return nil
	}

	// Extract and store schema definitions
	if defs, ok := flattenedSchema["$defs"].(map[string]any); ok {
		for name, def := range defs {
			components["schemas"].(map[string]any)[name] = FixSchemaRefs(def)
		}
	}

	content := map[string]any{
		"schema": removeDefsFromSchema(flattenedSchema),
	}
	if len(endpoint.RequestExamples) > 0 {
		content["examples"] = endpoint.RequestExamples
	}

	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": content,
		},
	}
}

// buildResponses creates the responses object for an endpoint
func (api *API) buildResponses(endpoint *EndpointSpec, components map[string]any) map[string]any {
	responses := make(map[string]any)

	for statusCode, resp := range endpoint.Responses {
		flattenedSchema, err := generateSchemaFromType(resp.Type)
		if err != nil {
			continue
		}

		// Extract and store schema definitions
		if defs, ok := flattenedSchema["$defs"].(map[string]any); ok {
			for name, def := range defs {
				components["schemas"].(map[string]any)[name] = FixSchemaRefs(def)
			}
		}

		content := map[string]any{
			"schema": removeDefsFromSchema(flattenedSchema),
		}
		if len(resp.Examples) > 0 {
			content["examples"] = resp.Examples
		}

		responses[strconv.Itoa(statusCode)] = map[string]any{
			"description": resp.Description,
			"content": map[string]any{
				"application/json": content,
			},
		}
	}

	return responses
}

// removeDefsFromSchema removes $defs from a schema since we move them to components
func removeDefsFromSchema(s map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range s {
		if k != "$defs" {
			result[k] = v
		}
	}
	return result
}

// MarshalOpenAPI returns the OpenAPI spec as JSON bytes
func (api *API) MarshalOpenAPI() ([]byte, error) {
	spec := api.GenerateOpenAPI()
	return json.MarshalIndent(spec, "", "  ")
}

// ConvertGinPathToOpenAPI converts Gin path format to OpenAPI format
// e.g., /users/:id -> /users/{id}
func ConvertGinPathToOpenAPI(ginPath string) string {
	result := ginPath
	// Replace :param with {param}
	for {
		start := strings.Index(result, ":")
		if start == -1 {
			break
		}
		// Find the end of the parameter (next / or end of string)
		end := start + 1
		for end < len(result) && result[end] != '/' {
			end++
		}
		paramName := result[start+1 : end]
		result = result[:start] + "{" + paramName + "}" + result[end:]
	}
	return result
}

// ExtractPathParameters extracts parameter names from an OpenAPI path
// e.g., /users/{id}/posts/{postId} -> ["id", "postId"]
func ExtractPathParameters(path string) []string {
	var params []string
	for {
		start := strings.Index(path, "{")
		if start == -1 {
			break
		}
		end := strings.Index(path[start:], "}")
		if end == -1 {
			break
		}
		paramName := path[start+1 : start+end]
		params = append(params, paramName)
		path = path[start+end+1:]
	}
	return params
}

// extractParametersFromType extracts parameter definitions from a type for any parameter location
// paramLocation: "path", "query", "header", or "cookie"
// paramNames: for path params, the list of param names from the URL pattern (to determine required status)
func extractParametersFromType(t reflect.Type, paramLocation string, paramNames []string) []any {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil
	}

	// Use godantic to scan field options
	fieldOptions := godantic.ScanTypeFieldOptions(t)
	params := make([]any, 0)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		jsonFieldName := strings.Split(jsonTag, ",")[0]
		fieldOpts, hasOpts := fieldOptions[field.Name]

		if len(paramNames) > 0 && !slices.Contains(paramNames, jsonFieldName) {
			continue
		}

		paramSchema := map[string]any{
			"type": reflectutil.JSONSchemaType(field.Type),
		}

		required := false
		if hasOpts {
			applyConstraintsToParamSchema(paramSchema, fieldOpts.Constraints)
			required = fieldOpts.Required
		}

		param := map[string]any{
			"name":   jsonFieldName,
			"in":     paramLocation,
			"schema": paramSchema,
		}

		if paramLocation == "path" || required {
			param["required"] = true
		}

		if hasOpts {
			if desc, ok := fieldOpts.Constraints["description"].(string); ok {
				param["description"] = desc
			}
		}

		params = append(params, param)
	}

	return params
}

// applyConstraintsToParamSchema applies godantic constraints to an OpenAPI parameter schema
func applyConstraintsToParamSchema(paramSchema map[string]any, constraints map[string]any) {
	constraintMap := map[string]string{
		"default":   "default",
		"minimum":   "minimum",
		"maximum":   "maximum",
		"minLength": "minLength",
		"maxLength": "maxLength",
		"pattern":   "pattern",
		"enum":      "enum",
	}

	for godanticKey, openAPIKey := range constraintMap {
		if val, ok := constraints[godanticKey]; ok {
			paramSchema[openAPIKey] = val
		}
	}
}

// FixSchemaRefs recursively fixes $ref paths and removes $schema property
func FixSchemaRefs(data any) any {
	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any)
		for key, value := range v {
			if key == "$schema" {
				continue
			}
			// Fix $ref paths
			if key == "$ref" {
				if refStr, ok := value.(string); ok {
					// Convert #/$defs/TypeName to #/components/schemas/TypeName
					if strings.HasPrefix(refStr, "#/$defs/") {
						refStr = "#/components/schemas/" + refStr[len("#/$defs/"):]
						result[key] = refStr
						continue
					}
				}
			}
			result[key] = FixSchemaRefs(value)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = FixSchemaRefs(item)
		}
		return result
	default:
		return v
	}
}

// generateSchemaFromType generates a JSON schema from a reflect.Type
// Uses godantic's schema package which includes validation metadata
func generateSchemaFromType(t reflect.Type) (map[string]any, error) {
	schemaMap, err := schema.GenerateForType(t)
	if err != nil {
		return nil, err
	}

	// Fix $ref paths and remove $schema for OpenAPI compatibility
	fixed := FixSchemaRefs(schemaMap)
	if fixedMap, ok := fixed.(map[string]any); ok {
		return fixedMap, nil
	}

	return schemaMap, nil
}
