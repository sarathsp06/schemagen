package schemagen

import (
	"context"
	"fmt"
	"maps"
	"slices"
)

// mergeSchemas combines two schemas into a new one, treating overlay as the
// stricter/later schema. Pure: neither input is mutated.
//
// Rules: required is unioned, type is intersected, numeric/length/item/property
// bounds take the tighter value, maps are unioned with overlay winning per key,
// and most scalar keywords are overlay-wins-when-set.
func (g *Generator) mergeSchemas(base, overlay *Schema) *Schema {
	if base == nil && overlay == nil {
		return &Schema{}
	}
	if base == nil {
		out := *overlay
		return &out
	}
	if overlay == nil {
		out := *base
		return &out
	}
	out := *base

	// type: intersection when both set; empty intersection → overlay wins
	if !overlay.Type.IsEmpty() {
		if base.Type.IsEmpty() {
			out.Type = overlay.Type
		} else {
			var inter []string
			for _, t := range base.Type.GetTypes() {
				if overlay.Type.Contains(t) {
					inter = append(inter, t)
				}
			}
			switch len(inter) {
			case 0:
				out.Type = overlay.Type
			case 1:
				out.Type = StringOrArray{Single: inter[0]}
			default:
				out.Type = StringOrArray{Multiple: inter, IsArray: true}
			}
		}
	}

	if overlay.Title != "" {
		out.Title = overlay.Title
	}

	// Generic keywords: overlay wins when set.
	// ponytail: enum/const intersection with base constraints is out of scope;
	// a true merge would intersect enum values against base bounds.
	if len(overlay.Enum) > 0 {
		out.Enum = overlay.Enum
	}
	if len(overlay.Const) > 0 {
		out.Const = overlay.Const
	}
	if len(overlay.Default) > 0 {
		out.Default = overlay.Default
	}
	if len(overlay.Examples) > 0 {
		out.Examples = overlay.Examples
	}

	// String: tighter bounds win.
	// ponytail: regex intersection is out of scope; overlay pattern wins.
	out.MinLength = maxIntPtr(base.MinLength, overlay.MinLength)
	out.MaxLength = minIntPtr(base.MaxLength, overlay.MaxLength)
	if overlay.Pattern != "" {
		out.Pattern = overlay.Pattern
	}
	if overlay.Format != "" {
		out.Format = overlay.Format
	}
	if overlay.XFaker != "" {
		out.XFaker = overlay.XFaker
	}

	// Number: tighter bounds win; multipleOf overlay wins when both set.
	out.Minimum = maxFloatPtr(base.Minimum, overlay.Minimum)
	out.Maximum = minFloatPtr(base.Maximum, overlay.Maximum)
	out.ExclusiveMinimum = maxFloatPtr(base.ExclusiveMinimum, overlay.ExclusiveMinimum)
	out.ExclusiveMaximum = minFloatPtr(base.ExclusiveMaximum, overlay.ExclusiveMaximum)
	if overlay.MultipleOf != nil {
		out.MultipleOf = overlay.MultipleOf
	}

	// Object: properties unioned with recursive merge on key collision.
	if len(overlay.Properties) > 0 {
		props := make(map[string]*Schema, len(base.Properties)+len(overlay.Properties))
		maps.Copy(props, base.Properties)
		for k, v := range overlay.Properties {
			if bv, ok := props[k]; ok {
				props[k] = g.mergeSchemas(bv, v)
			} else {
				props[k] = v
			}
		}
		out.Properties = props
	}
	if len(overlay.PatternProperties) > 0 {
		out.PatternProperties = overlay.PatternProperties
	}
	if overlay.PropertyNames != nil {
		out.PropertyNames = overlay.PropertyNames
	}
	if len(overlay.Required) > 0 {
		out.Required = unionStrings(base.Required, overlay.Required)
	}
	out.MinProperties = maxIntPtr(base.MinProperties, overlay.MinProperties)
	out.MaxProperties = minIntPtr(base.MaxProperties, overlay.MaxProperties)
	if len(overlay.AdditionalProperties) > 0 {
		out.AdditionalProperties = overlay.AdditionalProperties
	}
	if len(overlay.DependentRequired) > 0 {
		dr := make(map[string][]string, len(base.DependentRequired)+len(overlay.DependentRequired))
		maps.Copy(dr, base.DependentRequired)
		maps.Copy(dr, overlay.DependentRequired)
		out.DependentRequired = dr
	}
	out.DependentSchemas = mergeSchemaMap(base.DependentSchemas, overlay.DependentSchemas)

	// Array: tighter bounds win; item schemas overlay wins when set.
	if len(overlay.Items) > 0 {
		out.Items = overlay.Items
	}
	if len(overlay.PrefixItems) > 0 {
		out.PrefixItems = overlay.PrefixItems
	}
	if overlay.Contains != nil {
		out.Contains = overlay.Contains
	}
	out.MinContains = maxIntPtr(base.MinContains, overlay.MinContains)
	out.MaxContains = minIntPtr(base.MaxContains, overlay.MaxContains)
	out.MinItems = maxIntPtr(base.MinItems, overlay.MinItems)
	out.MaxItems = minIntPtr(base.MaxItems, overlay.MaxItems)
	if overlay.UniqueItems != nil {
		out.UniqueItems = overlay.UniqueItems
	}

	// Composition: overlay's keywords carry over when set so chained
	// composition still generates.
	if len(overlay.OneOf) > 0 {
		out.OneOf = overlay.OneOf
	}
	if len(overlay.AnyOf) > 0 {
		out.AnyOf = overlay.AnyOf
	}
	if len(overlay.AllOf) > 0 {
		out.AllOf = overlay.AllOf
	}
	if overlay.Not != nil {
		out.Not = overlay.Not
	}

	// Conditionals: overlay wins when set; applyConditionals clears them on
	// its own result so they never loop.
	if overlay.If != nil {
		out.If = overlay.If
	}
	if overlay.Then != nil {
		out.Then = overlay.Then
	}
	if overlay.Else != nil {
		out.Else = overlay.Else
	}

	// References: overlay wins; defs unioned so $ref targets stay reachable.
	if overlay.Ref != "" {
		out.Ref = overlay.Ref
	}
	out.Definitions = mergeSchemaMap(base.Definitions, overlay.Definitions)
	out.Defs = mergeSchemaMap(base.Defs, overlay.Defs)

	return &out
}

