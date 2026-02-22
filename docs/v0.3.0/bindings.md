# OmniQL Language Bindings — v0.3.0

All four language bindings follow the same three-step pattern:

1. **Register a driver** — tells the engine how to connect to the database.
2. **Route a target** — maps a collection/table name to the registered driver.
3. **Execute queries** — run OQL queries through the engine.

Build the native library before using any binding:

```bash
go build -buildmode=c-shared -o libomniql.so ./pkg/ffi
```

---

## Python

**Mechanism:** CFFI  
**Location:** `bindings/python/omniql.py`  
**Requires:** `cffi` (`pip install cffi`)

### Setup

Place `libomniql.so` (or `omniql.dll` on Windows) alongside `omniql.py`,
or on a path searched by `cffi.FFI.dlopen`.

### Async API (recommended)

```python
import asyncio
from omniql import OmniEngine, Query

async def main():
    engine = OmniEngine()

    # 1. Register a driver — returns the driver name string.
    driver = await engine.register_sqlite_driver(":memory:")
    # driver == "sqlite"

    # For Postgres:
    # driver = await engine.register_postgres_driver(
    #     "postgres://user:pass@localhost/mydb?sslmode=disable")

    # For MongoDB:
    # driver = await engine.register_mongo_driver(
    #     "mongodb://localhost:27017", "mydb")

    # 2. Route a target to the driver.
    await engine.route("users", driver)

    # 3. Execute queries.
    result = await engine.execute(Query(
        target="users",
        action="FIND",
        filter={"status": "active"},
        options={"limit": 10},
    ))

    print(result.data)   # list of dicts
    print(result.meta)   # OmniMeta(total=…, driver="sqlite", …)

asyncio.run(main())
```

### Synchronous API

```python
from omniql import OmniEngine, Query

engine = OmniEngine()
driver = engine.register_sqlite_driver_sync(":memory:")
engine.route_sync("products", driver)

result = engine.execute_sync(Query(
    target="products",
    action="COUNT",
    filter={"price": {"$lt": 100}},
))
print(result.data)
```

### API Reference

| Method | Description |
|--------|-------------|
| `register_sqlite_driver_sync(dsn)` / `await register_sqlite_driver(dsn)` | Register SQLite; returns `"sqlite"` |
| `register_postgres_driver_sync(conn_str)` / `await register_postgres_driver(conn_str)` | Register Postgres; returns `"postgres"` |
| `register_mongo_driver_sync(uri, db_name)` / `await register_mongo_driver(uri, db_name)` | Register MongoDB; returns `"mongo"` |
| `route_sync(target, driver_name)` / `await route(target, driver_name)` | Bind target to driver |
| `execute_sync(query)` / `await execute(query)` | Execute OQL query |
| `register_schema_sync(schema)` / `await register_schema(schema)` | Register collection schema |

---

## Java

**Mechanism:** JNI  
**Location:** `bindings/java/OmniEngine.java`  
**Requires:** `com.google.gson:gson` on the classpath

### Setup

1. Add `libomniql.so` (or `omniql.dll`) to `java.library.path`:
   ```bash
   java -Djava.library.path=/path/to/lib -jar myapp.jar
   ```
2. Add `gson` to your build (Maven / Gradle).

### Usage

```java
import io.omniql.OmniEngine;
import java.util.Map;

OmniEngine engine = OmniEngine.create();

// 1. Register a driver.
String driver = engine.registerSQLiteDriver(":memory:");
// driver == "sqlite"

// For Postgres:
// String driver = engine.registerPostgresDriver(
//     "host=localhost user=pg password=pg dbname=mydb sslmode=disable");

// For MongoDB:
// String driver = engine.registerMongoDriver(
//     "mongodb://localhost:27017", "mydb");

// 2. Route a target.
engine.route("users", driver);

// 3. Execute queries using the fluent builder.
OmniEngine.OmniResult result = engine.table("users")
    .find(Map.of("status", "active"))
    .limit(10)
    .execute();

result.data.forEach(row -> System.out.println(row.get("name")));

// Insert
engine.table("users")
    .insert(Map.of("name", "Alice", "status", "active"))
    .execute();

// Count
engine.table("users")
    .count(Map.of("status", "active"))
    .execute();

engine.close();
```

### API Reference

| Method | Description |
|--------|-------------|
| `OmniEngine.create()` | Create a new engine |
| `registerSQLiteDriver(dsn)` | Register SQLite; returns `"sqlite"` |
| `registerPostgresDriver(connStr)` | Register Postgres; returns `"postgres"` |
| `registerMongoDriver(uri, dbName)` | Register MongoDB; returns `"mongo"` |
| `route(target, driverName)` | Bind target to driver |
| `registerSchema(schema)` | Register collection schema |
| `execute(query)` | Execute raw `OQLQuery` |
| `table(target)` | Start a fluent `QueryBuilder` |
| `close()` | Release native engine handle (implements `AutoCloseable`) |

#### `QueryBuilder` methods

| Method | Description |
|--------|-------------|
| `.find(filter)` | Set action to FIND |
| `.insert(document)` | Set action to INSERT |
| `.update(document)` | Set action to UPDATE |
| `.delete(filter)` | Set action to DELETE |
| `.count(filter)` | Set action to COUNT |
| `.limit(n)` | Set result limit |
| `.skip(n)` | Set result offset |
| `.execute()` | Run the query and return `OmniResult` |

