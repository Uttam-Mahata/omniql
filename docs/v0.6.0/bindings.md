# OmniQL Language Bindings — v0.6.0

v0.6.0 adds a `batchInsert` convenience method to every language binding and
documents MySQL driver registration.

---

## Python

**Mechanism:** CFFI
**Location:** `bindings/python/omniql/__init__.py`
**Requires:** `pip install cffi`

### New: `batch_insert`

```python
import asyncio
from omniql import OmniEngine

async def main():
    engine = OmniEngine()
    driver = await engine.register_sqlite_driver(":memory:")
    await engine.route("products", driver)

    # batchInsert convenience method (new in v0.6.0)
    result = await engine.batch_insert("products", [
        {"name": "Laptop",  "price": 999.99, "category": "electronics"},
        {"name": "Keyboard","price":  79.99, "category": "electronics"},
        {"name": "Mouse",   "price":  29.99, "category": "electronics"},
    ])
    print(f"Inserted {result.meta.returned} rows")

asyncio.run(main())
```

### Synchronous form

```python
result = engine.batch_insert_sync("products", docs)
```

### MySQL driver

```python
driver = await engine.register_mysql_driver(
    "root:pass@tcp(127.0.0.1:3306)/shop?parseTime=true"
)
await engine.route("orders", driver)
```

---

## Java

**Mechanism:** JNI
**Location:** `bindings/java/src/main/java/io/omniql/OmniEngine.java`
**Requires:** `com.google.gson:gson`

### New: `batchInsert`

```java
import io.omniql.OmniEngine;
import java.util.List;
import java.util.Map;

OmniEngine engine = OmniEngine.create();
String driver = engine.registerSQLiteDriver(":memory:");
engine.route("products", driver);

// Via engine directly
OmniEngine.OmniResult result = engine.batchInsert("products", List.of(
    Map.of("name", "Laptop",   "price", 999.99),
    Map.of("name", "Keyboard", "price",  79.99),
    Map.of("name", "Mouse",    "price",  29.99)
));

// Via fluent QueryBuilder
OmniEngine.OmniResult r2 = engine.table("products")
    .batchInsert(List.of(
        Map.of("name", "Monitor", "price", 329.99)
    ));
```

### MySQL driver

```java
String mysqlDriver = engine.registerMySQLDriver(
    "root:pass@tcp(127.0.0.1:3306)/shop?parseTime=true"
);
engine.route("orders", mysqlDriver);

OmniEngine.OmniResult result = engine.table("orders")
    .find(Map.of("status", "pending"))
    .sort(Map.of("total", -1))
    .limit(20)
    .execute();
```

---

## C# / .NET

**Mechanism:** P/Invoke
**Location:** `bindings/csharp/OmniQL/OmniEngine.cs`
**Requires:** .NET 6+

### New: `BatchInsertAsync`

```csharp
using OmniQL;

using var engine = new OmniEngine();
string driver = engine.RegisterSQLiteDriver(":memory:");
engine.Route("products", driver);

// Via engine directly
var docs = new List<Dictionary<string, object>>
{
    new() { ["name"] = "Laptop",   ["price"] = 999.99 },
    new() { ["name"] = "Keyboard", ["price"] =  79.99 },
    new() { ["name"] = "Mouse",    ["price"] =  29.99 },
};

var result = await engine.BatchInsertAsync("products", docs);
Console.WriteLine($"Inserted {result.Meta.Returned} rows");

// Via fluent builder
var r2 = await engine.Table("products").BatchInsertAsync(docs);
```

### MySQL driver

```csharp
string mysqlDriver = engine.RegisterMySQLDriver(
    "root:pass@tcp(127.0.0.1:3306)/shop?parseTime=true"
);
engine.Route("orders", mysqlDriver);

var result = await engine
    .Table("orders")
    .Find(new Dictionary<string, object> { { "status", "pending" } })
    .Sort(new Dictionary<string, int> { { "total", -1 } })
    .Limit(20)
    .ExecuteAsync();
```

---

## TypeScript / Node.js

**Mechanism:** Native C++ NAPI bridge (`omniql_bridge.cc`)
**Location:** `bindings/typescript/omniql.ts`

> The bridge uses `signal(SIGURG, SIG_IGN)` before `dlopen` to suppress
> the Go preemptive scheduler signal that conflicts with libuv.

### New: `batchInsert`

```typescript
import { OmniEngine } from './omniql';

const engine = new OmniEngine();
const driver = engine.registerSQLiteDriver(':memory:');
engine.route('products', driver);

// batchInsert convenience method (new in v0.6.0)
const result = await engine.batchInsert('products', [
  { name: 'Laptop',   price: 999.99, category: 'electronics' },
  { name: 'Keyboard', price:  79.99, category: 'electronics' },
  { name: 'Mouse',    price:  29.99, category: 'electronics' },
]);
console.log(`Inserted ${result.meta.returned} rows`);
```

### MySQL driver

```typescript
const mysqlDriver = engine.registerMySQLDriver(
  'root:pass@tcp(127.0.0.1:3306)/shop?parseTime=true'
);
engine.route('orders', mysqlDriver);

const result = await engine.execute({
  target: 'orders',
  action: 'FIND',
  filter: { status: 'pending' },
  options: { sort: { total: -1 }, limit: 20 },
});
```

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

The `batchInsert` convenience methods shown above build this automatically.
