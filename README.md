# Godantic

[![Tests](https://github.com/deepankarm/godantic/workflows/Tests/badge.svg)](https://github.com/deepankarm/godantic/actions/workflows/test.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/deepankarm/godantic.svg)](https://pkg.go.dev/github.com/deepankarm/godantic) [![Go Report Card](https://goreportcard.com/badge/github.com/deepankarm/godantic)](https://goreportcard.com/report/github.com/deepankarm/godantic) [![Go Version](https://img.shields.io/github/go-mod/go-version/deepankarm/godantic)](go.mod)

**Validation, JSON Schema, and OpenAPI generation for Go.** Define your types once with validation rules, get runtime validation, JSON schemas, and complete OpenAPI specs—without struct tags. Full support for Union types with discriminators.

Inspired by Python's [Pydantic](https://github.com/pydantic/pydantic) and [FastAPI](https://github.com/fastapi/fastapi). Works with LLM structured output APIs (OpenAI/Anthropic/Gemini), Gin REST APIs, and any system that needs JSON Schema or OpenAPI 3.0.3 specs.

```go
func (u *User) FieldEmail() godantic.FieldOptions[string] {
    return godantic.Field(
        godantic.Required[string](),
        godantic.Email(),
        godantic.Description[string]("Primary contact email"),
    )
}
```

**Key Features:**

- **Single source of truth**: Define validation rules once in Go code, get runtime validation + JSON Schema + OpenAPI specs
- **No struct tags**: Uses `Field{Name}()` methods instead of tag syntax—works with your IDE, tests, and debugger
- **Union type support**: Full `anyOf`/`oneOf` schemas with discriminators for LLM structured outputs and OpenAPI 3.1
- **LLM-ready**: Works seamlessly with OpenAI, Anthropic, and Gemini structured output APIs
- **Gin integration**: Automatic OpenAPI generation and interactive docs (`/docs`, `/redoc`)
- **Type-safe**: Leverages Go generics for compile-time safety

```bash
go get github.com/deepankarm/godantic
```

## Quick Start

```go
import "github.com/deepankarm/godantic"

type User struct {
    Email    string
    Age      int
    Username string
}

// Define validation rules
func (u *User) FieldEmail() godantic.FieldOptions[string] {
    return godantic.Field(
        godantic.Required[string](),
        godantic.Email(),
    )
}

func (u *User) FieldAge() godantic.FieldOptions[int] {
    return godantic.Field(
        godantic.Min(0),
        godantic.Max(130),
    )
}

func (u *User) FieldUsername() godantic.FieldOptions[string] {
    return godantic.Field(
        godantic.Required[string](),
        godantic.MinLen(3),
        godantic.MaxLen(20),
    )
}

// Validate
validator := godantic.NewValidator[User]()
errs := validator.Validate(&user)
```

## Features

### Type-Safe Constraints

```go
// Numeric constraints
godantic.Field(
    godantic.Min(0),
    godantic.Max(100),
    godantic.MultipleOf(5),
)

// String constraints
godantic.Field(
    godantic.MinLen(3),
    godantic.MaxLen(20),
    godantic.Regex("^[a-zA-Z0-9]+$"),
    godantic.Email(),
    godantic.URL(),
)

// Enum validation
godantic.Field(
    godantic.OneOf("active", "inactive", "pending"),
)
```

### Custom Validation

```go
func (u *User) FieldPassword() godantic.FieldOptions[string] {
    return godantic.Field(
        godantic.Required[string](),
        godantic.MinLen(8),
        godantic.Validate(func(password string) error {
            // Your custom logic
            if !hasUppercase(password) {
                return fmt.Errorf("must contain uppercase")
            }
            return nil
        }),
    )
}
```

### Union Types

Go doesn't have native union types, and that's by design. However, when building systems that interact with external APIs, LLMs, or generate OpenAPI schemas, you often need to express "this field can be one of several types" in JSON Schema.

Godantic provides **runtime validation and JSON Schema generation** for union-like patterns using Go's existing type system (`any` and interfaces).

**1. Simple Unions (`anyOf`) - For `any` fields**

Runtime validation + schema generation for fields that accept multiple types:

```go
type APIRequest struct {
    Query any  // Accept string OR array of inputs
}

func (r *APIRequest) FieldQuery() godantic.FieldOptions[any] {
    return godantic.Field(
        godantic.Union[any]("string", []TextInput{}, []ImageInput{}),
        godantic.Description[any]("Text query or structured inputs"),
    )
}
```

**Generated JSON Schema:**
```json
{
  "Query": {
    "anyOf": [
      {"type": "string"},
      {"type": "array", "items": {"$ref": "#/$defs/TextInput"}},
      {"type": "array", "items": {"$ref": "#/$defs/ImageInput"}}
    ]
  }
}
```

**2. Discriminated Unions (`oneOf`) - For interface fields**

Runtime validation + schema generation with discriminator field (OpenAPI 3.1, LLM tool calls):

```go
type ResponseType interface {
    GetStatus() string
}

type SuccessResponse struct {
    Status string            // Must be "success"
    Data   map[string]string
}

type ErrorResponse struct {
    Status  string  // Must be "error"
    Message string
    Code    int
}

type APIResult struct {
    Response ResponseType  // Compile-time type safety via interface
}

func (a *APIResult) FieldResponse() godantic.FieldOptions[ResponseType] {
    return godantic.Field(
        godantic.DiscriminatedUnion[ResponseType]("Status", map[string]any{
            "success": SuccessResponse{},
            "error":   ErrorResponse{},
        }),
    )
}
```

**Generated JSON Schema:**
```json
{
  "Response": {
    "oneOf": [
      {"$ref": "#/$defs/SuccessResponse"},
      {"$ref": "#/$defs/ErrorResponse"}
    ],
    "discriminator": {
      "propertyName": "Status",
      "mapping": {
        "success": "#/$defs/SuccessResponse",
        "error": "#/$defs/ErrorResponse"
      }
    }
  }
}
```

**What validation provides:**

- Validates `Status` field is one of `["success", "error"]`
- Catches typos: `Status: "succes"` → validation error
- Prevents invalid states: `ErrorResponse{Status: "success"}` → validation error
- Works with Go's type system (interfaces for compile-time safety)
- Generates proper JSON Schema for external tools

**Real-world benefits:**

- **LLM integrations**: Send proper schemas for structured outputs (OpenAI, Anthropic, Gemini)
- **OpenAPI generation**: Create API specs that generate type-safe clients
- **API validation**: Catch data inconsistencies at runtime
- **Interoperability**: Work with systems that expect union types (TypeScript, Pydantic)

### Validating interfaces

Automatic type routing for interface types based on discriminator fields. Useful for polymorphic API requests, payment methods, or any domain where you need runtime type selection.

```go
type PaymentType string

const (
    PaymentTypeCreditCard   PaymentType = "credit_card"
    PaymentTypePayPal       PaymentType = "paypal"
    PaymentTypeBankTransfer PaymentType = "bank_transfer"
)

type PaymentMethod interface {
    GetType() PaymentType
}

type CreditCardPayment struct {
    Type       PaymentType `json:"type"`
    CardNumber string      `json:"card_number"`
}

type PayPalPayment struct {
    Type  PaymentType `json:"type"`
    Email string      `json:"email"`
}

type BankTransferPayment struct {
    Type        PaymentType `json:"type"`
    AccountName string      `json:"account_name"`
}

// Create validator with discriminator
validator := godantic.NewValidator[PaymentMethod](
    godantic.WithDiscriminatorTyped("type", map[PaymentType]any{
        PaymentTypeCreditCard:   CreditCardPayment{},
        PaymentTypePayPal:       PayPalPayment{},
        PaymentTypeBankTransfer: BankTransferPayment{},
    }),
)

// Automatically routes to correct type based on "type" field
payment, errs := validator.Marshal(jsonData)

// Type switch for handling
switch p := (*payment).(type) {
case CreditCardPayment:
    fmt.Printf("Credit card: %s\n", p.CardNumber)
case PayPalPayment:
    fmt.Printf("PayPal: %s\n", p.Email)
case BankTransferPayment:
    fmt.Printf("Bank: %s\n", p.AccountName)
}
```

**How it works:**

1. Examines discriminator field (`"type"`) to determine concrete type
2. Unmarshals JSON into the appropriate struct (e.g., `CreditCardPayment`)
3. Validates all fields according to their Field methods
4. Applies default values if defined
5. Returns as interface type with proper concrete value

**Key benefits:**

- No manual discriminator routing code required
- Compile-time type safety via Go interfaces
- All field validation rules applied automatically
- Clean separation of validation logic from business logic

See [`examples/payment-methods/`](./examples/payment-methods/) for a complete working example.

### JSON Schema Generation

Generate JSON Schema without struct tags:

```go
import "github.com/deepankarm/godantic/schema"

sg := schema.NewGenerator[User]()
s, err := sg.Generate()

// Or as JSON string
jsonSchema, err := sg.GenerateJSON()
```

All validation constraints (min, max, pattern, etc.) are automatically included in the schema.

**For LLM APIs (OpenAI, Gemini, Claude):**

LLM providers require a "flattened" schema format where the root object definition is at the top level instead of behind a `$ref`:

```go
// Generate flattened schema for LLM APIs
flatSchema, err := sg.GenerateFlattened()
```

This promotes the root object to the top level while preserving `$defs` for nested types, making it compatible with structured output APIs.

### JSON Marshal/Unmarshal with Validation

Godantic provides convenient methods for working with JSON that automatically apply defaults and validate:

**`Marshal` - JSON → Struct (with validation)**

Unmarshals JSON, applies defaults, and validates in one step:

```go
validator := godantic.NewValidator[User]()

// One-liner: unmarshal + defaults + validate
user, errs := validator.Marshal(jsonData)
if len(errs) > 0 {
    // Handle validation errors
}
// user is ready to use with all defaults applied
```

**`Unmarshal` - Struct → JSON (with validation)**

Validates, applies defaults, and marshals to JSON in one step:

```go
validator := godantic.NewValidator[User]()

// One-liner: validate + defaults + marshal
jsonData, errs := validator.Unmarshal(&user)
if len(errs) > 0 {
    // Handle validation errors
}
// jsonData is valid JSON with all defaults
```

### Lifecycle Hooks

Godantic provides hooks to transform data at different stages of validation and serialization:

**`BeforeValidate` - Transform JSON before validation**

Modify the raw JSON map before unmarshaling and validation (useful for backward compatibility):

```go
func (u *User) BeforeValidate(raw map[string]any) error {
    // Normalize legacy format
    if legacyField, ok := raw["old_field"]; ok {
        raw["new_field"] = legacyField
        delete(raw, "old_field")
    }
    return nil
}
```

**`AfterValidate` - Transform struct after validation**

Modify the struct after successful validation (useful for computed fields):

```go
func (u *User) AfterValidate() error {
    u.DisplayName = strings.ToLower(u.Username)
    return nil
}
```

**`BeforeSerialize` - Transform struct before marshaling**

Modify the struct before marshaling to JSON:

```go
func (u *User) BeforeSerialize() error {
    u.UpdatedAt = time.Now()
    return nil
}
```

**`AfterSerialize` - Transform JSON after marshaling**

Modify the JSON bytes after marshaling (useful for wrapping responses):

```go
func (u *User) AfterSerialize(data []byte) ([]byte, error) {
    // Wrap in envelope
    envelope := map[string]any{
        "data": json.RawMessage(data),
        "version": "1.0",
    }
    return json.Marshal(envelope)
}
```

**Execution order:**

- **Marshal** (JSON → Struct): `BeforeValidate` → unmarshal → validate → defaults → `AfterValidate`
- **Unmarshal** (Struct → JSON): `BeforeSerialize` → validate → defaults → marshal → `AfterSerialize`

### Complex Structures

Works with embedded structs, nested structs, pointers, slices, and maps:

```go
type Company struct {
    Name     string
    Address  Address        // Nested struct
    Contacts []Contact      // Slice of structs
    Settings map[string]int // Map
}
```

## LLM Structured Output

**Godantic is an excellent companion for LLM structured outputs.** Use it to:
1. **Generate JSON Schema** from your Go types (with all constraints and union types)
2. **Send it to LLM APIs** (OpenAI, Gemini, Claude) to guide response structure
3. **Validate the LLM response** to ensure it matches your schema and constraints

**Complete workflow:**

```go
import (
    "github.com/deepankarm/godantic/"
    "github.com/deepankarm/godantic/schema"
)

// 1. Define your response structure
type TaskList struct {
    Tasks []Task
}

func (t *TaskList) FieldTasks() godantic.FieldOptions[[]Task] {
    return godantic.Field(
        godantic.Required[[]Task](),
        godantic.MinItems[Task](1),
    )
}

// 2. Generate flattened schema for LLM
schemaGen := schema.NewGenerator[TaskList]()
flatSchema, _ := schemaGen.GenerateFlattened()

// 3. Send to LLM API (OpenAI, Gemini, Claude)
response := callLLM(prompt, flatSchema)

// 4. Validate LLM response
validator := godantic.NewValidator[TaskList]()
var result TaskList
json.Unmarshal(response, &result)
if errs := validator.Validate(&result); len(errs) > 0 {
    // Handle validation errors
}
```

**Why this matters:**
- LLMs sometimes return invalid data (missing fields, wrong types, out-of-range values)
- Godantic catches these issues before they reach your business logic
- Same schema definition for both generation and validation (single source of truth)

See [`examples/openai-structured-output/`](./examples/openai-structured-output/) and [`examples/gemini-structured-output/`](./examples/gemini-structured-output/) for complete working examples.

### Streaming Partial JSON

Parse incomplete JSON as it streams from LLM APIs. Essential for real-time UI updates during long-running generation.

```go
// Create streaming parser
parser := godantic.NewStreamParser[Response]()

// Feed chunks as they arrive from LLM API
for chunk := range llmStream {
    result, state, _ := parser.Feed(chunk)
    
    if state.IsComplete {
        // Full response received and validated
        fmt.Println("Complete:", result)
    } else {
        // Show partial data in real-time
        fmt.Printf("Streaming... waiting for: %v\n", state.WaitingFor())
    }
}
```

**Features:**
- Repairs incomplete JSON (closes unclosed strings, arrays, objects)
- Tracks incomplete fields via `state.WaitingFor()`
- Skips validation for incomplete fields
- Applies defaults automatically

See [`examples/llm-partialjson-streaming/`](./examples/llm-partialjson-streaming/) for a complete working example with Gemini streaming.

## Gin Integration (gingodantic)

Automatic OpenAPI spec generation and validation for Gin APIs. Define your request/response types once, get runtime validation, OpenAPI specs, and interactive documentation.

```go
import "github.com/deepankarm/godantic/pkg/gingodantic"

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (c *CreateUserRequest) FieldName() godantic.FieldOptions[string] {
    return godantic.Field(
        godantic.Required[string](),
        godantic.MinLen(2),
    )
}

func (c *CreateUserRequest) FieldEmail() godantic.FieldOptions[string] {
    return godantic.Field(
        godantic.Required[string](),
        godantic.Email(),
    )
}

// Setup
router := gin.Default()
api := gingodantic.New("User API", "1.0.0")

// Register endpoint with schema
router.POST("/users",
    api.OpenAPISchema("POST", "/users",
        gingodantic.WithSummary("Create user"),
        gingodantic.WithRequest[CreateUserRequest](),  // Enables validation
        gingodantic.WithResponse[UserResponse](201, "Created"),
    ),
    func(c *gin.Context) {
        req, _ := gingodantic.GetValidated[CreateUserRequest](c)
        // req is validated and ready to use
    },
)

// Serve OpenAPI spec and documentation UIs
router.GET("/openapi.json", api.OpenAPIHandler())
router.GET("/docs", gingodantic.SwaggerUI("/openapi.json"))
router.GET("/redoc", gingodantic.ReDoc("/openapi.json"))  // Alternative UI
```

**Features:**

- **Automatic validation**: Request bodies, query params, path params, headers, and cookies
- **OpenAPI 3.0.3 generation**: Complete spec with all parameter types and constraints
- **Type-safe helpers**: `GetValidated[T]()`, `GetValidatedQuery[T]()`, `GetValidatedPath[T]()`, etc.
- **Validation by default**: Enabled automatically when request types are specified
- **Documentation UIs**: Built-in Swagger UI and ReDoc handlers
- **Zero boilerplate**: No manual schema writing or validation middleware

**Parameter types supported:**

```go
gingodantic.WithRequest[T]()        // Request body
gingodantic.WithQueryParams[T]()    // Query parameters
gingodantic.WithPathParams[T]()     // Path parameters (:id)
gingodantic.WithHeaderParams[T]()   // Request headers
gingodantic.WithCookieParams[T]()   // Cookies
gingodantic.WithResponse[T](code)   // Response schemas
```

All godantic constraints (min, max, regex, email, etc.) are automatically included in the OpenAPI spec.

See [`examples/gin-api/`](./examples/gin-api/) for a complete working API with all parameter types.

## Available Constraints

```go
godantic.Required[T]()              // required field

// numeric constraints
godantic.Min(value)                 // value >= min
godantic.Max(value)                 // value <= max
godantic.ExclusiveMin(value)        // value > min
godantic.ExclusiveMax(value)        // value < max
godantic.MultipleOf(value)          // value is multiple of

// string constraints
godantic.MinLen(length)             // minimum length
godantic.MaxLen(length)             // maximum length
godantic.Regex(pattern)             // regex pattern match
godantic.Email()                    // email format
godantic.URL()                      // URL format
godantic.ContentEncoding(encoding)  // e.g., "base64"
godantic.ContentMediaType(type)     // e.g., "application/json"

// array/slice constraints
godantic.MinItems[T](count)         // minimum number of items
godantic.MaxItems[T](count)         // maximum number of items
godantic.UniqueItems[T]()           // all items must be unique

// map/object constraints
godantic.MinProperties(count)       // minimum properties
godantic.MaxProperties(count)       // maximum properties

// union constraints
godantic.Union[T](type1, type2, ...) // any of the types
godantic.DiscriminatedUnion[T](discriminator, map[string]any{
    "type1": type1{},
    "type2": type2{},
    ...
}) // one of the types based on discriminator

// value constraints
godantic.OneOf(value1, value2, ...) // enum - one of allowed values
godantic.Const(value)               // must equal exactly this value
godantic.Default(value)             // default value (schema only)

// schema metadata
godantic.Description[T](text)       // field description
godantic.Example(value)             // example value
godantic.Title[T](text)             // field title
godantic.Format[T](format)          // format hint (e.g., "date-time")
godantic.ReadOnly[T]()              // read-only field
godantic.WriteOnly[T]()             // write-only field
godantic.Deprecated[T]()            // deprecated field
```

### Custom Validation
```go
godantic.Validate(func(val T) error {
    // Your custom validation logic
    return nil
})
```

## How it works

1. Define `Field{FieldName}()` methods that return `FieldOptions[T]`
2. Use `Field()` to compose validation constraints
3. Create a `Validator[T]` and call `Validate()`
4. Generate JSON Schema with `schema.NewGenerator[T]()`

Zero values (empty string, 0, nil) are treated as "not set" for required field checks.


## Testing

```bash
go test ./... -v -cover
```

## Examples

Check out [`examples/`](./examples/) for complete working examples:

- **[`openai-structured-output/`](./examples/openai-structured-output/)** - Using godantic with OpenAI's structured output API to extract meeting summaries
- **[`gemini-structured-output/`](./examples/gemini-structured-output/)** - Using godantic with Google Gemini to parse task lists with enums and unions
- **[`gin-api/`](./examples/gin-api/)** - Complete Gin REST API with automatic OpenAPI generation and Swagger UI
- **[`payment-methods/`](./examples/payment-methods/)** - Validating polymorphic payment requests using discriminated unions  
- **[`llm-partialjson-streaming/`](./examples/llm-partialjson-streaming/)** - Streaming partial JSON during long-running generation using Gemini

---

> Disclaimer: Some of the code and most of the tests & docs are written by Cursor.
