# OmniQL Language Bindings — v0.4.0

All language bindings now support the advanced OQL features introduced in v0.4.0:
1. **Logical Operators** (`$or`, `$and`)
2. **Query Sorting** (`sort` options)
3. **Field Projection** (`fields` options)
4. **Functional INSERT** (`INSERT` returns the created record/ID)

---

## Python

**Mechanism:** CFFI  
**Location:** `bindings/python/omniql.py`  
**Requires:** `cffi` (`pip install cffi`)

### Complex Query Example

```python
import asyncio
from omniql import OmniEngine, Query, QueryOptions

async def main():
    engine = OmniEngine()
    driver = await engine.register_sqlite_driver(":memory:")
    await engine.route("products", driver)

    # 1. Execute a complex FIND with logical operators, sort, and fields.
    result = await engine.execute(Query(
        target="products",
        action="FIND",
        filter={
            "$or": [
                {"category": "electronics"},
                {"price": {"$lt": 100}}
            ]
        },
        options=QueryOptions(
            limit=10,
            sort={"price": -1},
            fields={"name": 1, "price": 1}
        )
    ))

    # 2. INSERT now returns the ID.
    res = await engine.execute(Query(
        target="products",
        action="INSERT",
        document={"name": "New Phone", "price": 999, "category": "electronics"}
    ))
    print(f"Created record with ID: {res.data[0]['id']}")

asyncio.run(main())
```

---

## Java

**Mechanism:** JNI  
**Location:** `bindings/java/OmniEngine.java`  
**Requires:** `com.google.gson:gson` on the classpath

### Fluent API Example

```java
import io.omniql.OmniEngine;
import java.util.Map;
import java.util.List;

OmniEngine engine = OmniEngine.create();
String driver = engine.registerSQLiteDriver(":memory:");
engine.route("products", driver);

// Complex FIND with fluent builder
OmniEngine.OmniResult result = engine.table("products")
    .find(Map.of("$or", List.of(
        Map.of("category", "electronics"),
        Map.of("price", Map.of("$lt", 100))
    )))
    .sort(Map.of("price", -1))
    .fields(Map.of("name", 1, "price", 1))
    .limit(10)
    .execute();

// INSERT returning ID
OmniEngine.OmniResult ins = engine.table("products")
    .insert(Map.of("name", "Tablet", "price", 499))
    .execute();

System.out.println("Inserted ID: " + ins.data.get(0).get("id"));
```

---

## C# / .NET

**Mechanism:** P/Invoke  
**Location:** `bindings/csharp/OmniEngine.cs`  
**Requires:** .NET 6+

### Usage Example

```csharp
using OmniQL;

using var engine = new OmniEngine();
string driver = engine.RegisterSQLiteDriver(":memory:");
engine.Route("products", driver);

// Complex Query
var result = await engine
    .Table("products")
    .Find(new Dictionary<string, object> {
        { "$or", new List<object> {
            new Dictionary<string, object> { { "category", "electronics" } },
            new Dictionary<string, object> { { "price", new Dictionary<string, object> { { "$lt", 100 } } } }
        }}
    })
    .Sort(new Dictionary<string, int> { { "price", -1 } })
    .Fields(new Dictionary<string, object> { { "name", 1 }, { "price", 1 } })
    .Limit(10)
    .ExecuteAsync();

// Insert returning ID
var ins = await engine.Table("products")
    .Insert(new Dictionary<string, object> { { "name", "Laptop" }, { "price", 1200 } })
    .ExecuteAsync();

Console.WriteLine($"Inserted: {ins.Data[0]["id"]}");
```

---

## TypeScript / Node.js

**Mechanism:** ffi-napi (N-API)  
**Location:** `bindings/typescript/omniql.ts`

### Usage Example

```typescript
import { OmniEngine } from './omniql';

const engine = new OmniEngine();
const driver = engine.registerSQLiteDriver(':memory:');
engine.route('products', driver);

// Complex Query
const result = await engine.execute({
  target: 'products',
  action: 'FIND',
  filter: {
    $or: [
      { category: 'electronics' },
      { price: { $lt: 100 } }
    ]
  },
  options: {
    limit: 10,
    sort: { price: -1 },
    fields: { name: 1, price: 1 }
  }
});

// Insert returning ID
const ins = await engine.execute({
  target: 'products',
  action: 'INSERT',
  document: { name: 'Monitor', price: 300 }
});

console.log('New ID:', ins.data[0].id);
```
