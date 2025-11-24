package gingodantic

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSwaggerUI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/docs", SwaggerUI("/openapi.json"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/docs", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Check for essential Swagger UI elements
	requiredElements := []string{
		"<!DOCTYPE html>",
		"swagger-ui-bundle.js",
		"swagger-ui.css",
		"url: '/openapi.json'",
		"SwaggerUIBundle",
		"dom_id: '#swagger-ui'",
	}

	for _, element := range requiredElements {
		if !strings.Contains(body, element) {
			t.Errorf("Expected HTML to contain %q", element)
		}
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected content type text/html, got %s", contentType)
	}
}

func TestSwaggerUIWithConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	config := DefaultSwaggerUIConfig("/api/openapi.json")
	config.Title = "Custom API Docs"
	router.GET("/docs", SwaggerUIWithConfig(config))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/docs", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Check for custom configuration
	if !strings.Contains(body, "Custom API Docs") {
		t.Error("Expected HTML to contain custom title")
	}

	if !strings.Contains(body, "/api/openapi.json") {
		t.Error("Expected HTML to contain custom OpenAPI URL")
	}
}

func TestReDoc(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/redoc", ReDoc("/openapi.json"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/redoc", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Check for essential ReDoc elements
	requiredElements := []string{
		"<!DOCTYPE html>",
		"redoc.standalone.js",
		"spec-url=\"/openapi.json\"",
		"<redoc",
		"fonts.googleapis.com",
	}

	for _, element := range requiredElements {
		if !strings.Contains(body, element) {
			t.Errorf("Expected HTML to contain %q", element)
		}
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected content type text/html, got %s", contentType)
	}
}

func TestReDocWithConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	config := DefaultReDocConfig("/api/openapi.json")
	config.Title = "Custom ReDoc"
	config.WithGoogleFonts = false
	router.GET("/redoc", ReDocWithConfig(config))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/redoc", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Check for custom configuration
	if !strings.Contains(body, "Custom ReDoc") {
		t.Error("Expected HTML to contain custom title")
	}

	if !strings.Contains(body, "/api/openapi.json") {
		t.Error("Expected HTML to contain custom OpenAPI URL")
	}

	// Should not contain Google Fonts when disabled
	if strings.Contains(body, "fonts.googleapis.com") {
		t.Error("Expected HTML to not contain Google Fonts when disabled")
	}
}

func TestDefaultSwaggerUIConfig(t *testing.T) {
	config := DefaultSwaggerUIConfig("/test/openapi.json")

	if config.OpenAPIURL != "/test/openapi.json" {
		t.Errorf("Expected OpenAPIURL to be /test/openapi.json, got %s", config.OpenAPIURL)
	}

	if config.Title != "API Documentation" {
		t.Errorf("Expected default title, got %s", config.Title)
	}

	if !strings.Contains(config.SwaggerJSURL, "swagger-ui-dist") {
		t.Error("Expected default Swagger JS URL")
	}

	if !strings.Contains(config.SwaggerCSSURL, "swagger-ui-dist") {
		t.Error("Expected default Swagger CSS URL")
	}
}

func TestDefaultReDocConfig(t *testing.T) {
	config := DefaultReDocConfig("/test/openapi.json")

	if config.OpenAPIURL != "/test/openapi.json" {
		t.Errorf("Expected OpenAPIURL to be /test/openapi.json, got %s", config.OpenAPIURL)
	}

	if config.Title != "API Documentation" {
		t.Errorf("Expected default title, got %s", config.Title)
	}

	if !strings.Contains(config.ReDocJSURL, "redoc") {
		t.Error("Expected default ReDoc JS URL")
	}

	if !config.WithGoogleFonts {
		t.Error("Expected Google Fonts to be enabled by default")
	}
}