// applyConditionals folds if/then into the schema before generation.
// ponytail: we always satisfy the "if" branch — guaranteeing an if-violation
// (to exercise "else") is not generally possible, so "else" is never generated.
func (g *Generator) applyConditionals(schema *Schema) *Schema {
	if schema.If == nil {
		return schema
	}
	base := *schema
	base.If, base.Then, base.Else = nil, nil, nil
	merged := g.mergeSchemas(&base, schema.If)
	if schema.Then != nil {
		merged = g.mergeSchemas(merged, schema.Then)
	}
	merged.If, merged.Then, merged.Else = nil, nil, nil
	return merged
}

// handleOneOf picks a random oneOf branch and merges it with the parent's
// sibling constraints before generating.
func (g *Generator) handleOneOf(ctx context.Context, schema *Schema, depth int) (interface{}, error) {
	if len(schema.OneOf) == 0 {
		return nil, fmt.Errorf("oneOf array is empty")
	}
	chosen := &schema.OneOf[g.rand.IntN(len(schema.OneOf))]
	parent := *schema
	parent.OneOf, parent.AnyOf, parent.AllOf = nil, nil, nil
	return g.generateWithContext(ctx, g.mergeSchemas(&parent, chosen), depth)
}

// handleAnyOf picks a random anyOf branch and merges it with the parent's
// sibling constraints before generating.
func (g *Generator) handleAnyOf(ctx context.Context, schema *Schema, depth int) (interface{}, error) {
	if len(schema.AnyOf) == 0 {
		return nil, fmt.Errorf("anyOf array is empty")
	}
	chosen := &schema.AnyOf[g.rand.IntN(len(schema.AnyOf))]
	parent := *schema
	parent.OneOf, parent.AnyOf, parent.AllOf = nil, nil, nil
	return g.generateWithContext(ctx, g.mergeSchemas(&parent, chosen), depth)
}

// handleAllOf folds all allOf sub-schemas together, layers the parent's
// sibling constraints on top, and generates from the merged schema.
func (g *Generator) handleAllOf(ctx context.Context, schema *Schema, depth int) (interface{}, error) {
	if len(schema.AllOf) == 0 {
		return nil, fmt.Errorf("allOf array is empty")
	}
	merged := &Schema{}
	for i := range schema.AllOf {
		merged = g.mergeSchemas(merged, &schema.AllOf[i])
	}
	parent := *schema
	parent.OneOf, parent.AnyOf, parent.AllOf = nil, nil, nil
	return g.generateWithContext(ctx, g.mergeSchemas(merged, &parent), depth)
}

// unionStrings returns a ∪ b preserving order, deduplicated.
func unionStrings(a, b []string) []string {
	out := slices.Clone(a)
	for _, s := range b {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

// mergeSchemaMap unions two schema maps; overlay wins per key.
func mergeSchemaMap(base, overlay map[string]*Schema) map[string]*Schema {
	if len(overlay) == 0 {
		return base
	}
	out := make(map[string]*Schema, len(base)+len(overlay))
	maps.Copy(out, base)
	maps.Copy(out, overlay)
	return out
}

func maxFloatPtr(a, b *float64) *float64 {
	if a == nil {
		return b
	}
	if b == nil || *a >= *b {
		return a
	}
	return b
}

func minFloatPtr(a, b *float64) *float64 {
	if a == nil {
		return b
	}
	if b == nil || *a <= *b {
		return a
	}
	return b
}

func maxIntPtr(a, b *int) *int {
	if a == nil {
		return b
	}
	if b == nil || *a >= *b {
		return a
	}
	return b
}

func minIntPtr(a, b *int) *int {
	if a == nil {
		return b
	}
	if b == nil || *a <= *b {
		return a
	}
	return b
}
