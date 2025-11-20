package gingodantic_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deepankarm/godantic/pkg/gingodantic"
	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/gin-gonic/gin"
)

type TestRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func (r *TestRequest) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(3),
		godantic.Description[string]("User name"),
	)
}

func (r *TestRequest) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Email(),
		godantic.Description[string]("User email"),
	)
}

func (r *TestRequest) FieldAge() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Min(18),
		godantic.Max(120),
		godantic.Description[int]("User age"),
	)
}

type TestResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type TestErrorResponse struct {
	Error   string `json:"error"`
	Details any    `json:"details,omitempty"`
}

func TestOpenAPIGeneration(t *testing.T) {
	api := gingodantic.New("Test API", "1.0.0")

	// Register endpoint
	api.OpenAPISchema("POST", "/users",
		gingodantic.WithSummary("Create user"),
		gingodantic.WithDescription("Creates a new user"),
		gingodantic.WithTags("users"),
		gingodantic.WithRequest[TestRequest](),
		gingodantic.WithResponse[TestResponse](201, "Created"),
		gingodantic.WithResponse[TestErrorResponse](400, "Bad request"),
	)

	spec := api.GenerateOpenAPI()

	// Check basic info
	if spec["openapi"] != "3.0.3" {
		t.Errorf("Expected OpenAPI 3.0.3, got %v", spec["openapi"])
	}

	info := spec["info"].(map[string]any)
	if info["title"] != "Test API" {
		t.Errorf("Expected title 'Test API', got %v", info["title"])
	}

	// Check paths
	paths := spec["paths"].(map[string]any)
	if _, ok := paths["/users"]; !ok {
		t.Error("Expected /users path")
	}

	usersPath := paths["/users"].(map[string]any)
	postOp := usersPath["post"].(map[string]any)

	// Check operation details
	if postOp["summary"] != "Create user" {
		t.Errorf("Expected summary 'Create user', got %v", postOp["summary"])
	}

	tags := postOp["tags"].([]string)
	if len(tags) != 1 || tags[0] != "users" {
		t.Errorf("Expected tags ['users'], got %v", tags)
	}

	// Check request body
	requestBody := postOp["requestBody"].(map[string]any)
	if requestBody["required"] != true {
		t.Error("Expected requestBody to be required")
	}

	// Check responses
	responses := postOp["responses"].(map[string]any)
	if _, ok := responses["201"]; !ok {
		t.Error("Expected 201 response")
	}
	if _, ok := responses["400"]; !ok {
		t.Error("Expected 400 response")
	}

	// Check components/schemas
	components := spec["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	if _, ok := schemas["TestRequest"]; !ok {
		t.Error("Expected TestRequest in schemas")
	}
	if _, ok := schemas["TestResponse"]; !ok {
		t.Error("Expected TestResponse in schemas")
	}
}

func TestPathParameterExtraction(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "single parameter",
			path:     "/users/{id}",
			expected: []string{"id"},
		},
		{
			name:     "multiple parameters",
			path:     "/users/{userId}/posts/{postId}",
			expected: []string{"userId", "postId"},
		},
		{
			name:     "no parameters",
			path:     "/users",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := gingodantic.ExtractPathParameters(tc.path)
			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d params, got %d", len(tc.expected), len(result))
				return
			}
			for i, param := range result {
				if param != tc.expected[i] {
					t.Errorf("Expected param '%s', got '%s'", tc.expected[i], param)
				}
			}
		})
	}
}

