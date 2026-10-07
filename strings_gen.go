package schemagen

import (
	"fmt"
	"time"

	"github.com/lucasjones/reggen"
)

// generateString generates a random string conforming to schema constraints.
// Precedence: x-faker template → registered custom format → pattern →
// built-in format → length-constrained random string.
func (g *Generator) generateString(schema *Schema) (string, error) {
	if schema.XFaker != "" {
		if s, err := g.faker.Generate(schema.XFaker); err == nil {
			return s, nil
		}
		// bad template: fall through to other strategies
	}

	if schema.Format != "" {
		if fn, ok := g.formats[schema.Format]; ok {
			return fn(g.faker), nil
		}
	}

	if schema.Pattern != "" {
		return g.patternWithLength(schema)
	}

	if schema.Format != "" {
		// ponytail: formats ignore min/maxLength — format wins; constrain via pattern if both matter.
		if s, ok := g.builtinFormat(schema.Format); ok {
			return s, nil
		}
		if g.StrictFormats {
			return "", fmt.Errorf("unknown string format %q", schema.Format)
		}
		// unknown format: fall through to length-constrained random
	}

	return g.randomString(g.pickLength(schema)), nil
}

// pickLength chooses a length satisfying minLength/maxLength (default max 20).
func (g *Generator) pickLength(schema *Schema) int {
	minLen := 0
	maxLen := 20
	if schema.MinLength != nil {
		minLen = *schema.MinLength
	}
	if schema.MaxLength != nil {
		maxLen = *schema.MaxLength
	}
	if minLen > maxLen {
		maxLen = minLen
	}
	if maxLen > minLen {
		return minLen + g.rand.IntN(maxLen-minLen+1)
	}
	return minLen
}

// patternWithLength generates from pattern, retrying when min/maxLength are violated.
func (g *Generator) patternWithLength(schema *Schema) (string, error) {
	s, err := g.generateStringFromPattern(schema.Pattern)
	if err != nil {
		return "", err
	}
	okLen := func(s string) bool {
		if schema.MinLength != nil && len(s) < *schema.MinLength {
			return false
		}
		if schema.MaxLength != nil && len(s) > *schema.MaxLength {
			return false
		}
		return true
	}
	for range 50 {
		if okLen(s) {
			return s, nil
		}
		if s, err = g.generateStringFromPattern(schema.Pattern); err != nil {
			return "", err
		}
	}
	// ponytail: best-effort — pattern+length may be unsatisfiable; return last attempt.
	return s, nil
}

// generateStringFromPattern generates a string matching the regex pattern,
// caching compiled reggen generators per pattern.
func (g *Generator) generateStringFromPattern(pattern string) (string, error) {
	if g.patternCache == nil {
		g.patternCache = make(map[string]*reggen.Generator)
	}
	gen, ok := g.patternCache[pattern]
	if !ok {
		var err error
		gen, err = reggen.NewGenerator(pattern)
		if err != nil {
			return "", fmt.Errorf("invalid regex pattern: %w", err)
		}
		g.patternCache[pattern] = gen
	}
	return gen.Generate(10), nil // limit unbounded quantifiers to 10 repeats
}

// builtinFormat returns a value for a known built-in format, or ok=false.
func (g *Generator) builtinFormat(format string) (string, bool) {
	switch format {
	case "uuid":
		return g.faker.UUID(), true
	case "email", "idn-email":
		return g.faker.Email(), true
	case "date-time":
		return g.faker.Date().Format(time.RFC3339), true
	case "date":
		return g.faker.Date().Format("2006-01-02"), true
	case "time":
		return g.faker.Date().UTC().Format("15:04:05Z07:00"), true
	case "ipv4":
		return g.faker.IPv4Address(), true
	case "ipv6":
		return g.faker.IPv6Address(), true
	case "uri", "url", "iri":
		return g.faker.URL(), true
	case "hostname", "idn-hostname":
		return g.faker.DomainName(), true
	case "duration":
		return g.isoDuration(), true
	case "uri-reference", "iri-reference":
		return g.uriReference(), true
	case "json-pointer":
		return g.jsonPointer(), true
	case "relative-json-pointer":
		return fmt.Sprintf("%d%s", g.rand.IntN(5), g.jsonPointer()), true
	case "regex":
		return fmt.Sprintf("[a-z]+[0-9]{%d}", 1+g.rand.IntN(5)), true
	default:
		return "", false
	}
}

// isoDuration builds a valid ISO 8601 duration like PT3H25M10S or P2DT4H.
func (g *Generator) isoDuration() string {
	s := "P"
	if g.rand.IntN(2) == 1 {
		s += fmt.Sprintf("%dD", 1+g.rand.IntN(30))
	}
	s += fmt.Sprintf("T%dH%dM%dS", g.rand.IntN(24), g.rand.IntN(60), g.rand.IntN(60))
	return s
}

// uriReference builds a relative reference like /path/to?x=1, or a full URL.
func (g *Generator) uriReference() string {
	if g.rand.IntN(3) == 0 {
		return g.faker.URL()
	}
	s := "/" + g.faker.Word() + "/" + g.faker.Word()
	if g.rand.IntN(2) == 1 {
		s += "?" + g.faker.Word() + "=" + g.faker.Word()
	}
	return s
}

// jsonPointer builds a pointer like /foo/0/bar.
func (g *Generator) jsonPointer() string {
	s := ""
	for range 1 + g.rand.IntN(3) {
		if g.rand.IntN(3) == 0 {
			s += fmt.Sprintf("/%d", g.rand.IntN(10))
		} else {
			s += "/" + g.faker.Word()
		}
	}
	return s
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

	// For longer strings, generate words with spaces for readability
	result := g.faker.Word()

	// If we need more characters, add more words with spaces
	for len(result) < length {
		result += " " + g.faker.Word()
	}

	// Truncate to exact length if needed
	if len(result) > length {
		return result[:length]
	}

	return result
}
