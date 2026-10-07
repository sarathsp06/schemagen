# SchemaGen - JSON Schema Generator for Go

[![Go Report Card](https://goreportcard.com/badge/github.com/sarathsp06/schemagen)](https://goreportcard.com/report/github.com/sarathsp06/schemagen)
[![GoDoc](https://pkg.go.dev/badge/github.com/sarathsp06/schemagen.svg)](https://pkg.go.dev/github.com/sarathsp06/schemagen)
[![Go Version](https://img.shields.io/github/go-mod/go-version/sarathsp06/schemagen)](https://github.com/sarathsp06/schemagen/blob/main/go.mod)
[![License](https://img.shields.io/github/license/sarathsp06/schemagen)](https://github.com/sarathsp06/schemagen/blob/main/LICENSE)

A robust, performant Golang library for generating random, schema-compliant JSON data from JSON Schema definitions. Perfect for testing, fuzzing, and mock data generation.

## Features

- ✅ **JSON Schema Compliant**: Supports Draft 2020-12 and Draft-07, including `$ref`, `prefixItems`, `contains`, `patternProperties`, `dependentRequired`/`dependentSchemas`, and `if`/`then`
- ✅ **Realistic Fake Data**: Uses [gofakeit](https://github.com/brianvoe/gofakeit) for generating realistic mock data, plus `x-faker` templates and custom format handlers
- ✅ **Deterministic Generation**: Seedable random generation with sorted property iteration for reproducible results
- ✅ **Type Safe**: Strong typing with `json.RawMessage`-backed polymorphic fields
- ✅ **Comprehensive Validation**: Collects all schema errors via `ValidationErrors`, validates negative constraints, required-in-properties, and more
- ✅ **Context Support**: Full `context.Context` propagation for cancellation and timeouts
- ✅ **Configurable**: Optional-field probability, defaults/examples reuse, depth limits, strict formats, and more
- ✅ **Bulk + CLI**: `GenerateN` for many documents from one parse, and a `schemagen` CLI emitting NDJSON
- ✅ **Well Tested**: Unit, compliance (validated against [jsonschema/v6](https://github.com/santhosh-tekuri/jsonschema)), and fuzz tests

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
    SetMaxDepth(10).                 // Limit recursion depth
    SetGenerateAllFields(true)       // Generate optional fields too
```

### Generator Options

| Option | Default | Description |
|--------|---------|-------------|
| `SetSeed(int64)` | Current timestamp | Set seed for deterministic generation |
| `SetMaxDepth(int)` | 10 | Maximum recursion depth for nested objects |
| `SetGenerateAllFields(bool)` | false | Generate all fields vs. only required ones |
| `SetOptionalProbability(float64)` | unset | Per-optional-property inclusion probability in [0,1]; overrides `GenerateAllFields` |
| `SetUseDefaults(bool)` | false | Return the schema's `default` value when present |
| `SetUseExamples(bool)` | false | Return a random `examples` entry when present |
| `SetStrictFormats(bool)` | false | Error on unknown string formats instead of falling back to a random string |
| `SetLenientDepth(bool)` | false | Return minimal values (`{}`, `[]`, `""`, …) at the depth limit instead of erroring |
| `RegisterFormat(name, fn)` | — | Register or override a string format generator |

## Concurrency

A `Generator` instance is **not** safe for concurrent use by multiple goroutines. Each goroutine should create its own `Generator` instance:

```go
// Correct: each goroutine gets its own generator
for i := 0; i < 10; i++ {
    go func() {
        gen := schemagen.NewGenerator()
        result, _ := gen.Generate([]byte(schema))
        // ...
    }()
}
```

## Supported JSON Schema Keywords

### Type Keywords

| Keyword | Support | Description |
|---------|---------|-------------|
| `type` | ✅ | Single or array of types: `string`, `number`, `integer`, `boolean`, `object`, `array`, `null` |
| `enum` | ✅ | Pick random value from enumerated list |
| `const` | ✅ | Return exact constant value (including `null` and `false`) |
| `default` | ✅ | Returned when `SetUseDefaults(true)` |
| `examples` | ✅ | Random entry returned when `SetUseExamples(true)` |
| `$ref` | ✅ | Local JSON pointers (`#`, `#/$defs/…`, `#/definitions/…`, nested) |
| `x-faker` | ✅ | gofakeit template, e.g. `{"x-faker": "{firstname} {lastname}"}` |

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
| `patternProperties` | ✅ | Keys generated from the regex, values from the schema |
| `propertyNames` | ✅ | Constrains invented keys for `additionalProperties` |
| `minProperties` / `maxProperties` | ✅ | Object size kept within bounds |
| `dependentRequired` | ✅ | Dependent fields added transitively when triggers present |
| `dependentSchemas` | ✅ | Dependent schema applied when trigger present |

### Array Keywords

| Keyword | Support | Example |
|---------|---------|---------|
| `items` | ✅ | Schema for array items (single or draft-07 tuple) |
| `prefixItems` | ✅ | 2020-12 positional schemas; `items` applies to the rest |
| `contains` | ✅ | With `minContains` (default 1) and `maxContains` |
| `minItems` | ✅ | `{"type": "array", "minItems": 2}` |
| `maxItems` | ✅ | `{"type": "array", "maxItems": 10}` |
| `uniqueItems` | ✅ | `{"type": "array", "uniqueItems": true}` |

### Composition Keywords

| Keyword | Support | Behavior |
|---------|---------|----------|
| `oneOf` | ✅ | Randomly select one sub-schema, merged with sibling keywords |
| `anyOf` | ✅ | Randomly select one sub-schema, merged with sibling keywords |
| `allOf` | ✅ | Deep-merge all sub-schemas (tighter constraint wins) |
| `if` / `then` | ✅ | Always generates the `if`+`then` branch; `else` is never generated |
| `not` | ⚠️ | Parsed and stored but not enforced during generation |

### Supported Formats

The library uses [gofakeit](https://github.com/brianvoe/gofakeit) to generate realistic data for these formats:

| Format | Example Output |
|--------|----------------|
| `uuid` | `550e8400-e29b-41d4-a716-446655440000` |
| `email` / `idn-email` | `john.doe@example.com` |
| `date-time` | `2023-10-15T14:30:00Z` |
| `date` | `2023-10-15` |
| `time` | `14:30:00Z` |
| `duration` | `P3DT2H30M` |
| `ipv4` | `192.168.1.1` |
| `ipv6` | `2001:0db8:85a3:0000:0000:8a2e:0370:7334` |
| `uri` / `url` / `iri` | `https://example.com/path` |
| `uri-reference` / `iri-reference` | `/path/to/resource?x=1` |
| `hostname` / `idn-hostname` | `example.com` |
| `json-pointer` | `/foo/0/bar` |
| `relative-json-pointer` | `1/foo` |
| `regex` | `^[a-z]{3}$` |

Unknown formats fall back to a length-respecting random string, or error with `SetStrictFormats(true)`. Register your own with:

```go
gen.RegisterFormat("employee-id", func(f *gofakeit.Faker) string {
    return fmt.Sprintf("EMP-%04d", f.Number(0, 9999))
})
```

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

### Generate Many Documents at Once

```go
gen := schemagen.NewGenerator().SetSeed(42)
docs, err := gen.GenerateN([]byte(schema), 100) // parses/validates once
```

### CLI

```bash
go install github.com/sarathsp06/schemagen/cmd/schemagen@latest

schemagen -n 10 -seed 42 schema.json          # 10 NDJSON documents
echo '{"type":"integer"}' | schemagen -n 3 -  # schema from stdin
schemagen -pretty -all schema.json            # indented, all optional fields
```

Flags: `-n` count, `-seed` seed, `-all` all fields, `-p` optional-field probability, `-depth` max depth, `-pretty` indent, `-defaults` use schema defaults, `-lenient` minimal values at depth limit.

### Generate JSON Bytes

```go
gen := schemagen.NewGenerator()
jsonBytes, err := gen.GenerateBytes([]byte(schema))
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(jsonBytes))
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

### Generate from a Pre-Parsed Schema

```go
schema, err := schemagen.ParseSchema([]byte(`{
    "type": "object",
    "properties": {
        "name": {"type": "string"},
        "age": {"type": "integer", "minimum": 0, "maximum": 120}
    },
    "required": ["name"]
}`))
if err != nil {
    log.Fatal(err)
}

gen := schemagen.NewGenerator()
result, err := gen.GenerateFromSchema(schema)
```

### Context-Aware Generation

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

gen := schemagen.NewGenerator()
result, err := gen.GenerateWithContext(ctx, []byte(schema))
if err != nil {
    // May be context.DeadlineExceeded or context.Canceled
    log.Printf("Generation failed: %v", err)
}
```

## Error Handling

The library validates schemas and returns detailed errors for:

- Invalid JSON Schema syntax
- Conflicting constraints (e.g., `minimum > maximum`)
- Negative constraint values (e.g., negative `minLength`, `maxItems`)
- Non-positive `multipleOf`
- Required fields not defined in `properties`
- Impossible `multipleOf` ranges
- Maximum recursion depth exceeded
- Context cancellation and timeouts

### ValidationErrors

`Validate()` returns a `ValidationErrors` value (which implements the `error` interface) containing **all** validation errors found in the schema, not just the first one:

```go
schema, _ := schemagen.ParseSchema([]byte(`{
    "type": "object",
    "properties": {
        "bad_string": {"type": "string", "minLength": 10, "maxLength": 5},
        "bad_number": {"type": "number", "minimum": 100, "maximum": 50}
    }
}`))

err := schema.Validate()
if err != nil {
    // err is a ValidationErrors containing both constraint violations
    fmt.Println(err) // "2 validation errors: [minLength (10) > maxLength (5); minimum (100) > maximum (50)]"
}
```

```go
gen := schemagen.NewGenerator().SetMaxDepth(3)
result, err := gen.Generate([]byte(deeplyNestedSchema))
if err != nil {
    log.Printf("Generation failed: %v", err)
}
```

## Limitations

### Current Limitations

- **`$ref`**: Local JSON pointers only; external/URL references return an error
- **`not` keyword**: Parsed and stored but not enforced during generation
- **`else` branch**: Generation always satisfies the `if`+`then` branch; `else` output is never produced
- **Formats vs. length**: Built-in and custom format output ignores `minLength`/`maxLength` (format wins); `pattern` with length bounds is retried best-effort
- **`"$ref": ""`**: An explicitly empty `$ref` is indistinguishable from an absent one (Go zero value) and is treated as "no ref" rather than a root self-reference
- **Case-insensitive keywords**: Go's `encoding/json` matches struct fields case-insensitively, so misspelled keys like `prefiXItems` are honored instead of ignored (validators correctly ignore them)
- **`maxContains`**: Regular items may incidentally also match `contains`, so the actual match count can exceed `maxContains` — best-effort only

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


## Roadmap

Future enhancements planned:

- [ ] `not` keyword enforcement during generation
- [ ] External `$ref` resolution (URLs, files)
- [ ] `else` branch generation for `if`/`then`/`else`
- [ ] Performance optimizations for large schemas

## Credits
Built with:

- [gofakeit](https://github.com/brianvoe/gofakeit) by Brian Voelker
- [reggen](https://github.com/lucasjones/reggen) by Lucas Jones
