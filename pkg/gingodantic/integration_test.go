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

type AuthHeaders struct {
	Authorization string `json:"Authorization"`
	APIKey        string `json:"X-API-Key"`
}

func (AuthHeaders) FieldAuthorization() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Regex("^Bearer .+$"),
	)
}

func (AuthHeaders) FieldAPIKey() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(32),
	)
}

type ListUsersQuery struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Search string `json:"search"`
}

func (ListUsersQuery) FieldPage() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Min(0),
		godantic.Default(0),
	)
}

func (ListUsersQuery) FieldLimit() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Min(1),
		godantic.Max(100),
		godantic.Default(10),
	)
}

func (ListUsersQuery) FieldSearch() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.MaxLen(100),
	)
}

type UserResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type ListUsersResponse struct {
	Users []UserResponse `json:"users"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
}

type UserPathParams struct {
	ID int `json:"id"`
}

func (UserPathParams) FieldID() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.Min(1),
	)
}

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (CreateUserRequest) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(2),
		godantic.MaxLen(100),
	)
}

func (CreateUserRequest) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Email(),
	)
}

func (CreateUserRequest) FieldRole() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.OneOf("admin", "user", "guest"),
	)
}

type OrderPathParams struct {
	UserID  int `json:"user_id"`
	OrderID int `json:"order_id"`
}

func (OrderPathParams) FieldUserID() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.Min(1),
	)
}

func (OrderPathParams) FieldOrderID() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Required[int](),
		godantic.Min(1),
	)
}

type UpdateOrderRequest struct {
	Status   string  `json:"status"`
	Priority int     `json:"priority"`
	Total    float64 `json:"total"`
	Notes    string  `json:"notes"`
}

func (UpdateOrderRequest) FieldStatus() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.OneOf("pending", "processing", "shipped", "delivered", "cancelled"),
	)
}

func (UpdateOrderRequest) FieldPriority() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Min(1),
		godantic.Max(5),
		godantic.Default(3),
	)
}

func (UpdateOrderRequest) FieldTotal() godantic.FieldOptions[float64] {
	return godantic.Field(
		godantic.Required[float64](),
		godantic.Min(0.01),
	)
}

func (UpdateOrderRequest) FieldNotes() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.MaxLen(500),
	)
}

type OrderResponse struct {
	OrderID   int     `json:"order_id"`
	UserID    int     `json:"user_id"`
	Status    string  `json:"status"`
	Priority  int     `json:"priority"`
	Total     float64 `json:"total"`
	UpdatedBy string  `json:"updated_by"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func setupIntegrationRouter() (*gin.Engine, *gingodantic.API) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := gingodantic.New("E-Commerce API", "1.0.0")

	router.GET("/health",
		api.OpenAPISchema("GET", "/health",
			gingodantic.WithSummary("Health check"),
			gingodantic.WithTags("system"),
			gingodantic.WithResponse[HealthResponse](200, "Service is healthy"),
		),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, HealthResponse{
				Status:  "ok",
				Version: "1.0.0",
			})
		},
	)

	router.GET("/users",
		api.OpenAPISchema("GET", "/users",
			gingodantic.WithSummary("List users"),
			gingodantic.WithTags("users"),
			gingodantic.WithQueryParams[ListUsersQuery](),
			gingodantic.WithHeaderParams[AuthHeaders](),
			gingodantic.WithResponse[ListUsersResponse](200, "List of users"),
			gingodantic.WithResponse[ErrorResponse](401, "Unauthorized"),
		),
		func(c *gin.Context) {
			query, _ := gingodantic.GetValidatedQuery[ListUsersQuery](c)
			headers, _ := gingodantic.GetValidatedHeaders[AuthHeaders](c)

			if !(headers.Authorization != "" && headers.APIKey != "") {
				c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid credentials"})
				return
			}

			c.JSON(http.StatusOK, ListUsersResponse{
				Users: []UserResponse{
					{ID: 1, Name: "Alice", Email: "alice@example.com", Role: "admin"},
					{ID: 2, Name: "Bob", Email: "bob@example.com", Role: "user"},
				},
				Total: 42,
				Page:  query.Page,
			})
		},
	)

	router.GET("/users/:id",
		api.OpenAPISchema("GET", "/users/:id",
			gingodantic.WithSummary("Get user by ID"),
			gingodantic.WithTags("users"),
			gingodantic.WithPathParams[UserPathParams](),
			gingodantic.WithHeaderParams[AuthHeaders](),
			gingodantic.WithResponse[UserResponse](200, "User found"),
			gingodantic.WithResponse[ErrorResponse](401, "Unauthorized"),
			gingodantic.WithResponse[ErrorResponse](404, "User not found"),
		),
		func(c *gin.Context) {
			pathParams, _ := gingodantic.GetValidatedPath[UserPathParams](c)
			headers, _ := gingodantic.GetValidatedHeaders[AuthHeaders](c)

			if headers.APIKey == "" {
				c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Missing API key"})
				return
			}

			if pathParams.ID == 999 {
				c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
				return
			}

			c.JSON(http.StatusOK, UserResponse{
				ID:    pathParams.ID,
				Name:  "John Doe",
				Email: "john@example.com",
				Role:  "user",
			})
		},
	)

	router.POST("/users",
		api.OpenAPISchema("POST", "/users",
			gingodantic.WithSummary("Create user"),
			gingodantic.WithTags("users"),
			gingodantic.WithHeaderParams[AuthHeaders](),
			gingodantic.WithRequest[CreateUserRequest](),
			gingodantic.WithResponse[UserResponse](201, "User created"),
			gingodantic.WithResponse[ErrorResponse](400, "Invalid request"),
			gingodantic.WithResponse[ErrorResponse](401, "Unauthorized"),
		),
		func(c *gin.Context) {
			headers, _ := gingodantic.GetValidatedHeaders[AuthHeaders](c)
			body, _ := gingodantic.GetValidated[CreateUserRequest](c)

			if !isAdminToken(headers.Authorization) {
				c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Admin access required"})
				return
			}

			c.JSON(http.StatusCreated, UserResponse{
				ID:    123,
				Name:  body.Name,
				Email: body.Email,
				Role:  body.Role,
			})
		},
	)

	router.PUT("/users/:user_id/orders/:order_id",
		api.OpenAPISchema("PUT", "/users/:user_id/orders/:order_id",
			gingodantic.WithSummary("Update order"),
			gingodantic.WithTags("orders"),
			gingodantic.WithPathParams[OrderPathParams](),
			gingodantic.WithHeaderParams[AuthHeaders](),
			gingodantic.WithRequest[UpdateOrderRequest](),
			gingodantic.WithResponse[OrderResponse](200, "Order updated"),
			gingodantic.WithResponse[ErrorResponse](400, "Invalid request"),
			gingodantic.WithResponse[ErrorResponse](401, "Unauthorized"),
			gingodantic.WithResponse[ErrorResponse](404, "Order not found"),
		),
		func(c *gin.Context) {
			pathParams, _ := gingodantic.GetValidatedPath[OrderPathParams](c)
			headers, _ := gingodantic.GetValidatedHeaders[AuthHeaders](c)
			body, _ := gingodantic.GetValidated[UpdateOrderRequest](c)

			updatedBy := extractUserFromToken(headers.Authorization)

			c.JSON(http.StatusOK, OrderResponse{
				OrderID:   pathParams.OrderID,
				UserID:    pathParams.UserID,
				Status:    body.Status,
				Priority:  body.Priority,
				Total:     body.Total,
				UpdatedBy: updatedBy,
			})
		},
	)

	router.GET("/openapi.json", api.OpenAPIHandler())
	return router, api
}

