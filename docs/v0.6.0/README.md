# OmniQL v0.6.0

> **Relational Parity + Distribution & Tooling release**
> MySQL driver, cross-driver compliance suite, `batchInsert` API for all bindings,
> `omniql.yaml` config, and GoReleaser binary distribution.

---

## What's New in v0.6.0

| Area | Change |
|------|--------|
| Driver | New MySQL adapter (`pkg/drivers/mysql`) |
| Bindings | `batchInsert` convenience method on all four language bindings |
| Testing | 50+ canonical compliance tests (`pkg/compliance`) — run against every driver |
| CLI | `-config omniql.yaml` mode for named connections and route tables |
| Distribution | GoReleaser: Linux deb/rpm, macOS Homebrew tap, Windows Scoop bucket |
| Bug fix | BatchInsert column ordering in SQLite and Postgres (map iteration determinism) |
| Bug fix | SQLite OFFSET-without-LIMIT syntax error |

For the full diff see [CHANGELOG.md](CHANGELOG.md).

---

## Architecture

```
Language Binding (Java / Python / C# / TypeScript)
        │
        │  JSON-encoded OQLQuery
        ▼
┌──────────────────────────────────────────────────┐
│               OmniQL Core Engine                 │
│  1. Schema Registry (optional validation)        │
│  2. Driver Selection (explicit route → fallback) │
│  3. AST Translation (OQL → native query)         │
│  4. Execution (native DB command)                │
│  5. Normalisation → OmniJSON                     │
└──────────────────────────────────────────────────┘
        │
   ┌────┴──────────────────────────┐
   │         │          │          │
   ▼         ▼          ▼          ▼
SQLite   Postgres    MongoDB     MySQL   (v0.6.0)
```

---

## Drivers

| Driver | Package | Placeholder | Quote style | JSON path |
|--------|---------|-------------|-------------|-----------|
| SQLite | `pkg/drivers/sqlite` | `?` | `"ident"` | `json_extract(col, '$.path')` |
| PostgreSQL | `pkg/drivers/postgres` | `$N` | `"ident"` | `col->>'path'` |
| MongoDB | `pkg/drivers/mongo` | BSON | — | Dot notation |
| MySQL | `pkg/drivers/mysql` | `?` | `` `ident` `` | `col->>'$.path'` |

---

## OQL Query Format

```json
{
  "target":    "orders",
  "action":    "FIND",
  "filter":    { "status": { "$in": ["pending", "shipped"] }, "total": { "$gt": 100 } },
  "document":  { "field": "value" },
  "documents": [ { "field": "value" } ],
  "options": {
    "limit":  20,
    "skip":   0,
    "sort":   { "total": -1 },
    "fields": { "id": 1, "status": 1, "total": 1 }
  }
}
```

### Actions

| Action | Description |
|--------|-------------|
| `FIND` | Query / select records |
| `INSERT` | Create a single record |
| `BATCH_INSERT` | Create multiple records in one operation |
| `UPDATE` | Modify existing records |
| `DELETE` | Remove records |
| `COUNT` | Count matching records |

### Filter Operators

| Operator | Meaning |
|----------|---------|
| `$eq` | Equal (also: bare value) |
| `$ne` | Not equal |
| `$lt` | Less than |
| `$lte` | Less than or equal |
| `$gt` | Greater than |
| `$gte` | Greater than or equal |
| `$in` | In set |
| `$nin` | Not in set |
| `$or` | Logical OR (recursive) |
| `$and` | Logical AND (recursive) |

---

## Getting Started (Go)

```go
import (
    "context"
    "github.com/Uttam-Mahata/omniql/pkg/core"
    drvmysql  "github.com/Uttam-Mahata/omniql/pkg/drivers/mysql"
    drvsqlite "github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite"
)

func main() {
    engine := core.NewEngine()

    // Register SQLite for local data
    sqliteDrv, _ := drvsqlite.New(":memory:")
    engine.RegisterDriver(sqliteDrv)
    engine.Route("sessions", sqliteDrv.Name())

    // Register MySQL for application data
    mysqlDrv, _ := drvmysql.New("user:pass@tcp(localhost:3306)/app?parseTime=true")
    engine.RegisterDriver(mysqlDrv)
    engine.Route("orders", mysqlDrv.Name())

    ctx := context.Background()

    // Single insert
    engine.Execute(ctx, core.OQLQuery{
        Target:   "orders",
        Action:   core.ActionInsert,
        Document: map[string]interface{}{"customer": "Alice", "total": 149.99},
    })

    // Batch insert
    engine.Execute(ctx, core.OQLQuery{
        Target: "sessions",
        Action: core.ActionBatchInsert,
        Documents: []map[string]interface{}{
            {"token": "abc", "user_id": 1},
            {"token": "def", "user_id": 2},
        },
    })

    // Filtered FIND
    result, _ := engine.Execute(ctx, core.OQLQuery{
        Target: "orders",
        Action: core.ActionFind,
        Filter: core.Filter{
            "$and": []interface{}{
                map[string]interface{}{"total": map[string]interface{}{"$gte": 100.0}},
                map[string]interface{}{"status": map[string]interface{}{"$ne": "cancelled"}},
            },
        },
        Options: core.QueryOptions{
            Limit: 10,
            Sort:  map[string]int{"total": -1},
        },
    })
    _ = result
}
```

---

## Config File Mode

Create `omniql.yaml` in your working directory:

```yaml
drivers:
  main_db:
    type: postgres
    dsn: "postgres://user:pass@localhost/mydb?sslmode=disable"
  local:
    type: sqlite
    dsn: "./local.db"
  analytics:
    type: mysql
    dsn: "reader:pass@tcp(analytics:3306)/stats"

routes:
  users:    main_db
  cache:    local
  events:   analytics
```

Then run queries without any `-driver`/`-dsn` flags:

```bash
omniql -query '{"target":"users","action":"FIND","filter":{"active":true}}'
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
# All tests (SQLite compliance always runs; others require env vars)
CGO_ENABLED=1 go test ./...

# Compliance suite only
CGO_ENABLED=1 go test ./pkg/compliance/...

# With PostgreSQL integration
POSTGRES_DSN="postgres://user:pass@localhost/testdb?sslmode=disable" \
  CGO_ENABLED=1 go test ./pkg/compliance/... ./pkg/drivers/postgres/...

# With MySQL integration
MYSQL_DSN="root:@tcp(127.0.0.1:3306)/test" \
  CGO_ENABLED=1 go test ./pkg/drivers/mysql/...
```

Expected output:

```
ok  github.com/Uttam-Mahata/omniql/pkg/compliance
ok  github.com/Uttam-Mahata/omniql/pkg/core
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/mongo
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/mysql
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/postgres
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite
ok  github.com/Uttam-Mahata/omniql/pkg/ffi
```
