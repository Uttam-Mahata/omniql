# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build CLI binary
go build -o omniql ./cmd/omniql

# Build FFI shared library (C ABI for language bindings)
go build -buildmode=c-shared -o libomniql.so ./pkg/ffi

# Run all tests (CGO required for sqlite/ffi)
CGO_ENABLED=1 go test ./...

# Run tests for a specific package
go test ./pkg/core/...
go test ./pkg/drivers/sqlite/...
go test ./pkg/drivers/postgres/...
go test ./pkg/drivers/mongo/...
go test ./pkg/drivers/mysql/...
go test ./pkg/compliance/...
go test ./pkg/ffi/...

# Run a single test
go test ./pkg/core/... -run TestEngineName

# MySQL / Postgres integration tests (require live DB)
MYSQL_DSN="root:@tcp(127.0.0.1:3306)/test" go test ./pkg/drivers/mysql/...
POSTGRES_DSN="postgres://user:pass@localhost/testdb?sslmode=disable" go test ./pkg/drivers/postgres/...

# Lint (standard Go tooling)
go vet ./...

# GoReleaser dry-run (produces ./dist without pushing to GitHub)
goreleaser release --snapshot --clean
```

> `go-sqlite3` requires CGO. Ensure `gcc` is available; tests for the sqlite, compliance,
> and ffi packages will fail without it.

## Architecture

OmniQL is a unified query routing layer. A caller submits a JSON `OQLQuery` and gets back a JSON `OmniJSON` response regardless of which database is underneath.

### Core pipeline (`pkg/core/`)

```
OQLQuery → SchemaRegistry.Validate → Engine.selectDriver → Driver.Execute → OmniJSON
```

Key types:
- **`OQLQuery`** (`oql.go`) — the canonical, database-agnostic query object (`target`, `action`, `filter`, `document`, `documents`, `options`).
- **`Driver`** interface (`driver.go`) — every database adapter must implement `Name()`, `Execute()`, `BatchInsert()`, `Ping()`, `Close()`.
- **`Engine`** (`engine.go`) — holds a `drivers` map and a `routes` map (target → driver name). `selectDriver` first checks explicit routes, then falls back to the first registered driver. Errors always return an `OmniJSON` with a non-nil `Error` field rather than a Go error.
- **`SchemaRegistry`** (`schema.go`) — optional schema validation; strict mode (`WithStrictSchema()` option) rejects queries against unregistered targets.
- **`OmniJSON`** / **`OmniMeta`** / **`OmniError`** (`omnijson.go`) — the normalised response format.

### Drivers (`pkg/drivers/`)

| Driver | Package | Placeholder | Quote style | Notes |
|--------|---------|-------------|-------------|-------|
| SQLite | `pkg/drivers/sqlite` | `?` | `"ident"` | CGO. `New(dsn)` or `NewFromDB(db)`. |
| PostgreSQL | `pkg/drivers/postgres` | `$N` | `"ident"` | `lib/pq`. Takes a pre-opened `*sql.DB`. |
| MongoDB | `pkg/drivers/mongo` | BSON | — | `mongo-driver`. OQL operators map 1:1 to BSON. |
| MySQL | `pkg/drivers/mysql` | `?` | `` `ident` `` | `go-sql-driver/mysql`. JSON path via `->>`. |

All four drivers follow the same pattern: a top-level `Execute` switch dispatches to private `find`, `insert`, `update`, `delete` methods. SQL drivers build parameterised queries via `buildWhere`. `$or` / `$and` are handled recursively. `Fields` (projection) only honours inclusion (`1`); `Sort` values `1`/`-1` map to ASC/DESC.

**Important:** When implementing `BatchInsert`, always collect column names into a sorted slice and use that slice for both the SQL statement and each row's value list. Go map iteration order is non-deterministic — iterating a map twice may produce different orderings.

### Compliance suite (`pkg/compliance/`)

`pkg/compliance/suite_test.go` contains 50+ canonical OQL tests run against every driver. SQLite always runs (in-memory). PostgreSQL and MongoDB run when `POSTGRES_DSN` / `MONGO_URI`+`MONGO_DB` environment variables are set. All new drivers must pass the compliance suite.

### FFI layer (`pkg/ffi/`)

`pkg/ffi` is declared `package main` — **required** by Go's `-buildmode=c-shared`. It exposes a C ABI over an internal `engines` map keyed by opaque integer handles.

Exported functions (v0.6.0):
`OmniQL_NewEngine`, `OmniQL_FreeEngine`, `OmniQL_Execute`, `OmniQL_Free`,
`OmniQL_RegisterSchema`, `OmniQL_Route`,
`OmniQL_RegisterSQLiteDriver`, `OmniQL_RegisterPostgresDriver`,
`OmniQL_RegisterMongoDriver`, `OmniQL_RegisterMySQLDriver`

`wrappers.go` provides Go-typed shims (`goNewEngine`, `goExecute`, …) so that test files can call FFI logic without importing `"C"` (forbidden in test files of packages that use `//export`).