func TestGinPathConversion(t *testing.T) {
	testCases := []struct {
		name     string
		ginPath  string
		expected string
	}{
		{
			name:     "single parameter",
			ginPath:  "/users/:id",
			expected: "/users/{id}",
		},
		{
			name:     "multiple parameters",
			ginPath:  "/users/:userId/posts/:postId",
			expected: "/users/{userId}/posts/{postId}",
		},
		{
			name:     "no parameters",
			ginPath:  "/users",
			expected: "/users",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := gingodantic.ConvertGinPathToOpenAPI(tc.ginPath)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestPathParametersInSpec(t *testing.T) {
	api := gingodantic.New("Test API", "1.0.0")

	api.OpenAPISchema("GET", "/users/:id",
		gingodantic.WithSummary("Get user by ID"),
		gingodantic.WithResponse[TestResponse](200, "OK"),
	)

	spec := api.GenerateOpenAPI()
	paths := spec["paths"].(map[string]any)

	// Should convert :id to {id}
	userPath := paths["/users/{id}"].(map[string]any)
	getOp := userPath["get"].(map[string]any)

	// Check parameters
	params := getOp["parameters"].([]any)
	if len(params) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(params))
	}

	param := params[0].(map[string]any)
	if param["name"] != "id" {
		t.Errorf("Expected parameter name 'id', got %v", param["name"])
	}
	if param["in"] != "path" {
		t.Errorf("Expected parameter in 'path', got %v", param["in"])
	}
	if param["required"] != true {
		t.Error("Expected parameter to be required")
	}
}

func TestValidationEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := gingodantic.New("Test API", "1.0.0")

	var receivedReq *TestRequest

	router.POST("/users",
		api.OpenAPISchema("POST", "/users",
			gingodantic.WithRequest[TestRequest](),
		),
		func(c *gin.Context) {
			req, ok := gingodantic.GetValidated[TestRequest](c)
			if !ok {
				c.JSON(500, gin.H{"error": "failed to get validated request"})
				return
			}
			receivedReq = req
			c.JSON(200, gin.H{"success": true})
		},
	)

	t.Run("valid request passes", func(t *testing.T) {
		body := `{"name":"John Doe","email":"john@example.com","age":25}`
		req := httptest.NewRequest("POST", "/users", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		if receivedReq == nil {
			t.Fatal("Expected request to be set")
		}
		if receivedReq.Name != "John Doe" {
			t.Errorf("Expected name 'John Doe', got '%s'", receivedReq.Name)
		}
	})

	t.Run("missing required field fails", func(t *testing.T) {
		body := `{"email":"john@example.com","age":25}`
		req := httptest.NewRequest("POST", "/users", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["error"] != "validation failed" {
			t.Errorf("Expected validation error, got %v", resp)
		}
	})

	t.Run("constraint violation fails", func(t *testing.T) {
		body := `{"name":"Jo","email":"invalid","age":15}`
		req := httptest.NewRequest("POST", "/users", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)

		details := resp["details"].([]any)
		if len(details) != 3 {
			t.Errorf("Expected exactly 3 validation errors, got %d", len(details))
		}

		// Check for specific validation errors
		errorFields := make(map[string]bool)
		for _, detail := range details {
			errMap := detail.(map[string]any)
			loc := errMap["Loc"].([]any)
			if len(loc) > 0 {
				fieldName := loc[0].(string)
				errorFields[fieldName] = true
			}
		}

		// Verify errors for name, email, and age
		expectedFields := []string{"Name", "Email", "Age"}
		for _, field := range expectedFields {
			if !errorFields[field] {
				t.Errorf("Expected validation error for field %s, but didn't find it", field)
			}
		}
	})
}

func TestValidationDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := gingodantic.New("Test API", "1.0.0")

	router.POST("/webhook",
		api.OpenAPISchema("POST", "/webhook",
			gingodantic.WithRequest[TestRequest](),
			gingodantic.WithSkipValidation(), // Disable validation
		),
		func(c *gin.Context) {
			// Should reach here even with invalid data
			c.JSON(200, gin.H{"success": true})
		},
	)

	body := `{"invalid":"data"}` // Invalid according to TestRequest
	req := httptest.NewRequest("POST", "/webhook", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should succeed because validation is disabled
	if w.Code != 200 {
		t.Errorf("Expected status 200 (validation disabled), got %d", w.Code)
	}
}

func TestGetValidated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	t.Run("returns false when not set", func(t *testing.T) {
		_, ok := gingodantic.GetValidated[TestRequest](c)
		if ok {
			t.Error("Expected GetValidated to return false when not set")
		}
	})

	t.Run("returns value when set", func(t *testing.T) {
		expected := &TestRequest{Name: "John", Email: "john@example.com", Age: 25}
		c.Set("validated_request", expected)

		result, ok := gingodantic.GetValidated[TestRequest](c)
		if !ok {
			t.Fatal("Expected GetValidated to return true")
		}
		if result != expected {
			t.Error("Expected to get the same pointer")
		}
	})
}

func TestSchemaRefFixes(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name: "fixes $ref paths",
			input: map[string]any{
				"$ref": "#/$defs/MyType",
			},
			expected: map[string]any{
				"$ref": "#/components/schemas/MyType",
			},
		},
		{
			name: "removes $schema property",
			input: map[string]any{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type":    "object",
			},
			expected: map[string]any{
				"type": "object",
			},
		},
		{
			name: "recursive fixing",
			input: map[string]any{
				"properties": map[string]any{
					"items": map[string]any{
						"$ref": "#/$defs/Item",
					},
				},
			},
			expected: map[string]any{
				"properties": map[string]any{
					"items": map[string]any{
						"$ref": "#/components/schemas/Item",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := gingodantic.FixSchemaRefs(tc.input)
			resultJSON, _ := json.Marshal(result)
			expectedJSON, _ := json.Marshal(tc.expected)

			if string(resultJSON) != string(expectedJSON) {
				t.Errorf("Expected %s, got %s", expectedJSON, resultJSON)
			}
		})
	}
}

func TestMultipleEndpoints(t *testing.T) {
	api := gingodantic.New("Test API", "1.0.0")

	api.OpenAPISchema("GET", "/users", gingodantic.WithSummary("List users"))
	api.OpenAPISchema("POST", "/users", gingodantic.WithSummary("Create user"))
	api.OpenAPISchema("GET", "/users/:id", gingodantic.WithSummary("Get user"))
	api.OpenAPISchema("PUT", "/users/:id", gingodantic.WithSummary("Update user"))
	api.OpenAPISchema("DELETE", "/users/:id", gingodantic.WithSummary("Delete user"))

	spec := api.GenerateOpenAPI()
	paths := spec["paths"].(map[string]any)

	if len(paths) != 2 { // /users and /users/{id}
		t.Errorf("Expected 2 paths, got %d", len(paths))
	}

	usersPath := paths["/users"].(map[string]any)
	if _, ok := usersPath["get"]; !ok {
		t.Error("Expected GET /users")
	}
	if _, ok := usersPath["post"]; !ok {
		t.Error("Expected POST /users")
	}

	userByIdPath := paths["/users/{id}"].(map[string]any)
	if _, ok := userByIdPath["get"]; !ok {
		t.Error("Expected GET /users/{id}")
	}
	if _, ok := userByIdPath["put"]; !ok {
		t.Error("Expected PUT /users/{id}")
	}
	if _, ok := userByIdPath["delete"]; !ok {
		t.Error("Expected DELETE /users/{id}")
	}
}

func TestHTTPMethodsLowercase(t *testing.T) {
	api := gingodantic.New("Test API", "1.0.0")

	api.OpenAPISchema("GET", "/users", gingodantic.WithSummary("Get users"))
	api.OpenAPISchema("POST", "/users", gingodantic.WithSummary("Create user"))
	api.OpenAPISchema("PUT", "/users/:id", gingodantic.WithSummary("Update user"))
	api.OpenAPISchema("DELETE", "/users/:id", gingodantic.WithSummary("Delete user"))

	spec := api.GenerateOpenAPI()
	paths := spec["paths"].(map[string]any)

	usersPath := paths["/users"].(map[string]any)

	// Methods should be lowercase
	if _, ok := usersPath["get"]; !ok {
		t.Error("Expected 'get' (lowercase)")
	}
	if _, ok := usersPath["post"]; !ok {
		t.Error("Expected 'post' (lowercase)")
	}

	// Should NOT have uppercase
	if _, ok := usersPath["GET"]; ok {
		t.Error("Should not have 'GET' (uppercase)")
	}
	if _, ok := usersPath["POST"]; ok {
		t.Error("Should not have 'POST' (uppercase)")
	}
}

// Query parameter tests
type TestListUsersQuery struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Search string `json:"search"`
}

