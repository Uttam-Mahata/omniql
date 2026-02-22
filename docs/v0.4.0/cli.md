# OmniQL CLI — v0.4.0

The `omniql` binary lets you run ad-hoc OQL queries from the command line
against any of the supported databases.

---

## Build

```bash
go build -o omniql ./cmd/omniql
```

---

## Usage

```
omniql -driver <sqlite|postgres|mongo> -dsn <dsn> [-db <dbname>] -query '<OQL JSON>'
```

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `-driver` | ✅ | Database driver: `sqlite`, `postgres`, or `mongo` |
| `-dsn` | ✅ | Connection string, file path, or MongoDB URI |
| `-db` | Only for `mongo` | Database name to use with the MongoDB driver |
| `-query` | ✅ | JSON-encoded OQL query |

---

## Examples

### SQLite with Sorting and Projection

```bash
# Find active users, sorted by price descending, returning only name and price
omniql -driver sqlite -dsn /var/data/app.db \
  -query '{
    "target": "users",
    "action": "FIND",
    "filter": {"status": "active"},
    "options": {
      "limit": 5,
      "sort": {"price": -1},
      "fields": {"name": 1, "price": 1}
    }
  }'
```

### PostgreSQL with Logical Operators

```bash
# Find orders that are either pending or high-priority
omniql -driver postgres \
  -dsn "postgres://user:pass@localhost/mydb?sslmode=disable" \
  -query '{
    "target": "orders",
    "action": "FIND",
    "filter": {
      "$or": [
        {"status": "pending"},
        {"priority": "high"}
      ]
    },
    "options": {"limit": 20}
  }'
```

### MongoDB Count with Complex Filter

```bash
# Count products with price between 10 and 100
omniql -driver mongo -dsn "mongodb://localhost:27017" -db mydb \
  -query '{
    "target": "products",
    "action": "COUNT",
    "filter": {
      "$and": [
        {"price": {"$gte": 10}},
        {"price": {"$lte": 100}}
      ]
    }
  }'
```

---

## Output Format

The CLI outputs a JSON-encoded OmniJSON object to stdout:

```json
{
  "data": [
    { "id": 1, "name": "Widget", "price": 9.99 }
  ],
  "meta": {
    "total": 1,
    "returned": 1,
    "driver": "sqlite",
    "target": "products"
  },
  "error": null
}
```

On error, the `error` field is populated:

```json
{
  "data": [],
  "meta": { "total": 0, "returned": 0, "driver": "", "target": "" },
  "error": {
    "code": "EXECUTION_ERROR",
    "message": "sqlite count: no such table: products"
  }
}
```

### Error Codes

| Code | Cause |
|------|-------|
| `SCHEMA_ERROR` | Query failed strict schema validation |
| `DRIVER_ERROR` | No driver registered for the target (should not occur in v0.3.0 CLI) |
| `EXECUTION_ERROR` | The driver returned an error (e.g. table not found, syntax error) |
| `PARSE_ERROR` | The `-query` flag value is not valid JSON |
