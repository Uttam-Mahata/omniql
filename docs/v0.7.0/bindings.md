# OmniQL Language Bindings — v0.7.0

v0.7.0 adds **terminal convenience methods** to every language binding and
documents the new SQL Server and Redis driver registration calls.

---

## Python

**Mechanism:** CFFI
**Location:** `bindings/python/omniql/__init__.py`
**Requires:** `pip install cffi`

### Terminal convenience methods (new in v0.7.0)

All six methods are available in both async and sync forms.

```python
import asyncio
from omniql import OmniEngine

async def main():
    engine = OmniEngine()
    driver = await engine.register_sqlite_driver(":memory:")
    await engine.route("products", driver)

    # find_many — returns list of dicts
    docs = await engine.find_many("products", {"category": "electronics"})
    docs = await engine.find_many("products", {"category": "electronics"},
                                  options={"limit": 10, "sort": {"price": -1}})

    # find_first — returns single dict or None
    item = await engine.find_first("products", {"name": "Laptop"})

    # count — returns int
    n = await engine.count("products")
    n = await engine.count("products", {"category": "electronics"})

    # insert_one — returns inserted doc dict
    row = await engine.insert_one("products", {"name": "Tablet", "price": 299.99})

    # update_many — returns OmniResult
    result = await engine.update_many(
        "products", {"name": "Tablet"}, {"price": 279.99}
    )

    # delete_many — returns OmniResult
    result = await engine.delete_many("products", {"name": "Tablet"})

asyncio.run(main())
```

### Synchronous form

Every async method has a `_sync` counterpart:

```python
docs   = engine.find_many_sync("products", {"category": "electronics"})
first  = engine.find_first_sync("products", {"name": "Laptop"})
n      = engine.count_sync("products")
row    = engine.insert_one_sync("products", {"name": "Monitor", "price": 429.99})
engine.update_many_sync("products", {"name": "Monitor"}, {"price": 399.99})
engine.delete_many_sync("products", {"name": "Monitor"})
```

### SQL Server driver

```python
driver = await engine.register_sqlserver_driver(
    "sqlserver://sa:Pass@localhost:1433?database=shop"
)
await engine.route("orders", driver)
```

### Redis driver

```python
driver = await engine.register_redis_driver("redis://localhost:6379/0")
await engine.route("sessions", driver)

# Insert with TTL (_ttl field in seconds)
await engine.insert_one("sessions", {"id": "tok:abc", "user_id": 7, "_ttl": 3600})
```

---

## Java

**Mechanism:** JNI
**Location:** `bindings/java/src/main/java/io/omniql/OmniEngine.java`
**Requires:** `com.google.gson:gson`

### Terminal convenience methods (new in v0.7.0)

```java
import io.omniql.OmniEngine;
import java.util.List;
import java.util.Map;

OmniEngine engine = OmniEngine.create();
String driver = engine.registerSQLiteDriver(":memory:");
engine.route("products", driver);

// findMany — returns List<Map<String, Object>>
List<Map<String, Object>> docs =
    engine.findMany("products", Map.of("category", "electronics"));

// findFirst — returns Map<String, Object> or null
Map<String, Object> item = engine.findFirst("products", Map.of("name", "Laptop"));

// count — returns long
long n = engine.count("products", null);

// insertOne — returns OmniResult
OmniEngine.OmniResult r = engine.insertOne(
    "products", Map.of("name", "Tablet", "price", 299.99)
);

// updateMany / deleteMany — return OmniResult
engine.updateMany("products", Map.of("name", "Tablet"), Map.of("price", 279.99));
engine.deleteMany("products", Map.of("name", "Tablet"));
```

### SQL Server driver

```java
String sqlsrvDriver = engine.registerSQLServerDriver(
    "sqlserver://sa:Pass@localhost:1433?database=shop"
);
engine.route("orders", sqlsrvDriver);

OmniEngine.OmniResult result = engine.table("orders")
    .find(Map.of("status", "pending"))
    .sort(Map.of("total", -1))
    .limit(20)
    .execute();
```

### Redis driver

```java
String redisDriver = engine.registerRedisDriver("redis://localhost:6379/0");
engine.route("sessions", redisDriver);

// Insert with TTL
engine.insertOne("sessions",
    Map.of("id", "tok:abc", "user_id", 7, "_ttl", 3600));
```

---

## C# / .NET

**Mechanism:** P/Invoke
**Location:** `bindings/csharp/OmniQL/OmniEngine.cs`
**Requires:** .NET 6+

### Terminal convenience methods (new in v0.7.0)

