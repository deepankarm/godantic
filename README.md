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
- Support for both simple unions (anyOf) and discriminated unions (oneOf)
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

### Union Types

Support for unions and discriminated unions with flexible type definitions:

```go
// 1. Union - accepts both primitive type names (strings) and complex Go types
//    Primitive types: "string", "integer", "number", "boolean", "object", "array", "null"

// Simple primitive union
type Config struct {
    Value any  // Can be string, integer, or object
}

func (c *Config) FieldValue() godantic.FieldOptions[any] {
    return godantic.Field(
        godantic.Union[any]("string", "integer", "object"),
        godantic.Description[any]("Can be a string, number, or object"),
    )
}

// Union with complex types (structs, slices)
type QueryPayload struct {
    Query any  // Can be string or []TextInput or []ImageInput
}

func (q *QueryPayload) FieldQuery() godantic.FieldOptions[any] {
    return godantic.Field(
        godantic.Union[any]("", []TextInput{}, []ImageInput{}),  // "" = string type
        godantic.Description[any]("Query can be string or array of inputs"),
    )
}

// Mix primitive and complex types
type MixedData struct {
    Data any  // Can be string, integer, or []CustomStruct
}

func (m *MixedData) FieldData() godantic.FieldOptions[any] {
    return godantic.Field(
        godantic.Union[any]("string", "integer", []CustomStruct{}),
        godantic.Description[any]("Flexible data field"),
    )
}

// 2. DiscriminatedUnion - type determined by discriminator field
type Response struct {
    Animal any  // Can be Cat, Dog, or Bird
}

func (r *Response) FieldAnimal() godantic.FieldOptions[any] {
    return godantic.Field(
        godantic.DiscriminatedUnion[any]("type", map[string]any{
            "cat":  Cat{},
            "dog":  Dog{},
            "bird": Bird{},
        }),
    )
}
```

All generate proper `anyOf` or `oneOf` with references in JSON Schema.

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
