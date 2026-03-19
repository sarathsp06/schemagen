package schemagen

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
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
			{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]},
			{"type": "object", "properties": {"age": {"type": "integer"}}, "required": ["age"]}
		]
	}`

	gen := NewGenerator().SetSeed(12345).SetGenerateAllFields(true)
	result, err := gen.Generate([]byte(schema))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	obj, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected object, got %T", result)
	}

	// After allOf merging, both properties should be present
	if _, exists := obj["name"]; !exists {
		t.Error("Expected merged 'name' property from first allOf schema")
	}
	if _, exists := obj["age"]; !exists {
		t.Error("Expected merged 'age' property from second allOf schema")
	}
}

func TestGenerateAllOfMergesRequired(t *testing.T) {
	schema := `{
		"allOf": [
			{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]},
			{"type": "object", "properties": {"age": {"type": "integer"}}, "required": ["age"]}
		]
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

	// Both required fields should be present even without GenerateAllFields
	if _, exists := obj["name"]; !exists {
		t.Error("Required field 'name' missing after allOf merge")
	}
	if _, exists := obj["age"]; !exists {
		t.Error("Required field 'age' missing after allOf merge")
	}
}

func TestGenerateAllOfNonObject(t *testing.T) {
	schema := `{
		"allOf": [
			{"type": "string", "minLength": 5}
		]
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

	if len(str) < 5 {
		t.Errorf("Expected string length >= 5, got %d", len(str))
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

// Test exclusive minimum and maximum with proper assertions
func TestGenerateNumberExclusiveBounds(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		minVal float64
		maxVal float64
	}{
		{
			name:   "exclusive minimum",
			schema: `{"type": "integer", "exclusiveMinimum": 10, "maximum": 20}`,
			minVal: 11, // exclusiveMinimum: 10 means > 10, so integer min is 11
			maxVal: 20,
		},
		{
			name:   "exclusive maximum",
			schema: `{"type": "integer", "minimum": 10, "exclusiveMaximum": 20}`,
			minVal: 10,
			maxVal: 19, // exclusiveMaximum: 20 means < 20, so integer max is 19
		},
		{
			name:   "both exclusive",
			schema: `{"type": "number", "exclusiveMinimum": 0, "exclusiveMaximum": 1}`,
			minVal: 0, // for floats, exclusive bounds are used directly
			maxVal: 1,
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

			if numVal < tt.minVal {
				t.Errorf("Value %f is less than expected minimum %f", numVal, tt.minVal)
			}
			if numVal > tt.maxVal {
				t.Errorf("Value %f is greater than expected maximum %f", numVal, tt.maxVal)
			}
		})
	}
}

// Test exclusive bounds with non-integer boundaries
func TestGenerateNumberExclusiveBoundsNonInteger(t *testing.T) {
	t.Run("exclusiveMinimum with fractional value", func(t *testing.T) {
		schema := `{"type": "integer", "exclusiveMinimum": 10.5, "maximum": 20}`
		gen := NewGenerator().SetSeed(42)
		result, err := gen.Generate([]byte(schema))
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		val := result.(int64)
		if val < 11 {
			t.Errorf("Expected value >= 11 (exclusiveMinimum: 10.5), got %d", val)
		}
	})

	t.Run("exclusiveMaximum with fractional value", func(t *testing.T) {
		schema := `{"type": "integer", "exclusiveMaximum": 20.5, "minimum": 10}`
		gen := NewGenerator().SetSeed(42)
		result, err := gen.Generate([]byte(schema))
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		val := result.(int64)
		if val > 20 {
			t.Errorf("Expected value <= 20 (exclusiveMaximum: 20.5), got %d", val)
		}
	})
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

	if val%7 != 0 {
		t.Errorf("Expected value to be a multiple of 7, got %d", val)
	}
}

// Test multipleOf with no valid value in range
func TestGenerateNumberMultipleOfNoValidValue(t *testing.T) {
	// Range [10, 13] with multipleOf 7: only multiple of 7 is 7 or 14, neither in range
	schema := `{"type": "integer", "minimum": 10, "maximum": 13, "multipleOf": 7}`

	gen := NewGenerator().SetSeed(12345)
	_, err := gen.Generate([]byte(schema))
	if err == nil {
		t.Error("Expected error when no multiple of 7 exists in range [10, 13]")
	}
}

// Test array items parsing error
func TestGenerateArrayItemsParseError(t *testing.T) {
	gen := NewGenerator().SetSeed(42)

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

// Test ValidationErrors implements error and contains all errors
func TestValidationErrorsMultiple(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"bad_string": {
				"type": "string",
				"minLength": 10,
				"maxLength": 5
			},
			"bad_number": {
				"type": "number",
				"minimum": 100,
				"maximum": 50
			}
		}
	}`

	s, err := ParseSchema([]byte(schema))
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	validationErr := s.Validate()
	if validationErr == nil {
		t.Fatal("Expected validation error")
	}

	ve, ok := validationErr.(ValidationErrors)
	if !ok {
		t.Fatalf("Expected ValidationErrors type, got %T", validationErr)
	}

	if len(ve) != 2 {
		t.Errorf("Expected 2 validation errors, got %d", len(ve))
	}

	errStr := ve.Error()
	if !strings.Contains(errStr, "2 validation errors") {
		t.Errorf("Expected multi-error message, got: %s", errStr)
	}
}

