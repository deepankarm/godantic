package gingodantic_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deepankarm/godantic/pkg/gingodantic"
	"github.com/deepankarm/godantic/pkg/godantic"
	"github.com/gin-gonic/gin"
)

// Test nested discriminated unions in OpenAPI generation

type RequestType interface {
	IsRequestType()
}

type RequestPayload interface {
	IsRequestPayload()
}

type JSONPayload struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func (JSONPayload) IsRequestPayload() {}

type CreateRequest struct {
	Action  string         `json:"action"`
	Payload RequestPayload `json:"payload"` // Nested discriminated union
}

func (CreateRequest) IsRequestType() {}

func (c *CreateRequest) FieldPayload() godantic.FieldOptions[RequestPayload] {
	return godantic.Field(
		godantic.Required[RequestPayload](),
		godantic.DiscriminatedUnion[RequestPayload]("type", map[string]any{
			"json": JSONPayload{},
		}),
	)
}

type APISchema struct {
	Request RequestType `json:"request"`
}

func (a *APISchema) FieldRequest() godantic.FieldOptions[RequestType] {
	return godantic.Field(
		godantic.DiscriminatedUnion[RequestType]("action", map[string]any{
			"create": CreateRequest{},
		}),
	)
}

func TestNestedDiscriminatedUnionInOpenAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	api := gingodantic.New("Test API", "1.0.0")
	router.GET("/api-schema",
		api.OpenAPISchema("GET", "/api-schema",
			gingodantic.WithResponse[APISchema](200, "API Schema"),
		),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, APISchema{})
		},
	)
	router.GET("/openapi.json", api.OpenAPIHandler())

	// Request OpenAPI spec
	req := httptest.NewRequest("GET", "/openapi.json", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Parse response
	var spec map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatalf("Failed to parse OpenAPI spec: %v", err)
	}

	// Check CreateRequest.payload schema
	components, ok := spec["components"].(map[string]any)
	if !ok {
		t.Fatal("Expected components in OpenAPI spec")
	}

	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Fatal("Expected schemas in components")
	}

	createReq, ok := schemas["CreateRequest"].(map[string]any)
	if !ok {
		t.Fatal("Expected CreateRequest schema")
	}

	props, ok := createReq["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected properties in CreateRequest schema")
	}

	payload := props["payload"]
	t.Logf("CreateRequest.payload: %T = %v", payload, payload)

	// The critical check: payload should be a map with oneOf, not just `true`
	payloadMap, ok := payload.(map[string]any)
	if !ok {
		t.Fatalf("Expected payload to be a schema object, got: %T = %v", payload, payload)
	}

	oneOf, ok := payloadMap["oneOf"].([]any)
	if !ok || len(oneOf) == 0 {
		t.Fatalf("Expected payload to have oneOf, got: %v", payloadMap)
	}

	t.Logf("✅ payload has oneOf with %d variants", len(oneOf))

	// Verify JSONPayload schema exists
	if _, exists := schemas["JSONPayload"]; !exists {
		t.Error("Expected JSONPayload schema in components/schemas")
	}
}
