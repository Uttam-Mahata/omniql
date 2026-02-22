# Changelog — v0.3.0

> Released: 2026-02-22  
> Compared to: v0.2.0

---

## Bug Fixes

### FFI: `pkg/ffi` renamed to `package main` _(critical)_
The `go build -buildmode=c-shared` command requires the entry package to be
`package main`. In v0.2.0 the package was declared as `package ffi`, making
the shared library completely unbuildable.

### FFI: No driver registration or routing exports _(critical)_
In v0.2.0 the FFI layer only exported `NewEngine`, `FreeEngine`, `Execute`,
`RegisterSchema`, and `Free`. Language bindings had no way to register a
database driver or configure target routing. Every `Execute` call returned:
```json
{"error": {"code": "DRIVER_ERROR", "message": "no drivers registered"}}
```
**Fixed:** Added `OmniQL_RegisterSQLiteDriver`, `OmniQL_RegisterPostgresDriver`,
`OmniQL_RegisterMongoDriver`, and `OmniQL_Route`.

### C#: Silent JSON serialisation bug _(critical)_
`OmniEngine.cs` serialized `OQLQuery` using `System.Text.Json` defaults, which
produced PascalCase JSON keys (`"Target"`, `"Action"`, `"Filter"`…). The Go
engine deserializes with `encoding/json` which expects camelCase (`"target"`,
`"action"`, `"filter"`…). All queries from C# were silently mis-parsed,
returning empty results.

**Fixed:** Added `[JsonPropertyName("...")]` attributes to all properties of
`OQLQuery`, `OQLOptions`, `OmniResult`, `OmniMeta`, and `OmniError`. The same
fix was applied to the deserialization side.

### CLI: Always returned `DRIVER_ERROR` _(critical)_
`cmd/omniql/main.go` created an engine but never registered a driver. Every
query returned:
```json
{"error": {"code": "DRIVER_ERROR", "message": "no drivers registered: \"target\""}}
```
**Fixed:** Added `-driver`, `-dsn`, `-db` flags. The CLI now constructs and
registers the appropriate driver at startup and routes the query target before
executing.

### Python: Deprecated `asyncio.get_event_loop()` _(minor)_
The async methods in `omniql.py` used `asyncio.get_event_loop()`, which is
deprecated in Python 3.10+ and raises a `DeprecationWarning`.

**Fixed:** Replaced with `asyncio.get_running_loop()` (falls back to
`asyncio.get_event_loop()` when called outside a running loop).

---

## New Features

### FFI: `OmniQL_Route`
Exposes the engine's `Route(target, driverName)` method to the C ABI. Required
for bindings to configure which driver handles which collection/table.

### FFI: Driver registration exports
Three new exported functions:

| Function | Driver |
|----------|--------|
| `OmniQL_RegisterSQLiteDriver(handle, dsn)` | SQLite via `go-sqlite3` |
| `OmniQL_RegisterPostgresDriver(handle, connStr)` | Postgres via `lib/pq` |
| `OmniQL_RegisterMongoDriver(handle, uri, dbName)` | MongoDB via `mongo-driver` |

Each returns `{"driver":"<name>"}` on success for easy driver-name extraction
in binding code.

### FFI: `wrappers.go` + `ffi_test.go`
Added Go-typed wrapper functions (`wrappers.go`) so the package can be tested
without importing `"C"` in test files (forbidden in packages that use
`//export`). Added four smoke tests:
- `TestNewFreeEngine` — handle uniqueness and registry cleanup
- `TestInvalidHandleReturnsError` — graceful unknown-handle handling
- `TestRegisterSQLiteDriverViaFFI` — registration + routing via FFI string API
- `TestRouteAndFindDirect` — full FIND round-trip with seeded in-memory SQLite
- `TestRegisterSchema` — schema registration returns empty success string

### C#: Driver registration and routing methods
Added to `OmniEngine`:
- `RegisterSQLiteDriver(dsn) → string`
- `RegisterPostgresDriver(connStr) → string`
- `RegisterMongoDriver(uri, dbName) → string`
- `Route(target, driverName)`

Added to `QueryBuilder`: `Update(document)`, `Delete(filter)`, `Count(filter?)`

### Python: Driver registration and routing methods
Added to `OmniEngine` (sync + async pairs):
- `register_sqlite_driver_sync(dsn)` / `await register_sqlite_driver(dsn)`
- `register_postgres_driver_sync(conn_str)` / `await register_postgres_driver(conn_str)`
- `register_mongo_driver_sync(uri, db_name)` / `await register_mongo_driver(uri, db_name)`
- `route_sync(target, driver_name)` / `await route(target, driver_name)`

### Java: Driver registration and routing methods
Added to `OmniEngine`:
- `registerSQLiteDriver(dsn) → String`
- `registerPostgresDriver(connStr) → String`
- `registerMongoDriver(uri, dbName) → String`
- `route(target, driverName)`

Added to `QueryBuilder`: `insert(document)`, `update(document)`,
`delete(filter)`, `count(filter)`

### TypeScript: Driver registration and routing methods
Added to `OmniEngine`:
- `registerSQLiteDriver(dsn) → string`
- `registerPostgresDriver(connStr) → string`
- `registerMongoDriver(uri, dbName) → string`
- `route(target, driverName)`
Updated `ffi.Library` map to declare all 4 new native symbols.

---

## Dependencies

| Dependency | Change |
|------------|--------|
| `github.com/lib/pq` v1.11.2 | Added — Postgres `database/sql` driver required by `OmniQL_RegisterPostgresDriver` |

---

## Known Remaining Limitations (targeting v0.4.0)

| Issue | Notes |
|-------|-------|
| `QueryOptions.Sort` silently ignored | Defined in `oql.go`; no driver builds `ORDER BY` / sort options |
| `QueryOptions.Fields` (projection) silently ignored | All drivers return `SELECT *` |
| INSERT returns empty `data` | Inserted ID/document not surfaced |
| MongoDB driver has no tests | Zero test coverage |
| `$or` / `$and` logical operators not supported | No multi-predicate logical grouping |
| TypeScript `execute` may block event loop | Sync FFI call wrapped in Promise without worker thread |
