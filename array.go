package schemagen

import (
	"context"
	"fmt"
)

// generateArray generates a random array conforming to schema, supporting
// prefixItems, items (single-schema rest or draft-07 tuple), contains with
// minContains/maxContains, and uniqueItems.
func (g *Generator) generateArray(ctx context.Context, schema *Schema, depth int) (interface{}, error) {
	single, tuple, err := schema.parseItems()
	if err != nil {
		return nil, fmt.Errorf("failed to parse items schema: %w", err)
	}

	// Positional schemas: 2020-12 prefixItems, falling back to draft-07
	// array-form items. The rest schema (single) applies beyond the prefix.
	prefix := schema.PrefixItems
	if len(prefix) == 0 && tuple != nil {
		prefix = tuple
	}

	minItems := 0
	maxItems := 5 // default
	if schema.MinItems != nil {
		minItems = *schema.MinItems
	}
	if schema.MaxItems != nil {
		maxItems = *schema.MaxItems
	}

	hasContains := schema.Contains != nil
	minContains := 1
	maxContains := -1 // unbounded
	if hasContains {
		if schema.MinContains != nil {
			minContains = *schema.MinContains
		}
		if schema.MaxContains != nil {
			maxContains = *schema.MaxContains
		}
		if schema.MaxItems != nil && minContains > *schema.MaxItems {
			return nil, fmt.Errorf("minContains (%d) cannot be greater than maxItems (%d)", minContains, *schema.MaxItems)
		}
		if minItems < minContains {
			minItems = minContains
		}
	}

	if minItems > maxItems {
		maxItems = minItems
	}

	length := minItems
	if maxItems > minItems {
		length = minItems + g.rand.IntN(maxItems-minItems+1)
	}

	// Pick how many items come from the contains schema and where they go.
	// ponytail: regular items may incidentally also match contains, which
	// could exceed maxContains; we don't validate that — best-effort only.
	containsAt := make(map[int]bool)
	if hasContains {
		hi := length
		if maxContains >= 0 && maxContains < hi {
			hi = maxContains
		}
		lo := min(minContains, hi)
		n := lo
		if hi > lo {
			n = lo + g.rand.IntN(hi-lo+1)
		}
		perm := g.rand.Perm(length)
		for i := range n {
			containsAt[perm[i]] = true
		}
	}

	// genAt generates the value for position i per its governing schema.
	genAt := func(i int) (interface{}, error) {
		switch {
		case containsAt[i]:
			// Contains items must also satisfy the positional/rest schema.
			base := single
			if i < len(prefix) {
				base = prefix[i]
			}
			return g.generateWithContext(ctx, g.mergeSchemas(base, schema.Contains), depth+1)
		case i < len(prefix):
			return g.generateWithContext(ctx, prefix[i], depth+1)
		case single != nil && tuple == nil:
			return g.generateWithContext(ctx, single, depth+1)
		default:
			return g.faker.Word(), nil
		}
	}

	isUnique := schema.UniqueItems != nil && *schema.UniqueItems
	seen := make(map[string]bool)
	maxAttempts := length * 10 // prevent infinite loops for uniqueItems

	result := make([]interface{}, 0, length)
	for i := range length {
		value, err := genAt(i)
		if err != nil {
			return nil, fmt.Errorf("failed to generate array item %d: %w", i, err)
		}
		if isUnique {
			key := jsonKey(value)
			for attempts := 0; seen[key] && attempts < maxAttempts; attempts++ {
				value, err = genAt(i)
				if err != nil {
					return nil, fmt.Errorf("failed to generate unique array item %d: %w", i, err)
				}
				key = jsonKey(value)
			}
			seen[key] = true
		}
		result = append(result, value)
	}

	return result, nil
}