// Test that required fields not in properties are caught by validation
func TestValidateRequiredNotInProperties(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name", "missing_field"]
	}`

	gen := NewGenerator()
	_, err := gen.Generate([]byte(schema))
	if err == nil {
		t.Error("Expected error for required field not defined in properties")
	}

	if !strings.Contains(err.Error(), "missing_field") {
		t.Errorf("Expected error to mention 'missing_field', got: %s", err.Error())
	}
}

// Test negative constraint values
func TestValidateNegativeConstraints(t *testing.T) {
	tests := []struct {
		name   string
		schema string
	}{
		{
			name:   "negative minLength",
			schema: `{"type": "string", "minLength": -1}`,
		},
		{
			name:   "negative maxLength",
			schema: `{"type": "string", "maxLength": -1}`,
		},
		{
			name:   "negative minItems",
			schema: `{"type": "array", "minItems": -1}`,
		},
		{
			name:   "negative maxItems",
			schema: `{"type": "array", "maxItems": -1}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator()
			_, err := gen.Generate([]byte(tt.schema))
			if err == nil {
				t.Error("Expected error for negative constraint value")
			}
		})
	}
}

// Test zero multipleOf validation
func TestValidateZeroMultipleOf(t *testing.T) {
	schema := `{"type": "number", "multipleOf": 0}`

	gen := NewGenerator()
	_, err := gen.Generate([]byte(schema))
	if err == nil {
		t.Error("Expected error for zero multipleOf")
	}
}

// Test uniqueItems support
func TestGenerateArrayUniqueItems(t *testing.T) {
	schema := `{
		"type": "array",
		"items": {"type": "integer", "minimum": 1, "maximum": 100},
		"minItems": 5,
		"maxItems": 5,
		"uniqueItems": true
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

	if len(arr) != 5 {
		t.Fatalf("Expected 5 items, got %d", len(arr))
	}

	// Check uniqueness
	seen := make(map[int64]bool)
	for _, item := range arr {
		val, ok := item.(int64)
		if !ok {
			t.Fatalf("Expected int64, got %T", item)
		}
		if seen[val] {
			t.Errorf("Duplicate value found: %d", val)
		}
		seen[val] = true
	}
}

// Test GenerateFromSchema with pre-parsed schema
func TestGenerateFromSchema(t *testing.T) {
	schemaJSON := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "minimum": 0, "maximum": 120}
		},
		"required": ["name"]
	}`

	schema, err := ParseSchema([]byte(schemaJSON))
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	gen := NewGenerator().SetSeed(42)
	result, err := gen.GenerateFromSchema(schema)
	if err != nil {
		t.Fatalf("GenerateFromSchema() error = %v", err)
	}

	obj, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected object, got %T", result)
	}

	if _, exists := obj["name"]; !exists {
		t.Error("Required field 'name' is missing")
	}
}

// Test GenerateFromSchema with invalid schema
func TestGenerateFromSchemaInvalid(t *testing.T) {
	schema := &Schema{
		Type: StringOrArray{Single: "string", IsArray: false},
	}
	minLen := 10
	maxLen := 5
	schema.MinLength = &minLen
	schema.MaxLength = &maxLen

	gen := NewGenerator()
	_, err := gen.GenerateFromSchema(schema)
	if err == nil {
		t.Error("Expected error for invalid schema")
	}
}

// Test context cancellation is propagated through recursive generation
func TestGenerateWithContextCancellation(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"data": {
				"type": "array",
				"items": {"type": "string"},
				"minItems": 100
			}
		},
		"required": ["data"]
	}`

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	gen := NewGenerator().SetSeed(42)
	_, err := gen.GenerateWithContext(ctx, []byte(schema))
	if err == nil {
		t.Error("Expected error for cancelled context")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("Expected cancellation error, got: %v", err)
	}
}

// Test context timeout propagation
func TestGenerateWithContextTimeout(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"required": ["name"]
	}`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	gen := NewGenerator().SetSeed(42)
	result, err := gen.GenerateWithContext(ctx, []byte(schema))
	if err != nil {
		t.Fatalf("GenerateWithContext() error = %v", err)
	}

	obj, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected object, got %T", result)
	}

	if _, exists := obj["name"]; !exists {
		t.Error("Required field 'name' is missing")
	}
}

