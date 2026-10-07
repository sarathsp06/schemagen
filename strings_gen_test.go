package schemagen

import (
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
)

func intPtr(n int) *int { return &n }

func TestBuiltinFormats(t *testing.T) {
	checks := map[string]func(t *testing.T, s string){
		"duration": func(t *testing.T, s string) {
			if !regexp.MustCompile(`^P(\d+D)?(T\d+H\d+M\d+S)?$`).MatchString(s) || s == "P" {
				t.Errorf("bad ISO duration %q", s)
			}
		},
		"json-pointer": func(t *testing.T, s string) {
			if !strings.HasPrefix(s, "/") {
				t.Errorf("json-pointer %q must start with /", s)
			}
		},
		"relative-json-pointer": func(t *testing.T, s string) {
			if !regexp.MustCompile(`^[0-9]+(/|#?$)`).MatchString(s) {
				t.Errorf("bad relative-json-pointer %q", s)
			}
		},
		"regex": func(t *testing.T, s string) {
			if _, err := regexp.Compile(s); err != nil {
				t.Errorf("regex format produced non-compiling %q: %v", s, err)
			}
		},
		"uri-reference": func(t *testing.T, s string) {
			if _, err := url.Parse(s); err != nil {
				t.Errorf("uri-reference %q does not parse: %v", s, err)
			}
		},
		"iri-reference": func(t *testing.T, s string) {
			if _, err := url.Parse(s); err != nil {
				t.Errorf("iri-reference %q does not parse: %v", s, err)
			}
		},
		"idn-email": func(t *testing.T, s string) {
			if !strings.Contains(s, "@") {
				t.Errorf("idn-email %q has no @", s)
			}
		},
		"iri": func(t *testing.T, s string) {
			u, err := url.Parse(s)
			if err != nil || u.Scheme == "" {
				t.Errorf("iri %q is not an absolute URL", s)
			}
		},
	}
	for format, check := range checks {
		t.Run(format, func(t *testing.T) {
			for seed := range int64(20) {
				g := NewGenerator().SetSeed(seed)
				s, err := g.generateString(&Schema{Format: format})
				if err != nil {
					t.Fatalf("seed %d: %v", seed, err)
				}
				check(t, s)
			}
		})
	}
}

func TestRegisterFormatWinsOverBuiltin(t *testing.T) {
	g := NewGenerator().SetSeed(1)
	g.RegisterFormat("employee-id", func(f *gofakeit.Faker) string { return "EMP-42" })
	g.RegisterFormat("uuid", func(f *gofakeit.Faker) string { return "custom-uuid" })

	s, err := g.generateString(&Schema{Format: "employee-id"})
	if err != nil || s != "EMP-42" {
		t.Fatalf("custom format: got %q, %v", s, err)
	}
	s, err = g.generateString(&Schema{Format: "uuid"})
	if err != nil || s != "custom-uuid" {
		t.Fatalf("custom format must override builtin: got %q, %v", s, err)
	}
}

func TestXFakerTemplate(t *testing.T) {
	for seed := range int64(20) {
		g := NewGenerator().SetSeed(seed)
		s, err := g.generateString(&Schema{XFaker: "{number:1,9}"})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if !regexp.MustCompile(`^[1-9]$`).MatchString(s) {
			t.Errorf("seed %d: x-faker template produced %q", seed, s)
		}
	}
}

func TestXFakerBadTemplateFallsThrough(t *testing.T) {
	g := NewGenerator().SetSeed(1)
	s, err := g.generateString(&Schema{XFaker: "{nosuchfunc}", MinLength: intPtr(3)})
	if err != nil {
		t.Fatalf("bad template must fall through: %v", err)
	}
	if len(s) < 3 {
		t.Errorf("fallback ignored minLength: %q", s)
	}
}

func TestStrictFormats(t *testing.T) {
	g := NewGenerator().SetSeed(1).SetStrictFormats(true)
	if _, err := g.generateString(&Schema{Format: "no-such-format"}); err == nil {
		t.Fatal("strict mode must error on unknown format")
	} else if !strings.Contains(err.Error(), "no-such-format") {
		t.Errorf("error must name the format: %v", err)
	}
}

func TestUnknownFormatFallbackRespectsLength(t *testing.T) {
	for seed := range int64(20) {
		g := NewGenerator().SetSeed(seed)
		s, err := g.generateString(&Schema{
			Format:    "no-such-format",
			MinLength: intPtr(30), MaxLength: intPtr(40),
		})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if len(s) < 30 || len(s) > 40 {
			t.Errorf("seed %d: length %d outside [30,40]: %q", seed, len(s), s)
		}
	}
}

func TestPatternWithLengthBounds(t *testing.T) {
	re := regexp.MustCompile(`^[a-z]{5}$`)
	for seed := range int64(20) {
		g := NewGenerator().SetSeed(seed)
		schema := &Schema{Pattern: "^[a-z]{5}$", MinLength: intPtr(5), MaxLength: intPtr(5)}
		s, err := g.generateString(schema)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if len(s) != 5 || !re.MatchString(s) {
			t.Errorf("seed %d: got %q", seed, s)
		}
	}
}

func TestPatternCacheReuse(t *testing.T) {
	g := NewGenerator().SetSeed(7)
	re := regexp.MustCompile(`^[A-Z]{3}[0-9]{2}$`)
	for range 10 {
		s, err := g.generateStringFromPattern("^[A-Z]{3}[0-9]{2}$")
		if err != nil {
			t.Fatal(err)
		}
		if !re.MatchString(s) {
			t.Errorf("got %q", s)
		}
	}
	if len(g.patternCache) != 1 {
		t.Errorf("expected 1 cached pattern, got %d", len(g.patternCache))
	}
}

func TestPatternNilCache(t *testing.T) {
	g := NewGenerator().SetSeed(1)
	g.patternCache = nil
	if _, err := g.generateStringFromPattern("[a-z]+"); err != nil {
		t.Fatal(err)
	}
}

func TestRandomLengthBounds(t *testing.T) {
	for seed := range int64(20) {
		g := NewGenerator().SetSeed(seed)
		s, err := g.generateString(&Schema{MinLength: intPtr(4), MaxLength: intPtr(8)})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if len(s) < 4 || len(s) > 8 {
			t.Errorf("seed %d: length %d outside [4,8]: %q", seed, len(s), s)
		}
	}
}
