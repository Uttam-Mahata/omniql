# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build CLI binary
go build -o omniql ./cmd/omniql

# Build FFI shared library (C ABI for language bindings)
go build -buildmode=c-shared -o libomniql.so ./pkg/ffi

# Run all tests
go test ./...

# Run tests for a specific package
go test ./pkg/core/...
go test ./pkg/drivers/sqlite/...
go test ./pkg/drivers/postgres/...
go test ./pkg/drivers/mongo/...
go test ./pkg/ffi/...

# Run a single test
go test ./pkg/core/... -run TestEngineName

# Lint (standard Go tooling)
go vet ./...
```

> `go-sqlite3` requires CGO. Ensure `gcc` is available; tests for the sqlite and ffi packages will fail without it.

## Architecture

OmniQL is a unified query routing layer. A caller submits a JSON `OQLQuery` and gets back a JSON `OmniJSON` response regardless of which database is underneath.

### Core pipeline (`pkg/core/`)

```
OQLQuery → SchemaRegistry.Validate → Engine.selectDriver → Driver.Execute → OmniJSON
```

Key types:
- **`OQLQuery`** (`oql.go`) — the canonical, database-agnostic query object (`target`, `action`, `filter`, `document`, `options`).
- **`Driver`** interface (`driver.go`) — every database adapter must implement `Name()`, `Execute()`, `Ping()`, `Close()`.
- **`Engine`** (`engine.go`) — holds a `drivers` map and a `routes` map (target → driver name). `selectDriver` first checks explicit routes, then falls back to the first registered driver. Errors always return an `OmniJSON` with a non-nil `Error` field rather than a Go error.
- **`SchemaRegistry`** (`schema.go`) — optional schema validation; strict mode rejects queries against unregistered targets.
- **`OmniJSON`** / **`OmniMeta`** / **`OmniError`** (`omnijson.go`) — the normalised response format.

### Drivers (`pkg/drivers/`)

| Driver | Package | Notes |
|--------|---------|-------|
| SQLite | `pkg/drivers/sqlite` | `database/sql` + `go-sqlite3` (CGO). Uses `?` placeholders. `New(dsn)` or `NewFromDB(db)`. |
| PostgreSQL | `pkg/drivers/postgres` | `database/sql` + `lib/pq`. Uses `$N` placeholders. Takes a pre-opened `*sql.DB`. |
| MongoDB | `pkg/drivers/mongo` | `mongo-driver`. OQL filter operators map 1:1 to MongoDB BSON operators. Takes a pre-connected `*mongo.Client`. |

All three drivers follow the same pattern: a top-level `Execute` switch dispatches to private `find`, `insert`, `update`, `delete` methods. SQL drivers build parameterised queries via `buildWhere` / `buildWhereFrom`. `$or` / `$and` are handled recursively. `Fields` (projection) only honours inclusion (`1`); `Sort` values `1`/`-1` map to ASC/DESC.

### FFI layer (`pkg/ffi/`)

`pkg/ffi` is declared `package main` — **required** by Go's `-buildmode=c-shared`. It exposes a C ABI (`OmniQL_NewEngine`, `OmniQL_Execute`, etc.) over an internal `engines` map keyed by opaque integer handles.

`wrappers.go` provides Go-typed shims (`goNewEngine`, `goExecute`, …) so that test files can call FFI logic without importing `"C"` (forbidden in test files of packages that use `//export`).

### CLI (`cmd/omniql/`)

Thin wrapper: parses `-driver`, `-dsn`, `-db`, `-query` flags → instantiates the chosen driver → calls `engine.Execute` → pretty-prints `OmniJSON` to stdout.

### Language bindings (`bindings/`)

Each binding (`java/`, `python/`, `csharp/`, `typescript/`) loads `libomniql.so` at runtime and wraps the C ABI. They all follow the same three-step pattern: register driver → route target → execute query. The bindings are standalone files and do not form part of the Go module.

## Implementing a new driver

1. Create `pkg/drivers/<name>/driver.go` with `package <name>`.
2. Define a `Driver` struct and implement the `core.Driver` interface: `Name() string`, `Execute(ctx, OQLQuery) ([]map[string]interface{}, int64, error)`, `Ping(ctx) error`, `Close() error`.
3. Translate `OQLQuery.Filter` into the native query format using the MongoDB-style operators (`$eq`, `$ne`, `$lt`, `$lte`, `$gt`, `$gte`, `$in`, `$nin`, `$or`, `$and`).
4. Register the driver in `cmd/omniql/main.go` and `pkg/ffi/ffi.go` (add `OmniQL_Register<Name>Driver`).
