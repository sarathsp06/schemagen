package schemagen

import (
	"encoding/json"
	"fmt"
	"slices"
)

// ValidationError represents a schema validation error with context
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

func (ve ValidationError) Error() string {
	if ve.Path != "" {
		return fmt.Sprintf("validation error at %s: %s", ve.Path, ve.Message)
	}
	return ve.Message
}

// Schema represents a JSON Schema with support for Draft 2020-12 and Draft-07
type Schema struct {
	// Meta
	Type  StringOrArray `json:"type,omitempty"`
	Title string        `json:"title,omitempty"`

	// Generic
	Enum  []interface{} `json:"enum,omitempty"`
	Const interface{}   `json:"const,omitempty"`

	// String
	MinLength *int   `json:"minLength,omitempty"`
	MaxLength *int   `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	Format    string `json:"format,omitempty"`

	// Number
	Minimum          *float64 `json:"minimum,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty"`
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty"`
	MultipleOf       *float64 `json:"multipleOf,omitempty"`

	// Object
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	AdditionalProperties interface{}        `json:"additionalProperties,omitempty"` // bool or Schema

	// Array
	Items    interface{} `json:"items,omitempty"` // Schema or array of Schemas
	MinItems *int        `json:"minItems,omitempty"`
	MaxItems *int        `json:"maxItems,omitempty"`

	// Composition
	OneOf []Schema `json:"oneOf,omitempty"`
	AnyOf []Schema `json:"anyOf,omitempty"`
	AllOf []Schema `json:"allOf,omitempty"`

	// References (for future support)
	Ref         string             `json:"$ref,omitempty"`
	Definitions map[string]*Schema `json:"definitions,omitempty"`
	Defs        map[string]*Schema `json:"$defs,omitempty"` // Draft 2020-12
}

// StringOrArray handles the polymorphic nature of the "type" field
// which can be either a single string or an array of strings
type StringOrArray struct {
	Single   string
	Multiple []string
	IsArray  bool
}

// UnmarshalJSON implements custom unmarshaling for StringOrArray
func (s *StringOrArray) UnmarshalJSON(data []byte) error {
	// Try unmarshaling as a string first
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		s.Single = single
		s.IsArray = false
		return nil
	}

	// Try as an array
	var multiple []string
	if err := json.Unmarshal(data, &multiple); err != nil {
		return fmt.Errorf("type must be either a string or array of strings: %w", err)
	}

	s.Multiple = multiple
	s.IsArray = true
	return nil
}

// MarshalJSON implements custom marshaling for StringOrArray
func (s StringOrArray) MarshalJSON() ([]byte, error) {
	if s.IsArray {
		return json.Marshal(s.Multiple)
	}
	return json.Marshal(s.Single)
}

// Contains checks if the StringOrArray contains a specific type
func (s *StringOrArray) Contains(typeName string) bool {
	if !s.IsArray {
		return s.Single == typeName
	}
	return slices.Contains(s.Multiple, typeName)
}

// GetTypes returns all types as a slice
func (s *StringOrArray) GetTypes() []string {
	if !s.IsArray {
		if s.Single == "" {
			return []string{}
		}
		return []string{s.Single}
	}
	return s.Multiple
}

// IsEmpty checks if no type is specified
func (s *StringOrArray) IsEmpty() bool {
	if s.IsArray {
		return len(s.Multiple) == 0
	}
	return s.Single == ""
}

// ParseSchema parses a JSON Schema from bytes
func ParseSchema(schemaJSON []byte) (*Schema, error) {
	var schema Schema
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}
	return &schema, nil
}

// Validate performs comprehensive validation on the schema constraints
func (s *Schema) Validate() error {
	errors := s.ValidateWithDetails("")
	if len(errors) > 0 {
		// Return first error for backward compatibility
		return errors[0]
	}
	return nil
}

// ValidateWithDetails performs comprehensive validation and returns all errors with path context
func (s *Schema) ValidateWithDetails(basePath string) []ValidationError {
	var errors []ValidationError

	// Check for impossible number constraints
	if s.Minimum != nil && s.Maximum != nil {
		if *s.Minimum > *s.Maximum {
			errors = append(errors, ValidationError{
				Path:    basePath,
				Message: fmt.Sprintf("minimum (%f) cannot be greater than maximum (%f)", *s.Minimum, *s.Maximum),
			})
		}
	}

	if s.ExclusiveMinimum != nil && s.ExclusiveMaximum != nil {
		if *s.ExclusiveMinimum >= *s.ExclusiveMaximum {
			errors = append(errors, ValidationError{
				Path:    basePath,
				Message: fmt.Sprintf("exclusiveMinimum (%f) must be less than exclusiveMaximum (%f)", *s.ExclusiveMinimum, *s.ExclusiveMaximum),
			})
		}
	}

	// Check for impossible string length constraints
	if s.MinLength != nil && s.MaxLength != nil {
		if *s.MinLength > *s.MaxLength {
			errors = append(errors, ValidationError{
				Path:    basePath,
				Message: fmt.Sprintf("minLength (%d) cannot be greater than maxLength (%d)", *s.MinLength, *s.MaxLength),
			})
		}
	}

	// Check for impossible array length constraints
	if s.MinItems != nil && s.MaxItems != nil {
		if *s.MinItems > *s.MaxItems {
			errors = append(errors, ValidationError{
				Path:    basePath,
				Message: fmt.Sprintf("minItems (%d) cannot be greater than maxItems (%d)", *s.MinItems, *s.MaxItems),
			})
		}
	}

	// Validate nested schemas
	for propName, propSchema := range s.Properties {
		propPath := basePath
		if propPath == "" {
			propPath = propName
		} else {
			propPath = propPath + "." + propName
		}
		errors = append(errors, propSchema.ValidateWithDetails(propPath)...)
	}

	// Validate composition schemas
	for i, schema := range s.OneOf {
		schemaPath := fmt.Sprintf("%s.oneOf[%d]", basePath, i)
		errors = append(errors, schema.ValidateWithDetails(schemaPath)...)
	}

	for i, schema := range s.AnyOf {
		schemaPath := fmt.Sprintf("%s.anyOf[%d]", basePath, i)
		errors = append(errors, schema.ValidateWithDetails(schemaPath)...)
	}

	for i, schema := range s.AllOf {
		schemaPath := fmt.Sprintf("%s.allOf[%d]", basePath, i)
		errors = append(errors, schema.ValidateWithDetails(schemaPath)...)
	}

	return errors
}
