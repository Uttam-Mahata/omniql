# OmniQL CLI — v0.3.0

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

### SQLite

```bash
# In-memory database
omniql -driver sqlite -dsn :memory: \
  -query '{"target":"users","action":"FIND","filter":{},"options":{"limit":5}}'

# File-based database
omniql -driver sqlite -dsn /var/data/app.db \
  -query '{"target":"orders","action":"COUNT","filter":{"status":"pending"}}'
```

### PostgreSQL

```bash
omniql -driver postgres \
  -dsn "postgres://user:pass@localhost/mydb?sslmode=disable" \
  -query '{"target":"orders","action":"FIND","filter":{"status":"pending"},"options":{"limit":20}}'

# With $lt filter
omniql -driver postgres \
  -dsn "host=localhost user=pg password=pg dbname=shop sslmode=disable" \
  -query '{"target":"products","action":"FIND","filter":{"price":{"$lt":100}}}'
```

### MongoDB

```bash
omniql -driver mongo -dsn "mongodb://localhost:27017" -db mydb \
  -query '{"target":"products","action":"COUNT","filter":{"price":{"$lt":100}}}'

omniql -driver mongo -dsn "mongodb://user:pass@host:27017" -db analytics \
  -query '{"target":"events","action":"FIND","filter":{"type":{"$in":["click","view"]}},"options":{"limit":50}}'
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
