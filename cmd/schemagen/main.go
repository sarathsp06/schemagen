// Command schemagen generates random JSON data conforming to a JSON Schema.
//
// Usage: schemagen [flags] [schema-file]
//
// With no schema-file argument, or with "-", the schema is read from stdin.
// Output is one JSON document per line (NDJSON) on stdout.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sarathsp06/schemagen"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("schemagen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: schemagen [flags] [schema-file]")
		fmt.Fprintln(stderr, "reads schema from stdin when no file is given or file is \"-\"")
		fs.PrintDefaults()
	}
	n := fs.Int("n", 1, "number of documents to generate")
	seed := fs.Int64("seed", 0, "random seed (default: time-based)")
	all := fs.Bool("all", false, "generate all optional fields")
	p := fs.Float64("p", -1, "probability [0,1] of including each optional field")
	depth := fs.Int("depth", 10, "maximum recursion depth")
	pretty := fs.Bool("pretty", false, "indent output")
	defaults := fs.Bool("defaults", false, "use schema default values when present")
	lenient := fs.Bool("lenient", false, "emit minimal values at depth limit instead of erroring")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	var schemaJSON []byte
	var err error
	if file := fs.Arg(0); file == "" || file == "-" {
		schemaJSON, err = io.ReadAll(stdin)
	} else {
		schemaJSON, err = os.ReadFile(file)
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	g := schemagen.NewGenerator().
		SetMaxDepth(*depth).
		SetGenerateAllFields(*all).
		SetOptionalProbability(*p).
		SetUseDefaults(*defaults).
		SetLenientDepth(*lenient)
	seedSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			seedSet = true
		}
	})
	if seedSet {
		g.SetSeed(*seed)
	}

	docs, err := g.GenerateN(schemaJSON, *n)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	w := bufio.NewWriter(stdout)
	enc := json.NewEncoder(w)
	if *pretty {
		enc.SetIndent("", "  ")
	}
	for _, doc := range docs {
		if err := enc.Encode(doc); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}
