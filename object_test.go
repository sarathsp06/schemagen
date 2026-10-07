package schemagen

import (
	"regexp"
	"testing"
)

func genObj(t *testing.T, g *Generator, schema string) map[string]interface{} {
	t.Helper()
	v, err := g.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	obj, ok := v.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object, got %T", v)
	}
	return obj
}

const optSchema = `{
	"type": "object",
	"properties": {
		"a": {"type": "string"},
		"b": {"type": "string"},
		"c": {"type": "string"}
	},
	"required": ["a"]
}`

func TestOptionalProbability(t *testing.T) {
	t.Run("zero means only required", func(t *testing.T) {
		for seed := range int64(20) {
			g := NewGenerator().SetSeed(seed).SetOptionalProbability(0)
			obj := genObj(t, g, optSchema)
			if len(obj) != 1 {
				t.Fatalf("seed %d: expected only required key, got %v", seed, obj)
			}
			if _, ok := obj["a"]; !ok {
				t.Fatalf("seed %d: required key a missing: %v", seed, obj)
			}
		}
	})

	t.Run("one means all declared", func(t *testing.T) {
		for seed := range int64(20) {
			g := NewGenerator().SetSeed(seed).SetOptionalProbability(1)
			obj := genObj(t, g, optSchema)
			for _, k := range []string{"a", "b", "c"} {
				if _, ok := obj[k]; !ok {
					t.Fatalf("seed %d: key %s missing: %v", seed, k, obj)
				}
			}
		}
	})

	t.Run("half includes and excludes across seeds", func(t *testing.T) {
		included, excluded := false, false
		for seed := range int64(200) {
			g := NewGenerator().SetSeed(seed).SetOptionalProbability(0.5)
			obj := genObj(t, g, optSchema)
			if _, ok := obj["a"]; !ok {
				t.Fatalf("seed %d: required key a missing", seed)
			}
			if _, ok := obj["b"]; ok {
				included = true
			} else {
				excluded = true
			}
		}
		if !included || !excluded {
			t.Fatalf("expected both inclusion and exclusion of optional b: included=%v excluded=%v", included, excluded)
		}
	})
}

func TestDependentRequired(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"credit_card": {"type": "string"},
			"billing_address": {"type": "string"}
		},
		"required": ["credit_card"],
		"dependentRequired": {"credit_card": ["billing_address"]}
	}`
	for seed := range int64(20) {
		g := NewGenerator().SetSeed(seed)
		obj := genObj(t, g, schema)
		if _, ok := obj["billing_address"]; !ok {
			t.Fatalf("seed %d: credit_card present without billing_address: %v", seed, obj)
		}
	}
}

func TestDependentSchemas(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {"name": {"type": "string"}},
		"required": ["name"],
		"dependentSchemas": {
			"name": {
				"properties": {"age": {"type": "integer"}},
				"required": ["age"]
			}
		}
	}`
	for seed := range int64(20) {
		g := NewGenerator().SetSeed(seed)
		obj := genObj(t, g, schema)
		if _, ok := obj["age"]; !ok {
			t.Fatalf("seed %d: dependent schema required property age missing: %v", seed, obj)
		}
	}
}

func TestPatternProperties(t *testing.T) {
	schema := `{
		"type": "object",
		"patternProperties": {"^S_": {"type": "string"}}
	}`
	re := regexp.MustCompile(`^S_`)
	for seed := range int64(20) {
		g := NewGenerator().SetSeed(seed).SetGenerateAllFields(true)
		obj := genObj(t, g, schema)
		found := false
		for k, v := range obj {
			if re.MatchString(k) {
				found = true
				if _, ok := v.(string); !ok {
					t.Fatalf("seed %d: key %s value not a string: %T", seed, k, v)
				}
			}
		}
		if !found {
			t.Fatalf("seed %d: no key matching ^S_ generated: %v", seed, obj)
		}
	}
}

func TestPropertyNames(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {"Declared": {"type": "string"}},
		"required": ["Declared"],
		"propertyNames": {"pattern": "^[a-z]+$"},
		"additionalProperties": {"type": "string"}
	}`
	re := regexp.MustCompile(`^[a-z]+$`)
	for seed := range int64(20) {
		g := NewGenerator().SetSeed(seed).SetGenerateAllFields(true)
		obj := genObj(t, g, schema)
		for k := range obj {
			if k == "Declared" {
				continue
			}
			if !re.MatchString(k) {
				t.Fatalf("seed %d: invented key %q does not match propertyNames pattern", seed, k)
			}
		}
	}
}

func TestMinMaxProperties(t *testing.T) {
	t.Run("minProperties tops up from optionals", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"a": {"type": "string"},
				"b": {"type": "string"},
				"c": {"type": "string"}
			},
			"required": ["a"],
			"minProperties": 3
		}`
		for seed := range int64(20) {
			g := NewGenerator().SetSeed(seed)
			obj := genObj(t, g, schema)
			if len(obj) < 3 {
				t.Fatalf("seed %d: expected >= 3 properties, got %v", seed, obj)
			}
		}
	})

	t.Run("maxProperties trims optionals", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {
				"a": {"type": "string"},
				"b": {"type": "string"},
				"c": {"type": "string"}
			},
			"required": ["a"],
			"maxProperties": 1
		}`
		for seed := range int64(20) {
			g := NewGenerator().SetSeed(seed).SetGenerateAllFields(true)
			obj := genObj(t, g, schema)
			if len(obj) != 1 {
				t.Fatalf("seed %d: expected exactly 1 property, got %v", seed, obj)
			}
			if _, ok := obj["a"]; !ok {
				t.Fatalf("seed %d: required key a dropped: %v", seed, obj)
			}
		}
	})

	t.Run("maxProperties below required count errors", func(t *testing.T) {
		schema := `{
			"type": "object",
			"properties": {"a": {"type": "string"}, "b": {"type": "string"}},
			"required": ["a", "b"],
			"maxProperties": 1
		}`
		g := NewGenerator().SetSeed(1)
		if _, err := g.Generate([]byte(schema)); err == nil {
			t.Fatal("expected error when required count exceeds maxProperties")
		}
	})

	t.Run("minProperties unsatisfiable errors", func(t *testing.T) {
		schema := `{
			"type": "object",
			"minProperties": 2,
			"additionalProperties": false
		}`
		g := NewGenerator().SetSeed(1)
		if _, err := g.Generate([]byte(schema)); err == nil {
			t.Fatal("expected error when minProperties cannot be satisfied")
		}
	})
}
