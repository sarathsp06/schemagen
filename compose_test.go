package schemagen

import (
	"testing"
)

func asFloat(t *testing.T, v interface{}) float64 {
	t.Helper()
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	}
	t.Fatalf("not a number: %T (%v)", v, v)
	return 0
}

func asObject(t *testing.T, v interface{}) map[string]interface{} {
	t.Helper()
	m, ok := v.(map[string]interface{})
	if !ok {
		t.Fatalf("not an object: %T (%v)", v, v)
	}
	return m
}

func TestAllOfNumericBounds(t *testing.T) {
	schema := []byte(`{"type":"integer","allOf":[{"minimum":10},{"maximum":20}]}`)
	for seed := range int64(20) {
		v, err := NewGenerator().SetSeed(seed).Generate(schema)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		n, ok := v.(int64)
		if !ok {
			t.Fatalf("seed %d: expected int64, got %T (%v)", seed, v, v)
		}
		if n < 10 || n > 20 {
			t.Errorf("seed %d: %d out of [10,20]", seed, n)
		}
	}
}

func TestAllOfObjectMerge(t *testing.T) {
	schema := []byte(`{
		"allOf": [
			{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]},
			{"properties":{"b":{"type":"integer"}},"required":["b"]}
		]
	}`)
	for seed := range int64(20) {
		v, err := NewGenerator().SetSeed(seed).Generate(schema)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		m := asObject(t, v)
		if _, ok := m["a"].(string); !ok {
			t.Errorf("seed %d: missing/wrong-type required property a: %v", seed, m["a"])
		}
		if _, ok := m["b"]; !ok {
			t.Errorf("seed %d: missing required property b", seed)
		}
	}
}

func TestOneOfSiblingConstraints(t *testing.T) {
	schema := []byte(`{
		"type":"object",
		"required":["id"],
		"properties":{"id":{"type":"string"}},
		"oneOf":[
			{"properties":{"a":{"const":1}},"required":["a"]},
			{"properties":{"b":{"const":2}},"required":["b"]}
		]
	}`)
	for seed := range int64(20) {
		v, err := NewGenerator().SetSeed(seed).Generate(schema)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		m := asObject(t, v)
		if _, ok := m["id"].(string); !ok {
			t.Errorf("seed %d: missing/wrong-type sibling required id: %v", seed, m["id"])
		}
		a, hasA := m["a"]
		b, hasB := m["b"]
		if hasA == hasB {
			t.Errorf("seed %d: expected exactly one branch key, got a=%v b=%v", seed, hasA, hasB)
			continue
		}
		if hasA && asFloat(t, a) != 1 {
			t.Errorf("seed %d: a = %v, want 1", seed, a)
		}
		if hasB && asFloat(t, b) != 2 {
			t.Errorf("seed %d: b = %v, want 2", seed, b)
		}
	}
}

func TestAnyOfSiblingType(t *testing.T) {
	schema := []byte(`{"type":"integer","anyOf":[{"minimum":100},{"maximum":-100}]}`)
	for seed := range int64(20) {
		v, err := NewGenerator().SetSeed(seed).Generate(schema)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		n, ok := v.(int64)
		if !ok {
			t.Fatalf("seed %d: expected int64, got %T (%v)", seed, v, v)
		}
		if n < 100 && n > -100 {
			t.Errorf("seed %d: %d satisfies neither branch", seed, n)
		}
	}
}

func TestIfThen(t *testing.T) {
	schema := []byte(`{
		"type":"object",
		"properties":{"country":{"const":"US"},"zip":{"type":"string"}},
		"required":["country"],
		"if":{"properties":{"country":{"const":"US"}}},
		"then":{"required":["zip"]}
	}`)
	for seed := range int64(20) {
		v, err := NewGenerator().SetSeed(seed).Generate(schema)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		m := asObject(t, v)
		if m["country"] != "US" {
			t.Errorf("seed %d: country = %v, want US", seed, m["country"])
		}
		if _, ok := m["zip"].(string); !ok {
			t.Errorf("seed %d: then-required zip missing or not string: %v", seed, m["zip"])
		}
	}
}

func TestIfWithoutThen(t *testing.T) {
	schema := []byte(`{
		"type":"object",
		"properties":{"x":{"type":"boolean"}},
		"if":{"properties":{"x":{"const":true}},"required":["x"]}
	}`)
	for seed := range int64(20) {
		v, err := NewGenerator().SetSeed(seed).Generate(schema)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		m := asObject(t, v)
		// if-branch merged in: x is required and const true
		if m["x"] != true {
			t.Errorf("seed %d: x = %v, want true", seed, m["x"])
		}
	}
}

func TestAllOfStringLength(t *testing.T) {
	schema := []byte(`{"type":"string","allOf":[{"minLength":5},{"maxLength":8}]}`)
	for seed := range int64(20) {
		v, err := NewGenerator().SetSeed(seed).Generate(schema)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		s, ok := v.(string)
		if !ok {
			t.Fatalf("seed %d: expected string, got %T", seed, v)
		}
		if n := len([]rune(s)); n < 5 || n > 8 {
			t.Errorf("seed %d: len(%q) = %d, want [5,8]", seed, s, n)
		}
	}
}

func TestMergeSchemasTypeIntersection(t *testing.T) {
	g := NewGenerator()
	base := &Schema{Type: StringOrArray{Multiple: []string{"string", "integer"}, IsArray: true}}
	overlay := &Schema{Type: StringOrArray{Multiple: []string{"integer", "number"}, IsArray: true}}
	got := g.mergeSchemas(base, overlay)
	if got.Type.IsArray || got.Type.Single != "integer" {
		t.Errorf("type intersection = %v, want integer", got.Type.GetTypes())
	}

	// empty intersection → overlay wins
	disjoint := g.mergeSchemas(
		&Schema{Type: StringOrArray{Single: "string"}},
		&Schema{Type: StringOrArray{Single: "integer"}},
	)
	if disjoint.Type.Single != "integer" {
		t.Errorf("disjoint type = %v, want overlay integer", disjoint.Type.GetTypes())
	}

	// purity: inputs untouched
	if len(base.Type.GetTypes()) != 2 || overlay.Type.GetTypes()[0] != "integer" {
		t.Error("mergeSchemas mutated its inputs")
	}
}