---

## C# / .NET

**Mechanism:** P/Invoke  
**Location:** `bindings/csharp/OmniEngine.cs`  
**Requires:** .NET 6+ (uses `System.Text.Json`)

### Setup

Copy `omniql.dll` (Windows) or `libomniql.so` (Linux) to the application
output directory, or set `LD_LIBRARY_PATH` / `PATH` appropriately.

### Usage

```csharp
using OmniQL;

using var engine = new OmniEngine();

// 1. Register a driver.
string driver = engine.RegisterSQLiteDriver(":memory:");
// driver == "sqlite"

// For Postgres:
// string driver = engine.RegisterPostgresDriver(
//     "host=localhost user=pg password=pg dbname=mydb sslmode=disable");

// For MongoDB:
// string driver = engine.RegisterMongoDriver(
//     "mongodb://localhost:27017", "mydb");

// 2. Route a target.
engine.Route("users", driver);

// 3. Execute queries using the fluent builder.
var result = await engine
    .Table("users")
    .Find(new Dictionary<string, object> { { "status", "active" } })
    .Limit(10)
    .ExecuteAsync();

foreach (var row in result.Data)
    Console.WriteLine(row["name"]);

// Insert
await engine.Table("users")
    .Insert(new Dictionary<string, object> { { "name", "Alice" }, { "status", "active" } })
    .ExecuteAsync();

// Count
await engine.Table("users")
    .Count(new Dictionary<string, object> { { "status", "active" } })
    .ExecuteAsync();
```

### API Reference

| Method | Description |
|--------|-------------|
| `new OmniEngine()` | Create a new engine (implements `IDisposable`) |
| `RegisterSQLiteDriver(dsn)` | Register SQLite; returns `"sqlite"` |
| `RegisterPostgresDriver(connStr)` | Register Postgres; returns `"postgres"` |
| `RegisterMongoDriver(uri, dbName)` | Register MongoDB; returns `"mongo"` |
| `Route(target, driverName)` | Bind target to driver |
| `RegisterSchema(schema)` | Register collection schema |
| `ExecuteAsync(query)` | Execute raw `OQLQuery` asynchronously |
| `Table(target)` | Start a fluent `QueryBuilder` |
| `Dispose()` | Release native engine handle |

#### `QueryBuilder` methods

| Method | Description |
|--------|-------------|
| `.Find(filter)` | Set action to FIND |
| `.Insert(document)` | Set action to INSERT |
| `.Update(document)` | Set action to UPDATE |
| `.Delete(filter)` | Set action to DELETE |
| `.Count(filter?)` | Set action to COUNT |
| `.Limit(n)` | Set result limit |
| `.Skip(n)` | Set result offset |
| `.ExecuteAsync()` | Run the query and return `Task<OmniResult>` |

> **v0.2.0 note:** The C# binding silently dropped all queries in v0.2.0 because
> `OQLQuery` serialized with PascalCase JSON keys but the Go engine expected
> camelCase. This is fixed in v0.3.0 via `[JsonPropertyName]` attributes.

---

## TypeScript / Node.js

**Mechanism:** ffi-napi (N-API)  
**Location:** `bindings/typescript/omniql.ts`  
**Requires:** `ffi-napi`, `ref-napi` (`npm install ffi-napi ref-napi`)

### Setup

Place `libomniql.so` in the same directory as the compiled JS or on
`LD_LIBRARY_PATH`.

### Usage

```typescript
import { OmniEngine } from './omniql';

const engine = new OmniEngine();

// 1. Register a driver.
const driver = engine.registerSQLiteDriver(':memory:');
// driver == "sqlite"

// For Postgres:
// const driver = engine.registerPostgresDriver(
//     'host=localhost user=pg password=pg dbname=mydb sslmode=disable');

// For MongoDB:
// const driver = engine.registerMongoDriver(
//     'mongodb://localhost:27017', 'mydb');

// 2. Route a target.
engine.route('users', driver);

// 3. Execute queries.
const result = await engine.execute({
  target: 'users',
  action: 'FIND',
  filter: { status: 'active' },
  options: { limit: 10 },
});

console.log(result.data);   // Array of record objects
console.log(result.meta);   // { total, returned, driver, target }

engine.close();
```

### API Reference

| Method | Description |
|--------|-------------|
| `new OmniEngine()` | Create a new engine |
| `registerSQLiteDriver(dsn)` | Register SQLite; returns `"sqlite"` |
| `registerPostgresDriver(connStr)` | Register Postgres; returns `"postgres"` |
| `registerMongoDriver(uri, dbName)` | Register MongoDB; returns `"mongo"` |
| `route(target, driverName)` | Bind target to driver |
| `registerSchema(schema)` | Register collection schema |
| `execute<T>(query)` | Execute query; returns `Promise<OmniResult<T>>` |
| `close()` | Release native engine handle |

### TypeScript Types

```typescript
type Action = 'FIND' | 'INSERT' | 'UPDATE' | 'DELETE' | 'COUNT';

interface Query {
  target:    string;
  action?:   Action;           // default: 'FIND'
  filter?:   Record<string, unknown>;
  document?: Record<string, unknown>;
  options?:  { limit?: number; skip?: number };
}

interface OmniResult<T = Record<string, unknown>> {
  data:   T[];
  meta:   { total: number; returned: number; driver: string; target: string };
  error?: { code: string; message: string };
}
```