func isAdminToken(token string) bool {
	return token == "Bearer admin_token_123"
}

func extractUserFromToken(token string) string {
	if token == "Bearer admin_token_123" {
		return "admin@example.com"
	}
	return "user@example.com"
}

func TestIntegration_HealthCheck(t *testing.T) {
	router, _ := setupIntegrationRouter()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response HealthResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Status != "ok" {
		t.Errorf("Expected status 'ok', got %s", response.Status)
	}
}

func TestIntegration_ListUsers(t *testing.T) {
	router, _ := setupIntegrationRouter()

	tests := []struct {
		name         string
		query        string
		setupHeaders func(*http.Request)
		expectedCode int
		checkPage    int
	}{
		{
			name:  "valid_with_defaults",
			query: "",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusOK,
			checkPage:    0,
		},
		{
			name:  "valid_with_pagination",
			query: "?page=2&limit=20",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusOK,
			checkPage:    2,
		},
		{
			name:  "missing_auth_header",
			query: "",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:  "invalid_auth_format",
			query: "",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "InvalidFormat")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:  "api_key_too_short",
			query: "",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token123")
				req.Header.Set("X-API-Key", "short")
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:  "invalid_page_zero",
			query: "?page=-1",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:  "invalid_limit_exceeds_max",
			query: "?limit=200",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/users"+tt.query, nil)
			tt.setupHeaders(req)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedCode, w.Code, w.Body.String())
				return
			}

			if tt.expectedCode == http.StatusOK {
				var response ListUsersResponse
				json.Unmarshal(w.Body.Bytes(), &response)
				if response.Page != tt.checkPage {
					t.Errorf("Expected page %d, got %d", tt.checkPage, response.Page)
				}
			}
		})
	}
}

