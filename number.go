package schemagen

import (
	"fmt"
	"math"
)

// generateNumber generates a random number (integer or float) conforming to constraints
func (g *Generator) generateNumber(schema *Schema, isInteger bool) (interface{}, error) {
	var min, max float64
	hasMin, hasMax := true, true

	// Determine minimum
	if schema.Minimum != nil {
		min = *schema.Minimum
	} else if schema.ExclusiveMinimum != nil {
		if isInteger {
			// exclusiveMinimum: 10 means > 10, so smallest integer is 11
			// exclusiveMinimum: 10.5 means > 10.5, so smallest integer is 11
			min = math.Floor(*schema.ExclusiveMinimum) + 1
		} else {
			min = *schema.ExclusiveMinimum
		}
	} else {
		hasMin = false
	}

	// Determine maximum
	if schema.Maximum != nil {
		max = *schema.Maximum
	} else if schema.ExclusiveMaximum != nil {
		if isInteger {
			// exclusiveMaximum: 20 means < 20, so largest integer is 19
			// exclusiveMaximum: 20.5 means < 20.5, so largest integer is 20
			max = math.Ceil(*schema.ExclusiveMaximum) - 1
		} else {
			max = *schema.ExclusiveMaximum
		}
	} else {
		hasMax = false
	}

	// Default the unset side relative to the set one so one-sided schemas
	// (e.g. only maximum: -100 or only minimum: 5000) stay satisfiable.
	switch {
	case !hasMin && !hasMax:
		min, max = 0, 1000
	case !hasMin:
		min = max - 1000
	case !hasMax:
		max = min + 1000
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
			result = float64(intMin + g.rand.Int64N(intMax-intMin+1))
		}
	} else {
		result = min + g.rand.Float64()*(max-min)
	}

	// Handle multipleOf constraint
	if schema.MultipleOf != nil && *schema.MultipleOf > 0 {
		multiple := *schema.MultipleOf
		result = math.Round(result/multiple) * multiple

		// Ensure result is still within bounds using a loop
		for result < min {
			result += multiple
		}
		for result > max {
			result -= multiple
		}

		// If no valid multiple exists in the range, return an error
		if result < min || result > max {
			return nil, fmt.Errorf("no multiple of %f exists in range [%f, %f]", multiple, min, max)
		}
	}

	if isInteger {
		return int64(result), nil
	}
	return result, nil
}
