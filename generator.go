package schemagen

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/lucasjones/reggen"
)

// FormatFunc produces a string value for a custom string format.
type FormatFunc func(f *gofakeit.Faker) string

// Generator configuration for generating random JSON data.
//
// A Generator is NOT safe for concurrent use by multiple goroutines.
// Each goroutine should create its own Generator instance.
type Generator struct {
	MaxDepth          int
	Seed              int64
	rand              *rand.Rand
	faker             *gofakeit.Faker
	GenerateAllFields bool // If false, only generate required fields

	// OptionalProbability, when in [0, 1], includes each optional property
	// with this probability. A negative value (default) means unset, in which
	// case GenerateAllFields decides.
	OptionalProbability float64

	// UseDefaults returns the schema's "default" value when present.
	UseDefaults bool
	// UseExamples returns a random value from the schema's "examples" when present.
	UseExamples bool
	// StrictFormats makes unknown string formats an error instead of
	// falling back to a generic word.
	StrictFormats bool
	// LenientDepth emits a minimal valid value at the depth limit instead of
	// returning an error.
	LenientDepth bool

	formats      map[string]FormatFunc
	patternCache map[string]*reggen.Generator
	root         *Schema // root schema of the current generation, for $ref resolution
}

// NewGenerator creates a new Generator with default settings
func NewGenerator() *Generator {
	seed := time.Now().UnixNano()
	g := &Generator{
		MaxDepth:            10,
		GenerateAllFields:   false,
		OptionalProbability: -1,
		formats:             make(map[string]FormatFunc),
		patternCache:        make(map[string]*reggen.Generator),
	}
	g.SetSeed(seed)
	return g
}

// SetSeed sets a specific seed for deterministic generation
func (g *Generator) SetSeed(seed int64) *Generator {
	g.Seed = seed
	g.rand = rand.New(rand.NewPCG(uint64(seed), uint64(seed)))
	g.faker = gofakeit.New(uint64(seed))
	return g
}

// SetMaxDepth sets the maximum recursion depth
func (g *Generator) SetMaxDepth(depth int) *Generator {
	g.MaxDepth = depth
	return g
}

// SetGenerateAllFields controls whether to generate all fields or just required ones
func (g *Generator) SetGenerateAllFields(all bool) *Generator {
	g.GenerateAllFields = all
	return g
}

// SetOptionalProbability sets the probability ([0, 1]) that each optional
// property is included. Overrides GenerateAllFields when set.
func (g *Generator) SetOptionalProbability(p float64) *Generator {
	g.OptionalProbability = p
	return g
}

// SetUseDefaults makes generation return the schema's "default" value when present.
func (g *Generator) SetUseDefaults(use bool) *Generator {
	g.UseDefaults = use
	return g
}

// SetUseExamples makes generation return a random "examples" entry when present.
func (g *Generator) SetUseExamples(use bool) *Generator {
	g.UseExamples = use
	return g
}

// SetStrictFormats makes unknown string formats an error instead of a fallback word.
func (g *Generator) SetStrictFormats(strict bool) *Generator {
	g.StrictFormats = strict
	return g
}

// SetLenientDepth makes the generator emit a minimal valid value when the
// depth limit is reached instead of returning an error.
func (g *Generator) SetLenientDepth(lenient bool) *Generator {
	g.LenientDepth = lenient
	return g
}

// RegisterFormat registers a custom generator for a string "format" value.
// Registered formats take precedence over built-in ones.
func (g *Generator) RegisterFormat(name string, fn FormatFunc) *Generator {
	g.formats[name] = fn
	return g
}

// Generate generates random JSON data that conforms to the provided schema
func (g *Generator) Generate(schemaJSON []byte) (interface{}, error) {
	return g.GenerateWithContext(context.Background(), schemaJSON)
}

// GenerateFromSchema generates random JSON data from a pre-parsed Schema
func (g *Generator) GenerateFromSchema(schema *Schema) (interface{}, error) {
	if err := schema.Validate(); err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	g.root = schema
	return g.generateWithContext(context.Background(), schema, 0)
}

// GenerateBytes generates random JSON data and returns it as bytes
func (g *Generator) GenerateBytes(schemaJSON []byte) ([]byte, error) {
	result, err := g.Generate(schemaJSON)
	if err != nil {
		return nil, err
	}

	return json.Marshal(result)
}

// GenerateWithContext generates random JSON data with context support for cancellation
func (g *Generator) GenerateWithContext(ctx context.Context, schemaJSON []byte) (interface{}, error) {
	schema, err := ParseSchema(schemaJSON)
	if err != nil {
		return nil, err
	}

	if err := schema.Validate(); err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	g.root = schema
	return g.generateWithContext(ctx, schema, 0)
}

