# Godantic

**Validation and schema generation in one place.** Inspired by Python's [Pydantic](https://github.com/pydantic/pydantic), Godantic brings type-safe validation and automatic JSON Schema generation to Go — without struct tags.

Most Go validation libraries use struct tags, requiring separate tools for validation and schema generation. This creates duplication, limits flexibility, and makes testing difficult. Godantic solves this by defining both validation and schema in `Field{FieldName}()` methods:

```go
func (u *User) FieldEmail() godantic.FieldOptions[string] {
    return godantic.Field(
        godantic.Required[string](),
        godantic.Email(),
        godantic.Description[string]("Primary contact email"),
    )
}
```

**Benefits:**
- Single source of truth for validation + schema
- Full Go language power (conditionals, custom functions, external calls)
- Type-safe with generics (compile-time checks)
- Testable validation logic
- No tag parsing, easier debugging

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
        // handle error
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
    return godantic.Field(
        godantic.Required[string](),
        godantic.Description[string]("User's full name"),
        godantic.MinLen(2),
        godantic.MaxLen(50),
    )
}

sg := schema.NewGenerator[User]()
jsonSchema, err := sg.GenerateJSON()
```

## Testing

```bash
go test ./... -v -cover
```

---

> Disclaimer: Most of the code and tests are written by Cursor.
