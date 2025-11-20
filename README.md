# Godantic

Godantic provides runtime validation and automatic JSON Schema generation with Union type support, without struct tags. Inspired by Python's [Pydantic](https://github.com/pydantic/pydantic). Built for developers working with LLM APIs (OpenAI, Anthropic, Gemini), building REST APIs with OpenAPI specs, or validating complex structured data in Go.


```go
func (u *User) FieldEmail() godantic.FieldOptions[string] {
    return godantic.Field(
        godantic.Required[string](),
        godantic.Email(),
        godantic.Description[string]("Primary contact email"),
    )
}
```

**Why Godantic?:**
- **Single source of truth**: Define validation rules once, use them for both runtime checks and schema generation
- **Beyond struct tags**: Validation libraries and JSON Schema generators use different tag syntaxes. Godantic uses Go code, so your IDE, tests, and debugger all work naturally. Critical when iterating on LLM schemas.
- **Union type support**: Generate proper `anyOf`/`oneOf` schemas—critical for LLM structured outputs and OpenAPI
- **Type-safe**: Go generics catch errors at compile time instead of runtime
- **Testable**: Validation is plain Go code you can unit test

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

## Gin Integration (gingodantic)

**Automatic OpenAPI spec generation and validation for Gin APIs.** Define your request/response types once with godantic, get OpenAPI specs and validation for free.

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

// Serve OpenAPI spec
router.GET("/openapi.json", api.OpenAPIHandler())
// TODO: Add Swagger UI
```

**Features:**

- **Automatic validation**: Request bodies, query params, path params, headers, and cookies
- **OpenAPI 3.0.3 generation**: Complete spec with all parameter types and constraints
- **Type-safe helpers**: `GetValidated[T]()`, `GetValidatedQuery[T]()`, `GetValidatedPath[T]()`, etc.
- **Validation by default**: Enabled automatically when request types are specified
- **Swagger UI included**: Built-in handler for API documentation
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

- **[`gin-api/`](./examples/gin-api/)** - Complete Gin REST API with automatic OpenAPI generation, validation for all parameter types (path, query, headers, cookies, body), and Swagger UI
- **[`payment-methods/`](./examples/payment-methods/)** - Validating polymorphic payment requests using discriminated unions at the interface level
- **[`openai-structured-output/`](./examples/openai-structured-output/)** - Using godantic with OpenAI's structured output API to extract meeting summaries from text
- **[`gemini-structured-output/`](./examples/gemini-structured-output/)** - Using godantic with Google Gemini to parse task lists with enums, dates, and unions  

---

> Disclaimer: Most of the code and tests are written by Cursor.
