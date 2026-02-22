# OmniQL v0.3.0

> **Bug-fix & completeness release**  
> Fixes all critical blockers from v0.2.0: FFI driver registration, CLI, C# JSON serialisation, and adds a full FFI test suite.

---

## What's New in v0.3.0

| Area | Change |
|------|--------|
| FFI | `pkg/ffi` renamed to `package main` — `c-shared` builds now work |
| FFI | New `OmniQL_Route` export — bindings can map targets to drivers |
| FFI | New `OmniQL_RegisterSQLiteDriver`, `OmniQL_RegisterPostgresDriver`, `OmniQL_RegisterMongoDriver` exports |
| FFI | New `wrappers.go` (Go-typed shims for the test suite) |
| FFI | New `ffi_test.go` with 4 smoke tests |
| CLI | `-driver`, `-dsn`, `-db` flags; SQLite/Postgres/Mongo wired up at startup |
| C# | `[JsonPropertyName]` attributes on all DTOs — fixes silent JSON casing bug |
| C# | `RegisterSQLiteDriver`, `RegisterPostgresDriver`, `RegisterMongoDriver`, `Route` added to `OmniEngine` |
| C# | `Update`, `Delete`, `Count` added to `QueryBuilder` |
| Python | Updated CFFI declarations for 4 new symbols |
| Python | `route`, `register_sqlite_driver`, `register_postgres_driver`, `register_mongo_driver` (sync + async) |
| Python | Fixed deprecated `asyncio.get_event_loop()` |
| Java | New native declarations for routing and driver registration |
| Java | `route`, `registerSQLiteDriver`, `registerPostgresDriver`, `registerMongoDriver` on `OmniEngine` |
| Java | `insert`, `update`, `delete`, `count` added to `QueryBuilder` |
| TypeScript | `ffi.Library` map updated with 4 new symbols |
| TypeScript | `route`, `registerSQLiteDriver`, `registerPostgresDriver`, `registerMongoDriver` on `OmniEngine` |
| Dependencies | `github.com/lib/pq` added for Postgres `database/sql` support |

For the full diff see [CHANGELOG.md](CHANGELOG.md).

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

| Action   | Description             |
|----------|-------------------------|
| `FIND`   | Query / select records  |
| `INSERT` | Create a new record     |
| `UPDATE` | Modify existing records |
| `DELETE` | Remove records          |
| `COUNT`  | Count matching records  |

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

    // result.Data  — []map[string]interface{}
    // result.Meta  — driver name, total count, …
}
```

---

## Further Reading

- [CLI Reference](cli.md)
- [FFI / C API Reference](ffi.md)
- [Language Bindings Guide](bindings.md)
- [Changelog](CHANGELOG.md)

---

## Running Tests

```bash
go test ./...
# Expected output:
# ok  github.com/Uttam-Mahata/omniql/pkg/core
# ok  github.com/Uttam-Mahata/omniql/pkg/drivers/postgres
# ok  github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite
# ok  github.com/Uttam-Mahata/omniql/pkg/ffi
```
