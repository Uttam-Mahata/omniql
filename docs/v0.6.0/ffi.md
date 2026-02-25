# OmniQL FFI / C API Reference — v0.6.0

The FFI layer (`pkg/ffi`) exports OmniQL as a C shared library so that any
language with a C FFI mechanism can use it.

---

## Building the Shared Library

> `pkg/ffi` uses `package main` — required by Go's `-buildmode=c-shared` flag.

```bash
# Linux / macOS
go build -buildmode=c-shared -o libomniql.so ./pkg/ffi

# Windows
go build -buildmode=c-shared -o omniql.dll ./pkg/ffi
```

This produces `libomniql.so` (or `.dll`) and a `libomniql.h` header file.

---

## Exported Functions

### Engine lifecycle

#### `OmniQL_NewEngine() → int`
Creates a new Core Engine and returns an opaque integer handle.
Pass this handle to every subsequent API call.

```c
int handle = OmniQL_NewEngine();
```

#### `OmniQL_FreeEngine(int handle)`
Releases the engine identified by `handle` and removes it from the registry.

```c
OmniQL_FreeEngine(handle);
```

---

### Driver registration

All register functions return a JSON string on both success and failure.
**The caller must free the returned pointer with `OmniQL_Free`.**

#### `OmniQL_RegisterSQLiteDriver(int handle, const char* dsn) → char*`
Opens a SQLite connection at the given DSN (file path or `":memory:"`) and
registers the driver with the engine.

Returns `{"driver":"sqlite"}` on success.

```c
char* r = OmniQL_RegisterSQLiteDriver(handle, ":memory:");
OmniQL_Free(r);
```

#### `OmniQL_RegisterPostgresDriver(int handle, const char* connStr) → char*`
Opens a PostgreSQL connection using the given connection string.

Returns `{"driver":"postgres"}` on success.

```c
char* r = OmniQL_RegisterPostgresDriver(handle,
    "postgres://user:pass@localhost/mydb?sslmode=disable");
OmniQL_Free(r);
```

#### `OmniQL_RegisterMongoDriver(int handle, const char* uri, const char* dbName) → char*`
Connects to MongoDB using the given URI and selects the specified database.

Returns `{"driver":"mongo"}` on success.

```c
char* r = OmniQL_RegisterMongoDriver(handle,
    "mongodb://localhost:27017", "mydb");
OmniQL_Free(r);
```

#### `OmniQL_RegisterMySQLDriver(int handle, const char* dsn) → char*` *(new in v0.6.0)*
Opens a MySQL connection using the given DSN and registers the driver.

DSN format: `"user:password@tcp(host:port)/dbname?parseTime=true"`

Returns `{"driver":"mysql"}` on success.

```c
char* r = OmniQL_RegisterMySQLDriver(handle,
    "root:pass@tcp(127.0.0.1:3306)/mydb?parseTime=true");
OmniQL_Free(r);
```

---

### Routing

#### `OmniQL_Route(int handle, const char* target, const char* driverName) → char*`
Binds a collection/table target name to a registered driver name.
Returns `""` on success or a JSON error on failure.

```c
char* r = OmniQL_Route(handle, "orders", "mysql");
OmniQL_Free(r);
```

If no explicit route exists for a target, the engine falls back to the first
registered driver automatically.

---

### Query execution

#### `OmniQL_Execute(int handle, const char* queryJSON) → char*`
Executes a JSON-encoded OQL query and returns a JSON-encoded OmniJSON response.
**The caller must free the returned pointer with `OmniQL_Free`.**

```c
const char* query =
    "{\"target\":\"orders\","
    "\"action\":\"FIND\","
    "\"filter\":{\"status\":\"pending\"},"
    "\"options\":{\"limit\":10,\"sort\":{\"total\":-1}}}";

char* response = OmniQL_Execute(handle, query);
// parse response JSON …
OmniQL_Free(response);
```

**BATCH_INSERT request:**

```json
{
  "target":    "log_entries",
  "action":    "BATCH_INSERT",
  "documents": [
    { "level": "info",  "msg": "started" },
    { "level": "error", "msg": "crashed" }
  ]
}
```

**Response format:**

```json
{
  "data": [ { "id": 1, "status": "pending", "total": 249.99 } ],
  "meta": { "total": 1, "returned": 1, "driver": "mysql", "target": "orders" },
  "error": null
}
```

**Error response:**

```json
{
  "data":  [],
  "meta":  { "total": 0, "returned": 0, "driver": "", "target": "" },
  "error": { "code": "EXECUTION_ERROR", "message": "mysql: table 'orders' doesn't exist" }
}
```

---

### Schema registration

#### `OmniQL_RegisterSchema(int handle, const char* schemaJSON) → char*`
Registers a collection schema for optional validation.
Returns `""` on success. **Free the returned pointer with `OmniQL_Free`.**

```c
const char* schema =
    "{\"name\":\"orders\","
    "\"fields\":{"
    "\"id\":{\"type\":\"int\",\"required\":true},"
    "\"status\":{\"type\":\"string\",\"required\":true},"
    "\"total\":{\"type\":\"float\",\"required\":false}}}";

char* r = OmniQL_RegisterSchema(handle, schema);
OmniQL_Free(r);
```

---

### Memory management

#### `OmniQL_Free(char* ptr)`
Frees a C string previously returned by `OmniQL_Execute`,
`OmniQL_RegisterSchema`, `OmniQL_Route`, or any `OmniQL_Register*Driver`
function. **Never call `free()` directly on these pointers.**

---

## Multi-engine usage

Multiple engines can coexist within the same process, each with its own
drivers and route table.

```c
int engine1 = OmniQL_NewEngine();
int engine2 = OmniQL_NewEngine();

OmniQL_RegisterPostgresDriver(engine1, "postgres://...");
OmniQL_RegisterMySQLDriver(engine2, "root:pass@tcp(localhost:3306)/db");

// engine1 and engine2 are completely independent.

OmniQL_FreeEngine(engine1);
OmniQL_FreeEngine(engine2);
```

---

## Error codes

| Code | Cause |
|------|-------|
| `INVALID_HANDLE` | The provided handle does not map to a registered engine |
| `PARSE_ERROR` | The JSON input could not be parsed |
| `SCHEMA_ERROR` | Query failed strict schema validation |
| `DRIVER_ERROR` | No driver registered / routed for the target |
| `DRIVER_INIT_ERROR` | Driver constructor failed (bad DSN, connection refused, etc.) |
| `EXECUTION_ERROR` | The driver returned an error during execution |
| `MARSHAL_ERROR` | Result could not be serialised to JSON (should never occur) |

---

## All Exported Symbols (v0.6.0)

| Symbol | Since |
|--------|-------|
| `OmniQL_NewEngine` | v0.2.0 |
| `OmniQL_FreeEngine` | v0.2.0 |
| `OmniQL_Execute` | v0.2.0 |
| `OmniQL_Free` | v0.2.0 |
| `OmniQL_RegisterSchema` | v0.2.0 |
| `OmniQL_Route` | v0.3.0 |
| `OmniQL_RegisterSQLiteDriver` | v0.3.0 |
| `OmniQL_RegisterPostgresDriver` | v0.3.0 |
| `OmniQL_RegisterMongoDriver` | v0.3.0 |
| `OmniQL_RegisterMySQLDriver` | **v0.6.0** |
