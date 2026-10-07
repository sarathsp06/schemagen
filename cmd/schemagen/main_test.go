package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const testSchema = `{
	"type": "object",
	"properties": {
		"id": {"type": "integer", "minimum": 1},
		"name": {"type": "string", "minLength": 3}
	},
	"required": ["id", "name"]
}`

func runCLI(t *testing.T, args []string, stdin string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestRunNDJSON(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"-n", "3", "-seed", "7"}, testSchema)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errOut)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}
	for i, line := range lines {
		var v map[string]interface{}
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Fatalf("line %d not valid JSON: %v", i, err)
		}
		if _, ok := v["id"]; !ok {
			t.Errorf("line %d missing id", i)
		}
	}
}

func TestRunDeterministic(t *testing.T) {
	_, a, _ := runCLI(t, []string{"-n", "5", "-seed", "7"}, testSchema)
	_, b, _ := runCLI(t, []string{"-n", "5", "-seed", "7"}, testSchema)
	if a != b {
		t.Fatal("same seed produced different output")
	}
}

func TestRunInvalidSchema(t *testing.T) {
	code, _, errOut := runCLI(t, nil, `{not json`)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if errOut == "" {
		t.Fatal("expected stderr output")
	}
}

func TestRunPretty(t *testing.T) {
	code, out, errOut := runCLI(t, []string{"-n", "2", "-seed", "7", "-pretty"}, testSchema)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errOut)
	}
	dec := json.NewDecoder(bufio.NewReader(strings.NewReader(out)))
	count := 0
	for dec.More() {
		var v interface{}
		if err := dec.Decode(&v); err != nil {
			t.Fatalf("doc %d: %v", count, err)
		}
		count++
	}
	if count != 2 {
		t.Fatalf("got %d docs, want 2", count)
	}
	if !strings.Contains(out, "\n  ") {
		t.Fatal("output not indented")
	}
}
