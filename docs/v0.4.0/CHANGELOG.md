# Changelog — v0.4.0

> Released: 2026-02-22  
> Compared to: v0.3.0

---

## Bug Fixes

### INSERT now returns data
In v0.3.0 the `INSERT` action always returned an empty `data` array. In v0.4.0, all three drivers now return the inserted document or its identifier:
- **SQLite:** Returns `[{"id": <last_insert_id>}]`.
- **Postgres:** Returns the full inserted row via `RETURNING *`.
- **MongoDB:** Returns `[{"_id": <inserted_id>}]`.

### MongoDB driver tests
Added a comprehensive suite of integration tests for the MongoDB driver (`pkg/drivers/mongo/driver_test.go`) covering `FIND`, `INSERT`, `UPDATE`, `DELETE`, and `COUNT` actions with various filters and operators.

---

## New Features

### Logical Operators: `$or` and `$and`
All database drivers now support the `$or` and `$and` logical operators in OQL filters for multi-predicate grouping.
```json
{
  "filter": {
    "$or": [
      { "category": "electronics" },
      { "price": { "$lt": 100 } }
    ]
  }
}
```

### Query Sorting
`QueryOptions.Sort` is now implemented in all drivers. It supports multiple fields with directions (1 for ascending, -1 for descending).
```json
{
  "options": {
    "sort": { "price": -1, "name": 1 }
  }
}
}
```

### Field Projection
`QueryOptions.Fields` is now implemented in all drivers. It allows selecting specific fields to be returned (inclusion only).
```json
{
  "options": {
    "fields": { "name": 1, "price": 1 }
  }
}
```

---

## Known Remaining Limitations (targeting v0.5.0)

| Issue | Notes |
|-------|-------|
| TypeScript `execute` blocks event loop | Sync FFI call wrapped in Promise; needs worker thread or async FFI call |
| Projection (exclusion) | All drivers only support field inclusion (`{"field": 1}`) |
| Schema Migration | No automated schema evolution or migration tool |
| Joint/Join Queries | OQL remains focused on single-collection/table operations |
