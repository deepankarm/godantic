package main

import (
	"net/http"

	"github.com/deepankarm/godantic/pkg/gingodantic"
	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/gin-gonic/gin"
)

// Request/Response types with godantic validation
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func (r *CreateUserRequest) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.MinLen(3),
		godantic.Description[string]("User's full name"),
	)
}

func (r *CreateUserRequest) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(
		godantic.Required[string](),
		godantic.Regex(`^[^\s@]+@[^\s@]+\.[^\s@]+$`),
		godantic.Description[string]("User's email address"),
	)
}

func (r *CreateUserRequest) FieldAge() godantic.FieldOptions[int] {
	return godantic.Field(
		godantic.Min(18),
		godantic.Max(120),
		godantic.Description[int]("User's age"),
	)
}

type UserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

type ErrorResponse struct {
	Error   string  `json:"error"`
	Details *string `json:"details,omitempty"`
}

type ListUsersResponse struct {
	Users []UserResponse `json:"users"`
	Total int            `json:"total"`
}

func main() {
	// Create normal Gin router
	router := gin.Default()

	// Create API spec builder
	api := gingodantic.New("User Management API", "1.0.0")

	// Register routes with Gin as normal, but add schema middleware
	// Validation is ENABLED by default when Request type is specified
	router.POST("/users",
		api.OpenAPISchema("POST", "/users",
			gingodantic.WithSummary("Create a new user"),
			gingodantic.WithDescription("Creates a new user account with the provided information"),
			gingodantic.WithTags("users"),
			gingodantic.WithRequest[CreateUserRequest](), // This enables validation automatically
			gingodantic.WithResponse[UserResponse](201, "User created successfully"),
			gingodantic.WithResponse[ErrorResponse](400, "Invalid request"),
		),
		createUser, // Handler receives validated data via gingodantic.GetValidated()
	)

	router.GET("/users",
		api.OpenAPISchema("GET", "/users",
			gingodantic.WithSummary("List all users"),
			gingodantic.WithTags("users"),
			gingodantic.WithResponse[ListUsersResponse](200, "List of users"),
		),
		listUsers,
	)

	router.GET("/users/:id",
		api.OpenAPISchema("GET", "/users/:id",
			gingodantic.WithSummary("Get user by ID"),
			gingodantic.WithTags("users"),
			gingodantic.WithResponse[UserResponse](200, "User found"),
			gingodantic.WithResponse[ErrorResponse](404, "User not found"),
		),
		getUser,
	)

	// Health check endpoint without schema (optional)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Serve OpenAPI spec
	router.GET("/openapi.json", api.OpenAPIHandler())

	// Serve Swagger UI (interactive API documentation)
	router.GET("/docs", gingodantic.SwaggerUI("/openapi.json"))

	// Serve ReDoc (alternative API documentation)
	router.GET("/redoc", gingodantic.ReDoc("/openapi.json"))

	// Print startup message
	println("Server starting on :8080")
	println("API Documentation:")
	println("  Swagger UI:  http://localhost:8080/docs")
	println("  ReDoc:       http://localhost:8080/redoc")
	println("  OpenAPI Spec: http://localhost:8080/openapi.json")
	println("\nTest with:")
	println("  curl -X POST http://localhost:8080/users -H 'Content-Type: application/json' -d '{\"name\":\"John Doe\",\"email\":\"john@example.com\",\"age\":25}'")

	router.Run(":8080")
}

func createUser(c *gin.Context) {
	// Get validated request from context (already validated by middleware)
	req, ok := gingodantic.GetValidated[CreateUserRequest](c)
	if !ok {
		// This shouldn't happen if validation is enabled
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get validated request",
		})
		return
	}

	user := UserResponse{
		ID:    "user_123",
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
	}

	c.JSON(http.StatusCreated, user)
}

func listUsers(c *gin.Context) {
	users := []UserResponse{
		{ID: "user_1", Name: "Alice", Email: "alice@example.com", Age: 30},
		{ID: "user_2", Name: "Bob", Email: "bob@example.com", Age: 25},
	}

	c.JSON(http.StatusOK, ListUsersResponse{
		Users: users,
		Total: len(users),
	})
}

func getUser(c *gin.Context) {
	id := c.Param("id")

	if id != "user_123" {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "User not found",
		})
		return
	}

	user := UserResponse{
		ID:    id,
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}

	c.JSON(http.StatusOK, user)
}