```csharp
using OmniQL;
using System.Collections.Generic;

using var engine = new OmniEngine();
string driver = engine.RegisterSQLiteDriver(":memory:");
engine.Route("products", driver);

// FindManyAsync — returns List<Dictionary<string, JsonElement>>
var docs = await engine.FindManyAsync("products",
    new Dictionary<string, object> { ["category"] = "electronics" });

// FindFirstAsync — returns Dictionary<string, JsonElement>? (null if not found)
var item = await engine.FindFirstAsync("products",
    new Dictionary<string, object> { ["name"] = "Laptop" });

// CountAsync — returns long
long n = await engine.CountAsync("products");

// InsertOneAsync — returns OmniResult
var r = await engine.InsertOneAsync("products",
    new Dictionary<string, object> { ["name"] = "Tablet", ["price"] = 299.99 });

// UpdateManyAsync / DeleteManyAsync — return OmniResult
await engine.UpdateManyAsync("products",
    new Dictionary<string, object> { ["name"] = "Tablet" },
    new Dictionary<string, object> { ["price"] = 279.99 });

await engine.DeleteManyAsync("products",
    new Dictionary<string, object> { ["name"] = "Tablet" });
```

### SQL Server driver

```csharp
string sqlsrvDriver = engine.RegisterSQLServerDriver(
    "sqlserver://sa:Pass@localhost:1433?database=shop"
);
engine.Route("orders", sqlsrvDriver);

var result = await engine
    .Table("orders")
    .Find(new Dictionary<string, object> { { "status", "pending" } })
    .Sort(new Dictionary<string, int> { { "total", -1 } })
    .Limit(20)
    .ExecuteAsync();
```

### Redis driver

```csharp
string redisDriver = engine.RegisterRedisDriver("redis://localhost:6379/0");
engine.Route("sessions", redisDriver);

// Insert with TTL
await engine.InsertOneAsync("sessions", new Dictionary<string, object>
{
    ["id"]      = "tok:abc",
    ["user_id"] = 7,
    ["_ttl"]    = 3600,
});
```

---

## TypeScript / Node.js

**Mechanism:** Native C++ NAPI bridge (`omniql_bridge.cc`)
**Location:** `bindings/typescript/omniql.ts`

> The bridge uses `signal(SIGURG, SIG_IGN)` before `dlopen` to suppress
> the Go preemptive scheduler signal that conflicts with libuv.

### Terminal convenience methods (new in v0.7.0)

```typescript
import { OmniEngine } from './omniql';

const engine = new OmniEngine();
const driver = engine.registerSQLiteDriver(':memory:');
engine.route('products', driver);

// findMany — returns T[]
const docs = await engine.findMany('products', { category: 'electronics' });
const paged = await engine.findMany('products',
  { category: 'electronics' },
  { limit: 10, sort: { price: -1 } }
);

// findFirst — returns T | undefined
const item = await engine.findFirst('products', { name: 'Laptop' });

// count — returns number
const n = await engine.count('products');
const n2 = await engine.count('products', { category: 'electronics' });

// insertOne — returns OmniResult<T>
const r = await engine.insertOne('products', { name: 'Tablet', price: 299.99 });

// updateMany / deleteMany — return OmniResult
await engine.updateMany('products', { name: 'Tablet' }, { price: 279.99 });
await engine.deleteMany('products', { name: 'Tablet' });

engine.close();
```

### SQL Server driver

```typescript
const sqlsrvDriver = engine.registerSQLServerDriver(
  'sqlserver://sa:Pass@localhost:1433?database=shop'
);
engine.route('orders', sqlsrvDriver);

const orders = await engine.findMany('orders', { status: 'pending' }, {
  limit: 20,
  sort: { total: -1 },
});
```

### Redis driver

```typescript
const redisDriver = engine.registerRedisDriver('redis://localhost:6379/0');
engine.route('sessions', redisDriver);

// Insert with TTL (_ttl in seconds)
await engine.insertOne('sessions', { id: 'tok:abc', user_id: 7, _ttl: 3600 });

// Lookup by id
const session = await engine.findFirst('sessions', { id: 'tok:abc' });
```

---

## Redis Key Semantics

The Redis driver uses a `<target>:<id>` key pattern. The `id` or `_id`
field in the document determines the key suffix:

| Operation | Key used |
|-----------|----------|
| `INSERT { id: "abc", … }` | `sessions:abc` |
| `FIND { id: "abc" }` | `GET sessions:abc` |
| `FIND {}` | `SCAN sessions:*` |
| `DELETE { id: "abc" }` | `DEL sessions:abc` |
| `UPDATE { id: "abc" }` | `GET sessions:abc` → merge → `SET sessions:abc` |

The optional `_ttl` field (integer seconds) triggers a Redis `EXPIRE` after
each `SET`. The `_ttl` field is stored in the JSON value and is also honoured
on `UPDATE`.

---

## Complete BATCH_INSERT example (raw OQL)

For all bindings, the underlying OQL wire format for `BATCH_INSERT` is:

```json
{
  "target":    "products",
  "action":    "BATCH_INSERT",
  "documents": [
    { "name": "Laptop",   "price": 999.99, "category": "electronics" },
    { "name": "Keyboard", "price":  79.99, "category": "electronics" }
  ]
}
```

The `batchInsert` convenience method (introduced in v0.6.0) builds this automatically.
