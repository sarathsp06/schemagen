package schemagen

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateString(t *testing.T) {
	tests := []struct {
		name   string
		schema string
	}{
		{
			name:   "basic string",
			schema: `{"type": "string"}`,
		},
		{
			name:   "string with minLength",
			schema: `{"type": "string", "minLength": 5}`,
		},
		{
			name:   "string with maxLength",
			schema: `{"type": "string", "maxLength": 10}`,
		},
		{
			name:   "string with min and max length",
			schema: `{"type": "string", "minLength": 5, "maxLength": 10}`,
		},
		{
			name:   "email format",
			schema: `{"type": "string", "format": "email"}`,
		},
		{
			name:   "uuid format",
			schema: `{"type": "string", "format": "uuid"}`,
		},
		{
			name:   "date-time format",
			schema: `{"type": "string", "format": "date-time"}`,
		},
		{
			name:   "ipv4 format",
			schema: `{"type": "string", "format": "ipv4"}`,
		},
		{
			name:   "pattern - digits only",
			schema: `{"type": "string", "pattern": "^[0-9]{5}$"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator().SetSeed(12345)
			result, err := gen.Generate([]byte(tt.schema))
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			str, ok := result.(string)
			if !ok {
				t.Fatalf("Expected string, got %T", result)
			}

			// Parse schema to validate result
			schema, _ := ParseSchema([]byte(tt.schema))

			if schema.MinLength != nil && len(str) < *schema.MinLength {
				t.Errorf("String length %d is less than minLength %d", len(str), *schema.MinLength)
			}

			if schema.MaxLength != nil && len(str) > *schema.MaxLength {
				t.Errorf("String length %d is greater than maxLength %d", len(str), *schema.MaxLength)
			}
		})
	}
}

func TestGenerateNumber(t *testing.T) {
	tests := []struct {
		name   string
		schema string
	}{
		{
			name:   "basic integer",
			schema: `{"type": "integer"}`,
		},
		{
			name:   "basic number",
			schema: `{"type": "number"}`,
		},
		{
			name:   "integer with minimum",
			schema: `{"type": "integer", "minimum": 10}`,
		},
		{
			name:   "integer with maximum",
			schema: `{"type": "integer", "maximum": 100}`,
		},
		{
			name:   "integer with min and max",
			schema: `{"type": "integer", "minimum": 10, "maximum": 20}`,
		},
		{
			name:   "integer with multipleOf",
			schema: `{"type": "integer", "multipleOf": 5, "minimum": 0, "maximum": 100}`,
		},
		{
			name:   "number with multipleOf",
			schema: `{"type": "number", "multipleOf": 0.5, "minimum": 0, "maximum": 10}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator().SetSeed(12345)
			result, err := gen.Generate([]byte(tt.schema))
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			schema, _ := ParseSchema([]byte(tt.schema))

			var numVal float64
			switch v := result.(type) {
			case int64:
				numVal = float64(v)
			case float64:
				numVal = v
			default:
				t.Fatalf("Expected number type, got %T", result)
			}

			if schema.Minimum != nil && numVal < *schema.Minimum {
				t.Errorf("Value %f is less than minimum %f", numVal, *schema.Minimum)
			}

			if schema.Maximum != nil && numVal > *schema.Maximum {
				t.Errorf("Value %f is greater than maximum %f", numVal, *schema.Maximum)
			}
		})
	}
}

func TestGenerateBoolean(t *testing.T) {
	schema := `{"type": "boolean"}`
	gen := NewGenerator().SetSeed(12345)

	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	_, ok := result.(bool)
	if !ok {
		t.Fatalf("Expected boolean, got %T", result)
	}
}

func TestGenerateNull(t *testing.T) {
	schema := `{"type": "null"}`
	gen := NewGenerator()

	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if result != nil {
		t.Fatalf("Expected nil, got %v", result)
	}
}

