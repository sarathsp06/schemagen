package schemagen

import (
	"strings"
	"testing"
)

func refSchema(ref string) *Schema { return &Schema{Ref: ref} }

func strSchema(minLen int) *Schema {
	return &Schema{Type: StringOrArray{Single: "string"}, MinLength: &minLen}
}

func TestResolveRefPointers(t *testing.T) {
	root := &Schema{
		Properties: map[string]*Schema{
			"user": {
				Properties: map[string]*Schema{
					"name": strSchema(3),
				},
			},
		},
		Defs: map[string]*Schema{
			"str": strSchema(5),
			"a/b": strSchema(2),
			"a":   refSchema("#/$defs/b"),
			"b":   strSchema(4),
		},
		Definitions: map[string]*Schema{
			"legacy": strSchema(7),
		},
	}
	g := NewGenerator()
	g.root = root

	tests := []struct {
		name    string
		ref     string
		wantMin int
	}{
		{"defs", "#/$defs/str", 5},
		{"definitions", "#/definitions/legacy", 7},
		{"nested properties", "#/properties/user/properties/name", 3},
		{"tilde escape", "#/$defs/a~1b", 2},
		{"percent escape", "#/%24defs/str", 5},
		{"chained ref", "#/$defs/a", 4},
		{"root", "#", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := g.resolveRef(refSchema(tt.ref))
			if err != nil {
				t.Fatalf("resolveRef(%q): %v", tt.ref, err)
			}
			if tt.ref == "#" {
				if got != root {
					t.Fatalf("resolveRef(%q) != root", tt.ref)
				}
				return
			}
			if got.MinLength == nil || *got.MinLength != tt.wantMin {
				t.Fatalf("resolveRef(%q) minLength = %v, want %d", tt.ref, got.MinLength, tt.wantMin)
			}
		})
	}
}

func TestResolveRefErrors(t *testing.T) {
	root := &Schema{
		Defs: map[string]*Schema{
			"a": refSchema("#/$defs/a"),
		},
	}
	g := NewGenerator()
	g.root = root

	for _, ref := range []string{
		"#/$defs/missing",
		"http://example.com/schema.json#/$defs/a",
		"#/$defs/a", // self-loop: hop limit
		"#/nonsense/x",
	} {
		if _, err := g.resolveRef(refSchema(ref)); err == nil {
			t.Errorf("resolveRef(%q): want error, got nil", ref)
		} else if !strings.Contains(err.Error(), ref) && !strings.Contains(err.Error(), "#/$defs/a") {
			t.Errorf("resolveRef(%q): error %q does not mention the ref", ref, err)
		}
	}

	// No root set
	if _, err := (&Generator{}).resolveRef(refSchema("#")); err == nil {
		t.Error("resolveRef with nil root: want error, got nil")
	}
}

func TestResolveRefSiblingOverlay(t *testing.T) {
	g := NewGenerator()
	maxLen := 6
	g.root = &Schema{Defs: map[string]*Schema{"str": strSchema(5)}}

	got, err := g.resolveRef(&Schema{Ref: "#/$defs/str", MaxLength: &maxLen})
	if err != nil {
		t.Fatal(err)
	}
	if got.MinLength == nil || *got.MinLength != 5 {
		t.Errorf("overlay lost target minLength: %v", got.MinLength)
	}
	if got.MaxLength == nil || *got.MaxLength != 6 {
		t.Errorf("overlay missed sibling maxLength: %v", got.MaxLength)
	}
	// Target must not be mutated.
	if g.root.Defs["str"].MaxLength != nil {
		t.Error("overlay mutated the target schema")
	}
}

func TestGenerateWithDefsRef(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {"name": {"$ref": "#/$defs/str"}},
		"required": ["name"],
		"$defs": {"str": {"type": "string", "minLength": 5}}
	}`)
	for seed := range int64(20) {
		g := NewGenerator().SetSeed(seed)
		v, err := g.Generate(schemaJSON)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		obj, ok := v.(map[string]interface{})
		if !ok {
			t.Fatalf("seed %d: got %T, want object", seed, v)
		}
		name, ok := obj["name"].(string)
		if !ok || len(name) < 5 {
			t.Fatalf("seed %d: name = %#v, want string with minLength 5", seed, obj["name"])
		}
	}
}

func TestGenerateRecursiveRef(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"value": {"type": "string"},
			"children": {"type": "array", "minItems": 1, "items": {"$ref": "#"}}
		},
		"required": ["value", "children"]
	}`)

	g := NewGenerator().SetSeed(1).SetMaxDepth(4).SetLenientDepth(true)
	if _, err := g.Generate(schemaJSON); err != nil {
		t.Fatalf("lenient recursive generation failed: %v", err)
	}

	g = NewGenerator().SetSeed(1).SetMaxDepth(4)
	if _, err := g.Generate(schemaJSON); err == nil {
		t.Fatal("strict recursive generation: want depth error, got nil")
	}
}