func (q *TestListUsersQuery) FieldPage() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Default(1),
		godantic.Min(1),
		godantic.Description[int]("Page number"),
	)
}

func (q *TestListUsersQuery) FieldLimit() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Default(10),
		godantic.Min(1),
		godantic.Max(100),
		godantic.Description[int]("Items per page"),
	)
}

func (q *TestListUsersQuery) FieldSearch() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Description[string]("Search query"),
	)
}

func TestQueryParameters(t *testing.T) {
	api := gingodantic.New("Test API", "1.0.0")

	api.OpenAPISchema("GET", "/users",
		gingodantic.WithSummary("List users with filtering"),
		gingodantic.WithQueryParams[TestListUsersQuery](),
		gingodantic.WithResponse[TestResponse](200, "OK"),
	)

	spec := api.GenerateOpenAPI()
	paths := spec["paths"].(map[string]any)
	usersPath := paths["/users"].(map[string]any)
	getOp := usersPath["get"].(map[string]any)

	// Check parameters
	params, ok := getOp["parameters"].([]any)
	if !ok {
		t.Fatal("Expected parameters array")
	}

	if len(params) != 3 {
		t.Fatalf("Expected 3 query parameters, got %d", len(params))
	}

	// Verify each parameter
	paramMap := make(map[string]map[string]any)
	for _, p := range params {
		param := p.(map[string]any)
		name := param["name"].(string)
		paramMap[name] = param
	}

	// Check 'page' parameter
	page := paramMap["page"]
	if page["in"] != "query" {
		t.Errorf("Expected 'page' in 'query', got %v", page["in"])
	}
	if page["required"] == true {
		t.Error("Expected 'page' to be optional (has default)")
	}
	schema := page["schema"].(map[string]any)
	if schema["type"] != "integer" {
		t.Errorf("Expected 'page' type 'integer', got %v", schema["type"])
	}
	if schema["default"] != 1 {
		t.Errorf("Expected 'page' default 1, got %v", schema["default"])
	}

	// Check 'limit' parameter
	limit := paramMap["limit"]
	if limit["in"] != "query" {
		t.Errorf("Expected 'limit' in 'query', got %v", limit["in"])
	}
	limitSchema := limit["schema"].(map[string]any)
	if limitSchema["minimum"] != 1 {
		t.Errorf("Expected 'limit' minimum 1, got %v", limitSchema["minimum"])
	}
	if limitSchema["maximum"] != 100 {
		t.Errorf("Expected 'limit' maximum 100, got %v", limitSchema["maximum"])
	}

	// Check 'search' parameter
	search := paramMap["search"]
	if search["in"] != "query" {
		t.Errorf("Expected 'search' in 'query', got %v", search["in"])
	}
}

