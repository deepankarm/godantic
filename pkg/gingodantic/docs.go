package gingodantic

import (
	"github.com/gin-gonic/gin"
)

// SwaggerUIConfig holds configuration for Swagger UI
type SwaggerUIConfig struct {
	// OpenAPIURL is the URL to the OpenAPI spec JSON
	OpenAPIURL string
	// Title is the HTML page title
	Title string
	// SwaggerJSURL is the URL to Swagger UI JavaScript bundle
	SwaggerJSURL string
	// SwaggerCSSURL is the URL to Swagger UI CSS
	SwaggerCSSURL string
	// FaviconURL is the URL to the favicon
	FaviconURL string
}

// DefaultSwaggerUIConfig returns default Swagger UI configuration
func DefaultSwaggerUIConfig(openAPIURL string) SwaggerUIConfig {
	return SwaggerUIConfig{
		OpenAPIURL:    openAPIURL,
		Title:         "API Documentation",
		SwaggerJSURL:  "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js",
		SwaggerCSSURL: "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css",
		FaviconURL:    "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/favicon-32x32.png",
	}
}

// SwaggerUI returns a Gin handler that serves Swagger UI
//
// Example:
//
//	router.GET("/docs", gingodantic.SwaggerUI("/openapi.json"))
//
// You can customize the configuration:
//
//	config := gingodantic.DefaultSwaggerUIConfig("/openapi.json")
//	config.Title = "My API Docs"
//	router.GET("/docs", gingodantic.SwaggerUIWithConfig(config))
func SwaggerUI(openAPIURL string) gin.HandlerFunc {
	return SwaggerUIWithConfig(DefaultSwaggerUIConfig(openAPIURL))
}

// SwaggerUIWithConfig returns a Gin handler that serves Swagger UI with custom configuration
func SwaggerUIWithConfig(config SwaggerUIConfig) gin.HandlerFunc {
	html := `<!DOCTYPE html>
<html>
<head>
    <link type="text/css" rel="stylesheet" href="` + config.SwaggerCSSURL + `">
    <link rel="shortcut icon" href="` + config.FaviconURL + `">
    <title>` + config.Title + `</title>
</head>
<body>
<div id="swagger-ui"></div>
<script src="` + config.SwaggerJSURL + `"></script>
<script>
const ui = SwaggerUIBundle({
    url: '` + config.OpenAPIURL + `',
    dom_id: '#swagger-ui',
    layout: 'BaseLayout',
    deepLinking: true,
    showExtensions: true,
    showCommonExtensions: true,
    presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset
    ],
})
</script>
</body>
</html>`

	return func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, html)
	}
}

// ReDocConfig holds configuration for ReDoc
type ReDocConfig struct {
	// OpenAPIURL is the URL to the OpenAPI spec JSON
	OpenAPIURL string
	// Title is the HTML page title
	Title string
	// ReDocJSURL is the URL to ReDoc JavaScript bundle
	ReDocJSURL string
	// FaviconURL is the URL to the favicon
	FaviconURL string
	// WithGoogleFonts enables Google Fonts
	WithGoogleFonts bool
}

// DefaultReDocConfig returns default ReDoc configuration
func DefaultReDocConfig(openAPIURL string) ReDocConfig {
	return ReDocConfig{
		OpenAPIURL:      openAPIURL,
		Title:           "API Documentation",
		ReDocJSURL:      "https://cdn.jsdelivr.net/npm/redoc@2/bundles/redoc.standalone.js",
		FaviconURL:      "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/favicon-32x32.png",
		WithGoogleFonts: true,
	}
}

// ReDoc returns a Gin handler that serves ReDoc UI (alternative to Swagger UI)
//
// Example:
//
//	router.GET("/redoc", gingodantic.ReDoc("/openapi.json"))
func ReDoc(openAPIURL string) gin.HandlerFunc {
	return ReDocWithConfig(DefaultReDocConfig(openAPIURL))
}

// ReDocWithConfig returns a Gin handler that serves ReDoc UI with custom configuration
func ReDocWithConfig(config ReDocConfig) gin.HandlerFunc {
	googleFonts := ""
	if config.WithGoogleFonts {
		googleFonts = `<link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">`
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <title>` + config.Title + `</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    ` + googleFonts + `
    <link rel="shortcut icon" href="` + config.FaviconURL + `">
    <style>
      body {
        margin: 0;
        padding: 0;
      }
    </style>
</head>
<body>
    <noscript>
        ReDoc requires Javascript to function. Please enable it to browse the documentation.
    </noscript>
    <redoc spec-url="` + config.OpenAPIURL + `"></redoc>
    <script src="` + config.ReDocJSURL + `"></script>
</body>
</html>`

	return func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, html)
	}
}
