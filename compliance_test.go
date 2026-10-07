package schemagen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// compileSchema compiles a schema JSON string under draft 2020-12 with
// format assertion enabled.
func compileSchema(t *testing.T, schemaJSON string) *jsonschema.Schema {
	t.Helper()
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(schemaJSON))
	if err != nil {
		t.Fatalf("unmarshal schema: %v\nschema: %s", err, schemaJSON)
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.AssertFormat()
	if err := c.AddResource("schema.json", doc); err != nil {
		t.Fatalf("add resource: %v", err)
	}
	sch, err := c.Compile("schema.json")
	if err != nil {
		t.Fatalf("compile schema: %v\nschema: %s", err, schemaJSON)
	}
	return sch
}

type complianceCase struct {
	name    string
	schema  string
	lenient bool // pair with SetLenientDepth(true) for recursive schemas
}

var complianceCases = []complianceCase{
	// --- types ---
	{name: "type string", schema: `{"type":"string"}`},
	{name: "type null", schema: `{"type":"null"}`},
	{name: "type boolean", schema: `{"type":"boolean"}`},
	{name: "type array of types", schema: `{"type":["string","integer","null"]}`},

	// --- strings ---
	{name: "string min/maxLength", schema: `{"type":"string","minLength":5,"maxLength":10}`},
	{name: "string pattern", schema: `{"type":"string","pattern":"^[A-Z]{2}[0-9]{4}$"}`},

	// --- numbers ---
	{name: "integer bounds", schema: `{"type":"integer","minimum":-5,"maximum":5}`},
	{name: "number bounds", schema: `{"type":"number","minimum":1.5,"maximum":2.5}`},
	{name: "integer exclusive bounds", schema: `{"type":"integer","exclusiveMinimum":0,"exclusiveMaximum":10}`},
	{name: "number exclusive bounds", schema: `{"type":"number","exclusiveMinimum":0.5,"exclusiveMaximum":1.5}`},
	{name: "integer multipleOf", schema: `{"type":"integer","multipleOf":3,"minimum":10,"maximum":100}`},
	{name: "number multipleOf", schema: `{"type":"number","multipleOf":0.5,"minimum":0,"maximum":10}`},

	// --- enum / const ---
	{name: "enum mixed", schema: `{"enum":["red","green",42,null]}`},
	{name: "const object", schema: `{"const":{"a":1}}`},
	{name: "const null", schema: `{"const":null}`},
	{name: "const false", schema: `{"const":false}`},

	// --- objects ---
	{name: "object required/properties", schema: `{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer","minimum":0}},"required":["name","age"]}`},
	{name: "object additionalProperties schema", schema: `{"type":"object","properties":{"id":{"type":"integer"}},"required":["id"],"additionalProperties":{"type":"string"},"minProperties":3}`},
	{name: "object additionalProperties true", schema: `{"type":"object","additionalProperties":true,"minProperties":2}`},
	{name: "object patternProperties", schema: `{"type":"object","patternProperties":{"^num_":{"type":"integer"},"^str_":{"type":"string"}},"additionalProperties":false,"minProperties":1}`},
	{name: "object propertyNames", schema: `{"type":"object","propertyNames":{"pattern":"^[a-z]{3,8}$"},"additionalProperties":{"type":"integer"},"minProperties":2,"maxProperties":4}`},
	{name: "object min/maxProperties", schema: `{"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"string"},"c":{"type":"boolean"}},"minProperties":2,"maxProperties":3}`},
	{name: "object dependentRequired", schema: `{"type":"object","properties":{"credit":{"type":"string"},"billing":{"type":"string"}},"dependentRequired":{"credit":["billing"]}}`},
	{name: "object dependentSchemas", schema: `{"type":"object","properties":{"name":{"type":"string"}},"dependentSchemas":{"name":{"properties":{"age":{"type":"integer","minimum":0}},"required":["age"]}}}`},

	// --- arrays ---
	{name: "array items min/maxItems", schema: `{"type":"array","items":{"type":"string"},"minItems":1,"maxItems":3}`},
	{name: "array uniqueItems", schema: `{"type":"array","items":{"type":"integer","minimum":0,"maximum":1000000},"uniqueItems":true,"minItems":3,"maxItems":5}`},
	{name: "array prefixItems with rest", schema: `{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}],"items":{"type":"boolean"},"minItems":2,"maxItems":5}`},
	{name: "array contains", schema: `{"type":"array","items":{"type":"integer","minimum":0,"maximum":200},"contains":{"minimum":150},"minContains":1,"minItems":2,"maxItems":6}`},
	{name: "array min/maxContains", schema: `{"type":"array","items":{"type":"integer"},"contains":{"type":"integer"},"minContains":2,"maxContains":5,"minItems":2,"maxItems":5}`},

	// --- composition ---
	{name: "oneOf with siblings", schema: `{"type":"string","minLength":3,"oneOf":[{"maxLength":5},{"minLength":10,"maxLength":20}]}`},
	{name: "anyOf", schema: `{"anyOf":[{"type":"string","minLength":5},{"type":"integer","minimum":10}]}`},
	{name: "allOf", schema: `{"allOf":[{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]},{"properties":{"b":{"type":"integer"}},"required":["b"]}]}`},
	{name: "if/then", schema: `{"type":"object","properties":{"kind":{"type":"string","enum":["a","b"]},"aVal":{"type":"string"}},"required":["kind"],"if":{"properties":{"kind":{"const":"a"}},"required":["kind"]},"then":{"required":["aVal"]}}`},

	// --- $ref ---
	{name: "ref to $defs", schema: `{"type":"object","properties":{"addr":{"$ref":"#/$defs/address"}},"required":["addr"],"$defs":{"address":{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}}}`},
	// recursive: node has no required fields so the lenient depth-cap value {} stays valid
	{name: "recursive ref", schema: `{"$ref":"#/$defs/node","$defs":{"node":{"type":"object","properties":{"name":{"type":"string"},"child":{"$ref":"#/$defs/node"}}}}}`, lenient: true},

	// --- nested combo ---
	{name: "nested combo", schema: `{"type":"object","properties":{"users":{"type":"array","minItems":1,"maxItems":3,"items":{"type":"object","properties":{"id":{"type":"string","format":"uuid"},"score":{"type":"number","minimum":0,"maximum":1}},"required":["id","score"]}}},"required":["users"]}`},
}

