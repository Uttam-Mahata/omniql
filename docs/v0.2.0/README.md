# OmniQL v0.2.0

> **Initial release**

OmniQL v0.2.0 establishes the core architecture: a unified query engine that
routes OQL queries to pluggable database drivers and returns results in the
standardised OmniJSON format.

---

## What's Included

| Component | Status |
|-----------|--------|
| Core Engine (`pkg/core`) | ✅ Complete |
| SQLite driver | ✅ Complete (integration tests) |
| PostgreSQL driver | ✅ Complete (unit tests for filter builder) |
| MongoDB driver | ✅ Complete (no tests) |
| FFI shared library (`pkg/ffi`) | ✅ Exports `NewEngine`, `FreeEngine`, `Execute`, `RegisterSchema`, `Free` |
| Python binding | ✅ Sync + async API via CFFI |
| Java binding | ✅ JNI + Gson, fluent `QueryBuilder` |
| C# binding | ✅ P/Invoke + `System.Text.Json`, async |
| TypeScript binding | ✅ ffi-napi, Promise-based |
| CLI (`cmd/omniql`) | ⚠️ Parses and routes queries but never registers a driver |

---

## Known Limitations in v0.2.0

### Critical (broken by default)
- **CLI always fails** with `DRIVER_ERROR` — no driver is registered at startup.
- **FFI has no driver registration exports** — language bindings can create an engine and register schemas, but can never connect to a real database. Every `Execute` call returns `DRIVER_ERROR`.
- **FFI has no `OmniQL_Route` export** — bindings cannot configure target-to-driver routing.
- **C# JSON casing bug** — `OQLQuery` properties are serialized with PascalCase keys (`Target`, `Action`…) but the Go engine expects camelCase (`target`, `action`…). All queries from C# are silently mis-parsed.
- **`pkg/ffi` declared as `package ffi`** — the `go build -buildmode=c-shared` command fails; the package must be `package main`.

### Minor
- `QueryOptions.Sort` defined in types but silently ignored by all drivers.
- `QueryOptions.Fields` (projection) defined but all drivers always return `SELECT *`.
- INSERT returns an empty `data` array — inserted ID/document is not surfaced.
- Python async uses deprecated `asyncio.get_event_loop()` (deprecated Python ≥ 3.10).
- TypeScript `execute` wraps a synchronous FFI call in a Promise without dispatching to a worker thread — blocks the event loop under load.
- MongoDB driver has zero test coverage.
- No FFI layer tests.

---

## Architecture

```
Language Binding (Java / Python / C# / TypeScript)
        │
        │  JSON-encoded OQLQuery
        ▼
┌──────────────────────────────────────┐
│          OmniQL Core Engine          │
│  1. Schema Resolver (validation)     │
│  2. Driver Selection (routing)       │
│  3. AST Translation (via driver)     │
│  4. Execution (native DB command)    │
│  5. Normalization → OmniJSON         │
└──────────────────────────────────────┘
        │
   ┌────┴────────────────────┐
   │                         │
   ▼                         ▼
Postgres Driver          SQLite Driver   … (Mongo, …)
```

---

## OQL Query Format

```json
{
  "target": "analytics_data",
  "action": "FIND",
  "filter": {
    "category": { "$in": ["electronics", "books"] },
    "price":    { "$lt": 500 }
  },
  "options": { "limit": 20 }
}
```

### Actions

| Action   | Description            |
|----------|------------------------|
| `FIND`   | Query / select records |
| `INSERT` | Create a new record    |
| `UPDATE` | Modify existing records |
| `DELETE` | Remove records         |
| `COUNT`  | Count matching records |

### Filter Operators

| Operator | Meaning            |
|----------|--------------------|
| `$eq`    | Equal              |
| `$ne`    | Not equal          |
| `$lt`    | Less than          |
| `$lte`   | Less than or eq    |
| `$gt`    | Greater than       |
| `$gte`   | Greater than or eq |
| `$in`    | In set             |
| `$nin`   | Not in set         |

---

## Getting Started (Go)

```go
import (
    "context"
    "github.com/Uttam-Mahata/omniql/pkg/core"
    "github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite"
)

func main() {
    drv, _ := sqlite.New(":memory:")

    engine := core.NewEngine()
    engine.RegisterDriver(drv)
    engine.Route("products", drv.Name())

    result, _ := engine.Execute(context.Background(), core.OQLQuery{
        Target: "products",
        Action: core.ActionFind,
        Filter: core.Filter{"price": map[string]interface{}{"$lt": 500}},
        Options: core.QueryOptions{Limit: 10},
    })
}
```

---

## Building the FFI Shared Library

> ⚠️ In v0.2.0 this command fails because `pkg/ffi` uses `package ffi` instead
> of `package main`. Fixed in v0.3.0.

```bash
# Fails in v0.2.0:
go build -buildmode=c-shared -o libomniql.so ./pkg/ffi
```

---

## FFI API (v0.2.0)

| Function | Description |
|----------|-------------|
| `OmniQL_NewEngine() → int` | Create engine, returns handle |
| `OmniQL_FreeEngine(handle)` | Release engine |
| `OmniQL_Execute(handle, queryJSON) → char*` | Execute query — always returns `DRIVER_ERROR` in bindings (no registration API) |
| `OmniQL_RegisterSchema(handle, schemaJSON) → char*` | Register collection schema |
| `OmniQL_Free(ptr)` | Free returned string |

---

## Running Tests

```bash
go test ./pkg/core/...
go test ./pkg/drivers/sqlite/...
go test ./pkg/drivers/postgres/...
go test ./...
```