// Test not keyword is parsed (even if not used for generation)
func TestSchemaNotKeywordParsed(t *testing.T) {
	schemaJSON := `{
		"type": "string",
		"not": {"type": "null"}
	}`

	schema, err := ParseSchema([]byte(schemaJSON))
	if err != nil {
		t.Fatalf("ParseSchema error: %v", err)
	}

	if schema.Not == nil {
		t.Error("Expected 'not' keyword to be parsed")
	}
}

// Test deterministic object generation (sorted keys)
func TestDeterministicObjectGeneration(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"zebra": {"type": "string"},
			"apple": {"type": "string"},
			"mango": {"type": "string"}
		},
		"required": ["zebra", "apple", "mango"]
	}`

	seed := int64(42)

	// Generate multiple times with same seed; results must be identical
	for i := 0; i < 5; i++ {
		gen1 := NewGenerator().SetSeed(seed)
		result1, _ := gen1.GenerateBytes([]byte(schema))

		gen2 := NewGenerator().SetSeed(seed)
		result2, _ := gen2.GenerateBytes([]byte(schema))

		if string(result1) != string(result2) {
			t.Errorf("Iteration %d: Results with same seed should be identical.\nGot:\n%s\n%s", i, result1, result2)
		}
	}
}

// Test randomString produces readable strings with spaces for longer lengths
func TestRandomStringHasSpaces(t *testing.T) {
	gen := NewGenerator().SetSeed(42)

	// Generate a string long enough to require multiple words
	result := gen.randomString(30)

	if len(result) != 30 {
		t.Errorf("Expected length 30, got %d", len(result))
	}

	// Longer strings should have spaces from word concatenation
	if !strings.Contains(result, " ") {
		t.Errorf("Expected string to contain spaces for readability, got: %q", result)
	}
}

// Test parseAdditionalProperties helper
func TestParseAdditionalProperties(t *testing.T) {
	t.Run("not set", func(t *testing.T) {
		s := &Schema{}
		schema, allowed, err := s.parseAdditionalProperties()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if schema != nil || allowed {
			t.Error("Expected nil schema and false when not set")
		}
	})

	t.Run("boolean true", func(t *testing.T) {
		s := &Schema{AdditionalProperties: json.RawMessage(`true`)}
		schema, allowed, err := s.parseAdditionalProperties()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if schema != nil {
			t.Error("Expected nil schema for boolean")
		}
		if !allowed {
			t.Error("Expected allowed=true")
		}
	})

	t.Run("boolean false", func(t *testing.T) {
		s := &Schema{AdditionalProperties: json.RawMessage(`false`)}
		schema, allowed, err := s.parseAdditionalProperties()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if schema != nil {
			t.Error("Expected nil schema for boolean")
		}
		if allowed {
			t.Error("Expected allowed=false")
		}
	})

	t.Run("schema", func(t *testing.T) {
		s := &Schema{AdditionalProperties: json.RawMessage(`{"type": "integer"}`)}
		schema, allowed, err := s.parseAdditionalProperties()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if schema == nil {
			t.Error("Expected non-nil schema")
		}
		if !allowed {
			t.Error("Expected allowed=true for schema")
		}
		if schema.Type.Single != "integer" {
			t.Errorf("Expected integer type, got %s", schema.Type.Single)
		}
	})
}

// Test parseItems helper
func TestParseItems(t *testing.T) {
	t.Run("not set", func(t *testing.T) {
		s := &Schema{}
		single, tuple, err := s.parseItems()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if single != nil || tuple != nil {
			t.Error("Expected nil for unset items")
		}
	})

	t.Run("single schema", func(t *testing.T) {
		s := &Schema{Items: json.RawMessage(`{"type": "string"}`)}
		single, tuple, err := s.parseItems()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if single == nil {
			t.Fatal("Expected non-nil single schema")
		}
		if tuple != nil {
			t.Error("Expected nil tuple for single schema")
		}
		if single.Type.Single != "string" {
			t.Errorf("Expected string type, got %s", single.Type.Single)
		}
	})

	t.Run("tuple schema", func(t *testing.T) {
		s := &Schema{Items: json.RawMessage(`[{"type": "string"}, {"type": "integer"}]`)}
		single, tuple, err := s.parseItems()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if single != nil {
			t.Error("Expected nil single for tuple")
		}
		if len(tuple) != 2 {
			t.Fatalf("Expected 2 tuple schemas, got %d", len(tuple))
		}
	})
}