// formats exercised individually so failures localize per format.
var formatNames = []string{
	"uuid", "email", "date-time", "date", "time",
	"ipv4", "ipv6", "uri", "hostname", "duration", "json-pointer",
}

func TestCompliance(t *testing.T) {
	cases := complianceCases
	for _, f := range formatNames {
		cases = append(cases, complianceCase{
			name:   "format " + f,
			schema: `{"type":"string","format":"` + f + `"}`,
		})
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sch := compileSchema(t, tc.schema)
			for _, all := range []bool{false, true} {
				for seed := int64(1); seed <= 15; seed++ {
					g := NewGenerator().SetSeed(seed).SetGenerateAllFields(all)
					if tc.lenient {
						g.SetLenientDepth(true)
					}
					out, err := g.GenerateBytes([]byte(tc.schema))
					if err != nil {
						t.Errorf("generate failed: seed=%d all=%v err=%v\nschema: %s", seed, all, err, tc.schema)
						continue
					}
					inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(out))
					if err != nil {
						t.Errorf("output not valid JSON: seed=%d all=%v err=%v\nschema: %s\noutput: %s", seed, all, err, tc.schema, out)
						continue
					}
					if err := sch.Validate(inst); err != nil {
						t.Errorf("output violates schema: seed=%d all=%v\nschema: %s\noutput: %s\nerr: %v", seed, all, tc.schema, out, err)
					}
				}
			}
		})
	}
}

// Regression for the json.RawMessage const fix: const null must yield nil,
// const false must yield false (not be mistaken for "absent").
func TestConstNullAndFalse(t *testing.T) {
	g := NewGenerator().SetSeed(1)

	v, err := g.Generate([]byte(`{"const":null}`))
	if err != nil {
		t.Fatalf("const null: %v", err)
	}
	if v != nil {
		t.Errorf("const null: got %#v, want nil", v)
	}

	v, err = g.Generate([]byte(`{"const":false}`))
	if err != nil {
		t.Fatalf("const false: %v", err)
	}
	b, ok := v.(bool)
	if !ok || b {
		t.Errorf("const false: got %#v, want false", v)
	}
}
