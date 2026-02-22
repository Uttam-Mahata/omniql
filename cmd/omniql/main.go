// Package main is the OmniQL CLI entry-point.
//
// It provides a minimal REPL-style interface for ad-hoc OQL query execution
// against any registered driver.  In production deployments the Core Engine
// is typically embedded via the FFI layer rather than used directly.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Uttam-Mahata/omniql/pkg/core"
)

func main() {
	queryFlag := flag.String("query", "", "JSON-encoded OQL query to execute")
	flag.Parse()

	if *queryFlag == "" {
		fmt.Fprintln(os.Stderr, "Usage: omniql -query '<OQL JSON>'")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example:")
		fmt.Fprintln(os.Stderr, `  omniql -query '{"target":"users","action":"FIND","filter":{"status":"active"},"options":{"limit":10}}'`)
		os.Exit(1)
	}

	var query core.OQLQuery
	if err := json.Unmarshal([]byte(*queryFlag), &query); err != nil {
		log.Fatalf("failed to parse query: %v", err)
	}

	engine := core.NewEngine()

	result, err := engine.Execute(context.Background(), query)
	if err != nil {
		log.Fatalf("execution error: %v", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		log.Fatalf("failed to encode result: %v", err)
	}
}
