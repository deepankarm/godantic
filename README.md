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

See [`examples/`](./examples/) for complete working examples with OpenAI and Gemini.

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

Check out [`examples/`](./examples/) for complete working examples.

---

> Disclaimer: Most of the code and tests are written by Cursor.
