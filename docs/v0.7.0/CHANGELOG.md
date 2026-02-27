# OmniQL v0.7.0 — Changelog

> **Paradigm Coverage + DX release**
> SQL Server and Redis drivers, shared SQL utilities, terminal convenience
> methods for all language bindings, MySQL added to the compliance suite,
> and fuzz tests for the core engine and FFI layer.

---

## New Features

### 1. SQL Server Driver (`pkg/drivers/sqlserver/`)

A new production-ready SQL Server / Azure SQL adapter.

| Property | Value |
|----------|-------|
| Package | `pkg/drivers/sqlserver` |
| Placeholder style | `@pN` (`@p1`, `@p2`, …) |
| Identifier quoting | Double-quotes (`"ident"`) |
| JSON path | `JSON_VALUE(col, '$.path')` (SQL Server 2016+) |
| Pagination | `ORDER BY … OFFSET N ROWS FETCH NEXT M ROWS ONLY` |
| Dependency | `github.com/microsoft/go-mssqldb` |

When no `sort` is given but pagination is requested, the driver injects
`ORDER BY (SELECT NULL)` — SQL Server requires ORDER BY before OFFSET/FETCH.

**CLI:**
```bash
omniql -driver sqlserver \
  -dsn "sqlserver://sa:Pass@localhost:1433?database=mydb" \
  -query '{"target":"orders","action":"FIND","filter":{"status":"pending"}}'
```

**FFI:**
```c
char* r = OmniQL_RegisterSQLServerDriver(handle,
    "sqlserver://sa:Pass@localhost:1433?database=mydb");
OmniQL_Free(r);
```

**Go embed:**
```go
import drvsqlserver "github.com/Uttam-Mahata/omniql/pkg/drivers/sqlserver"

drv, err := drvsqlserver.New("sqlserver://sa:Pass@localhost:1433?database=mydb")
engine.RegisterDriver(drv)
engine.Route("orders", drv.Name())
```

---

### 2. Redis Driver (`pkg/drivers/redis/`)

A new Key-Value / Cache driver built on `go-redis/v9`.

| Property | Value |
|----------|-------|
| Package | `pkg/drivers/redis` |
| Key format | `<target>:<id>` (e.g. `products:laptop`) |
| Dependency | `github.com/redis/go-redis/v9` |

**OQL → Redis action mapping:**

| OQL Action | Redis command |
|---|---|
| `FIND` with `{id: x}` filter | `GET <target>:<id>` |
| `FIND` without filter | `SCAN <target>:*` + `GET` each key |
| `INSERT` | `SET <target>:<id> <json>` ; optional `EXPIRE` if `_ttl` field present |
| `UPDATE` | `GET` → merge → `SET` |
| `DELETE` | `DEL <target>:<id>` (requires `id` or `_id` in filter) |
| `COUNT` | `SCAN <target>:*` + count |
| `BATCH_INSERT` | Pipeline of `SET` commands |

DSN format: standard Redis URL — `redis://:password@host:6379/0`

**CLI:**
```bash
omniql -driver redis -dsn "redis://localhost:6379/0" \
  -query '{"target":"sessions","action":"FIND","filter":{"id":"abc123"}}'
```

**FFI:**
```c
char* r = OmniQL_RegisterRedisDriver(handle, "redis://localhost:6379/0");
OmniQL_Free(r);
```

---

### 3. Shared SQL Utilities (`pkg/drivers/sqlutil/`)

The `buildWhere` logic previously duplicated across SQLite, PostgreSQL, and
MySQL has been extracted into a single shared package.

```go
// pkg/drivers/sqlutil/where.go
type PlaceholderFn func(n int) string

func Question(_ int) string          { return "?" }          // SQLite, MySQL
func Dollar(n int) string            { return fmt.Sprintf("$%d", n) }  // Postgres
func AtParam(n int) string           { return fmt.Sprintf("@p%d", n) } // SQL Server

func BuildWhere(
    filter       map[string]interface{},
    placeholder  PlaceholderFn,
    quoteIdent   func(string) string,
    translateCol func(string) string,
    startIdx     int,
) (clause string, args []interface{}, nextIdx int, err error)
```

All four SQL drivers (SQLite, PostgreSQL, MySQL, SQL Server) now call
`sqlutil.BuildWhere` for consistent operator support and easier maintenance.

---

### 4. Terminal Convenience Methods — All Bindings

Every language binding now exposes ergonomic one-call methods so callers
do not need to construct raw query objects for common operations.

| Method | Maps to | Returns |
|--------|---------|---------|
| `find_many(target, filter?, options?)` | `FIND` | list of docs |
| `find_first(target, filter?)` | `FIND` limit=1 | single doc or None/null |
| `count(target, filter?)` | `COUNT` | integer |
| `insert_one(target, doc)` | `INSERT` | inserted doc / OmniResult |
| `update_many(target, filter, update)` | `UPDATE` | OmniResult |
| `delete_many(target, filter)` | `DELETE` | OmniResult |

Each binding exposes both **sync** and **async** variants where applicable
(Python, TypeScript); Java and C# expose async-only variants.

---

### 5. MySQL Added to Compliance Suite

MySQL is now exercised by the cross-driver compliance suite.

```bash
MYSQL_DSN="root:@tcp(127.0.0.1:3306)/test" go test ./pkg/compliance/...
```

When `MYSQL_DSN` is not set the MySQL compliance tests are automatically skipped.

---

### 6. Fuzz Tests

Two fuzz targets protect against panics in the core hot paths:

| Target | Package | What it fuzzes |
|--------|---------|----------------|
| `FuzzExecute` | `pkg/core` | Random bytes → `OQLQuery` → `engine.Execute` |
| `FuzzFFIExecute` | `pkg/ffi` | Random handle values + malformed JSON → `OmniQL_Execute` |

```bash
go test -fuzz=FuzzExecute    -fuzztime=30s ./pkg/core/...
go test -fuzz=FuzzFFIExecute -fuzztime=30s ./pkg/ffi/...
```

---

## Dependency Changes

| Dependency | Change |
|------------|--------|
| `github.com/microsoft/go-mssqldb v1.9.6` | Added — SQL Server driver |
| `github.com/redis/go-redis/v9 v9.18.0` | Added — Redis driver |
| Various Azure SDK / xxhash transitive deps | Added automatically by go-mssqldb |

---

## Test Results

```
ok  github.com/Uttam-Mahata/omniql/pkg/compliance        (SQLite always; MySQL/Postgres/Mongo with env vars)
ok  github.com/Uttam-Mahata/omniql/pkg/core
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/mongo
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/mysql     (skipped without MYSQL_DSN)
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/postgres
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/redis     (skipped without REDIS_URL)
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/sqlserver (skipped without SQLSERVER_DSN)
ok  github.com/Uttam-Mahata/omniql/pkg/ffi
```