func TestIntegration_GetUserByID(t *testing.T) {
	router, _ := setupIntegrationRouter()

	tests := []struct {
		name         string
		userID       string
		setupHeaders func(*http.Request)
		expectedCode int
		checkID      int
	}{
		{
			name:   "valid_user",
			userID: "42",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusOK,
			checkID:      42,
		},
		{
			name:   "user_not_found",
			userID: "999",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:   "missing_api_key",
			userID: "42",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token123")
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "invalid_user_id_zero",
			userID: "0",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "invalid_user_id_string",
			userID: "abc",
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer token123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/users/"+tt.userID, nil)
			tt.setupHeaders(req)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedCode, w.Code, w.Body.String())
				return
			}

			if tt.expectedCode == http.StatusOK {
				var response UserResponse
				json.Unmarshal(w.Body.Bytes(), &response)
				if response.ID != tt.checkID {
					t.Errorf("Expected ID %d, got %d", tt.checkID, response.ID)
				}
			}
		})
	}
}

func TestIntegration_CreateUser(t *testing.T) {
	router, _ := setupIntegrationRouter()

	tests := []struct {
		name         string
		body         CreateUserRequest
		setupHeaders func(*http.Request)
		expectedCode int
		checkName    string
	}{
		{
			name: "valid_admin_token",
			body: CreateUserRequest{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				Role:  "user",
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer admin_token_123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusCreated,
			checkName:    "Jane Doe",
		},
		{
			name: "non_admin_token",
			body: CreateUserRequest{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				Role:  "user",
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer user_token_456")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "missing_required_name",
			body: CreateUserRequest{
				Email: "jane@example.com",
				Role:  "user",
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer admin_token_123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid_email",
			body: CreateUserRequest{
				Name:  "Jane Doe",
				Email: "invalid_email",
				Role:  "user",
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer admin_token_123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid_role",
			body: CreateUserRequest{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				Role:  "superuser",
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer admin_token_123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyJSON, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/users", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")
			tt.setupHeaders(req)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedCode, w.Code, w.Body.String())
				return
			}

			if tt.expectedCode == http.StatusCreated {
				var response UserResponse
				json.Unmarshal(w.Body.Bytes(), &response)
				if response.Name != tt.checkName {
					t.Errorf("Expected name %s, got %s", tt.checkName, response.Name)
				}
			}
		})
	}
}

func TestIntegration_UpdateOrder(t *testing.T) {
	router, _ := setupIntegrationRouter()

	tests := []struct {
		name         string
		path         string
		body         UpdateOrderRequest
		setupHeaders func(*http.Request)
		expectedCode int
		checkOrderID int
		checkStatus  string
	}{
		{
			name: "valid_order_update",
			path: "/users/42/orders/100",
			body: UpdateOrderRequest{
				Status:   "processing",
				Priority: 1,
				Total:    99.99,
				Notes:    "Rush order",
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer admin_token_123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusOK,
			checkOrderID: 100,
			checkStatus:  "processing",
		},
		{
			name: "invalid_status",
			path: "/users/42/orders/100",
			body: UpdateOrderRequest{
				Status:   "invalid_status",
				Priority: 1,
				Total:    99.99,
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer admin_token_123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "priority_out_of_range",
			path: "/users/42/orders/100",
			body: UpdateOrderRequest{
				Status:   "processing",
				Priority: 10,
				Total:    99.99,
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer admin_token_123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "total_zero",
			path: "/users/42/orders/100",
			body: UpdateOrderRequest{
				Status:   "processing",
				Priority: 1,
				Total:    0,
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer admin_token_123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "notes_too_long",
			path: "/users/42/orders/100",
			body: UpdateOrderRequest{
				Status:   "processing",
				Priority: 1,
				Total:    99.99,
				Notes:    string(make([]byte, 501)),
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer admin_token_123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid_user_id_zero",
			path: "/users/0/orders/100",
			body: UpdateOrderRequest{
				Status:   "processing",
				Priority: 1,
				Total:    99.99,
			},
			setupHeaders: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer admin_token_123")
				req.Header.Set("X-API-Key", "abcdefghijklmnopqrstuvwxyz123456")
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyJSON, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("PUT", tt.path, bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")
			tt.setupHeaders(req)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedCode, w.Code, w.Body.String())
				return
			}

			if tt.expectedCode == http.StatusOK {
				var response OrderResponse
				json.Unmarshal(w.Body.Bytes(), &response)
				if response.OrderID != tt.checkOrderID {
					t.Errorf("Expected order ID %d, got %d", tt.checkOrderID, response.OrderID)
				}
				if response.Status != tt.checkStatus {
					t.Errorf("Expected status %s, got %s", tt.checkStatus, response.Status)
				}
				// Verify handler used the auth header
				if response.UpdatedBy == "" {
					t.Error("Expected UpdatedBy to be populated from auth header")
				}
			}
		})
	}
}

func TestIntegration_OpenAPISpec(t *testing.T) {
	router, _ := setupIntegrationRouter()

	req := httptest.NewRequest("GET", "/openapi.json", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var spec map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatalf("Failed to unmarshal OpenAPI spec: %v", err)
	}

	paths := spec["paths"].(map[string]any)

	// Verify all endpoints are present
	expectedPaths := []string{"/health", "/users", "/users/{id}", "/users/{user_id}/orders/{order_id}"}
	for _, path := range expectedPaths {
		if paths[path] == nil {
			t.Errorf("Expected path %s to exist in OpenAPI spec", path)
		}
	}

	// Verify health endpoint has no parameters (no auth required)
	healthEndpoint := paths["/health"].(map[string]any)["get"].(map[string]any)
	if params, ok := healthEndpoint["parameters"]; ok && params != nil {
		paramsList := params.([]any)
		if len(paramsList) > 0 {
			t.Error("Health endpoint should not have parameters")
		}
	}

	// Verify /users endpoint has query params and headers
	usersEndpoint := paths["/users"].(map[string]any)["get"].(map[string]any)
	params := usersEndpoint["parameters"].([]any)
	if len(params) < 5 { // 3 query + 2 headers
		t.Errorf("Expected at least 5 parameters for /users, got %d", len(params))
	}
}
