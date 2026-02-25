# OmniQL v0.7.0

> **Paradigm Coverage + DX release**
> SQL Server and Redis drivers, shared SQL utilities, terminal convenience
> methods for all language bindings, MySQL in the compliance suite, and
> fuzz tests for the core engine and FFI layer.

---

## What's New in v0.7.0

| Area | Change |
|------|--------|
| Driver | New SQL Server adapter (`pkg/drivers/sqlserver`) — `@pN` placeholders, `JSON_VALUE`, OFFSET/FETCH pagination |
| Driver | New Redis adapter (`pkg/drivers/redis`) — KV/cache paradigm, `<target>:<id>` key pattern, TTL support |
| Utilities | Shared `buildWhere` extracted to `pkg/drivers/sqlutil` — used by SQLite, Postgres, MySQL, SQL Server |
| Bindings | Terminal convenience methods on all four language bindings (`find_many`, `find_first`, `count`, `insert_one`, `update_many`, `delete_many`) |
| Testing | MySQL added to compliance suite (skip when `MYSQL_DSN` unset) |
| Testing | Fuzz tests for `pkg/core` (`FuzzExecute`) and `pkg/ffi` (`FuzzFFIExecute`) |

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
   ┌────┴──────────────────────────────────┐
   │       │          │         │          │
   ▼       ▼          ▼         ▼          ▼
SQLite  Postgres   MongoDB    MySQL    SQL Server   Redis
                                       (v0.7.0)   (v0.7.0)
```

---

## Drivers

| Driver | Package | Placeholder | Quote style | JSON path |
|--------|---------|-------------|-------------|-----------|
| SQLite | `pkg/drivers/sqlite` | `?` | `"ident"` | `json_extract(col, '$.path')` |
| PostgreSQL | `pkg/drivers/postgres` | `$N` | `"ident"` | `col->>'path'` |
| MongoDB | `pkg/drivers/mongo` | BSON | — | Dot notation |
| MySQL | `pkg/drivers/mysql` | `?` | `` `ident` `` | `col->>'$.path'` |
| SQL Server | `pkg/drivers/sqlserver` | `@pN` | `"ident"` | `JSON_VALUE(col, '$.path')` |
| Redis | `pkg/drivers/redis` | — | — | Key pattern `<target>:<id>` |

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
    drvsqlserver "github.com/Uttam-Mahata/omniql/pkg/drivers/sqlserver"
    drvredis     "github.com/Uttam-Mahata/omniql/pkg/drivers/redis"
)

func main() {
    engine := core.NewEngine()

    // SQL Server for transactional data
    sqlDrv, _ := drvsqlserver.New("sqlserver://sa:Pass@localhost:1433?database=app")
    engine.RegisterDriver(sqlDrv)
    engine.Route("orders", sqlDrv.Name())

    // Redis for session cache
    redisDrv, _ := drvredis.New("redis://localhost:6379/0")
    engine.RegisterDriver(redisDrv)
    engine.Route("sessions", redisDrv.Name())

    ctx := context.Background()

    // Terminal convenience: find without building OQLQuery manually
    result, _ := engine.Execute(ctx, core.OQLQuery{
        Target: "orders",
        Action: core.ActionFind,
        Filter: core.Filter{"status": "pending"},
        Options: core.QueryOptions{Limit: 10, Sort: map[string]int{"total": -1}},
    })
    _ = result
}
```

---

## Terminal Convenience Methods (Bindings)

### Python

```python
engine = OmniEngine()
driver = await engine.register_sqlite_driver(":memory:")
await engine.route("products", driver)

# Convenience methods — no Query object needed
docs  = await engine.find_many("products", {"category": "electronics"})
first = await engine.find_first("products", {"name": "Laptop"})
n     = await engine.count("products")
row   = await engine.insert_one("products", {"name": "Tablet", "price": 299.99})
await engine.update_many("products", {"name": "Tablet"}, {"price": 279.99})
await engine.delete_many("products", {"name": "Tablet"})
```

### Java

```java
OmniEngine engine = OmniEngine.create();
String driver = engine.registerSQLiteDriver(":memory:");
engine.route("products", driver);

List<Map<String, Object>> docs = engine.findMany("products", Map.of("category", "electronics"));
Map<String, Object> first = engine.findFirst("products", Map.of("name", "Laptop"));
long n = engine.count("products", null);
engine.insertOne("products", Map.of("name", "Tablet", "price", 299.99));
engine.updateMany("products", Map.of("name", "Tablet"), Map.of("price", 279.99));
engine.deleteMany("products", Map.of("name", "Tablet"));
```

### C#

```csharp
using var engine = new OmniEngine();
string driver = engine.RegisterSQLiteDriver(":memory:");
engine.Route("products", driver);

var docs  = await engine.FindManyAsync("products", new() { ["category"] = "electronics" });
var first = await engine.FindFirstAsync("products", new() { ["name"] = "Laptop" });
long n    = await engine.CountAsync("products");
await engine.InsertOneAsync("products", new() { ["name"] = "Tablet", ["price"] = 299.99 });
await engine.UpdateManyAsync("products", new() { ["name"] = "Tablet" }, new() { ["price"] = 279.99 });
await engine.DeleteManyAsync("products", new() { ["name"] = "Tablet" });
```

### TypeScript

```typescript
const engine = new OmniEngine();
const driver = engine.registerSQLiteDriver(':memory:');
engine.route('products', driver);

const docs  = await engine.findMany('products', { category: 'electronics' });
const first = await engine.findFirst('products', { name: 'Laptop' });
const n     = await engine.count('products');
await engine.insertOne('products', { name: 'Tablet', price: 299.99 });
await engine.updateMany('products', { name: 'Tablet' }, { price: 279.99 });
await engine.deleteMany('products', { name: 'Tablet' });
```

---

## Config File Mode

```yaml
# omniql.yaml
drivers:
  main_db:
    type: sqlserver
    dsn: "sqlserver://sa:Pass@localhost:1433?database=mydb"
  cache:
    type: redis
    dsn: "redis://localhost:6379/0"
  local:
    type: sqlite
    dsn: "./local.db"
  analytics:
    type: mysql
    dsn: "reader:pass@tcp(analytics:3306)/stats"

routes:
  orders:   main_db
  sessions: cache
  logs:     local
  events:   analytics
```

---

## Running Tests

```bash
# All tests (SQLite compliance always runs; others require env vars)
CGO_ENABLED=1 go test ./...

# SQL Server integration
SQLSERVER_DSN="sqlserver://sa:Pass@localhost:1433?database=test" \
  go test ./pkg/drivers/sqlserver/...

# Redis integration
REDIS_URL="redis://localhost:6379/0" go test ./pkg/drivers/redis/...

# MySQL in compliance suite
MYSQL_DSN="root:@tcp(127.0.0.1:3306)/test" go test ./pkg/compliance/...

# Fuzz (short run)
go test -fuzz=FuzzExecute    -fuzztime=30s ./pkg/core/...
go test -fuzz=FuzzFFIExecute -fuzztime=30s ./pkg/ffi/...
```

---

## Further Reading

- [CLI Reference](cli.md)
- [FFI / C API Reference](ffi.md)
- [Language Bindings Guide](bindings.md)
- [Changelog](CHANGELOG.md)
