# SchemaGen - JSON Schema Generator for Go

A robust, performant Golang library for generating random, schema-compliant JSON data from JSON Schema definitions. Perfect for testing, fuzzing, and mock data generation.

## Features

- ✅ **JSON Schema Compliant**: Supports Draft 2020-12 and Draft-07
- ✅ **Realistic Fake Data**: Uses [gofakeit](https://github.com/brianvoe/gofakeit) for generating realistic mock data
- ✅ **Deterministic Generation**: Seedable random generation for reproducible results
- ✅ **Type Safe**: Strong typing with Go structs
- ✅ **Comprehensive**: Supports most JSON Schema keywords
- ✅ **Configurable**: Control depth limits, field generation, and more
- ✅ **Well Tested**: Extensive test coverage

## Installation

```bash
go get github.com/sarathsp06/schemagen
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/sarathsp06/schemagen"
)

func main() {
    schema := `{
        "type": "object",
        "properties": {
            "name": {"type": "string", "minLength": 3},
            "email": {"type": "string", "format": "email"},
            "age": {"type": "integer", "minimum": 18, "maximum": 100}
        },
        "required": ["name", "email"]
    }`
    
    gen := schemagen.NewGenerator()
    result, err := gen.Generate([]byte(schema))
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("%+v\n", result)
    // Output: map[age:42 email:john.doe@example.com name:Alice]
}
```

## Configuration

### Basic Configuration

```go
gen := schemagen.NewGenerator().
    SetSeed(12345).                  // For deterministic output
    SetMaxDepth(10).                  // Limit recursion depth
    SetGenerateAllFields(true)        // Generate optional fields too
```

### Generator Options

| Option | Default | Description |
|--------|---------|-------------|
| `SetSeed(int64)` | Current timestamp | Set seed for deterministic generation |
| `SetMaxDepth(int)` | 10 | Maximum recursion depth for nested objects |
| `SetGenerateAllFields(bool)` | false | Generate all fields vs. only required ones |

## Supported JSON Schema Keywords

### Type Keywords

| Keyword | Support | Description |
|---------|---------|-------------|
| `type` | ✅ | Single or array of types: `string`, `number`, `integer`, `boolean`, `object`, `array`, `null` |
| `enum` | ✅ | Pick random value from enumerated list |
| `const` | ✅ | Return exact constant value |

### String Keywords

| Keyword | Support | Example |
|---------|---------|---------|
| `minLength` | ✅ | `{"type": "string", "minLength": 5}` |
| `maxLength` | ✅ | `{"type": "string", "maxLength": 10}` |
| `pattern` | ✅ | `{"type": "string", "pattern": "^[0-9]{5}$"}` |
| `format` | ✅ | See [Supported Formats](#supported-formats) |

### Number Keywords

| Keyword | Support | Example |
|---------|---------|---------|
| `minimum` | ✅ | `{"type": "integer", "minimum": 0}` |
| `maximum` | ✅ | `{"type": "integer", "maximum": 100}` |
| `exclusiveMinimum` | ✅ | `{"type": "number", "exclusiveMinimum": 0}` |
| `exclusiveMaximum` | ✅ | `{"type": "number", "exclusiveMaximum": 1}` |
| `multipleOf` | ✅ | `{"type": "integer", "multipleOf": 5}` |

### Object Keywords

| Keyword | Support | Example |
|---------|---------|---------|
| `properties` | ✅ | Define object fields with schemas |
| `required` | ✅ | List of required field names |
| `additionalProperties` | ✅ | Allow extra properties (boolean or schema) |

### Array Keywords

| Keyword | Support | Example |
|---------|---------|---------|
| `items` | ✅ | Schema for array items (single or tuple) |
| `minItems` | ✅ | `{"type": "array", "minItems": 2}` |
| `maxItems` | ✅ | `{"type": "array", "maxItems": 10}` |

### Composition Keywords

| Keyword | Support | Behavior |
|---------|---------|----------|
| `oneOf` | ✅ | Randomly select one sub-schema |
| `anyOf` | ✅ | Randomly select one sub-schema |
| `allOf` | ✅ | Generate from first schema (MVP) |

### Supported Formats

The library uses [gofakeit](https://github.com/brianvoe/gofakeit) to generate realistic data for these formats:

| Format | Example Output |
|--------|----------------|
| `uuid` | `550e8400-e29b-41d4-a716-446655440000` |
| `email` | `john.doe@example.com` |
| `date-time` | `2023-10-15T14:30:00Z` |
| `date` | `2023-10-15` |
| `time` | `14:30:00` |
| `ipv4` | `192.168.1.1` |
| `ipv6` | `2001:0db8:85a3:0000:0000:8a2e:0370:7334` |
| `uri` / `url` | `https://example.com/path` |
| `hostname` | `example.com` |

## Usage Examples

### Generate Complex Nested Objects

```go
schema := `{
    "type": "object",
    "properties": {
        "user": {
            "type": "object",
            "properties": {
                "id": {"type": "string", "format": "uuid"},
                "name": {"type": "string", "minLength": 3},
                "email": {"type": "string", "format": "email"},
                "age": {"type": "integer", "minimum": 18, "maximum": 120}
            },
            "required": ["id", "name", "email"]
        },
        "tags": {
            "type": "array",
            "items": {"type": "string"},
            "minItems": 1,
            "maxItems": 5
        },
        "active": {"type": "boolean"}
    },
    "required": ["user", "active"]
}`

gen := schemagen.NewGenerator()
result, _ := gen.Generate([]byte(schema))
```

### Generate JSON Bytes

```go
gen := schemagen.NewGenerator()
jsonBytes, err := gen.GenerateBytes([]byte(schema))
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(jsonBytes))
```

### Using Enums and Constants

```go
schema := `{
    "type": "object",
    "properties": {
        "status": {
            "type": "string",
            "enum": ["active", "inactive", "pending"]
        },
        "version": {
            "type": "string",
            "const": "1.0.0"
        }
    }
}`
```

### Pattern-Based Strings

```go
schema := `{
    "type": "object",
    "properties": {
        "zipcode": {
            "type": "string",
            "pattern": "^[0-9]{5}$"
        },
        "phone": {
            "type": "string",
            "pattern": "^\\+1-[0-9]{3}-[0-9]{3}-[0-9]{4}$"
        }
    }
}`
```

### Deterministic Generation for Testing

```go
func TestMyFunction(t *testing.T) {
    gen := schemagen.NewGenerator().SetSeed(12345)
    
    // Generate same data every time for reproducible tests
    result1, _ := gen.Generate([]byte(schema))
    
    // Reset with same seed
    gen.SetSeed(12345)
    result2, _ := gen.Generate([]byte(schema))
    
    // result1 and result2 will be identical
}
```

### Composition with OneOf

```go
schema := `{
    "oneOf": [
        {
            "type": "object",
            "properties": {
                "type": {"const": "user"},
                "username": {"type": "string"}
            }
        },
        {
            "type": "object",
            "properties": {
                "type": {"const": "admin"},
                "adminId": {"type": "integer"}
            }
        }
    ]
}`
```

## Error Handling

The library validates schemas and returns errors for:

- Invalid JSON Schema syntax
- Conflicting constraints (e.g., `minimum > maximum`)
- Maximum recursion depth exceeded
- Unsupported schema features

```go
gen := schemagen.NewGenerator().SetMaxDepth(3)
result, err := gen.Generate([]byte(deeplyNestedSchema))
if err != nil {
    // Handle error: might be depth exceeded or invalid schema
    log.Printf("Generation failed: %v", err)
}
```

## API Reference

### Generator

```go
type Generator struct {
    MaxDepth          int
    Seed              int64
    GenerateAllFields bool
}
```

#### Methods

##### `NewGenerator() *Generator`

Creates a new Generator with default settings.

##### `SetSeed(seed int64) *Generator`

Sets a specific seed for deterministic generation. Returns the generator for method chaining.

##### `SetMaxDepth(depth int) *Generator`

Sets the maximum recursion depth for nested objects. Returns the generator for method chaining.

##### `SetGenerateAllFields(all bool) *Generator`

Controls whether to generate all fields or just required ones. Returns the generator for method chaining.

##### `Generate(schemaJSON []byte) (interface{}, error)`

Generates random JSON data conforming to the schema. Returns a Go interface{} (typically map or slice).

##### `GenerateBytes(schemaJSON []byte) ([]byte, error)`

Generates random JSON data and returns it as JSON bytes.

### Schema

The `Schema` struct represents a parsed JSON Schema with full support for:
- Meta properties (type, title)
- Validation keywords (min/max, length, patterns)
- Structural properties (properties, items)
- Composition (oneOf, anyOf, allOf)

### Helper Functions

##### `ParseSchema(schemaJSON []byte) (*Schema, error)`

Parses a JSON Schema from bytes into a Schema struct.

## Implementation Details

### Recursion and Depth Limiting

The generator uses depth-first recursion to generate nested structures. The `MaxDepth` setting prevents stack overflow with deeply nested or circular schemas.

### Random Data Generation

- **Numbers**: Generated within specified bounds using `math/rand`
- **Strings**: Generated using [gofakeit](https://github.com/brianvoe/gofakeit) for realistic data
- **Patterns**: Uses [reggen](https://github.com/lucasjones/reggen) to generate strings matching regex patterns
- **Formats**: Uses gofakeit's built-in format generators (email, UUID, etc.)

### Constraint Satisfaction

The generator respects all specified constraints:
- String length bounds are strictly enforced
- Number ranges are inclusive/exclusive as specified
- MultipleOf constraints align generated numbers
- Required fields are always generated
- Enum/const values are exactly matched

## Limitations

### Current Limitations

- **$ref**: Reference resolution not yet implemented (future enhancement)
- **allOf**: Currently generates from first schema only (complete merge planned)
- **additionalProperties**: Limited support (generates 0-2 extra properties when enabled)

### Edge Cases

The library validates constraints and returns errors for impossible schemas:

```go
// This will return an error
schema := `{"type": "integer", "minimum": 100, "maximum": 10}`
```

## Testing

Run the test suite:

```bash
go test -v
```

Run tests with coverage:

```bash
go test -cover
```

## Dependencies

- [github.com/brianvoe/gofakeit/v7](https://github.com/brianvoe/gofakeit) - Realistic fake data generation
- [github.com/lucasjones/reggen](https://github.com/lucasjones/reggen) - Regex pattern string generation

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

### Development

1. Clone the repository
2. Run tests: `go test -v`
3. Make your changes
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - See LICENSE file for details

## Roadmap

Future enhancements planned:

- [ ] Full `$ref` and definitions support
- [ ] Complete `allOf` schema merging
- [ ] More format types (email variants, phone numbers, etc.)
- [ ] Custom format handlers
- [ ] Performance optimizations for large schemas
- [ ] CLI tool for generating test data

## Credits

Built with:
- [gofakeit](https://github.com/brianvoe/gofakeit) by Brian Voelker
- [reggen](https://github.com/lucasjones/reggen) by Lucas Jones
