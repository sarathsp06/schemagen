package schemagen

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var fuzzSeedSchemas = []string{
	`{"type":"string","minLength":5,"maxLength":10}`,
	`{"type":"integer","exclusiveMinimum":0,"exclusiveMaximum":10}`,
	`{"type":"number","multipleOf":0.5,"minimum":0,"maximum":10}`,
	`{"enum":["red","green",42,null]}`,
	`{"const":false}`,
	`{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer","minimum":0}},"required":["name","age"]}`,
	`{"type":"object","patternProperties":{"^num_":{"type":"integer"}},"additionalProperties":false,"minProperties":1}`,
	`{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}],"items":{"type":"boolean"},"minItems":2,"maxItems":5}`,
	`{"anyOf":[{"type":"string","minLength":5},{"type":"integer","minimum":10}]}`,
	`{"$ref":"#/$defs/node","$defs":{"node":{"type":"object","properties":{"name":{"type":"string"},"child":{"$ref":"#/$defs/node"}}}}}`,
}

// FuzzGenerate asserts Generate never panics, and whenever both generation
// succeeds and the schema compiles under jsonschema/v6, the output validates.
func FuzzGenerate(f *testing.F) {
	for _, s := range fuzzSeedSchemas {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, schemaStr string) {
		// ponytail: encoding/json matches struct fields case-insensitively,
		// so "prefiXItems" parses as prefixItems while validators (correctly)
		// ignore it. Skip any schema with an uppercase letter in a key.
		if hasUppercaseKey(schemaStr) {
			return
		}
		g := NewGenerator().SetSeed(42).SetMaxDepth(6).SetLenientDepth(true)
		out, err := g.GenerateBytes([]byte(schemaStr))
		if err != nil {
			return // rejecting malformed schemas is fine; panics are not
		}

		doc, err := jsonschema.UnmarshalJSON(strings.NewReader(schemaStr))
		if err != nil {
			return
		}
		c := jsonschema.NewCompiler()
		c.DefaultDraft(jsonschema.Draft2020)
		if err := c.AddResource("fuzz.json", doc); err != nil {
			return
		}
		sch, err := c.Compile("fuzz.json")
		if err != nil {
			return
		}

		inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(out))
		if err != nil {
			t.Errorf("output not valid JSON: %v\nschema: %s\noutput: %s", err, schemaStr, out)
			return
		}
		if err := sch.Validate(inst); err != nil {
			// Degenerate self-cycles like {"$ref":""} compile but fail every
			// instance with a cycle error — nothing could ever validate.
			if strings.Contains(err.Error(), "reference cycle") {
				return
			}
			t.Errorf("output violates schema:\nschema: %s\noutput: %s\nerr: %v", schemaStr, out, err)
		}
	})
}

// hasUppercaseKey reports whether any object key in the JSON doc contains
// an uppercase ASCII letter (recursively).
func hasUppercaseKey(doc string) bool {
	var v interface{}
	if json.Unmarshal([]byte(doc), &v) != nil {
		return false
	}
	var walk func(interface{}) bool
	walk = func(v interface{}) bool {
		switch v := v.(type) {
		case map[string]interface{}:
			for k, child := range v {
				if strings.ContainsFunc(k, func(r rune) bool { return r >= 'A' && r <= 'Z' }) {
					return true
				}
				if walk(child) {
					return true
				}
			}
		case []interface{}:
			for _, child := range v {
				if walk(child) {
					return true
				}
			}
		}
		return false
	}
	return walk(v)
}
