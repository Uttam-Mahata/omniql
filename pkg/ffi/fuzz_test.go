package main

import (
	"testing"
)

// FuzzFFIExecute feeds random handle values and malformed JSON to the
// goExecute wrapper and verifies that the FFI layer never panics.
//
// The FFI layer must always return a valid JSON string — either a successful
// OmniJSON response or a JSON-encoded error — regardless of the inputs.
func FuzzFFIExecute(f *testing.F) {
	// Seed with some representative inputs.
	seeds := []struct {
		handle    int
		queryJSON string
	}{
		{0, `{"target":"t","action":"FIND"}`},
		{1, `{"target":"t","action":"INSERT","document":{"x":1}}`},
		{-1, `{}`},
		{999999, `not-json`},
		{0, ``},
		{0, `{"target":"","action":""}`},
		{0, `null`},
		{0, `[]`},
	}

	for _, s := range seeds {
		f.Add(s.handle, s.queryJSON)
	}

	// Create a real engine handle so that some inputs exercise actual driver
	// dispatch code paths.
	handle := goNewEngine()
	defer goFreeEngine(handle)

	// Register an in-memory SQLite driver so FIND/INSERT/etc. are exercised.
	goRegisterSQLiteDriver(handle, ":memory:")

	f.Fuzz(func(t *testing.T, fuzzHandle int, queryJSON string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic with handle=%d query=%q: %v", fuzzHandle, queryJSON, r)
			}
		}()

		// Exercise with the fuzzed handle (likely invalid → error path).
		result := goExecute(fuzzHandle, queryJSON)
		if result == "" {
			t.Error("expected non-empty JSON response")
		}

		// Also exercise with the valid handle to hit driver code paths.
		result2 := goExecute(handle, queryJSON)
		if result2 == "" {
			t.Error("expected non-empty JSON response for valid handle")
		}
	})
}
