# OmniQL v0.5.0 — Implementation Changelog

This document describes the concrete changes shipped in v0.5.0.

---

## Bug Fixes & Code-Quality Improvements

### 1. Deterministic fallback driver (`pkg/core/engine.go`)

**Problem:** When no explicit `Route()` was set for a target, `selectDriver` iterated the
`drivers` map, whose ordering is randomised by the Go runtime. With two or more drivers
registered this made the routing non-deterministic.

**Fix:** `Engine` now stores a `defaultDriver` field that is set to the name of the
_first_ driver passed to `RegisterDriver`. The fallback always resolves to that driver,
making the behaviour predictable and documentable.

---

### 2. Schema validator recurses into `$or`/`$and` (`pkg/core/schema.go`)

**Problem:** The filter validator iterated top-level keys only. A filter such as
`{"$or": [{"undeclared_field": 1}]}` was either rejected because `$or` itself is not a
schema field, or passed through unchecked when `$or`/`$and` were whitelisted.

**Fix:** Introduced `validateFilter`, a recursive helper that:
- Skips `$or`/`$and` keys at the current level (they are logical operators, not fields).
- Recursively validates every object inside the operator's array.
- Returns an error as soon as an undeclared field is found at any nesting level.

---

### 3. Removed unreachable `ENGINE_ERROR` branch in FFI (`pkg/ffi/ffi.go`)

**Problem:** `engine.Execute` never returns a non-nil `error` — every failure is wrapped
inside the returned `*OmniJSON` value. The `if err != nil { return errorJSON("ENGINE_ERROR", …) }`
block was therefore dead code.

**Fix:** Changed the assignment to `result, _ := engine.Execute(…)`, making the
contract explicit and eliminating the unreachable branch.

---

### 4. `BATCH_INSERT` action wired end-to-end (`pkg/core/oql.go`, `pkg/core/engine.go`)

**Status:** Already implemented in the previous commit.  
`OQLQuery.Documents`, `ActionBatchInsert`, and the engine dispatch to `Driver.BatchInsert`
are all present and tested.

---

### 5. SQL drivers return an error for field exclusion (`pkg/drivers/sqlite`, `pkg/drivers/postgres`)

**Problem:** When `QueryOptions.Fields` contained a value of `0` (exclusion), the SQL
drivers silently fell back to `SELECT *`, so the caller received all fields with no
indication that their projection was ignored.

**Fix:** Both SQLite and Postgres `find()` methods now return a descriptive error when
any field value equals `0`, directing users to use inclusion-only projection (`1`) with
SQL backends.

---

### 6. SQL drivers reject UPDATE/DELETE with an empty filter (`pkg/drivers/sqlite`, `pkg/drivers/postgres`)

**Problem:** An `UPDATE` or `DELETE` query with an empty `filter` silently affected
every row in the target table — a common accidental bulk-mutation footgun.

**Fix:** Both SQLite and Postgres `update()` and `delete()` methods now return an error
when `query.Filter` is empty, requiring callers to supply an explicit filter.

---

## Testing

All existing tests pass. New tests were added for:

- `TestSchemaRegistry_Validate_OrWithKnownFields` — `$or` with declared fields passes.
- `TestSchemaRegistry_Validate_OrWithUnknownField` — `$or` containing an undeclared field is rejected.
- `TestEngine_DefaultDriverIsDeterministic` — fallback always resolves to the first-registered driver.
- `TestSQLiteDriver_FieldExclusionReturnsError` — exclusion projection returns an error.
- `TestSQLiteDriver_UpdateEmptyFilterReturnsError` — UPDATE without filter returns an error.
- `TestSQLiteDriver_DeleteEmptyFilterReturnsError` — DELETE without filter returns an error.

Run the full suite with:

```bash
go test ./pkg/...
```
