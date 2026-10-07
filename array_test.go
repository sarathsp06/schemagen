package schemagen

import "testing"

func genArray(t *testing.T, seed int64, schemaJSON string) []interface{} {
	t.Helper()
	v, err := NewGenerator().SetSeed(seed).Generate([]byte(schemaJSON))
	if err != nil {
		t.Fatalf("seed %d: %v", seed, err)
	}
	arr, ok := v.([]interface{})
	if !ok {
		t.Fatalf("seed %d: expected array, got %T", seed, v)
	}
	return arr
}

func TestPrefixItemsWithRestItems(t *testing.T) {
	schema := `{
		"type": "array",
		"prefixItems": [{"type": "string"}, {"type": "integer"}],
		"items": {"type": "boolean"},
		"minItems": 4
	}`
	for seed := range int64(20) {
		arr := genArray(t, seed, schema)
		if len(arr) < 4 {
			t.Fatalf("seed %d: len %d < minItems 4", seed, len(arr))
		}
		if _, ok := arr[0].(string); !ok {
			t.Errorf("seed %d: arr[0] = %T, want string", seed, arr[0])
		}
		if _, ok := arr[1].(int64); !ok {
			t.Errorf("seed %d: arr[1] = %T, want int64", seed, arr[1])
		}
		for i, v := range arr[2:] {
			if _, ok := v.(bool); !ok {
				t.Errorf("seed %d: arr[%d] = %T, want bool", seed, i+2, v)
			}
		}
	}
}

func TestPrefixItemsTruncatedByMaxItems(t *testing.T) {
	schema := `{
		"type": "array",
		"prefixItems": [{"type": "string"}, {"type": "string"}, {"type": "string"}, {"type": "string"}],
		"maxItems": 2
	}`
	for seed := range int64(20) {
		arr := genArray(t, seed, schema)
		if len(arr) > 2 {
			t.Errorf("seed %d: len %d > maxItems 2", seed, len(arr))
		}
	}
}

func TestContainsMinMax(t *testing.T) {
	schema := `{
		"type": "array",
		"contains": {"const": "X"},
		"minContains": 2,
		"maxContains": 3,
		"items": {"const": "Y"},
		"minItems": 5
	}`
	for seed := range int64(20) {
		arr := genArray(t, seed, schema)
		if len(arr) < 5 {
			t.Fatalf("seed %d: len %d < minItems 5", seed, len(arr))
		}
		count := 0
		for _, v := range arr {
			switch v {
			case "X":
				count++
			case "Y":
			default:
				t.Fatalf("seed %d: unexpected value %v", seed, v)
			}
		}
		if count < 2 || count > 3 {
			t.Errorf("seed %d: contains count %d, want [2,3]", seed, count)
		}
	}
}

func TestContainsMinZero(t *testing.T) {
	schema := `{
		"type": "array",
		"contains": {"const": "X"},
		"minContains": 0,
		"items": {"type": "string"},
		"maxItems": 3
	}`
	for seed := range int64(20) {
		genArray(t, seed, schema) // zero contains-items is fine; just no error
	}
}

func TestUniqueItemsIntegers(t *testing.T) {
	schema := `{
		"type": "array",
		"items": {"type": "integer", "minimum": 0, "maximum": 1000},
		"minItems": 5,
		"maxItems": 8,
		"uniqueItems": true
	}`
	for seed := range int64(20) {
		arr := genArray(t, seed, schema)
		seen := make(map[string]bool, len(arr))
		for _, v := range arr {
			key := jsonKey(v)
			if seen[key] {
				t.Errorf("seed %d: duplicate item %v", seed, v)
			}
			seen[key] = true
		}
	}
}

func TestMinContainsExceedsMaxItems(t *testing.T) {
	schema := `{
		"type": "array",
		"contains": {"const": "X"},
		"minContains": 4,
		"maxItems": 2
	}`
	if _, err := NewGenerator().SetSeed(1).Generate([]byte(schema)); err == nil {
		t.Fatal("expected error for minContains > maxItems")
	}
}
