# OmniQL CLI — v0.6.0

The `omniql` binary lets you run ad-hoc OQL queries from the command line
against any supported database. v0.6.0 adds MySQL support and a config-file
mode so you can manage multiple named connections in `omniql.yaml`.

---

## Build

```bash
go build -o omniql ./cmd/omniql
```

---

## Modes

### Flag mode — single driver on the command line

```
omniql -driver <driver> -dsn <dsn> [-db <dbname>] -query '<OQL JSON>'
```

### Config mode — named connections from `omniql.yaml`

```
omniql [-config <path>] -query '<OQL JSON>'
```

When `-driver` is absent the CLI looks for `omniql.yaml` in the current
directory (or the path given by `-config`).

---

## Flags

| Flag | Required | Description |
|------|----------|-------------|
| `-query` | ✅ always | JSON-encoded OQL query |
| `-driver` | flag mode only | Driver: `sqlite`, `postgres`, `mongo`, `mysql` |
| `-dsn` | flag mode only | Connection string, file path, or URI |
| `-db` | flag mode, mongo | MongoDB database name |
| `-config` | config mode | Path to config file (default: `./omniql.yaml`) |

---

## Flag Mode Examples

### SQLite (in-memory)

```bash
omniql -driver sqlite -dsn :memory: \
  -query '{"target":"t","action":"FIND","filter":{}}'
```

### PostgreSQL with logical operators

```bash
omniql -driver postgres \
  -dsn "postgres://user:pass@localhost/mydb?sslmode=disable" \
  -query '{
    "target": "orders",
    "action": "FIND",
    "filter": {
      "$or": [
        {"status": "pending"},
        {"priority": {"$gte": 5}}
      ]
    },
    "options": {"limit": 20, "sort": {"created_at": -1}}
  }'
```

### MongoDB count

```bash
omniql -driver mongo -dsn "mongodb://localhost:27017" -db mydb \
  -query '{
    "target": "products",
    "action": "COUNT",
    "filter": {"$and": [{"price":{"$gte":10}},{"price":{"$lte":100}}]}
  }'
```

### MySQL (new in v0.6.0)

```bash
omniql -driver mysql \
  -dsn "root:pass@tcp(127.0.0.1:3306)/shop?parseTime=true" \
  -query '{
    "target": "items",
    "action": "FIND",
    "filter": {"category": {"$in": ["electronics", "books"]}},
    "options": {"limit": 10, "sort": {"price": 1}}
  }'
```

---

## Config Mode Examples

### omniql.yaml

```yaml
drivers:
  main_db:
    type: postgres
    dsn: "postgres://user:pass@localhost/mydb?sslmode=disable"
  cache:
    type: sqlite
    dsn: "./cache.db"
  shop:
    type: mysql
    dsn: "root:pass@tcp(localhost:3306)/shop?parseTime=true"
  mongo_db:
    type: mongo
    dsn: "mongodb://localhost:27017"
    db:  "analytics"

routes:
  users:    main_db
  sessions: cache
  orders:   shop
  events:   mongo_db
```

### Queries using the config file

```bash
# Default config path (./omniql.yaml)
omniql -query '{"target":"users","action":"FIND","filter":{"active":true}}'

# Explicit config path
omniql -config /etc/omniql/prod.yaml \
  -query '{"target":"orders","action":"COUNT","filter":{"status":"pending"}}'
```

---

## BATCH_INSERT via CLI

```bash
omniql -driver sqlite -dsn ./app.db -query '{
  "target": "log_entries",
  "action": "BATCH_INSERT",
  "documents": [
    {"level": "info",  "msg": "service started"},
    {"level": "warn",  "msg": "high memory usage"},
    {"level": "error", "msg": "connection timeout"}
  ]
}'
```

---

## Output Format

```json
{
  "data": [
    { "id": 1, "name": "Widget", "price": 9.99 }
  ],
  "meta": {
    "total":    1,
    "returned": 1,
    "driver":   "mysql",
    "target":   "products"
  },
  "error": null
}
```

On error:

```json
{
  "data": [],
  "meta": { "total": 0, "returned": 0, "driver": "", "target": "" },
  "error": {
    "code":    "EXECUTION_ERROR",
    "message": "mysql: table 'products' doesn't exist"
  }
}
```

---

## Error Codes

| Code | Cause |
|------|-------|
| `SCHEMA_ERROR` | Query failed strict schema validation |
| `DRIVER_ERROR` | No driver registered/routed for the target |
| `EXECUTION_ERROR` | The driver returned an error |
| `PARSE_ERROR` | The `-query` value is not valid JSON |

---

## Installing Pre-built Binaries (v0.6.0+)

### Homebrew (macOS)

```bash
brew tap Uttam-Mahata/omniql
brew install omniql
```

### Scoop (Windows)

```powershell
scoop bucket add omniql https://github.com/Uttam-Mahata/scoop-omniql
scoop install omniql
```

### Linux (.deb / .rpm)

Download the appropriate package from the [GitHub Releases](https://github.com/Uttam-Mahata/omniql/releases)
page, then:

```bash
# Debian / Ubuntu
sudo dpkg -i omniql_0.6.0_linux_amd64.deb

# Fedora / RHEL
sudo rpm -i omniql_0.6.0_linux_amd64.rpm
```
