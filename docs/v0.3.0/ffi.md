# OmniQL FFI / C API Reference — v0.3.0

The FFI layer (`pkg/ffi`) exports OmniQL as a C shared library so that any
language with a C FFI mechanism can use it.

---

## Building the Shared Library

> `pkg/ffi` uses `package main` — this is required by Go's `-buildmode=c-shared`
> flag and does not affect any other package in the module.

```bash
# Linux / macOS
go build -buildmode=c-shared -o libomniql.so ./pkg/ffi

# Windows
go build -buildmode=c-shared -o omniql.dll ./pkg/ffi
```

This produces both a `.so` / `.dll` file and a `.h` header file.
The `.h` file is auto-generated; commit it alongside the binary if you
vendor the library.

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
Call this when the engine is no longer needed.

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

Returns `{"driver":"sqlite"}` on success, or a JSON error object on failure.

```c
char* result = OmniQL_RegisterSQLiteDriver(handle, ":memory:");
// use result ...
OmniQL_Free(result);
```

#### `OmniQL_RegisterPostgresDriver(int handle, const char* connStr) → char*`
Opens a PostgreSQL connection using the given connection string and registers
the driver.

Connection string formats accepted:
- DSN: `"host=localhost user=pg password=pg dbname=mydb sslmode=disable"`
- URL: `"postgres://user:pass@localhost/mydb?sslmode=disable"`

Returns `{"driver":"postgres"}` on success.

```c
char* result = OmniQL_RegisterPostgresDriver(handle,
    "postgres://user:pass@localhost/mydb?sslmode=disable");
OmniQL_Free(result);
```

#### `OmniQL_RegisterMongoDriver(int handle, const char* uri, const char* dbName) → char*`
Connects to MongoDB using the given URI and selects the specified database.

Returns `{"driver":"mongo"}` on success.

```c
char* result = OmniQL_RegisterMongoDriver(handle,
    "mongodb://localhost:27017", "mydb");
OmniQL_Free(result);
```

---

### Routing

#### `OmniQL_Route(int handle, const char* target, const char* driverName) → char*`
Binds a collection/table target name to a registered driver name.
Must be called after registering a driver. Returns `""` (empty string) on
success or a JSON error on failure.

```c
char* result = OmniQL_Route(handle, "users", "sqlite");
OmniQL_Free(result);
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
    "{\"target\":\"users\",\"action\":\"FIND\",\"filter\":{},\"options\":{\"limit\":10}}";

char* response = OmniQL_Execute(handle, query);
// parse response JSON ...
OmniQL_Free(response);
```

**Request format:**

```json
{
  "target":   "collection_or_table_name",
  "action":   "FIND",
  "filter":   { "field": { "$operator": value } },
  "document": { "field": value },
  "options":  { "limit": 10, "skip": 0 }
}
```

**Response format:**

```json
{
  "data": [ { "id": 1, "name": "Alice" } ],
  "meta": {
    "total":    1,
    "returned": 1,
    "driver":   "sqlite",
    "target":   "users"
  },
  "error": null
}
```

**Error response format:**

```json
{
  "data":  [],
  "meta":  { "total": 0, "returned": 0, "driver": "", "target": "" },
  "error": { "code": "EXECUTION_ERROR", "message": "no such table: users" }
}
```

---

### Schema registration

#### `OmniQL_RegisterSchema(int handle, const char* schemaJSON) → char*`
Registers a collection schema with the engine for optional validation.
Returns `""` on success or a JSON error on failure.
**The caller must free the returned pointer with `OmniQL_Free`.**

```c
const char* schema =
    "{\"name\":\"users\",\"fields\":"
    "{\"id\":{\"type\":\"int\",\"required\":true},"
    "\"name\":{\"type\":\"string\",\"required\":false}}}";

char* result = OmniQL_RegisterSchema(handle, schema);
OmniQL_Free(result);
```

---

### Memory management

#### `OmniQL_Free(char* ptr)`
Frees a C string previously returned by `OmniQL_Execute`,
`OmniQL_RegisterSchema`, `OmniQL_Route`, or any `OmniQL_Register*Driver`
function. **Never call `free()` directly on these pointers.**

```c
OmniQL_Free(ptr);
```

---

## Multi-engine usage

Multiple engines can coexist within the same process, each with its own
registered drivers and route table. Handles are unique integers allocated
by the engine registry.

```c
int engine1 = OmniQL_NewEngine();
int engine2 = OmniQL_NewEngine();

OmniQL_RegisterSQLiteDriver(engine1, "db1.sqlite");
OmniQL_RegisterSQLiteDriver(engine2, "db2.sqlite");

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
| `ENGINE_ERROR` | Unexpected engine-level error |
| `MARSHAL_ERROR` | Result could not be serialized to JSON (should never occur) |
