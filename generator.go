package schemagen

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/lucasjones/reggen"
)

// Generator configuration for generating random JSON data
type Generator struct {
	MaxDepth          int
	Seed              int64
	rand              *rand.Rand
	faker             *gofakeit.Faker
	GenerateAllFields bool // If false, only generate required fields
}

// NewGenerator creates a new Generator with default settings
func NewGenerator() *Generator {
	seed := time.Now().UnixNano()
	return &Generator{
		MaxDepth:          10,
		Seed:              seed,
		rand:              rand.New(rand.NewSource(seed)),
		faker:             gofakeit.New(uint64(seed)),
		GenerateAllFields: false,
	}
}

// SetSeed sets a specific seed for deterministic generation
func (g *Generator) SetSeed(seed int64) *Generator {
	g.Seed = seed
	g.rand = rand.New(rand.NewSource(seed))
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

// Generate generates random JSON data that conforms to the provided schema
func (g *Generator) Generate(schemaJSON []byte) (interface{}, error) {
	schema, err := ParseSchema(schemaJSON)
	if err != nil {
		return nil, err
	}

	if err := schema.Validate(); err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	return g.generate(schema, 0)
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

	return g.generateWithContext(ctx, schema, 0)
}

// generate is the core recursive generation function
func (g *Generator) generate(schema *Schema, depth int) (interface{}, error) {
	return g.generateWithContext(context.Background(), schema, depth)
}

// generateWithContext is the core recursive generation function with context support
func (g *Generator) generateWithContext(ctx context.Context, schema *Schema, depth int) (interface{}, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("generation cancelled: %w", ctx.Err())
	default:
	}

	// Check depth limit
	if depth >= g.MaxDepth {
		return nil, fmt.Errorf("maximum recursion depth (%d) exceeded", g.MaxDepth)
	}

	// Handle const - must return exact value
	if schema.Const != nil {
		return schema.Const, nil
	}

	// Handle enum - pick one random value
	if len(schema.Enum) > 0 {
		return schema.Enum[g.rand.Intn(len(schema.Enum))], nil
	}

	// Handle composition keywords
	if len(schema.OneOf) > 0 {
		return g.handleOneOf(schema, depth)
	}

	if len(schema.AnyOf) > 0 {
		return g.handleAnyOf(schema, depth)
	}

	if len(schema.AllOf) > 0 {
		return g.handleAllOf(schema, depth)
	}

	// Handle type-based generation
	if !schema.Type.IsEmpty() {
		return g.generateByType(schema, depth)
	}

	// If no type specified, try to infer from other properties
	if schema.Properties != nil {
		return g.generateObject(schema, depth)
	}

	if schema.Items != nil {
		return g.generateArray(schema, depth)
	}

	// Default to generating an object if we have no other info
	return map[string]interface{}{}, nil
}

// generateByType generates data based on the type field
func (g *Generator) generateByType(schema *Schema, depth int) (interface{}, error) {
	types := schema.Type.GetTypes()

	// If multiple types, randomly choose one
	if len(types) > 1 {
		chosenType := types[g.rand.Intn(len(types))]
		modifiedSchema := *schema
		modifiedSchema.Type = StringOrArray{Single: chosenType, IsArray: false}
		return g.generateByType(&modifiedSchema, depth)
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
		return g.generateObject(schema, depth)
	case "array":
		return g.generateArray(schema, depth)
	case "null":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", typeName)
	}
}

// generateString generates a random string conforming to schema constraints
func (g *Generator) generateString(schema *Schema) (string, error) {
	// Check pattern first (highest priority)
	if schema.Pattern != "" {
		return g.generateStringFromPattern(schema.Pattern)
	}

	// Check format
	if schema.Format != "" {
		return g.generateStringFromFormat(schema.Format)
	}

	// Generate random string with length constraints
	minLen := 0
	maxLen := 20 // default max length

	if schema.MinLength != nil {
		minLen = *schema.MinLength
	}
	if schema.MaxLength != nil {
		maxLen = *schema.MaxLength
	}

	// Ensure min <= max
	if minLen > maxLen {
		maxLen = minLen
	}

	length := minLen
	if maxLen > minLen {
		length = minLen + g.rand.Intn(maxLen-minLen+1)
	}

	return g.randomString(length), nil
}

