# OmniQL — One Query Language, Every Database.

> **Version:** v0.8.0
> **Module:** `github.com/Uttam-Mahata/omniql`

OmniQL is a unified data access layer that abstracts away the complexity of
database-specific protocols and languages. It exposes a single, model-agnostic
query format — **OQL (OmniQL Query Language)** — and routes queries to
pluggable database drivers, returning results in a standardised **OmniJSON**
format.

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
Postgres Driver          SQLite Driver   … (Mongo, Redis, ES, …)
```

The engine is a **Traffic Controller**: it never knows the details of any
specific database. Each driver implements the `core.Driver` interface and
handles its own query translation and execution.

---

## Supported Databases (Roadmap)

| Category           | Target Databases                                         | Unified Action                              |
|--------------------|----------------------------------------------------------|---------------------------------------------|
| Relational (SQL)   | PostgreSQL, MySQL, SQLite, MariaDB, SQL Server, Oracle   | `SELECT / INSERT / UPDATE / DELETE`         |
| Document (NoSQL)   | MongoDB, CouchDB, DynamoDB, Firestore                    | BSON/JSON-based lookup and filters          |
| Key-Value / Cache  | Redis, Dragonfly, Memcached, Valkey                      | `GET / SET` with TTL support                |
| Search Engines     | Elasticsearch, Meilisearch, Algolia, Typesense           | Full-text search and vector queries         |
| Time-Series        | InfluxDB, TimescaleDB, QuestDB                           | Time-windowed aggregations                  |
| Graph              | Neo4j, SurrealDB, Memgraph                               | Relationship / traversal queries            |

**v0.8.0 Standard Library drivers:** PostgreSQL · MongoDB · SQLite

---

## OQL Query Format

OQL is **model-agnostic**: the same query object works regardless of whether
the underlying store is a table, a collection, or a search index.

```json
{
  "target": "analytics_data",
  "action": "FIND",
  "filter": {
    "category": { "$in": ["electronics", "books"] },
    "price":    { "$lt": 500 }
  },
  "options": {
    "limit": 20,
    "sort": { "price": -1 },
    "fields": { "name": 1, "price": 1 }
  }
}
```

### Actions

| Action   | Description               |
|----------|---------------------------|
| `FIND`   | Query / select records    |
| `INSERT` | Create a new record       |
| `UPDATE` | Modify existing records   |
| `DELETE` | Remove records            |
| `COUNT`  | Count matching records    |

### Filter operators

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

## Project Layout

```
omniql/
├── pkg/
│   ├── core/               # Engine, Driver interface, Schema, OQL types, OmniJSON
│   ├── drivers/
│   │   ├── postgres/       # PostgreSQL driver (database/sql + lib/pq)
│   │   ├── mongo/          # MongoDB driver (mongo-driver)
│   │   └── sqlite/         # SQLite driver (go-sqlite3)
│   └── ffi/                # C-shared library (FFI for language bindings)
│       ├── ffi.go          # Exported C ABI (package main — required for c-shared)
│       ├── wrappers.go     # Go-typed shims used by the test suite
│       └── ffi_test.go     # FFI smoke tests
├── bindings/
│   ├── java/               # JNI wrapper — Maven Central
│   ├── python/             # CFFI extension — PyPI
│   ├── csharp/             # P/Invoke wrapper — NuGet
│   └── typescript/         # Node-API addon — npm
└── cmd/omniql/             # CLI entry-point
```

---

## Getting Started (Go)

```go
import (
    "context"
    "github.com/Uttam-Mahata/omniql/pkg/core"
    "github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite"
)

func main() {
    // 1. Create a driver.
    drv, _ := sqlite.New(":memory:")

    // 2. Create and configure the engine.
    engine := core.NewEngine()
    engine.RegisterDriver(drv)
    engine.Route("products", drv.Name())

    // 3. Execute an OQL query.
    result, _ := engine.Execute(context.Background(), core.OQLQuery{
        Target: "products",
        Action: core.ActionFind,
        Filter: core.Filter{
            "price": map[string]interface{}{"$lt": 500},
        },
        Options: core.QueryOptions{
            Limit: 10,
            Sort: map[string]int{"price": 1},
        },
    })

    // result.Data  — []map[string]interface{}
    // result.Meta  — driver name, total count, …
}
```

---

## Multi-Language Bindings

All bindings follow the same three-step pattern:

1. **Register a driver** — tells the engine how to connect to the database.
2. **Route a target** — maps a collection/table name to the registered driver.
3. **Execute queries** — run OQL queries through the engine.

### Python

Install via pip:

```bash
pip install omniql
```

Usage:

```python
import asyncio
from omniql import OmniEngine, Query

