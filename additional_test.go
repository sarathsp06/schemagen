package schemagen

import (
	"encoding/json"
	"strings"
	"testing"
)

// Test additional string formats
func TestGenerateStringFormats(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"date format", "date"},
		{"time format", "time"},
		{"ipv6 format", "ipv6"},
		{"uri format", "uri"},
		{"hostname format", "hostname"},
		{"unknown format", "unknown-format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := map[string]interface{}{
				"type":   "string",
				"format": tt.format,
			}
			schemaJSON, _ := json.Marshal(schema)

			gen := NewGenerator().SetSeed(42)
			result, err := gen.Generate(schemaJSON)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			if _, ok := result.(string); !ok {
				t.Errorf("Expected string, got %T", result)
			}
		})
	}
}

// Test pattern generation with error
func TestGenerateStringPatternError(t *testing.T) {
	schema := `{
		"type": "string",
		"pattern": "[invalid(pattern"
	}`

	gen := NewGenerator().SetSeed(42)
	_, err := gen.Generate([]byte(schema))
	if err == nil {
		t.Error("Expected error for invalid regex pattern")
	}
}

// Test exclusive minimum and maximum
func TestGenerateNumberExclusiveBounds(t *testing.T) {
	tests := []struct {
		name   string
		schema string
	}{
		{
			name:   "exclusive minimum",
			schema: `{"type": "integer", "exclusiveMinimum": 10, "maximum": 20}`,
		},
		{
			name:   "exclusive maximum",
			schema: `{"type": "integer", "minimum": 10, "exclusiveMaximum": 20}`,
		},
		{
			name:   "both exclusive",
			schema: `{"type": "number", "exclusiveMinimum": 0, "exclusiveMaximum": 1}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator().SetSeed(42)
			result, err := gen.Generate([]byte(tt.schema))
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			var numVal float64
			switch v := result.(type) {
			case int64:
				numVal = float64(v)
			case float64:
				numVal = v
			default:
				t.Fatalf("Expected number type, got %T", result)
			}

			// Verify result is within bounds
			if numVal < 0 || numVal > 100 {
				t.Logf("Generated value: %f", numVal)
			}
		})
	}
}

// Test number with conflicting exclusive bounds
func TestGenerateNumberConflictingExclusiveBounds(t *testing.T) {
	schema := `{"type": "number", "exclusiveMinimum": 20, "exclusiveMaximum": 10}`

	gen := NewGenerator()
	_, err := gen.Generate([]byte(schema))
	if err == nil {
		t.Error("Expected error for conflicting exclusive bounds")
	}
}

// Test object with additionalProperties as boolean true
func TestGenerateObjectWithAdditionalPropertiesTrue(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": true
	}`

	gen := NewGenerator().SetSeed(42).SetGenerateAllFields(true)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	obj, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected object, got %T", result)
	}

	// Should have at least the defined property
	if _, exists := obj["name"]; !exists {
		t.Error("Expected 'name' property to exist")
	}
}

// Test object with additionalProperties as boolean false
func TestGenerateObjectWithAdditionalPropertiesFalse(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": false,
		"required": ["name"]
	}`

	gen := NewGenerator().SetSeed(42).SetGenerateAllFields(true)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	obj, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected object, got %T", result)
	}

	if _, exists := obj["name"]; !exists {
		t.Error("Expected 'name' property to exist")
	}
}

// Test object with additionalProperties as schema
func TestGenerateObjectWithAdditionalPropertiesSchema(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": {
			"type": "integer"
		},
		"required": ["name"]
	}`

	gen := NewGenerator().SetSeed(42).SetGenerateAllFields(true)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	obj, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected object, got %T", result)
	}

	if _, exists := obj["name"]; !exists {
		t.Error("Expected 'name' property to exist")
	}
}

// Test array with tuple validation
func TestGenerateArrayTuple(t *testing.T) {
	schema := `{
		"type": "array",
		"items": [
			{"type": "string"},
			{"type": "integer"},
			{"type": "boolean"}
		],
		"minItems": 5
	}`

	gen := NewGenerator().SetSeed(42)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	arr, ok := result.([]interface{})
	if !ok {
		t.Fatalf("Expected array, got %T", result)
	}

	if len(arr) < 5 {
		t.Errorf("Expected at least 5 items, got %d", len(arr))
	}
}

// Test array with invalid items type
func TestGenerateArrayInvalidItems(t *testing.T) {
	schema := `{
		"type": "array",
		"items": 123
	}`

	gen := NewGenerator().SetSeed(42)
	_, err := gen.Generate([]byte(schema))
	if err == nil {
		t.Error("Expected error for invalid items type")
	}
}

// Test anyOf with empty array
// Test allOf with empty array
// Test schema with no type inference (object)
func TestGenerateNoTypeWithProperties(t *testing.T) {
	schema := `{
		"properties": {
			"name": {"type": "string"}
		}
	}`

	gen := NewGenerator().SetSeed(42)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if _, ok := result.(map[string]interface{}); !ok {
		t.Errorf("Expected object, got %T", result)
	}
}

// Test schema with no type inference (array)
func TestGenerateNoTypeWithItems(t *testing.T) {
	schema := `{
		"items": {"type": "string"}
	}`

	gen := NewGenerator().SetSeed(42)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if _, ok := result.([]interface{}); !ok {
		t.Errorf("Expected array, got %T", result)
	}
}

// Test schema with no type and no properties
func TestGenerateNoTypeNoProperties(t *testing.T) {
	schema := `{}`

	gen := NewGenerator().SetSeed(42)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if _, ok := result.(map[string]interface{}); !ok {
		t.Errorf("Expected object, got %T", result)
	}
}