func TestPathAndQueryParameters(t *testing.T) {
	api := gingodantic.New("Test API", "1.0.0")

	api.OpenAPISchema("GET", "/users/:id/posts",
		gingodantic.WithSummary("Get user posts"),
		gingodantic.WithQueryParams[ListUsersQuery](),
		gingodantic.WithResponse[TestResponse](200, "OK"),
	)

	spec := api.GenerateOpenAPI()
	paths := spec["paths"].(map[string]any)
	postsPath := paths["/users/{id}/posts"].(map[string]any)
	getOp := postsPath["get"].(map[string]any)

	params := getOp["parameters"].([]any)

	// Should have 4 parameters: 1 path (id) + 3 query (page, limit, search)
	if len(params) != 4 {
		t.Fatalf("Expected 4 parameters (1 path + 3 query), got %d", len(params))
	}

	// Count by type
	pathCount := 0
	queryCount := 0
	for _, p := range params {
		param := p.(map[string]any)
		paramIn := param["in"].(string)
		switch paramIn {
		case "path":
			pathCount++
			// Path params should be required
			if param["required"] != true {
				t.Error("Expected path parameter to be required")
			}
		case "query":
			queryCount++
		}
	}

	if pathCount != 1 {
		t.Errorf("Expected 1 path parameter, got %d", pathCount)
	}
	if queryCount != 3 {
		t.Errorf("Expected 3 query parameters, got %d", queryCount)
	}
}

func TestQueryParameterValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := gingodantic.New("Test API", "1.0.0")

	var receivedQuery *TestListUsersQuery

	router.GET("/users",
		api.OpenAPISchema("GET", "/users",
			gingodantic.WithQueryParams[TestListUsersQuery](),
		),
		func(c *gin.Context) {
			query, ok := gingodantic.GetValidatedQuery[TestListUsersQuery](c)
			if !ok {
				c.JSON(500, gin.H{"error": "failed to get validated query"})
				return
			}
			receivedQuery = query
			c.JSON(200, gin.H{"success": true})
		},
	)

	t.Run("valid query parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users?page=2&limit=20&search=john", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		if receivedQuery == nil {
			t.Fatal("Expected query to be set")
		}
		if receivedQuery.Page != 2 {
			t.Errorf("Expected page 2, got %d", receivedQuery.Page)
		}
		if receivedQuery.Limit != 20 {
			t.Errorf("Expected limit 20, got %d", receivedQuery.Limit)
		}
		if receivedQuery.Search != "john" {
			t.Errorf("Expected search 'john', got '%s'", receivedQuery.Search)
		}
	})

	t.Run("defaults applied when missing", func(t *testing.T) {
		receivedQuery = nil
		req := httptest.NewRequest("GET", "/users", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		if receivedQuery == nil {
			t.Fatal("Expected query to be set")
		}
		if receivedQuery.Page != 1 {
			t.Errorf("Expected page default 1, got %d", receivedQuery.Page)
		}
		if receivedQuery.Limit != 10 {
			t.Errorf("Expected limit default 10, got %d", receivedQuery.Limit)
		}
	})

	t.Run("constraint violations", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users?page=0&limit=200", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Errorf("Expected status 400 for constraint violations, got %d", w.Code)
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["error"] != "validation failed" {
			t.Errorf("Expected validation error, got %v", resp)
		}
	})
}

