package schemagen

import (
	"context"
	"fmt"
)

// GenerateN generates n random values conforming to the schema,
// parsing and validating the schema only once.
func (g *Generator) GenerateN(schemaJSON []byte, n int) ([]interface{}, error) {
	return g.GenerateNWithContext(context.Background(), schemaJSON, n)
}

// GenerateNWithContext is GenerateN with context support for cancellation.
func (g *Generator) GenerateNWithContext(ctx context.Context, schemaJSON []byte, n int) ([]interface{}, error) {
	schema, err := ParseSchema(schemaJSON)
	if err != nil {
		return nil, err
	}
	if err := schema.Validate(); err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	g.root = schema
	if n <= 0 {
		return []interface{}{}, nil
	}
	results := make([]interface{}, 0, n)
	for range n {
		v, err := g.generateWithContext(ctx, schema, 0)
		if err != nil {
			return nil, err
		}
		results = append(results, v)
	}
	return results, nil
}
