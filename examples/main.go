package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/sarathsp06/schemagen"
)

func main() {
	fmt.Println("=== SchemaGen Examples ===")

	// Example 1: Simple User Object
	example1()

	// Example 2: Complex Nested Object
	example2()

	// Example 3: Array of Products
	example3()

	// Example 4: Using Pattern for Formatted Strings
	example4()

	// Example 5: OneOf Composition
	example5()

	// Example 6: Deterministic Generation
	example6()

	// Example 7: Context-aware Generation
	example7()

	// Example 8: Enhanced Validation Errors
	example8()
}

func example1() {
	fmt.Println("1. Simple User Object:")
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string", "minLength": 3},
			"email": {"type": "string", "format": "email"},
			"age": {"type": "integer", "minimum": 18, "maximum": 100}
		},
		"required": ["name", "email"]
	}`

	gen := schemagen.NewGenerator().SetSeed(42)
	result, err := gen.GenerateBytes([]byte(schema))
	if err != nil {
		log.Fatal(err)
	}
	printJSON(result)
}

func example2() {
	fmt.Println("\n2. Complex Nested Object:")
	schema := `{
		"type": "object",
		"properties": {
			"user": {
				"type": "object",
				"properties": {
					"id": {"type": "string", "format": "uuid"},
					"name": {"type": "string", "minLength": 3},
					"email": {"type": "string", "format": "email"},
					"profile": {
						"type": "object",
						"properties": {
							"bio": {"type": "string", "maxLength": 50},
							"website": {"type": "string", "format": "url"}
						}
					}
				},
				"required": ["id", "name", "email"]
			},
			"tags": {
				"type": "array",
				"items": {"type": "string"},
				"minItems": 2,
				"maxItems": 5
			},
			"status": {
				"type": "string",
				"enum": ["active", "inactive", "pending"]
			},
			"createdAt": {
				"type": "string",
				"format": "date-time"
			}
		},
		"required": ["user", "status", "createdAt"]
	}`

	gen := schemagen.NewGenerator().SetSeed(42)
	result, err := gen.GenerateBytes([]byte(schema))
	if err != nil {
		log.Fatal(err)
	}
	printJSON(result)
}

func example3() {
	fmt.Println("\n3. Array of Products:")
	schema := `{
		"type": "array",
		"items": {
			"type": "object",
			"properties": {
				"id": {"type": "integer", "minimum": 1000},
				"name": {"type": "string"},
				"price": {"type": "number", "minimum": 0, "maximum": 1000},
				"inStock": {"type": "boolean"}
			},
			"required": ["id", "name", "price"]
		},
		"minItems": 2,
		"maxItems": 4
	}`

	gen := schemagen.NewGenerator().SetSeed(42)
	result, err := gen.GenerateBytes([]byte(schema))
	if err != nil {
		log.Fatal(err)
	}
	printJSON(result)
}

func example4() {
	fmt.Println("\n4. Formatted Strings with Patterns:")
	schema := `{
		"type": "object",
		"properties": {
			"zipcode": {
				"type": "string",
				"pattern": "^[0-9]{5}$"
			},
			"productCode": {
				"type": "string",
				"pattern": "^[A-Z]{3}-[0-9]{4}$"
			},
			"ipAddress": {
				"type": "string",
				"format": "ipv4"
			}
		},
		"required": ["zipcode", "productCode", "ipAddress"]
	}`

	gen := schemagen.NewGenerator().SetSeed(42)
	result, err := gen.GenerateBytes([]byte(schema))
	if err != nil {
		log.Fatal(err)
	}
	printJSON(result)
}

func example5() {
	fmt.Println("\n5. OneOf Composition (Union Types):")
	schema := `{
		"oneOf": [
			{
				"type": "object",
				"properties": {
					"type": {"const": "text"},
					"content": {"type": "string"}
				},
				"required": ["type", "content"]
			},
			{
				"type": "object",
				"properties": {
					"type": {"const": "number"},
					"value": {"type": "integer"}
				},
				"required": ["type", "value"]
			},
			{
				"type": "object",
				"properties": {
					"type": {"const": "boolean"},
					"flag": {"type": "boolean"}
				},
				"required": ["type", "flag"]
			}
		]
	}`

	gen := schemagen.NewGenerator().SetSeed(42)
	result, err := gen.GenerateBytes([]byte(schema))
	if err != nil {
		log.Fatal(err)
	}
	printJSON(result)
}

func example6() {
	fmt.Println("\n6. Deterministic Generation (same seed = same output):")
	schema := `{
		"type": "object",
		"properties": {
			"id": {"type": "integer", "minimum": 1, "maximum": 1000},
			"name": {"type": "string", "minLength": 5, "maxLength": 5}
		},
		"required": ["id", "name"]
	}`

	gen1 := schemagen.NewGenerator().SetSeed(12345)
	result1, _ := gen1.GenerateBytes([]byte(schema))
	fmt.Print("First generation:  ")
	printJSON(result1)

	gen2 := schemagen.NewGenerator().SetSeed(12345)
	result2, _ := gen2.GenerateBytes([]byte(schema))
	fmt.Print("Second generation: ")
	printJSON(result2)

	fmt.Println("\nBoth outputs are identical! ✓")
}

func example7() {
	fmt.Println("\n7. Context-aware Generation (with timeout):")
	schema := `{
		"type": "object",
		"properties": {
			"id": {"type": "integer", "minimum": 1, "maximum": 1000},
			"name": {"type": "string", "minLength": 5, "maxLength": 10}
		},
		"required": ["id", "name"]
	}`

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	gen := schemagen.NewGenerator().SetSeed(12345)
	result, err := gen.GenerateWithContext(ctx, []byte(schema))
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	formatted, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(formatted))
}

func example8() {
	fmt.Println("\n8. Enhanced Validation Errors:")
	
	// Example with conflicting constraints
	invalidSchema := `{
		"type": "object",
		"properties": {
			"invalidString": {
				"type": "string",
				"minLength": 10,
				"maxLength": 5
			},
			"invalidNumber": {
				"type": "number",
				"minimum": 100,
				"maximum": 50
			}
		}
	}`

	schema, err := schemagen.ParseSchema([]byte(invalidSchema))
	if err != nil {
		log.Printf("Parse error: %v", err)
		return
	}

	// Get detailed validation errors
	errors := schema.ValidateWithDetails("")
	fmt.Printf("Found %d validation errors:\n", len(errors))
	for _, err := range errors {
		fmt.Printf("  - %s\n", err.Error())
	}
}

func printJSON(data []byte) {
	var prettyJSON interface{}
	if err := json.Unmarshal(data, &prettyJSON); err != nil {
		fmt.Println(string(data))
		return
	}

	formatted, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		fmt.Println(string(data))
		return
	}

	fmt.Println(string(formatted))
}