// Test types for path parameters
type TestUserPathParams struct {
	ID int `json:"id"`
}

func (p *TestUserPathParams) FieldID() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.Min(1),
		godantic.Description[int]("User ID"),
	)
}

func TestPathParameterValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := gingodantic.New("Test API", "1.0.0")

	var receivedPath *TestUserPathParams

	router.GET("/users/:id",
		api.OpenAPISchema("GET", "/users/:id",
			gingodantic.WithPathParams[TestUserPathParams](),
		),
		func(c *gin.Context) {
			path, ok := gingodantic.GetValidatedPath[TestUserPathParams](c)
			if !ok {
				c.JSON(500, gin.H{"error": "failed to get validated path"})
				return
			}
			receivedPath = path
			c.JSON(200, gin.H{"id": path.ID})
		},
	)

	t.Run("valid path param", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		if receivedPath == nil {
			t.Fatal("Expected path to be set")
		}
		if receivedPath.ID != 123 {
			t.Errorf("Expected ID 123, got %d", receivedPath.ID)
		}
	})

	t.Run("invalid path param - constraint violation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/0", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Errorf("Expected status 400 for constraint violation, got %d", w.Code)
		}
	})
}

// Test types for header parameters
type TestAuthHeaders struct {
	Authorization string `json:"Authorization"`
	UserAgent     string `json:"User-Agent"`
}

func (h *TestAuthHeaders) FieldAuthorization() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Regex(`^Bearer .+`),
		godantic.Description[string]("Bearer token"),
	)
}

func TestHeaderParameterValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := gingodantic.New("Test API", "1.0.0")

	var receivedHeaders *TestAuthHeaders

	router.GET("/protected",
		api.OpenAPISchema("GET", "/protected",
			gingodantic.WithHeaderParams[TestAuthHeaders](),
		),
		func(c *gin.Context) {
			headers, ok := gingodantic.GetValidatedHeaders[TestAuthHeaders](c)
			if !ok {
				c.JSON(500, gin.H{"error": "failed to get validated headers"})
				return
			}
			receivedHeaders = headers
			c.JSON(200, gin.H{"success": true})
		},
	)

	t.Run("valid headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer token123")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		if receivedHeaders == nil {
			t.Fatal("Expected headers to be set")
		}
		if receivedHeaders.Authorization != "Bearer token123" {
			t.Errorf("Expected Authorization 'Bearer token123', got %s", receivedHeaders.Authorization)
		}
	})

	t.Run("invalid headers - missing required", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Errorf("Expected status 400 for missing header, got %d", w.Code)
		}
	})
}

// Test types for cookie parameters
type SessionCookies struct {
	SessionID string `json:"session_id"`
}

func (c *SessionCookies) FieldSessionID() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(10),
		godantic.Description[string]("Session ID"),
	)
}

func TestCookieParameterValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := gingodantic.New("Test API", "1.0.0")

	var receivedCookies *SessionCookies

	router.GET("/dashboard",
		api.OpenAPISchema("GET", "/dashboard",
			gingodantic.WithCookieParams[SessionCookies](),
		),
		func(c *gin.Context) {
			cookies, ok := gingodantic.GetValidatedCookies[SessionCookies](c)
			if !ok {
				c.JSON(500, gin.H{"error": "failed to get validated cookies"})
				return
			}
			receivedCookies = cookies
			c.JSON(200, gin.H{"success": true})
		},
	)

	t.Run("valid cookies", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/dashboard", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc1234567"})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		if receivedCookies == nil {
			t.Fatal("Expected cookies to be set")
		}
		if receivedCookies.SessionID != "abc1234567" {
			t.Errorf("Expected SessionID 'abc1234567', got %s", receivedCookies.SessionID)
		}
	})

	t.Run("invalid cookies - too short", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/dashboard", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "short"})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != 400 {
			t.Errorf("Expected status 400 for constraint violation, got %d", w.Code)
		}
	})
}

