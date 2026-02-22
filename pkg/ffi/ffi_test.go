// Package ffi tests exercise the FFI helper logic and driver wiring from the
// Go side.  They don't build a shared library — they call the Go-typed wrapper
// functions defined in wrappers.go that delegate to the C-typed FFI exports.
package main

import (
	"database/sql"
	"encoding/json"
	"testing"

	drvsqlite "github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

// -------------------------------------------------------------------------
// Tests
// -------------------------------------------------------------------------

// TestNewFreeEngine verifies that handles increment and that FreeEngine
// removes the entry from the registry.
func TestNewFreeEngine(t *testing.T) {
	h1 := goNewEngine()
	h2 := goNewEngine()

	if h1 <= 0 || h2 <= 0 {
		t.Fatalf("expected positive handles, got %d and %d", h1, h2)
	}
	if h1 == h2 {
		t.Fatalf("handles must be unique, both are %d", h1)
	}

	goFreeEngine(h1)
	goFreeEngine(h2)

	// After freeing, goGetEngine must return nil.
	if goGetEngine(h1) != nil {
		t.Fatalf("engine h1 should be nil after free")
	}
}

// TestInvalidHandleReturnsError verifies that using an unknown handle
// returns an INVALID_HANDLE error in the JSON response.
func TestInvalidHandleReturnsError(t *testing.T) {
	raw := goExecute(99999, `{"target":"x","action":"FIND","filter":{}}`)

	var resp struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v\nraw: %s", err, raw)
	}
	if resp.Error == nil || resp.Error.Code != "INVALID_HANDLE" {
		t.Fatalf("expected INVALID_HANDLE error, got: %s", raw)
	}
}

// TestRegisterSQLiteDriverViaFFI exercises goRegisterSQLiteDriver and
// goRoute, then executes a FIND query through the full FFI pipeline.
func TestRegisterSQLiteDriverViaFFI(t *testing.T) {
	handle := goNewEngine()
	defer goFreeEngine(handle)

	// Register an in-memory SQLite driver via the wrapper.
	regRaw := goRegisterSQLiteDriver(handle, ":memory:")
	var regResp map[string]string
	if err := json.Unmarshal([]byte(regRaw), &regResp); err != nil {
		t.Fatalf("parse RegisterSQLiteDriver response: %v\nraw: %s", err, regRaw)
	}
	driverName, ok := regResp["driver"]
	if !ok || driverName != "sqlite" {
		t.Fatalf("expected driver=sqlite, got %q (raw: %s)", driverName, regRaw)
	}

	// Route the target via the wrapper.
	routeRaw := goRoute(handle, "widgets", driverName)
	if routeRaw != "" {
		t.Fatalf("goRoute should return empty string on success, got: %s", routeRaw)
	}

	// The in-memory SQLite DB started empty; FIND against a missing table
	// returns EXECUTION_ERROR — correct, proves routing worked.
	queryRaw := goExecute(handle, `{"target":"widgets","action":"FIND","filter":{}}`)
	var result struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(queryRaw), &result); err != nil {
		t.Fatalf("parse Execute response: %v\nraw: %s", err, queryRaw)
	}
	// Must NOT see DRIVER_ERROR (no driver registered).
	if result.Error != nil && result.Error.Code == "DRIVER_ERROR" {
		t.Fatalf("driver was not registered, got DRIVER_ERROR: %s", queryRaw)
	}
}

// TestRouteAndFindDirect seeds an in-memory SQLite DB with a table, then runs
// a full FIND round-trip to validate the engine wiring used by the FFI layer.
func TestRouteAndFindDirect(t *testing.T) {
	const target = "products"

	handle := goNewEngine()
	defer goFreeEngine(handle)

	// Build and seed the SQLite DB directly so we control the schema.
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, price REAL)`); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO products (name, price) VALUES (?, ?)`, "Widget", 9.99); err != nil {
		t.Fatalf("INSERT seed: %v", err)
	}

	// Register driver + route directly on the engine via Go API.
	eng := goGetEngine(handle)
	drv := drvsqlite.NewFromDB(db)
	eng.RegisterDriver(drv)
	eng.Route(target, drv.Name())

	// Execute a FIND via the FFI wrapper.
	raw := goExecute(handle, `{"target":"products","action":"FIND","filter":{},"options":{"limit":10}}`)

	var result struct {
		Data  []map[string]interface{} `json:"data"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Meta struct {
			Driver string `json:"driver"`
			Target string `json:"target"`
		} `json:"meta"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("json.Unmarshal: %v\nraw: %s", err, raw)
	}
	if result.Error != nil {
		t.Fatalf("unexpected error: %+v\nraw: %s", result.Error, raw)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 row, got %d\nraw: %s", len(result.Data), raw)
	}
	if result.Meta.Driver != "sqlite" {
		t.Fatalf("expected meta.driver=sqlite, got %q", result.Meta.Driver)
	}
	if result.Meta.Target != target {
		t.Fatalf("expected meta.target=%q, got %q", target, result.Meta.Target)
	}
}

// TestRegisterSchema verifies goRegisterSchema returns an empty string on success.
func TestRegisterSchema(t *testing.T) {
	handle := goNewEngine()
	defer goFreeEngine(handle)

	schema := `{"name":"items","fields":{"id":{"type":"int","required":true},"name":{"type":"string","required":false}}}`
	raw := goRegisterSchema(handle, schema)
	if raw != "" {
		t.Fatalf("expected empty success response, got: %s", raw)
	}
}