### CLI (`cmd/omniql/`)

Two modes of operation:

1. **Flag mode** — `-driver <type> -dsn <dsn> [-db <dbname>] -query '<JSON>'`
2. **Config mode** — `-config omniql.yaml -query '<JSON>'` (or no flags; defaults to `./omniql.yaml`)

`config.go` handles YAML loading. `main.go` registers drivers and applies routes then calls `engine.Execute`.

### Language bindings (`bindings/`)

Each binding (`java/`, `python/`, `csharp/`, `typescript/`) loads `libomniql.so` at runtime and wraps the C ABI. All four follow the same three-step pattern: register driver → route target → execute query. All bindings expose a `batchInsert` convenience method (v0.6.0). The bindings are standalone files and do not form part of the Go module.

## CI/CD & Release Strategy

Releases are tag-driven. **Do not push to `dev` to publish packages** — the publish workflow only runs on tags and the nightly schedule.

### Release types & tag conventions

| Release type | Tag format | Example |
|---|---|---|
| Stable | `v<major>.<minor>.<patch>` | `v0.6.0` |
| Release Candidate | `v<version>-rc.<n>` | `v0.6.0-rc.1` |
| Beta | `v<version>-beta.<n>` | `v0.6.0-beta.1` |
| Nightly | *(scheduled — no tag needed)* | runs daily at 02:00 UTC |

### Cutting a release

```bash
# Beta
git tag v0.6.0-beta.1 && git push origin v0.6.0-beta.1

# Release candidate
git tag v0.6.0-rc.1 && git push origin v0.6.0-rc.1

# Stable
git tag v0.6.0 && git push origin v0.6.0
```

### Version matrix per ecosystem

| Type | Python / NuGet / Maven JAR | npm dist-tag | Maven | CLI binaries |
|---|---|---|---|---|
| nightly | `0.6.0-nightly.YYYYMMDD` | `nightly` | `0.6.0-SNAPSHOT` | — |
| beta | `0.6.0-beta.1` | `beta` | `0.6.0-beta.1` | GoReleaser |
| rc | `0.6.0-rc.1` | `next` | `0.6.0-rc.1` | GoReleaser |
| stable | `0.6.0` | `latest` | `0.6.0` | GoReleaser |

The base version is the source of truth in `bindings/python/pyproject.toml`. Update it there before tagging.

### Workflow files

- `.github/workflows/publish.yml` — builds Go shared library, publishes PyPI/npm/NuGet/Maven, runs GoReleaser for CLI binaries on tags.
- `.goreleaser.yml` — GoReleaser config: Linux/macOS/Windows binaries, `.deb`/`.rpm`, Homebrew tap, Scoop bucket.

---

## Implementing a new driver

1. Create `pkg/drivers/<name>/driver.go` with `package <name>`.
2. Define a `Driver` struct and implement the `core.Driver` interface:
   - `Name() string`
   - `Execute(ctx, OQLQuery) ([]map[string]interface{}, int64, error)`
   - `BatchInsert(ctx, target string, docs []map[string]interface{}) ([]map[string]interface{}, error)`
   - `Ping(ctx) error`
   - `Close() error`
3. Translate `OQLQuery.Filter` into the native query format using the MongoDB-style operators (`$eq`, `$ne`, `$lt`, `$lte`, `$gt`, `$gte`, `$in`, `$nin`, `$or`, `$and`).
4. In `BatchInsert`: collect column names into a **sorted slice** before building the SQL statement. Use the same slice for both statement construction and per-row value ordering.
5. Register the driver in `cmd/omniql/main.go` (add `-driver <name>` case), `cmd/omniql/config.go` (`registerDriver` switch), and `pkg/ffi/ffi.go` (add `OmniQL_Register<Name>Driver` export + wrapper in `wrappers.go`).
6. Add `driver_test.go` and verify the driver passes `go test ./pkg/compliance/...`.
