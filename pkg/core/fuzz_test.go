package core_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Uttam-Mahata/omniql/pkg/core"
)

// FuzzExecute feeds random bytes as an OQLQuery to the engine and verifies
// that it never panics.  The engine must always return an OmniJSON response
// (possibly with an error field) rather than crashing.
func FuzzExecute(f *testing.F) {
	// Seed with some valid-ish queries.
	seeds := []string{
		`{"target":"t","action":"FIND","filter":{}}`,
		`{"target":"t","action":"INSERT","document":{"x":1}}`,
		`{"target":"t","action":"UPDATE","filter":{"id":1},"document":{"x":2}}`,
		`{"target":"t","action":"DELETE","filter":{"id":1}}`,
		`{"target":"t","action":"COUNT","filter":{}}`,
		`{"target":"","action":""}`,
		`{}`,
		`{"target":"t","action":"FIND","filter":{"$or":[{"a":1},{"b":2}]}}`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// The engine must not panic regardless of input.
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic with input %q: %v", data, r)
			}
		}()

		var query core.OQLQuery
		if err := json.Unmarshal(data, &query); err != nil {
			// Invalid JSON — skip; the fuzz target is the engine, not the parser.
			return
		}

		// Create a minimal engine with a no-op-style configuration.
		engine := core.NewEngine()

		ctx := context.Background()
		result, err := engine.Execute(ctx, query)
		// We do not care about the result content — only that it does not panic.
		// Engine.Execute should never return a Go error (it wraps errors in OmniJSON).
		_ = result
		_ = err
	})
}