func TestDeprecatedFlag(t *testing.T) {
	api := gingodantic.New("Test API", "1.0.0")

	api.OpenAPISchema("GET", "/old-endpoint",
		gingodantic.WithSummary("Old endpoint"),
		gingodantic.WithDeprecated(),
		gingodantic.WithResponse[TestResponse](200, "OK"),
	)

	spec := api.GenerateOpenAPI()
	paths := spec["paths"].(map[string]any)
	endpoint := paths["/old-endpoint"].(map[string]any)
	getOp := endpoint["get"].(map[string]any)

	if deprecated, ok := getOp["deprecated"].(bool); !ok || !deprecated {
		t.Error("Expected deprecated flag to be true")
	}
}

func TestRequestExamples(t *testing.T) {
	api := gingodantic.New("Test API", "1.0.0")

	examples := map[string]any{
		"example1": map[string]any{
			"value": map[string]any{
				"name":  "John",
				"email": "john@example.com",
				"age":   30,
			},
		},
	}

	api.OpenAPISchema("POST", "/users",
		gingodantic.WithRequest[TestRequest](),
		gingodantic.WithRequestExamples(examples),
		gingodantic.WithResponse[TestResponse](201, "Created"),
	)

	spec := api.GenerateOpenAPI()
	paths := spec["paths"].(map[string]any)
	endpoint := paths["/users"].(map[string]any)
	postOp := endpoint["post"].(map[string]any)
	requestBody := postOp["requestBody"].(map[string]any)
	content := requestBody["content"].(map[string]any)
	jsonContent := content["application/json"].(map[string]any)

	if _, ok := jsonContent["examples"]; !ok {
		t.Error("Expected examples in request body")
	}
}

func TestResponseExamples(t *testing.T) {
	api := gingodantic.New("Test API", "1.0.0")

	examples := map[string]any{
		"example1": map[string]any{
			"value": map[string]any{
				"id":    "123",
				"name":  "John",
				"email": "john@example.com",
			},
		},
	}

	api.OpenAPISchema("GET", "/users/123",
		gingodantic.WithResponse[TestResponse](200, "OK"),
		gingodantic.WithResponseExamples(200, examples),
	)

	spec := api.GenerateOpenAPI()
	paths := spec["paths"].(map[string]any)
	endpoint := paths["/users/123"].(map[string]any)
	getOp := endpoint["get"].(map[string]any)
	responses := getOp["responses"].(map[string]any)
	response200 := responses["200"].(map[string]any)
	content := response200["content"].(map[string]any)
	jsonContent := content["application/json"].(map[string]any)

	if _, ok := jsonContent["examples"]; !ok {
		t.Error("Expected examples in response")
	}
}

func TestPathParamsInOpenAPISpec(t *testing.T) {
	api := gingodantic.New("Test API", "1.0.0")

	api.OpenAPISchema("GET", "/users/:id",
		gingodantic.WithPathParams[UserPathParams](),
		gingodantic.WithResponse[TestResponse](200, "OK"),
	)

	spec := api.GenerateOpenAPI()
	paths := spec["paths"].(map[string]any)
	endpoint := paths["/users/{id}"].(map[string]any)
	getOp := endpoint["get"].(map[string]any)
	params := getOp["parameters"].([]any)

	// Should have 1 path parameter
	if len(params) != 1 {
		t.Fatalf("Expected 1 path parameter, got %d", len(params))
	}

	param := params[0].(map[string]any)
	if param["name"] != "id" {
		t.Errorf("Expected param name 'id', got %v", param["name"])
	}
	if param["in"] != "path" {
		t.Errorf("Expected param in 'path', got %v", param["in"])
	}
	if param["required"] != true {
		t.Error("Expected path param to be required")
	}

	schema := param["schema"].(map[string]any)
	if schema["type"] != "integer" {
		t.Errorf("Expected param type 'integer', got %v", schema["type"])
	}
	if schema["minimum"] != 1 {
		t.Errorf("Expected param minimum 1, got %v", schema["minimum"])
	}
}
