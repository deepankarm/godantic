package gingodantic_test

import (
	"fmt"

	"github.com/deepankarm/godantic/pkg/gingodantic"
	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/gin-gonic/gin"
)

// Types for ExampleAPI_OpenAPISchema
type exampleUserPath struct {
	ID int `json:"id"`
}

func (p *exampleUserPath) FieldID() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Required[int](), godantic.Min(1))
}

type exampleListQuery struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

func (q *exampleListQuery) FieldPage() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Default(1), godantic.Min(1))
}

func (q *exampleListQuery) FieldLimit() godantic.FieldOptions[int] {
	return godantic.Field(godantic.Default(10), godantic.Min(1), godantic.Max(100))
}

type exampleCreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (r *exampleCreateUserRequest) FieldName() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.MinLen(2))
}

func (r *exampleCreateUserRequest) FieldEmail() godantic.FieldOptions[string] {
	return godantic.Field(godantic.Required[string](), godantic.Email())
}

type exampleUserResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ExampleAPI_OpenAPISchema demonstrates registering endpoints with automatic
// OpenAPI schema generation and validation for path parameters, query parameters,
// request bodies, and responses.
func ExampleAPI_OpenAPISchema() {
	api := gingodantic.New("User API", "1.0.0")
	router := gin.New()

	// GET /users/:id - path param + query params
	router.GET("/users/:id",
		api.OpenAPISchema("GET", "/users/:id",
			gingodantic.WithPathParams[exampleUserPath](),
			gingodantic.WithQueryParams[exampleListQuery](),
			gingodantic.WithResponse[exampleUserResponse](200, "User found"),
		),
		func(c *gin.Context) {
			path, _ := gingodantic.GetValidatedPath[exampleUserPath](c)
			query, _ := gingodantic.GetValidatedQuery[exampleListQuery](c)
			// path.ID, query.Page, query.Limit are validated
			_ = path
			_ = query
		},
	)

	// POST /users - request body
	router.POST("/users",
		api.OpenAPISchema("POST", "/users",
			gingodantic.WithRequest[exampleCreateUserRequest](),
			gingodantic.WithResponse[exampleUserResponse](201, "User created"),
		),
		func(c *gin.Context) {
			req, _ := gingodantic.GetValidated[exampleCreateUserRequest](c)
			// req.Name, req.Email are validated
			_ = req
		},
	)

	// Get OpenAPI spec
	spec := api.GenerateOpenAPI()
	fmt.Println(spec["openapi"])
	// Output: 3.0.3
}
