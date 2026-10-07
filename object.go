package schemagen

import (
	"context"
	"fmt"
	"maps"
	"slices"
)

// generateObject generates a random object conforming to schema.
func (g *Generator) generateObject(ctx context.Context, schema *Schema, depth int) (interface{}, error) {
	result := make(map[string]interface{})

	requiredMap := make(map[string]bool, len(schema.Required))
	for _, name := range schema.Required {
		requiredMap[name] = true
	}

	apSchema, apAllowed, err := schema.parseAdditionalProperties()
	if err != nil {
		return nil, fmt.Errorf("failed to parse additionalProperties: %w", err)
	}
	apUnset := len(schema.AdditionalProperties) == 0
	apExplicitlyFalse := !apUnset && !apAllowed

	// includeOptional decides whether a single optional property is included.
	includeOptional := func() bool {
		if g.OptionalProbability >= 0 {
			return g.rand.Float64() < g.OptionalProbability
		}
		return g.GenerateAllFields
	}

	// genNamed generates a value for a named property: declared schema if any,
	// else the additionalProperties schema, else a plain word.
	genNamed := func(name string, local map[string]*Schema) (interface{}, error) {
		if local != nil {
			if ps, ok := local[name]; ok && ps != nil {
				return g.generateWithContext(ctx, ps, depth+1)
			}
		}
		if ps, ok := schema.Properties[name]; ok && ps != nil {
			return g.generateWithContext(ctx, ps, depth+1)
		}
		if apSchema != nil {
			return g.generateWithContext(ctx, apSchema, depth+1)
		}
		return g.faker.Word(), nil
	}

	// Declared properties in deterministic sorted order.
	propNames := slices.Sorted(maps.Keys(schema.Properties))
	for _, name := range propNames {
		if requiredMap[name] || includeOptional() {
			value, err := g.generateWithContext(ctx, schema.Properties[name], depth+1)
			if err != nil {
				return nil, fmt.Errorf("failed to generate field %s: %w", name, err)
			}
			result[name] = value
		}
	}

	// Required names not declared in properties (validation permits this when
	// patternProperties/additionalProperties could supply them).
	for _, name := range schema.Required {
		if _, ok := result[name]; ok {
			continue
		}
		value, err := genNamed(name, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to generate required field %s: %w", name, err)
		}
		result[name] = value
	}

	// dependentRequired: fixpoint — a generated dependent may itself trigger more.
	if len(schema.DependentRequired) > 0 {
		triggers := slices.Sorted(maps.Keys(schema.DependentRequired))
		for changed := true; changed; {
			changed = false
			for _, trigger := range triggers {
				if _, ok := result[trigger]; !ok {
					continue
				}
				for _, dep := range schema.DependentRequired[trigger] {
					if _, ok := result[dep]; ok {
						continue
					}
					value, err := genNamed(dep, nil)
					if err != nil {
						return nil, fmt.Errorf("failed to generate dependent field %s: %w", dep, err)
					}
					result[dep] = value
					changed = true
				}
			}
		}
	}

	// dependentSchemas: when the trigger key is present, generate the dependent
	// schema's required properties (its optionals follow the inclusion rule).
	if len(schema.DependentSchemas) > 0 {
		for _, trigger := range slices.Sorted(maps.Keys(schema.DependentSchemas)) {
			if _, ok := result[trigger]; !ok {
				continue
			}
			ds := schema.DependentSchemas[trigger]
			if ds == nil {
				continue
			}
			for _, name := range ds.Required {
				if _, ok := result[name]; ok {
					continue
				}
				value, err := genNamed(name, ds.Properties)
				if err != nil {
					return nil, fmt.Errorf("failed to generate dependent field %s: %w", name, err)
				}
				result[name] = value
			}
			dsRequired := make(map[string]bool, len(ds.Required))
			for _, name := range ds.Required {
				dsRequired[name] = true
			}
			for _, name := range slices.Sorted(maps.Keys(ds.Properties)) {
				if _, ok := result[name]; ok {
					continue
				}
				if dsRequired[name] || includeOptional() {
					value, err := g.generateWithContext(ctx, ds.Properties[name], depth+1)
					if err != nil {
						return nil, fmt.Errorf("failed to generate dependent field %s: %w", name, err)
					}
					result[name] = value
				}
			}
		}
	}

	generateExtras := g.GenerateAllFields || g.OptionalProbability > 0
	sortedPatterns := slices.Sorted(maps.Keys(schema.PatternProperties))

	// genPatternKey adds one fresh key matching pattern; reports success.
	genPatternKey := func(pattern string) (bool, error) {
		for range 10 {
			key, err := g.generateStringFromPattern(pattern)
			if err != nil {
				return false, nil // ponytail: unsupported regex → skip this pattern
			}
			if _, exists := result[key]; exists {
				continue
			}
			if _, declared := schema.Properties[key]; declared {
				continue
			}
			value, err := g.generateWithContext(ctx, schema.PatternProperties[pattern], depth+1)
			if err != nil {
				return false, fmt.Errorf("failed to generate patternProperties value for %q: %w", pattern, err)
			}
			result[key] = value
			return true, nil
		}
		return false, nil
	}

	if len(sortedPatterns) > 0 && generateExtras {
		for _, pattern := range sortedPatterns {
			n := 1 + g.rand.IntN(2)
			for range n {
				if _, err := genPatternKey(pattern); err != nil {
					return nil, err
				}
			}
		}
	}

	// inventKey produces a fresh additional-property key honoring propertyNames.
	inventKey := func() (string, bool, error) {
		for range 10 {
			var key string
			if schema.PropertyNames != nil {
				k, err := g.generateString(schema.PropertyNames)
				if err != nil {
					return "", false, fmt.Errorf("failed to generate property name: %w", err)
				}
				key = k
			} else {
				key = g.faker.Word()
			}
			if key == "" {
				continue
			}
			if _, exists := result[key]; exists {
				continue
			}
			if _, declared := schema.Properties[key]; declared {
				continue
			}
			return key, true, nil
		}
		return "", false, nil
	}

	// genAdditional adds one additionalProperties-style entry; reports success.
	genAdditional := func() (bool, error) {
		key, ok, err := inventKey()
		if err != nil || !ok {
			return false, err
		}
		if apSchema != nil {
			value, err := g.generateWithContext(ctx, apSchema, depth+1)
			if err != nil {
				return false, fmt.Errorf("failed to generate additional property %s: %w", key, err)
			}
			result[key] = value
		} else {
			result[key] = g.faker.Word()
		}
		return true, nil
	}

	// additionalProperties extras (only when explicitly set and allowed).
	if !apUnset && apAllowed && g.GenerateAllFields {
		numExtra := g.rand.IntN(3)
		for range numExtra {
			if _, err := genAdditional(); err != nil {
				return nil, err
			}
		}
	}

	// minProperties: top up from declared optionals, then patterns, then extras.
	if schema.MinProperties != nil {
		minProps := *schema.MinProperties
		for _, name := range propNames {
			if len(result) >= minProps {
				break
			}
			if _, ok := result[name]; ok {
				continue
			}
			value, err := g.generateWithContext(ctx, schema.Properties[name], depth+1)
			if err != nil {
				return nil, fmt.Errorf("failed to generate field %s: %w", name, err)
			}
			result[name] = value
		}
		for len(result) < minProps && len(sortedPatterns) > 0 {
			added := false
			for _, pattern := range sortedPatterns {
				if len(result) >= minProps {
					break
				}
				ok, err := genPatternKey(pattern)
				if err != nil {
					return nil, err
				}
				added = added || ok
			}
			if !added {
				break
			}
		}
		for len(result) < minProps {
			if apExplicitlyFalse {
				return nil, fmt.Errorf("cannot satisfy minProperties %d: additionalProperties is false and no other property source", minProps)
			}
			ok, err := genAdditional()
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("cannot satisfy minProperties %d: unable to invent more property keys", minProps)
			}
		}
	}

	// maxProperties: drop droppable keys; protected = required plus dependents
	// forced by present protected triggers.
	if schema.MaxProperties != nil {
		maxProps := *schema.MaxProperties
		if len(result) > maxProps {
			protected := make(map[string]bool, len(schema.Required))
			for _, name := range schema.Required {
				if _, ok := result[name]; ok {
					protected[name] = true
				}
			}
			for changed := true; changed; {
				changed = false
				for trigger := range protected {
					for _, dep := range schema.DependentRequired[trigger] {
						if _, ok := result[dep]; ok && !protected[dep] {
							protected[dep] = true
							changed = true
						}
					}
					if ds := schema.DependentSchemas[trigger]; ds != nil {
						for _, dep := range ds.Required {
							if _, ok := result[dep]; ok && !protected[dep] {
								protected[dep] = true
								changed = true
							}
						}
					}
				}
			}
			if len(protected) > maxProps {
				return nil, fmt.Errorf("maxProperties %d is less than %d required properties", maxProps, len(protected))
			}
			// Drop non-protected trigger keys first: once gone, every remaining
			// trigger is protected and so are its dependents, so later drops
			// cannot break dependentRequired/dependentSchemas.
			isTrigger := func(name string) bool {
				_, a := schema.DependentRequired[name]
				_, b := schema.DependentSchemas[name]
				return a || b
			}
			names := slices.Sorted(maps.Keys(result))
			for _, triggersFirst := range []bool{true, false} {
				for _, name := range names {
					if len(result) <= maxProps {
						break
					}
					if !protected[name] && isTrigger(name) == triggersFirst {
						delete(result, name)
					}
				}
			}
		}
	}

	return result, nil
}