async def main():
    engine = OmniEngine()
    driver = await engine.register_sqlite_driver(":memory:")
    await engine.route("users", driver)

    result = await engine.execute(Query(
        target="users",
        action="FIND",
        filter={"status": "active"}
    ))
    print(result.data)

asyncio.run(main())
```

### Java

Add the following dependency to your `pom.xml`:

```xml
<dependency>
    <groupId>io.github.uttam-mahata</groupId>
    <artifactId>omniql</artifactId>
    <version>0.8.0</version>
</dependency>
```

Usage:

```java
import io.omniql.OmniEngine;
import java.util.Map;

OmniEngine engine = OmniEngine.create();
String driver = engine.registerSQLiteDriver(":memory:");
engine.route("users", driver);

var result = engine.table("users")
    .find(Map.of("status", "active"))
    .execute();
```

### C# / .NET

Install via NuGet:

```bash
dotnet add package OmniQL
```

Usage:

```csharp
using OmniQL;

using var engine = new OmniEngine();
string driver = engine.RegisterSQLiteDriver(":memory:");
engine.Route("users", driver);

var result = await engine.Table("users")
    .Find(new Dictionary<string, object> { { "status", "active" } })
    .ExecuteAsync();
```

### TypeScript / Node.js

Install via npm:

```bash
npm install omniql
```

Usage:

```typescript
import { OmniEngine } from 'omniql';

const engine = new OmniEngine();
const driver = engine.registerSQLiteDriver(':memory:');
engine.route('users', driver);

const result = await engine.execute({
  target: 'users',
  action: 'FIND',
  filter: { status: 'active' }
});

console.log(result.data);
engine.close();
```

---

## CLI

The `omniql` binary lets you run ad-hoc OQL queries from the command line.

```bash
# Build
go build -o omniql ./cmd/omniql

# Usage
omniql -driver <sqlite|postgres|mongo> -dsn <dsn> [-db <dbname>] -query '<OQL JSON>'
```

### Flags

| Flag      | Description                                                      |
|-----------|------------------------------------------------------------------|
| `-driver` | Database driver: `sqlite`, `postgres`, or `mongo`               |
| `-dsn`    | Connection string / file path / MongoDB URI                      |
| `-db`    | Database name (required for `mongo`)                             |
| `-query`  | JSON-encoded OQL query                                           |

### Examples

```bash
# SQLite (in-memory)
omniql -driver sqlite -dsn :memory: \
  -query '{"target":"users","action":"FIND","filter":{},"options":{"limit":5}}'

# PostgreSQL
omniql -driver postgres \
  -dsn "postgres://user:pass@localhost/mydb?sslmode=disable" \
  -query '{"target":"orders","action":"FIND","filter":{"status":"pending"}}'

# MongoDB
omniql -driver mongo -dsn "mongodb://localhost:27017" -db mydb \
  -query '{"target":"products","action":"COUNT","filter":{"price":{"$lt":100}}}'
```

---

## Running Tests

```bash
# Core engine tests
go test ./pkg/core/...

# SQLite driver integration tests (no external DB required)
go test ./pkg/drivers/sqlite/...

# PostgreSQL filter unit tests
go test ./pkg/drivers/postgres/...

# Mongo driver integration tests (using mtest)
go test ./pkg/drivers/mongo/...

# FFI smoke tests (no shared library build required)
go test ./pkg/ffi/...

# All tests
go test ./...
```

---

## Advanced Features (Roadmap)

### v0.8.0 (Released)
- **Mixed Projection** — Support mixed include/exclude projection (requires schema awareness).
- **Transactions** — `BeginTx` / `Commit` / `Rollback` as an optional `TransactionalDriver` interface.
- **Batch inserts** — insert multiple documents in a single round-trip.
- **`$not` operator** — support for logical negation.

### Future
- **Virtual DB / Federation** — join data from Postgres and MongoDB in a single query.
- **Unified Migration CLI** — `omniql-migrate up` applies schema changes across all databases simultaneously.
- **Real-time Sync (CDC)** — listen for changes in one database and automatically sync to another.
- **Additional drivers** — MySQL, Redis, Elasticsearch, InfluxDB, Neo4j, and more.

---

## License

Apache 2.0 — see [LICENSE](LICENSE).
