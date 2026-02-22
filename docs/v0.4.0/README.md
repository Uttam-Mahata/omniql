# OmniQL v0.4.0

> **Feature & completeness release**  
> Adds logical operators ($or/$and), sorting, projection, and fixes INSERT result behavior.

---

## What's New in v0.4.0

| Area | Change |
|------|--------|
| OQL | Support for logical operators `$or` and `$and` across all drivers. |
| OQL | Support for `Sort` options (ASC/DESC) in `FIND` queries. |
| OQL | Support for `Fields` (projection) options in `FIND` queries. |
| Drivers | `INSERT` actions now return the inserted document/id instead of an empty array. |
| Drivers | MongoDB driver now has a full integration test suite. |

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
    "$or": [
      { "category": "electronics" },
      { "price": { "$lt": 500 } }
    ]
  },
  "options": { 
    "limit": 20,
    "sort": { "price": -1 },
    "fields": { "name": 1, "price": 1 }
  }
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
| `$or`    | Logical OR         |
| `$and`   | Logical AND        |

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
        Options: core.QueryOptions{
            Limit:  10,
            Sort:   map[string]int{"price": -1},
            Fields: map[string]interface{}{"name": 1, "price": 1},
        },
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
# ok  github.com/Uttam-Mahata/omniql/pkg/drivers/mongo
# ok  github.com/Uttam-Mahata/omniql/pkg/drivers/postgres
# ok  github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite
# ok  github.com/Uttam-Mahata/omniql/pkg/ffi
```