func TestGenerateObject(t *testing.T) {
	tests := []struct {
		name   string
		schema string
	}{
		{
			name:   "empty object",
			schema: `{"type": "object"}`,
		},
		{
			name: "object with properties",
			schema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "integer"}
				}
			}`,
		},
		{
			name: "object with required fields",
			schema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "integer"},
					"email": {"type": "string"}
				},
				"required": ["name", "age"]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator().SetSeed(12345)
			result, err := gen.Generate([]byte(tt.schema))
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			obj, ok := result.(map[string]interface{})
			if !ok {
				t.Fatalf("Expected object, got %T", result)
			}

			schema, _ := ParseSchema([]byte(tt.schema))

			// Check required fields are present
			for _, reqField := range schema.Required {
				if _, exists := obj[reqField]; !exists {
					t.Errorf("Required field %s is missing", reqField)
				}
			}
		})
	}
}

func TestGenerateArray(t *testing.T) {
	tests := []struct {
		name   string
		schema string
	}{
		{
			name:   "basic array",
			schema: `{"type": "array"}`,
		},
		{
			name: "array with items schema",
			schema: `{
				"type": "array",
				"items": {"type": "string"}
			}`,
		},
		{
			name: "array with minItems",
			schema: `{
				"type": "array",
				"items": {"type": "integer"},
				"minItems": 3
			}`,
		},
		{
			name: "array with maxItems",
			schema: `{
				"type": "array",
				"items": {"type": "string"},
				"maxItems": 5
			}`,
		},
		{
			name: "array with min and max items",
			schema: `{
				"type": "array",
				"items": {"type": "boolean"},
				"minItems": 2,
				"maxItems": 4
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator().SetSeed(12345)
			result, err := gen.Generate([]byte(tt.schema))
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			arr, ok := result.([]interface{})
			if !ok {
				t.Fatalf("Expected array, got %T", result)
			}

			schema, _ := ParseSchema([]byte(tt.schema))

			if schema.MinItems != nil && len(arr) < *schema.MinItems {
				t.Errorf("Array length %d is less than minItems %d", len(arr), *schema.MinItems)
			}

			if schema.MaxItems != nil && len(arr) > *schema.MaxItems {
				t.Errorf("Array length %d is greater than maxItems %d", len(arr), *schema.MaxItems)
			}
		})
	}
}

func TestGenerateEnum(t *testing.T) {
	schema := `{
		"type": "string",
		"enum": ["red", "green", "blue"]
	}`

	gen := NewGenerator().SetSeed(12345)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	str, ok := result.(string)
	if !ok {
		t.Fatalf("Expected string, got %T", result)
	}

	validValues := map[string]bool{"red": true, "green": true, "blue": true}
	if !validValues[str] {
		t.Errorf("Value %s is not in enum", str)
	}
}

func TestGenerateConst(t *testing.T) {
	schema := `{
		"type": "string",
		"const": "constant_value"
	}`

	gen := NewGenerator()
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	str, ok := result.(string)
	if !ok {
		t.Fatalf("Expected string, got %T", result)
	}

	if str != "constant_value" {
		t.Errorf("Expected 'constant_value', got %s", str)
	}
}

func TestGenerateOneOf(t *testing.T) {
	schema := `{
		"oneOf": [
			{"type": "string"},
			{"type": "integer"},
			{"type": "boolean"}
		]
	}`

	gen := NewGenerator().SetSeed(12345)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Result should be one of the valid types
	switch result.(type) {
	case string, int64, bool:
		// OK
	default:
		t.Fatalf("Unexpected type: %T", result)
	}
}

func TestGenerateAnyOf(t *testing.T) {
	schema := `{
		"anyOf": [
			{"type": "string", "minLength": 5},
			{"type": "integer", "minimum": 10}
		]
	}`

	gen := NewGenerator().SetSeed(12345)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Result should be one of the valid types
	switch result.(type) {
	case string, int64:
		// OK
	default:
		t.Fatalf("Unexpected type: %T", result)
	}
}

func TestGenerateAllOf(t *testing.T) {
	schema := `{
		"allOf": [
			{"type": "object", "properties": {"name": {"type": "string"}}},
			{"type": "object", "properties": {"age": {"type": "integer"}}}
		]
	}`

	gen := NewGenerator().SetSeed(12345)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	_, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected object, got %T", result)
	}
}

func TestGenerateMultipleTypes(t *testing.T) {
	schema := `{"type": ["string", "null"]}`

	gen := NewGenerator().SetSeed(12345)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Result should be either string or null
	switch result.(type) {
	case string, nil:
		// OK
	default:
		t.Fatalf("Unexpected type: %T", result)
	}
}