// generateWithContext is the core recursive generation function with context support
func (g *Generator) generateWithContext(ctx context.Context, schema *Schema, depth int) (interface{}, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("generation cancelled: %w", ctx.Err())
	default:
	}

	// Resolve $ref against the root schema
	if schema.Ref != "" {
		resolved, err := g.resolveRef(schema)
		if err != nil {
			return nil, err
		}
		schema = resolved
	}

	// Check depth limit
	if depth >= g.MaxDepth {
		if g.LenientDepth {
			return minimalValue(schema), nil
		}
		return nil, fmt.Errorf("maximum recursion depth (%d) exceeded", g.MaxDepth)
	}

	// Honor default/examples when configured
	if g.UseDefaults && len(schema.Default) > 0 {
		return unmarshalRaw(schema.Default)
	}
	if g.UseExamples && len(schema.Examples) > 0 {
		return unmarshalRaw(schema.Examples[g.rand.IntN(len(schema.Examples))])
	}

	// Handle const - must return exact value (including null and false)
	if len(schema.Const) > 0 {
		return unmarshalRaw(schema.Const)
	}

	// Handle enum - pick one random value
	if schema.Enum != nil {
		if len(schema.Enum) == 0 {
			return nil, fmt.Errorf("enum is empty: no valid value exists")
		}
		return schema.Enum[g.rand.IntN(len(schema.Enum))], nil
	}

	// Fold if/then/else into the schema
	schema = g.applyConditionals(schema)

	// Handle composition keywords
	if len(schema.OneOf) > 0 {
		return g.handleOneOf(ctx, schema, depth)
	}

	if len(schema.AnyOf) > 0 {
		return g.handleAnyOf(ctx, schema, depth)
	}

	if len(schema.AllOf) > 0 {
		return g.handleAllOf(ctx, schema, depth)
	}

	// Handle type-based generation
	if !schema.Type.IsEmpty() {
		return g.generateByType(ctx, schema, depth)
	}

	// If no type specified, try to infer from other properties
	if schema.Properties != nil || schema.PatternProperties != nil ||
		len(schema.Required) > 0 || schema.MinProperties != nil || schema.MaxProperties != nil ||
		schema.PropertyNames != nil || len(schema.DependentRequired) > 0 || len(schema.DependentSchemas) > 0 {
		return g.generateObject(ctx, schema, depth)
	}

	if len(schema.Items) > 0 || len(schema.PrefixItems) > 0 || schema.Contains != nil ||
		schema.MinItems != nil || schema.MaxItems != nil {
		return g.generateArray(ctx, schema, depth)
	}

	// Default to generating an object if we have no other info
	return map[string]interface{}{}, nil
}

// generateByType generates data based on the type field
func (g *Generator) generateByType(ctx context.Context, schema *Schema, depth int) (interface{}, error) {
	types := schema.Type.GetTypes()

	// If multiple types, randomly choose one
	if len(types) > 1 {
		chosenType := types[g.rand.IntN(len(types))]
		modifiedSchema := *schema
		modifiedSchema.Type = StringOrArray{Single: chosenType, IsArray: false}
		return g.generateByType(ctx, &modifiedSchema, depth)
	}

	if len(types) == 0 {
		return nil, fmt.Errorf("no type specified")
	}

	typeName := types[0]

	switch typeName {
	case "string":
		return g.generateString(schema)
	case "number":
		return g.generateNumber(schema, false)
	case "integer":
		return g.generateNumber(schema, true)
	case "boolean":
		return g.generateBoolean()
	case "object":
		return g.generateObject(ctx, schema, depth)
	case "array":
		return g.generateArray(ctx, schema, depth)
	case "null":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", typeName)
	}
}

// generateBoolean generates a random boolean
func (g *Generator) generateBoolean() (bool, error) {
	return g.rand.IntN(2) == 1, nil
}

// minimalValue returns the smallest valid value for a schema's primary type.
// Used in lenient depth mode; best-effort, ignores nested constraints.
func minimalValue(schema *Schema) interface{} {
	types := schema.Type.GetTypes()
	if len(types) == 0 {
		return nil
	}
	switch types[0] {
	case "string":
		return ""
	case "number":
		return 0.0
	case "integer":
		return int64(0)
	case "boolean":
		return false
	case "array":
		return []interface{}{}
	case "object":
		return map[string]interface{}{}
	default:
		return nil
	}
}

// unmarshalRaw decodes a raw JSON value into an interface{}
func unmarshalRaw(raw json.RawMessage) (interface{}, error) {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("invalid JSON value: %w", err)
	}
	return v, nil
}

// jsonKey returns a string representation of a value for use as a uniqueness key
func jsonKey(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
