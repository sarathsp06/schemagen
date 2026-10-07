package schemagen

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// maxRefHops bounds chained $ref resolution ($ref → $ref → …) to detect cycles.
const maxRefHops = 32

// resolveRef resolves schema.Ref against g.root. Only local refs are
// supported: "#" (root) and JSON-pointer fragments like "#/$defs/x" or
// "#/properties/user/properties/name". Per 2020-12, sibling keywords on the
// referring schema are overlaid onto the resolved target.
func (g *Generator) resolveRef(schema *Schema) (*Schema, error) {
	if g.root == nil {
		return nil, fmt.Errorf("cannot resolve $ref %q: no root schema set", schema.Ref)
	}

	cur := schema
	for range maxRefHops {
		if cur.Ref == "" {
			return cur, nil
		}
		target, err := g.lookupRef(cur.Ref)
		if err != nil {
			return nil, err
		}
		cur = overlayRef(target, cur)
	}
	return nil, fmt.Errorf("$ref chain exceeds %d hops (cycle?) starting at %q", maxRefHops, schema.Ref)
}

// lookupRef finds the schema a local ref points to within g.root.
func (g *Generator) lookupRef(ref string) (*Schema, error) {
	if !strings.HasPrefix(ref, "#") {
		return nil, fmt.Errorf("unsupported $ref %q: only local refs (\"#...\") are supported", ref)
	}
	frag, err := url.PathUnescape(ref[1:])
	if err != nil {
		return nil, fmt.Errorf("invalid $ref %q: %w", ref, err)
	}
	if frag == "" {
		return g.root, nil
	}
	if !strings.HasPrefix(frag, "/") {
		return nil, fmt.Errorf("unsupported $ref %q: anchors are not supported, use JSON pointers", ref)
	}

	tokens := strings.Split(frag[1:], "/")
	for i, t := range tokens {
		tokens[i] = strings.ReplaceAll(strings.ReplaceAll(t, "~1", "/"), "~0", "~")
	}

	cur := g.root
	for i := 0; i < len(tokens); {
		next, consumed, err := walkPointer(cur, tokens[i:])
		if err != nil {
			return nil, fmt.Errorf("cannot resolve $ref %q: %w", ref, err)
		}
		cur = next
		i += consumed
	}
	return cur, nil
}

// walkPointer descends into a schema by one pointer step, which consumes one
// token (e.g. "contains") or two (e.g. "properties", "foo"). Returns the
// child schema and the number of tokens consumed.
func walkPointer(s *Schema, tokens []string) (*Schema, int, error) {
	kw := tokens[0]
	switch kw {
	case "items":
		single, _, err := s.parseItems()
		if err != nil || single == nil {
			return nil, 0, fmt.Errorf("%q: items is not a single schema", kw)
		}
		return single, 1, nil
	case "contains":
		return nonNil(s.Contains, kw)
	case "propertyNames":
		return nonNil(s.PropertyNames, kw)
	case "if":
		return nonNil(s.If, kw)
	case "then":
		return nonNil(s.Then, kw)
	case "else":
		return nonNil(s.Else, kw)
	case "not":
		return nonNil(s.Not, kw)
	}

	if len(tokens) < 2 {
		return nil, 0, fmt.Errorf("%q requires a key or index", kw)
	}
	key := tokens[1]

	var m map[string]*Schema
	switch kw {
	case "properties":
		m = s.Properties
	case "patternProperties":
		m = s.PatternProperties
	case "$defs":
		m = s.Defs
	case "definitions":
		m = s.Definitions
	case "dependentSchemas":
		m = s.DependentSchemas
	case "prefixItems":
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 0 || idx >= len(s.PrefixItems) {
			return nil, 0, fmt.Errorf("prefixItems index %q out of range", key)
		}
		return s.PrefixItems[idx], 2, nil
	case "oneOf", "anyOf", "allOf":
		var list []Schema
		switch kw {
		case "oneOf":
			list = s.OneOf
		case "anyOf":
			list = s.AnyOf
		case "allOf":
			list = s.AllOf
		}
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 0 || idx >= len(list) {
			return nil, 0, fmt.Errorf("%s index %q out of range", kw, key)
		}
		return &list[idx], 2, nil
	default:
		return nil, 0, fmt.Errorf("unknown segment %q", kw)
	}

	child, ok := m[key]
	if !ok || child == nil {
		return nil, 0, fmt.Errorf("%s/%s: not found", kw, key)
	}
	return child, 2, nil
}

func nonNil(s *Schema, token string) (*Schema, int, error) {
	if s == nil {
		return nil, 0, fmt.Errorf("%q: not present", token)
	}
	return s, 1, nil
}

// overlayRef copies target and applies any non-zero fields of the referring
// schema (except Ref) on top, implementing 2020-12 $ref-with-siblings.
// ponytail: reflection-based shallow overlay; hand-roll the field copy if
// this ever shows up in profiles.
func overlayRef(target, referrer *Schema) *Schema {
	rv := reflect.ValueOf(referrer).Elem()
	rt := rv.Type()
	hasSiblings := false
	for i := range rt.NumField() {
		if rt.Field(i).Name == "Ref" {
			continue
		}
		if !rv.Field(i).IsZero() {
			hasSiblings = true
			break
		}
	}
	if !hasSiblings {
		return target
	}

	merged := *target
	mv := reflect.ValueOf(&merged).Elem()
	for i := range rt.NumField() {
		if rt.Field(i).Name == "Ref" {
			continue
		}
		if f := rv.Field(i); !f.IsZero() {
			mv.Field(i).Set(f)
		}
	}
	return &merged
}
