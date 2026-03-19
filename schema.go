package schemagen

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// ValidationError represents a schema validation error with context
type ValidationError struct {
	Path    string      `json:"path"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

func (ve ValidationError) Error() string {
	if ve.Path != "" {
		return fmt.Sprintf("validation error at %s: %s", ve.Path, ve.Message)
	}
	return ve.Message
}

// ValidationErrors is a collection of validation errors that implements the error interface
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	if len(ve) == 1 {
		return ve[0].Error()
	}
	msgs := make([]string, len(ve))
	for i, e := range ve {
		msgs[i] = e.Error()
	}
	return fmt.Sprintf("%d validation errors: [%s]", len(ve), strings.Join(msgs, "; "))
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
	AdditionalProperties json.RawMessage    `json:"additionalProperties,omitempty"`

	// Array
	Items       json.RawMessage `json:"items,omitempty"`
	MinItems    *int            `json:"minItems,omitempty"`
	MaxItems    *int            `json:"maxItems,omitempty"`
	UniqueItems *bool           `json:"uniqueItems,omitempty"`

	// Composition
	OneOf []Schema `json:"oneOf,omitempty"`
	AnyOf []Schema `json:"anyOf,omitempty"`
	AllOf []Schema `json:"allOf,omitempty"`
	Not   *Schema  `json:"not,omitempty"`

	// References (for future support)
	Ref         string             `json:"$ref,omitempty"`
	Definitions map[string]*Schema `json:"definitions,omitempty"`
	Defs        map[string]*Schema `json:"$defs,omitempty"` // Draft 2020-12
}

// parseAdditionalProperties parses the AdditionalProperties field.
// Returns (nil, false) if not set, (nil, true/false) if boolean, (*Schema, true) if schema.
func (s *Schema) parseAdditionalProperties() (*Schema, bool, error) {
	if len(s.AdditionalProperties) == 0 {
		return nil, false, nil
	}

	// Try as boolean first
	var boolVal bool
	if err := json.Unmarshal(s.AdditionalProperties, &boolVal); err == nil {
		return nil, boolVal, nil
	}

	// Try as schema
	var schema Schema
	if err := json.Unmarshal(s.AdditionalProperties, &schema); err != nil {
		return nil, false, fmt.Errorf("additionalProperties must be a boolean or schema: %w", err)
	}
	return &schema, true, nil
}

// parseItems parses the Items field.
// Returns a single *Schema for uniform items, or []*Schema for tuple validation.
func (s *Schema) parseItems() (single *Schema, tuple []*Schema, err error) {
	if len(s.Items) == 0 {
		return nil, nil, nil
	}

	// Try as a single schema first
	var schema Schema
	if err := json.Unmarshal(s.Items, &schema); err == nil {
		return &schema, nil, nil
	}

	// Try as array of schemas (tuple validation)
	var schemas []Schema
	if err := json.Unmarshal(s.Items, &schemas); err != nil {
		return nil, nil, fmt.Errorf("items must be a schema or array of schemas: %w", err)
	}

	ptrs := make([]*Schema, len(schemas))
	for i := range schemas {
		ptrs[i] = &schemas[i]
	}
	return nil, ptrs, nil
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

// Validate performs comprehensive validation on the schema constraints.
// Returns a ValidationErrors (which implements error) containing all errors found,
// or nil if the schema is valid.
func (s *Schema) Validate() error {
	errors := s.ValidateWithDetails("")
	if len(errors) > 0 {
		return ValidationErrors(errors)
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

	// Validate non-negative constraint values
	if s.MinLength != nil && *s.MinLength < 0 {
		errors = append(errors, ValidationError{
			Path:    basePath,
			Message: fmt.Sprintf("minLength (%d) must be non-negative", *s.MinLength),
		})
	}
	if s.MaxLength != nil && *s.MaxLength < 0 {
		errors = append(errors, ValidationError{
			Path:    basePath,
			Message: fmt.Sprintf("maxLength (%d) must be non-negative", *s.MaxLength),
		})
	}
	if s.MinItems != nil && *s.MinItems < 0 {
		errors = append(errors, ValidationError{
			Path:    basePath,
			Message: fmt.Sprintf("minItems (%d) must be non-negative", *s.MinItems),
		})
	}
	if s.MaxItems != nil && *s.MaxItems < 0 {
		errors = append(errors, ValidationError{
			Path:    basePath,
			Message: fmt.Sprintf("maxItems (%d) must be non-negative", *s.MaxItems),
		})
	}

	// Validate multipleOf is positive
	if s.MultipleOf != nil && *s.MultipleOf <= 0 {
		errors = append(errors, ValidationError{
			Path:    basePath,
			Message: fmt.Sprintf("multipleOf (%f) must be greater than zero", *s.MultipleOf),
		})
	}

	// Validate required fields exist in properties (if properties are defined)
	if s.Properties != nil && len(s.Required) > 0 {
		for _, reqField := range s.Required {
			if _, exists := s.Properties[reqField]; !exists {
				errors = append(errors, ValidationError{
					Path:    basePath,
					Message: fmt.Sprintf("required field %q is not defined in properties", reqField),
				})
			}
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