// generateStringFromPattern generates a string matching the regex pattern
func (g *Generator) generateStringFromPattern(pattern string) (string, error) {
	gen, err := reggen.NewGenerator(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	return gen.Generate(10), nil // limit to 10 attempts
}

// generateStringFromFormat generates a string based on the format keyword
func (g *Generator) generateStringFromFormat(format string) (string, error) {
	switch format {
	case "uuid":
		return g.faker.UUID(), nil
	case "email":
		return g.faker.Email(), nil
	case "date-time":
		return g.faker.Date().Format(time.RFC3339), nil
	case "date":
		return g.faker.Date().Format("2006-01-02"), nil
	case "time":
		return g.faker.Date().Format("15:04:05"), nil
	case "ipv4":
		return g.faker.IPv4Address(), nil
	case "ipv6":
		return g.faker.IPv6Address(), nil
	case "uri", "url":
		return g.faker.URL(), nil
	case "hostname":
		return g.faker.DomainName(), nil
	default:
		// For unsupported formats, generate a generic string
		return g.faker.Word(), nil
	}
}

// generateNumber generates a random number (integer or float) conforming to constraints
func (g *Generator) generateNumber(schema *Schema, isInteger bool) (interface{}, error) {
	var min, max float64

	// Determine minimum
	if schema.Minimum != nil {
		min = *schema.Minimum
	} else if schema.ExclusiveMinimum != nil {
		min = *schema.ExclusiveMinimum
		if isInteger {
			min = math.Ceil(min)
		}
	} else {
		if isInteger {
			min = 0
		} else {
			min = 0.0
		}
	}

	// Determine maximum
	if schema.Maximum != nil {
		max = *schema.Maximum
	} else if schema.ExclusiveMaximum != nil {
		max = *schema.ExclusiveMaximum
		if isInteger {
			max = math.Floor(max) - 1
		}
	} else {
		if isInteger {
			max = 1000
		} else {
			max = 1000.0
		}
	}

	// Ensure min <= max
	if min > max {
		return nil, fmt.Errorf("minimum (%f) is greater than maximum (%f)", min, max)
	}

	var result float64

	if isInteger {
		intMin := int64(math.Ceil(min))
		intMax := int64(math.Floor(max))

		if intMin > intMax {
			result = float64(intMin)
		} else {
			result = float64(intMin + g.rand.Int63n(intMax-intMin+1))
		}
	} else {
		result = min + g.rand.Float64()*(max-min)
	}

	// Handle multipleOf constraint
	if schema.MultipleOf != nil && *schema.MultipleOf > 0 {
		multiple := *schema.MultipleOf
		result = math.Round(result/multiple) * multiple

		// Ensure result is still within bounds
		if result < min {
			result += multiple
		}
		if result > max {
			result -= multiple
		}
	}

	if isInteger {
		return int64(result), nil
	}
	return result, nil
}

// generateBoolean generates a random boolean
func (g *Generator) generateBoolean() (bool, error) {
	return g.rand.Intn(2) == 1, nil
}

// generateObject generates a random object conforming to schema
func (g *Generator) generateObject(schema *Schema, depth int) (interface{}, error) {
	result := make(map[string]interface{})

	if schema.Properties == nil {
		return result, nil
	}

	// Track which fields are required
	requiredMap := make(map[string]bool)
	for _, fieldName := range schema.Required {
		requiredMap[fieldName] = true
	}

	// Generate properties
	for fieldName, fieldSchema := range schema.Properties {
		// Generate field if it's required or if we're generating all fields
		if requiredMap[fieldName] || g.GenerateAllFields {
			value, err := g.generate(fieldSchema, depth+1)
			if err != nil {
				return nil, fmt.Errorf("failed to generate field %s: %w", fieldName, err)
			}
			result[fieldName] = value
		}
	}

	// Handle additionalProperties if configured
	if schema.AdditionalProperties != nil && g.GenerateAllFields {
		switch ap := schema.AdditionalProperties.(type) {
		case bool:
			if ap {
				// Generate a few random additional properties
				numExtra := g.rand.Intn(3)
				for i := 0; i < numExtra; i++ {
					key := g.faker.Word()
					result[key] = g.faker.Word()
				}
			}
		case map[string]interface{}:
			// AdditionalProperties is a schema
			apBytes, _ := json.Marshal(ap)
			apSchema, err := ParseSchema(apBytes)
			if err == nil {
				numExtra := g.rand.Intn(3)
				for i := 0; i < numExtra; i++ {
					key := g.faker.Word()
					value, err := g.generate(apSchema, depth+1)
					if err == nil {
						result[key] = value
					}
				}
			}
		}
	}

	return result, nil
}

// generateArray generates a random array conforming to schema
func (g *Generator) generateArray(schema *Schema, depth int) (interface{}, error) {
	minItems := 0
	maxItems := 5 // default

	if schema.MinItems != nil {
		minItems = *schema.MinItems
	}
	if schema.MaxItems != nil {
		maxItems = *schema.MaxItems
	}

	// Ensure min <= max
	if minItems > maxItems {
		maxItems = minItems
	}

	length := minItems
	if maxItems > minItems {
		length = minItems + g.rand.Intn(maxItems-minItems+1)
	}

	result := make([]interface{}, length)

	// Handle items schema
	if schema.Items == nil {
		// No items schema, generate arbitrary values
		for i := 0; i < length; i++ {
			result[i] = g.faker.Word()
		}
		return result, nil
	}

	// Items can be a single schema or an array of schemas
	switch items := schema.Items.(type) {
	case map[string]interface{}:
		// Single schema for all items
		itemsBytes, _ := json.Marshal(items)
		itemSchema, err := ParseSchema(itemsBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse items schema: %w", err)
		}

		for i := 0; i < length; i++ {
			value, err := g.generate(itemSchema, depth+1)
			if err != nil {
				return nil, fmt.Errorf("failed to generate array item %d: %w", i, err)
			}
			result[i] = value
		}
	case []interface{}:
		// Tuple validation - array of schemas
		for i := 0; i < length; i++ {
			if i < len(items) {
				itemBytes, _ := json.Marshal(items[i])
				itemSchema, err := ParseSchema(itemBytes)
				if err != nil {
					return nil, fmt.Errorf("failed to parse items schema at index %d: %w", i, err)
				}
				value, err := g.generate(itemSchema, depth+1)
				if err != nil {
					return nil, fmt.Errorf("failed to generate array item %d: %w", i, err)
				}
				result[i] = value
			} else {
				// Beyond tuple length, generate generic values
				result[i] = g.faker.Word()
			}
		}
	default:
		return nil, fmt.Errorf("unsupported items type: %T", items)
	}

	return result, nil
}

// handleOneOf randomly selects one schema from oneOf and generates data
func (g *Generator) handleOneOf(schema *Schema, depth int) (interface{}, error) {
	if len(schema.OneOf) == 0 {
		return nil, fmt.Errorf("oneOf array is empty")
	}

	// Pick a random schema
	chosen := &schema.OneOf[g.rand.Intn(len(schema.OneOf))]
	return g.generate(chosen, depth)
}

// handleAnyOf randomly selects one schema from anyOf and generates data
func (g *Generator) handleAnyOf(schema *Schema, depth int) (interface{}, error) {
	if len(schema.AnyOf) == 0 {
		return nil, fmt.Errorf("anyOf array is empty")
	}

	// Pick a random schema
	chosen := &schema.AnyOf[g.rand.Intn(len(schema.AnyOf))]
	return g.generate(chosen, depth)
}

// handleAllOf attempts to merge all schemas (simplified: use first schema for MVP)
func (g *Generator) handleAllOf(schema *Schema, depth int) (interface{}, error) {
	if len(schema.AllOf) == 0 {
		return nil, fmt.Errorf("allOf array is empty")
	}

	// For MVP: generate from the first schema
	// A complete implementation would merge all constraints
	return g.generate(&schema.AllOf[0], depth)
}

// randomString generates a random string of specified length using realistic words
func (g *Generator) randomString(length int) string {
	if length <= 0 {
		return ""
	}

	// For short lengths, use letter string
	if length <= 3 {
		return g.faker.LetterN(uint(length))
	}

	// For longer strings, generate a sentence and truncate if needed
	result := g.faker.Word()

	// If we need more characters, add more words
	for len(result) < length {
		result += g.faker.Word()
	}

	// Truncate to exact length if needed
	if len(result) > length {
		return result[:length]
	}

	return result
}
