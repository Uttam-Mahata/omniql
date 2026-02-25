# OmniQL v0.6.0 — Changelog

> **Relational Parity + Distribution & Tooling release**
> Adds the MySQL driver, cross-driver compliance tests, `batchInsert` convenience
> API for all language bindings, `omniql.yaml` config support, and GoReleaser-based
> binary distribution. Also fixes two correctness bugs in the SQLite/Postgres drivers.

---

## New Features

### 1. MySQL Driver (`pkg/drivers/mysql/`)

A new production-ready MySQL adapter implementing the full `core.Driver` interface.

| Property | Value |
|----------|-------|
| Package | `pkg/drivers/mysql` |
| Placeholder style | `?` |
| Identifier quoting | Backticks `` ` `` |
| JSON path | `->>` operator (MySQL 5.7+) |
| Dependency | `github.com/go-sql-driver/mysql` |

Supports all actions (FIND, COUNT, INSERT, BATCH_INSERT, UPDATE, DELETE), all 8 filter
operators, recursive `$or`/`$and`, pagination, sorting, and inclusion projection.

**CLI:**
```bash
omniql -driver mysql -dsn "root:pass@tcp(127.0.0.1:3306)/mydb" -query '...'
```

**FFI:**
```c
char* result = OmniQL_RegisterMySQLDriver(handle, "root:pass@tcp(127.0.0.1:3306)/mydb");
OmniQL_Free(result);
```

**Go embed:**
```go
import drvmysql "github.com/Uttam-Mahata/omniql/pkg/drivers/mysql"

drv, err := drvmysql.New("root:pass@tcp(127.0.0.1:3306)/mydb?parseTime=true")
engine.RegisterDriver(drv)
engine.Route("orders", drv.Name())
```

---

### 2. `batchInsert` Convenience API — All Bindings

Every language binding now exposes a first-class `batchInsert` method so callers
no longer need to construct `BATCH_INSERT` queries by hand.

| Binding | Method |
|---------|--------|
| Python | `engine.batch_insert(target, docs)` / `engine.batch_insert_sync(target, docs)` |
| Java | `engine.batchInsert(target, docs)` / `engine.table(t).batchInsert(docs)` |
| C# | `engine.BatchInsertAsync(target, docs)` / `builder.BatchInsertAsync(docs)` |
| TypeScript | `engine.batchInsert(target, docs)` |

---

### 3. Cross-Driver Compliance Suite (`pkg/compliance/`)

A canonical set of **50+ test cases** that every driver must pass. Tests cover:

- All 6 actions and all 8 filter operators
- Recursive `$or` / `$and` (including nested combinations)
- Pagination: `limit`, `skip`, limit+skip, skip-beyond-total, limit-larger-than-total
- Sort ASC/DESC, sort+limit, sort determinism across repeated queries
- Inclusion projection
- `BATCH_INSERT` (including empty batch)
- All error conditions (empty filter for UPDATE/DELETE, empty `$in`/`$nin`, field exclusion, unsupported action)
- Engine routing and strict-schema mode
- Per-category COUNT cross-check

SQLite always runs (in-memory). PostgreSQL/MongoDB run when `POSTGRES_DSN` / `MONGO_URI`+`MONGO_DB` environment variables are set.

```bash
# SQLite only (always)
go test ./pkg/compliance/...

# With PostgreSQL
POSTGRES_DSN="postgres://user:pass@localhost/testdb?sslmode=disable" go test ./pkg/compliance/...
```

---

### 4. `omniql.yaml` Config File (`cmd/omniql/config.go`)

The CLI can now load named connections and route tables from an `omniql.yaml` file,
eliminating the need to pass long DSN strings as flags.

```yaml
# omniql.yaml
drivers:
  main_db:
    type: postgres
    dsn: "postgres://user:pass@localhost/mydb?sslmode=disable"
  cache:
    type: sqlite
    dsn: "./cache.db"
  analytics:
    type: mysql
    dsn: "reader:pass@tcp(analytics-db:3306)/stats"

routes:
  users:    main_db
  sessions: cache
  events:   analytics
```

```bash
# Config mode — no -driver or -dsn needed
omniql -config omniql.yaml -query '{"target":"users","action":"FIND","filter":{}}'

# Default: looks for ./omniql.yaml automatically
omniql -query '{"target":"users","action":"FIND","filter":{}}'
```

---

### 5. GoReleaser Binary Distribution (`.goreleaser.yml`)

OmniQL CLI binaries are now distributed via GoReleaser on every stable/rc/beta tag:

| Platform | Formats |
|----------|---------|
| Linux (amd64, arm64) | `.tar.gz`, `.deb`, `.rpm` |
| macOS (amd64, arm64) | `.tar.gz`, Homebrew tap |
| Windows (amd64) | `.zip`, Scoop bucket |

Triggered automatically by the `goreleaser` job in `.github/workflows/publish.yml`
on any `v*` tag push.

```bash
# Dry-run snapshot locally (no GitHub token needed)
goreleaser release --snapshot --clean
```

---

## Bug Fixes

### BatchInsert column ordering (SQLite, Postgres)

**Problem:** `BatchInsert` built the column list and the value list by iterating the
same Go map in two separate loops. Go map iteration order is non-deterministic, so the
column names in the `INSERT` statement could end up in a different order than the
corresponding values, causing data to land in the wrong columns or trigger type errors.

**Fix:** Both drivers now collect column names once into a sorted `colOrder` slice and
use that slice consistently for both the SQL statement and every row's value list.

Files: `pkg/drivers/sqlite/driver.go`, `pkg/drivers/postgres/driver.go`

---

### SQLite OFFSET without LIMIT syntax error

**Problem:** Executing a FIND query with `options.skip > 0` but `options.limit == 0`
produced the SQL fragment `… OFFSET 100` without a preceding `LIMIT` clause. SQLite
rejects this syntax with a parse error.

**Fix:** When skip is set but limit is not, the driver now prepends `LIMIT -1` (SQLite's
idiom for "no upper bound") before the OFFSET clause.

File: `pkg/drivers/sqlite/driver.go`

---

## Dependency Changes

| Dependency | Change |
|------------|--------|
| `github.com/go-sql-driver/mysql v1.9.3` | Added — MySQL driver |
| `filippo.io/edwards25519 v1.1.0` | Added — transitive (mysql) |
| `gopkg.in/yaml.v3 v3.0.1` | Added — omniql.yaml config |

---

## Test Results

```
ok  github.com/Uttam-Mahata/omniql/pkg/compliance        (50+ tests)
ok  github.com/Uttam-Mahata/omniql/pkg/core
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/mongo
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/mysql     (skipped without MYSQL_DSN)
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/postgres
ok  github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite
ok  github.com/Uttam-Mahata/omniql/pkg/ffi
```
