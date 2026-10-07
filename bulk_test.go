package schemagen

import (
	"context"
	"reflect"
	"testing"
)

var bulkSchema = []byte(`{
	"type": "object",
	"properties": {
		"id": {"type": "integer", "minimum": 1},
		"name": {"type": "string", "minLength": 3}
	},
	"required": ["id", "name"]
}`)

func TestGenerateN(t *testing.T) {
	g := NewGenerator().SetSeed(42)
	docs, err := g.GenerateN(bulkSchema, 5)
	if err != nil {
		t.Fatalf("GenerateN: %v", err)
	}
	if len(docs) != 5 {
		t.Fatalf("got %d docs, want 5", len(docs))
	}
	for i, d := range docs {
		obj, ok := d.(map[string]interface{})
		if !ok {
			t.Fatalf("doc %d: not an object: %T", i, d)
		}
		if _, ok := obj["id"]; !ok {
			t.Errorf("doc %d: missing required id", i)
		}
		if _, ok := obj["name"]; !ok {
			t.Errorf("doc %d: missing required name", i)
		}
	}
}

func TestGenerateNZero(t *testing.T) {
	for _, n := range []int{0, -3} {
		docs, err := NewGenerator().SetSeed(1).GenerateN(bulkSchema, n)
		if err != nil {
			t.Fatalf("n=%d: %v", n, err)
		}
		if len(docs) != 0 {
			t.Fatalf("n=%d: got %d docs, want 0", n, len(docs))
		}
	}
}

func TestGenerateNInvalidSchema(t *testing.T) {
	if _, err := NewGenerator().GenerateN([]byte(`{not json`), 2); err == nil {
		t.Fatal("expected parse error")
	}
	if _, err := NewGenerator().GenerateN([]byte(`{"type":"number","minimum":5,"maximum":1}`), 2); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestGenerateNDeterministic(t *testing.T) {
	a, err := NewGenerator().SetSeed(99).GenerateN(bulkSchema, 10)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewGenerator().SetSeed(99).GenerateN(bulkSchema, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same seed produced different output")
	}
}

func TestGenerateNWithContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewGenerator().SetSeed(1).GenerateNWithContext(ctx, bulkSchema, 3); err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
