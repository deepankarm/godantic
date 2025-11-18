# Godantic

Go validation library inspired by Python's Pydantic, with type-safe constraints and automatic JSON Schema generation.

```bash
go get github.com/deepankarm/godantic
```

## Quick Start

```go
import "github.com/deepankarm/godantic/pkg/godantic"

type User struct {
    Email    string
    Age      int
    Username string
}

// Define validation rules
func (u *User) FieldEmail() godantic.FieldOptions[string] {
    return godantic.WithFieldOptions(
        godantic.Required[string](),
        godantic.Email(),
    )
}

func (u *User) FieldAge() godantic.FieldOptions[int] {
    return godantic.WithFieldOptions(
        godantic.Min(0),
        godantic.Max(130),
    )
}

func (u *User) FieldUsername() godantic.FieldOptions[string] {
    return godantic.WithFieldOptions(
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

### Native Go Types

No wrapper types needed - use regular `string`, `int`, pointers, etc.

```go
type User struct {
    Email string  // Not Field[string]!
    Age   int
}
```

### Type-Safe Constraints

```go
// Numeric constraints
godantic.WithFieldOptions(
    godantic.Min(0),
    godantic.Max(100),
    godantic.MultipleOf(5),
)

// String constraints
godantic.WithFieldOptions(
    godantic.MinLen(3),
    godantic.MaxLen(20),
    godantic.Regex("^[a-zA-Z0-9]+$"),
    godantic.Email(),
    godantic.URL(),
)

// Enum validation
godantic.WithFieldOptions(
    godantic.OneOf("active", "inactive", "pending"),
)
```

### Custom Validation

```go
func (u *User) FieldPassword() godantic.FieldOptions[string] {
    return godantic.WithFieldOptions(
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

### JSON Schema Generation

Generate JSON Schema without struct tags:

```go
import "github.com/deepankarm/godantic/pkg/godantic/schema"

sg := schema.NewGenerator[User]()
s, err := sg.Generate()

// Or as JSON string
jsonSchema, err := sg.GenerateJSON()
```

All validation constraints (min, max, pattern, etc.) are automatically included in the schema.

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

## Available Constraints

### Required Fields
```go
godantic.Required[T]()
```

### Numeric
```go
godantic.Min(value)                 // value >= min
godantic.Max(value)                 // value <= max
godantic.ExclusiveMin(value)        // value > min
godantic.ExclusiveMax(value)        // value < max
godantic.MultipleOf(value)          // value is multiple of
```

### String
```go
godantic.MinLen(length)             // minimum length
godantic.MaxLen(length)             // maximum length
godantic.Regex(pattern)             // regex pattern match
godantic.Email()                    // email format
godantic.URL()                      // URL format
godantic.ContentEncoding(encoding)  // e.g., "base64"
godantic.ContentMediaType(type)     // e.g., "application/json"
```

### Arrays/Slices
```go
godantic.MinItems[T](count)         // minimum number of items
godantic.MaxItems[T](count)         // maximum number of items
godantic.UniqueItems[T]()           // all items must be unique
```

### Maps/Objects
```go
godantic.MinProperties(count)       // minimum properties
godantic.MaxProperties(count)       // maximum properties
```

### Value Constraints
```go
godantic.OneOf(value1, value2, ...) // enum - one of allowed values
godantic.Const(value)               // must equal exactly this value
godantic.Default(value)             // default value (schema only)
```

### Schema Metadata
```go
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

## How It Works

1. Define `Field{FieldName}()` methods that return `FieldOptions[T]`
2. Use `WithFieldOptions()` to compose validation constraints
3. Create a `Validator[T]` and call `Validate()`
4. Generate JSON Schema with `schema.NewGenerator[T]()`

Zero values (empty string, 0, nil) are treated as "not set" for required field checks.

## Examples

### Basic Validation

```go
validator := godantic.NewValidator[User]()

user := User{
    Email:    "test@example.com",
    Username: "john_doe",
    Age:      25,
}

if errs := validator.Validate(&user); len(errs) > 0 {
    for _, err := range errs {
        fmt.Println(err)
    }
}
```

### Schema Generation

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

func (u *User) FieldName() godantic.FieldOptions[string] {
    return godantic.WithFieldOptions(
        godantic.Required[string](),
        godantic.Description[string]("User's full name"),
        godantic.MinLen(2),
        godantic.MaxLen(50),
    )
}

sg := schema.NewGenerator[User]()
jsonSchema, _ := sg.GenerateJSON()
// Use with LLMs, API docs, etc.
```

## Testing

```bash
# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Run specific package tests
go test ./pkg/godantic -v
go test ./pkg/godantic/schema -v
```

## Comparison with Pydantic

| Feature | Pydantic (Python) | Godantic (Go) |
|---------|-------------------|---------------|
| Native types | Yes | Yes |
| Type safety | Runtime | Compile-time (generics) |
| Validation | Field descriptors | `Field{Name}()` methods |
| Required fields | `Field(...)` | `Required[T]()` |
| Schema generation | Built-in | Built-in (no tags) |

## Development

Run the example:
```bash
go run ./cmd
```

The library uses:
- Go generics for type safety
- Reflection for field discovery
- Convention-based method naming (`Field{Name}()`)