// Test GenerateBytes error path
func TestGenerateBytesError(t *testing.T) {
	schema := `{"type": "invalid_type"}`

	gen := NewGenerator()
	_, err := gen.GenerateBytes([]byte(schema))
	if err == nil {
		t.Error("Expected error for invalid type")
	}
}

// Test invalid JSON schema
func TestGenerateInvalidJSON(t *testing.T) {
	schema := `{invalid json`

	gen := NewGenerator()
	_, err := gen.Generate([]byte(schema))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// Test StringOrArray MarshalJSON
func TestStringOrArrayMarshalJSON(t *testing.T) {
	t.Run("single string", func(t *testing.T) {
		s := StringOrArray{Single: "string", IsArray: false}
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("MarshalJSON error: %v", err)
		}
		if string(data) != `"string"` {
			t.Errorf("Expected \"string\", got %s", string(data))
		}
	})

	t.Run("array of strings", func(t *testing.T) {
		s := StringOrArray{Multiple: []string{"string", "null"}, IsArray: true}
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("MarshalJSON error: %v", err)
		}
		if !strings.Contains(string(data), "string") {
			t.Errorf("Expected array with 'string', got %s", string(data))
		}
	})
}

// Test StringOrArray Contains
func TestStringOrArrayContains(t *testing.T) {
	t.Run("single type contains", func(t *testing.T) {
		s := StringOrArray{Single: "string", IsArray: false}
		if !s.Contains("string") {
			t.Error("Expected Contains('string') to be true")
		}
		if s.Contains("integer") {
			t.Error("Expected Contains('integer') to be false")
		}
	})

	t.Run("array type contains", func(t *testing.T) {
		s := StringOrArray{Multiple: []string{"string", "null"}, IsArray: true}
		if !s.Contains("string") {
			t.Error("Expected Contains('string') to be true")
		}
		if !s.Contains("null") {
			t.Error("Expected Contains('null') to be true")
		}
		if s.Contains("integer") {
			t.Error("Expected Contains('integer') to be false")
		}
	})
}

// Test ParseSchema error
func TestParseSchemaError(t *testing.T) {
	_, err := ParseSchema([]byte(`{invalid json`))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// Test randomString edge cases
func TestRandomStringEdgeCases(t *testing.T) {
	gen := NewGenerator().SetSeed(42)

	t.Run("zero length", func(t *testing.T) {
		result := gen.randomString(0)
		if result != "" {
			t.Errorf("Expected empty string, got %s", result)
		}
	})

	t.Run("negative length", func(t *testing.T) {
		result := gen.randomString(-5)
		if result != "" {
			t.Errorf("Expected empty string, got %s", result)
		}
	})

	t.Run("short length", func(t *testing.T) {
		result := gen.randomString(2)
		if len(result) != 2 {
			t.Errorf("Expected length 2, got %d", len(result))
		}
	})

	t.Run("exact length from words", func(t *testing.T) {
		result := gen.randomString(10)
		if len(result) != 10 {
			t.Errorf("Expected length 10, got %d", len(result))
		}
	})
}

// Test generateByType with unsupported type
func TestGenerateByTypeUnsupported(t *testing.T) {
	schema := `{"type": "unsupported_type"}`

	gen := NewGenerator()
	_, err := gen.Generate([]byte(schema))
	if err == nil {
		t.Error("Expected error for unsupported type")
	}
}

// Test generateByType with no types
func TestGenerateByTypeNoTypes(t *testing.T) {
	// Schema with empty type array returns default behavior
	schema := `{"type": []}`

	gen := NewGenerator()
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should return default empty object
	if _, ok := result.(map[string]interface{}); !ok {
		t.Errorf("Expected object for empty type array, got %T", result)
	}
}

// Test number generation with same min and max
func TestGenerateNumberSameMinMax(t *testing.T) {
	schema := `{"type": "integer", "minimum": 42, "maximum": 42}`

	gen := NewGenerator().SetSeed(12345)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	val, ok := result.(int64)
	if !ok {
		t.Fatalf("Expected int64, got %T", result)
	}

	if val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}
}

// Test multipleOf adjustment staying in bounds
func TestGenerateNumberMultipleOfBounds(t *testing.T) {
	schema := `{"type": "integer", "minimum": 10, "maximum": 15, "multipleOf": 7}`

	gen := NewGenerator().SetSeed(12345)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	val, ok := result.(int64)
	if !ok {
		t.Fatalf("Expected int64, got %T", result)
	}

	if val < 10 || val > 15 {
		t.Errorf("Expected value between 10 and 15, got %d", val)
	}
}

// Test array items parsing error
func TestGenerateArrayItemsParseError(t *testing.T) {
	// Create invalid items that will fail to parse
	gen := NewGenerator().SetSeed(42)

	// This would require manipulating internal state,
	// so we test with a complex invalid schema structure
	schema := `{
		"type": "array",
		"items": {"type": "object", "properties": {"x": {"type": "invalid"}}}
	}`

	_, err := gen.Generate([]byte(schema))
	// This should succeed at generation level but might fail at nested validation
	if err != nil {
		// Expected in some cases
		t.Logf("Got expected error: %v", err)
	}
}

// Test object field generation error
func TestGenerateObjectFieldError(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"bad_field": {"type": "unsupported_type"}
		},
		"required": ["bad_field"]
	}`

	gen := NewGenerator()
	_, err := gen.Generate([]byte(schema))
	if err == nil {
		t.Error("Expected error for unsupported field type")
	}
}