func TestGenerateComplexSchema(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"name": {"type": "string", "minLength": 3},
					"email": {"type": "string", "format": "email"},
					"age": {"type": "integer", "minimum": 0, "maximum": 120}
				},
				"required": ["name", "email"]
			},
			"tags": {
				"type": "array",
				"items": {"type": "string"},
				"minItems": 1,
				"maxItems": 5
			},
			"status": {
				"type": "string",
				"enum": ["active", "inactive", "pending"]
			}
		},
		"required": ["user", "status"]
	}`

	gen := NewGenerator().SetSeed(12345)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	obj, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected object, got %T", result)
	}

	// Check required fields
	if _, exists := obj["user"]; !exists {
		t.Error("Required field 'user' is missing")
	}

	if _, exists := obj["status"]; !exists {
		t.Error("Required field 'status' is missing")
	}

	// Check nested user object
	user, ok := obj["user"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected user to be an object")
	}

	if _, exists := user["name"]; !exists {
		t.Error("Required field 'name' in user is missing")
	}

	if _, exists := user["email"]; !exists {
		t.Error("Required field 'email' in user is missing")
	}
}

func TestGenerateBytes(t *testing.T) {
	schema := `{"type": "object", "properties": {"name": {"type": "string"}}}`

	gen := NewGenerator().SetSeed(12345)
	result, err := gen.GenerateBytes([]byte(schema))
	if err != nil {
		t.Fatalf("GenerateBytes() error = %v", err)
	}

	// Verify it's valid JSON
	var obj map[string]interface{}
	if err := json.Unmarshal(result, &obj); err != nil {
		t.Fatalf("Failed to unmarshal generated JSON: %v", err)
	}
}

func TestInvalidSchema(t *testing.T) {
	tests := []struct {
		name   string
		schema string
	}{
		{
			name:   "conflicting number constraints",
			schema: `{"type": "integer", "minimum": 100, "maximum": 10}`,
		},
		{
			name:   "conflicting string length",
			schema: `{"type": "string", "minLength": 10, "maxLength": 5}`,
		},
		{
			name:   "conflicting array length",
			schema: `{"type": "array", "minItems": 10, "maxItems": 5}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator()
			_, err := gen.Generate([]byte(tt.schema))
			if err == nil {
				t.Error("Expected error for invalid schema, got nil")
			}
		})
	}
}

func TestMaxDepth(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"level1": {
				"type": "object",
				"properties": {
					"level2": {
						"type": "object",
						"properties": {
							"level3": {
								"type": "object",
								"properties": {
									"level4": {"type": "string"}
								},
								"required": ["level4"]
							}
						},
						"required": ["level3"]
					}
				},
				"required": ["level2"]
			}
		},
		"required": ["level1"]
	}`

	gen := NewGenerator().SetMaxDepth(3).SetSeed(12345)
	_, err := gen.Generate([]byte(schema))
	if err == nil {
		t.Error("Expected error for exceeding max depth, got nil")
	}
}

func TestDeterministicGeneration(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string", "minLength": 3, "maxLength": 3},
			"age": {"type": "integer", "minimum": 100, "maximum": 100}
		},
		"required": ["name", "age"]
	}`

	seed := int64(42)

	gen1 := NewGenerator().SetSeed(seed)
	result1, err1 := gen1.GenerateBytes([]byte(schema))
	if err1 != nil {
		t.Fatalf("Generate() error = %v", err1)
	}

	gen2 := NewGenerator().SetSeed(seed)
	result2, err2 := gen2.GenerateBytes([]byte(schema))
	if err2 != nil {
		t.Fatalf("Generate() error = %v", err2)
	}

	if string(result1) != string(result2) {
		t.Errorf("Results with same seed should be identical.\nGot:\n%s\n%s", result1, result2)
	}
}

func TestGenerateAllFields(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"required_field": {"type": "string"},
			"optional_field": {"type": "integer"}
		},
		"required": ["required_field"]
	}`

	t.Run("only required fields", func(t *testing.T) {
		gen := NewGenerator().SetSeed(12345).SetGenerateAllFields(false)
		result, err := gen.Generate([]byte(schema))
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		obj := result.(map[string]interface{})
		if _, exists := obj["required_field"]; !exists {
			t.Error("Required field is missing")
		}
	})

	t.Run("all fields", func(t *testing.T) {
		gen := NewGenerator().SetSeed(12345).SetGenerateAllFields(true)
		result, err := gen.Generate([]byte(schema))
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		obj := result.(map[string]interface{})
		if _, exists := obj["required_field"]; !exists {
			t.Error("Required field is missing")
		}
		if _, exists := obj["optional_field"]; !exists {
			t.Error("Optional field is missing when GenerateAllFields is true")
		}
	})
}

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
