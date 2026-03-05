OmniQL — Detailed Architectural Analysis                                                                                                                                           
                                                                                                                                                                                     
  1. What OmniQL Is                                                                                                                                                                  
                                                                                                                                                                                     
  OmniQL is a unified query routing layer written in Go. A caller submits a single JSON query (OQLQuery) and gets back a normalized JSON response (OmniJSON) regardless of which     
  database sits underneath — SQL, NoSQL, KV, or search.                                                                                                                              
                                                                                                                                                                                     
  OQLQuery → SchemaRegistry.Validate → Engine.selectDriver → Driver.Execute → OmniJSON

  It supports 7 drivers (SQLite, PostgreSQL, MongoDB, MySQL, SQL Server, Redis, Elasticsearch), 4 language bindings (Python, Java, TypeScript, C#), a CLI with interactive REPL, and
  a C ABI / FFI layer for cross-language consumption.

  ---
  2. Component Map

  ┌─────────────────────────────────────────────────────────┐
  │                     Consumers                           │
  │  CLI   Python   Java   TypeScript   C#   FFI/C-ABI     │
  └────────────────────┬────────────────────────────────────┘
                       │ JSON OQLQuery
                       ▼
  ┌─────────────────────────────────────────────────────────┐
  │                   pkg/core/engine.go                    │
  │  ┌──────────┐  ┌──────────────┐  ┌───────────────────┐ │
  │  │  Router   │  │ SchemaRegistry│  │  Auto-Schema      │ │
  │  │ (target→  │  │  (validate)  │  │  (EnsureTarget)   │ │
  │  │  driver)  │  │              │  │                   │ │
  │  └─────┬────┘  └──────┬───────┘  └────────┬──────────┘ │
  │        │              │                    │            │
  │        ▼              ▼                    ▼            │
  │  ┌─────────────────────────────────────────────────┐   │
  │  │            Driver.Execute(ctx, OQLQuery)         │   │
  │  │            Driver.BatchInsert(ctx, ...)          │   │
  │  └─────────────────────┬───────────────────────────┘   │
  │                        │                               │
  │  ┌─────────────────────▼───────────────────────────┐   │
  │  │              OmniJSON Response                    │   │
  │  │  { data: [...], meta: {...}, error?: {...} }      │   │
  │  └──────────────────────────────────────────────────┘   │
  └─────────────────────────────────────────────────────────┘
                           │
       ┌───────────┬───────┼────────┬───────────┬──────────┬──────────┐
       ▼           ▼       ▼        ▼           ▼          ▼          ▼
    SQLite     Postgres  Mongo    MySQL     SQLServer    Redis   Elasticsearch
    (CGO)      (lib/pq)  (BSON)  (mysql)   (go-mssql)  (go-redis) (REST/DSL)
       │           │                │           │
       └─────────┬─┘                └─────┬─────┘
                 ▼                        ▼
           sqlutil.BuildWhere()     sqlutil.BuildWhere()
           (shared WHERE builder)

  ---
  3. Core Design Decisions (and Their Consequences)

  ┌────────────────────────────────────────────┬──────────────────────────────────────────┬─────────────────────────────────────────────────────────────────────────────────┐
  │                  Decision                  │                Rationale                 │                                   Consequence                                   │
  ├────────────────────────────────────────────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────┤
  │ Execute() never returns Go error           │ Callers always get valid JSON            │ Signature (*OmniJSON, error) is misleading — error is always nil                │
  ├────────────────────────────────────────────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────┤
  │ MongoDB-style filter operators             │ Familiar to many devs, maps 1:1 to Mongo │ SQL drivers must recursively translate $or/$and — complex code                  │
  ├────────────────────────────────────────────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────┤
  │ First-registered driver = default fallback │ Deterministic without config             │ Implicit behavior; not configurable separately                                  │
  ├────────────────────────────────────────────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────┤
  │ map[string]int for Sort                    │ Simple JSON                              │ Go map iteration is non-deterministic — multi-field sort order is unpredictable │
  ├────────────────────────────────────────────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────┤
  │ No ID normalization                        │ Each driver is autonomous                │ _id vs id inconsistency breaks "write-once, run-anywhere"                       │
  ├────────────────────────────────────────────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────┤
  │ Auto-schema via SchemaAwareDriver          │ Zero-config table creation               │ ALTER TABLE on every INSERT if columns don't match — performance risk           │
  ├────────────────────────────────────────────┼──────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────┤
  │ sync.RWMutex on Engine                     │ Concurrent Execute() safe                │ SchemaRegistry itself has no locks — registration during execution is racy      │
  └────────────────────────────────────────────┴──────────────────────────────────────────┴─────────────────────────────────────────────────────────────────────────────────┘

  ---
  4. Identified Problems

  P1: Sort Order Non-Determinism (Critical)

  QueryOptions.Sort is map[string]int. Go maps have random iteration order.

  // oql.go:25
  Sort   map[string]int         `json:"sort,omitempty"`

  If a query sorts by {"price": -1, "name": 1}, the SQL might be ORDER BY price DESC, name ASC or ORDER BY name ASC, price DESC depending on the run. This is a correctness bug.

  Fix: Change Sort to an ordered type:
  type SortField struct {
      Field string `json:"field"`
      Order int    `json:"order"` // 1=ASC, -1=DESC
  }
  // Sort []SortField in QueryOptions
  This is a breaking API change — needs a major version bump or a migration period with both formats.

  ---
  P2: ID Field Inconsistency Across Drivers (Major)

  ┌───────────────┬──────────────────────────┬────────────────────────┐
  │    Driver     │      INSERT returns      │       Field name       │
  ├───────────────┼──────────────────────────┼────────────────────────┤
  │ SQLite        │ {"id": 5}                │ id (hardcoded)         │
  ├───────────────┼──────────────────────────┼────────────────────────┤
  │ MySQL         │ {"id": 5}                │ id (hardcoded)         │
  ├───────────────┼──────────────────────────┼────────────────────────┤
  │ SQL Server    │ {"id": 0}                │ id (may fail)          │
  ├───────────────┼──────────────────────────┼────────────────────────┤
  │ PostgreSQL    │ full row via RETURNING * │ whatever the column is │
  ├───────────────┼──────────────────────────┼────────────────────────┤
  │ MongoDB       │ {"_id": ObjectID(...)}   │ _id                    │
  ├───────────────┼──────────────────────────┼────────────────────────┤
  │ Elasticsearch │ {"_id": "abc123"}        │ _id                    │
  ├───────────────┼──────────────────────────┼────────────────────────┤
  │ Redis         │ {"id": val}              │ id                     │
  └───────────────┴──────────────────────────┴────────────────────────┘

  A consumer switching routes from Mongo to Postgres sees completely different response shapes. Filters using _id won't work on SQL drivers, and vice versa. There's no normalization
   layer in engine.go — driver results pass straight through.

  Fix: Add an ID normalization option at the engine level:
  - Define a canonical ID field (e.g., _id or configurable per-route)
  - Engine post-processes INSERT responses to normalize the field name
  - Provide a TranslateFields map in route config for aliasing

  ---
  P3: No Engine.Close() Lifecycle Method (Major)

  The engine holds references to all registered drivers but has no way to close them all. Callers must track and close each driver individually. This leads to resource leaks in
  practice.

  // Currently:
  drv1 := sqlite.New(":memory:")
  drv2 := postgres.New(db)
  engine.RegisterDriver(drv1)
  engine.RegisterDriver(drv2)
  // ... later, caller must remember to close drv1 and drv2 separately

  Fix: Add Engine.Close() that iterates all drivers and calls Close() on each.

  ---
  P4: Massive Code Duplication Across SQL Drivers (Design Debt)

  SQLite, PostgreSQL, MySQL, and SQL Server share nearly identical implementations for:

  - scanRows() — converting sql.Rows to []map[string]interface{}
  - fieldProjectionValue() — validating projection values
  - ensureColumns() / loadTableInfo() — dynamic schema management
  - Action dispatch pattern (Execute switch → find/insert/update/delete)
  - Deterministic column sorting for BatchInsert

  Only the quoting style, placeholder format, JSON path syntax, and a few SQL dialect quirks differ. This is ~400-500 lines duplicated four times.

  Fix: Extract a sqldriver base package:
  type SQLDriver struct {
      db           *sql.DB
      placeholder  sqlutil.PlaceholderFn
      quoteIdent   func(string) string
      translateCol func(string) string
      dialect      Dialect  // sqlite, postgres, mysql, sqlserver
  }
  Each driver embeds SQLDriver and overrides only what's unique (JSON path syntax, pagination quirks, etc.).

  ---
  P5: Redis and Elasticsearch Have Severely Limited Query Support (Design)

  Redis:
  - FIND without an id filter does SCAN * + loads ALL values into memory + in-memory filtering
  - Only supports = and $eq operators in filters
  - Sorting uses bubble sort in-memory
  - No projection support

  Elasticsearch:
  - Full Query DSL support (good)
  - But ListTargets() returns error (not implemented)

  Redis is essentially a KV store being forced into a query engine role. For any non-trivial filter, it scans the entire keyspace. This works for tiny datasets but is a performance
  trap.

  Fix: Either:
  1. Document the limitations clearly (Redis is ID-lookup-only; treat it as a cache layer)
  2. Add Redis secondary indexes (Sorted Sets for range queries, Sets for membership)
  3. Use RediSearch module for full query support

  ---
  P6: SchemaRegistry Is Not Thread-Safe (Concurrency Bug)

  engine.go protects driver/route maps with sync.RWMutex, but SchemaRegistry (schema.go) has no locks:

  type SchemaRegistry struct {
      schemas map[string]CollectionSchema  // unprotected
  }

  If RegisterSchema() is called concurrently with Execute() (which reads schemas for validation), there's a data race.

  Fix: Add sync.RWMutex to SchemaRegistry, or document that schemas must be registered before concurrent execution begins.

  ---
  P7: No Transaction Support in Driver Interface (Architectural Gap)

  The Driver interface has no concept of transactions:

  type Driver interface {
      Execute(ctx, OQLQuery) ([]map[string]interface{}, int64, error)
      BatchInsert(ctx, target, docs) ([]map[string]interface{}, error)
      // No BeginTx, Commit, Rollback
  }

  There's no way to do atomic multi-step operations (e.g., "debit account A and credit account B"). Each Execute call is independent.

  Fix: Add an optional TransactionalDriver interface:
  type TransactionalDriver interface {
      Driver
      BeginTx(ctx context.Context) (Tx, error)
  }
  type Tx interface {
      Execute(ctx context.Context, query OQLQuery) ([]map[string]interface{}, int64, error)
      Commit() error
      Rollback() error
  }

  ---
  P8: BatchInsert Response Inconsistency (API)

  ┌─────────────────┬─────────────────────────────────────────────┐
  │     Driver      │             BatchInsert returns             │
  ├─────────────────┼─────────────────────────────────────────────┤
  │ MongoDB         │ [{"_id": id1}, {"_id": id2}, ...] — all IDs │
  ├─────────────────┼─────────────────────────────────────────────┤
  │ All SQL drivers │ [{"count": N}] — just a count               │
  ├─────────────────┼─────────────────────────────────────────────┤
  │ Redis           │ [{"count": N}]                              │
  ├─────────────────┼─────────────────────────────────────────────┤
  │ Elasticsearch   │ [{"count": N}]                              │
  └─────────────────┴─────────────────────────────────────────────┘

  A consumer relying on inserted IDs works with MongoDB but silently loses that data on SQL. This is undocumented and surprising.

  Fix: Standardize: all drivers should return inserted IDs when possible. SQL drivers can use RETURNING (Postgres) or LastInsertId iteration. If a driver can't, document it
  explicitly.

  ---
  P9: No Default Query Limits (Safety)

  If Limit is 0, drivers treat it as "no limit." A FIND on a table with 10 million rows will try to load everything into memory as []map[string]interface{}.

  // oql.go:23
  Limit  int  `json:"limit,omitempty"`  // 0 = no limit

  Fix: Add a configurable MaxResultSize engine option (default 10,000) that caps results. Callers can override per-query with explicit Limit.

  ---
  P10: Compliance Suite Doesn't Cover All Drivers (Testing Gap)

  The compliance suite (50+ canonical tests) runs on:
  - SQLite (always)
  - PostgreSQL (with env var)
  - MySQL (with env var)

  Missing entirely: SQL Server, Redis, Elasticsearch, MongoDB (partially). These drivers are released but not validated against the canonical test cases. Behavioral drift is
  invisible.

  Fix: Add all 7 drivers to the compliance suite. For Redis/ES, mark known limitation tests as t.Skip("Redis does not support $lt operator") rather than omitting them.

  ---
  P11: Migration Has No Error Recovery (Reliability)

  Migrate() continues on error and increments a counter:

  // If batch fails, just count the error and continue
  result.Errors++

  There's no way to:
  - Know which records failed
  - Resume a failed migration from where it stopped
  - Roll back a partial migration
  - Get detailed error messages

  Fix: Add structured error reporting:
  type MigrateResult struct {
      RecordsRead    int64
      RecordsWritten int64
      Errors         []MigrateError  // not just a count
      LastOffset     int64           // for resume
  }
  type MigrateError struct {
      Offset  int64
      Message string
  }

  ---
  P12: FFI Memory Management Burden (Usability)

  Every FFI call that returns a *C.char requires the caller to call OmniQL_Free():

  char* result = OmniQL_Execute(handle, query);
  // use result...
  OmniQL_Free(result);  // caller MUST do this

  If any binding forgets to free, memory leaks. The Python binding uses threading.Lock() but relies on manual free. The TypeScript binding loads the native library at import time —
  a failure crashes the entire module.

  Fix: Consider arena-based allocation or auto-free wrappers in each binding. Python's CFFI gc callback can auto-free. Java's try-with-resources pattern should be used consistently.

  ---
  P13: Async Bindings Use Thread Pool, Not True Async I/O (Performance)

  All "async" methods in Python, C#, and TypeScript use run_in_executor / Task.Run / thread pool:

  # Python
  async def execute(self, query):
      loop = asyncio.get_event_loop()
      return await loop.run_in_executor(None, self.execute_sync, query)

  This blocks a thread per call. Under high concurrency, this exhausts the thread pool. It's not a coroutine-native solution.

  Fix: For Python, consider cffi release-GIL mode. For TypeScript, use N-API async workers. For C#, use TaskCompletionSource with callback-based FFI.

  ---
  5. Architectural Strengths

  Despite the problems, OmniQL has solid foundations:

  1. Clean Driver interface — 6 methods, clear contract, any database can be adapted
  2. Deterministic routing — explicit routes checked first, then first-registered fallback (no map randomness)
  3. All-JSON error handling — Execute() never returns Go errors; callers always get parseable JSON
  4. Shared sqlutil.BuildWhere — one parameterized WHERE builder for all SQL drivers, preventing SQL injection
  5. Concurrency-safe engine — sync.RWMutex protects registration/routing during concurrent execution
  6. Compliance suite — canonical test cases enforce behavioral parity across drivers
  7. Auto-schema with SchemaAwareDriver — zero-config table creation for rapid prototyping
  8. Cross-driver migration — built-in Migrate/MigrateAll with batching and dry-run
  9. Polyglot bindings — consistent API across Python, Java, TypeScript, C# with convenience methods
  10. Fuzz testing — crash resilience validated via native Go fuzzer

  ---
  6. Priority Recommendations

  ┌──────────┬────────────────────────────────────────┬─────────────────────┬───────────────────────────────────┐
  │ Priority │                Problem                 │       Effort        │              Impact               │
  ├──────────┼────────────────────────────────────────┼─────────────────────┼───────────────────────────────────┤
  │ P0       │ Sort order non-determinism (P1)        │ Small (type change) │ Correctness bug                   │
  ├──────────┼────────────────────────────────────────┼─────────────────────┼───────────────────────────────────┤
  │ P0       │ SchemaRegistry thread safety (P6)      │ Tiny (add mutex)    │ Race condition                    │
  ├──────────┼────────────────────────────────────────┼─────────────────────┼───────────────────────────────────┤
  │ P1       │ ID field normalization (P2)            │ Medium              │ "Write-once, run-anywhere" broken │
  ├──────────┼────────────────────────────────────────┼─────────────────────┼───────────────────────────────────┤
  │ P1       │ Default query limits (P9)              │ Small               │ OOM risk in production            │
  ├──────────┼────────────────────────────────────────┼─────────────────────┼───────────────────────────────────┤
  │ P1       │ Engine.Close() lifecycle (P3)          │ Small               │ Resource leaks                    │
  ├──────────┼────────────────────────────────────────┼─────────────────────┼───────────────────────────────────┤
  │ P2       │ SQL driver deduplication (P4)          │ Large               │ ~2000 lines of debt               │
  ├──────────┼────────────────────────────────────────┼─────────────────────┼───────────────────────────────────┤
  │ P2       │ Compliance suite for all drivers (P10) │ Medium              │ Untested drivers in production    │
  ├──────────┼────────────────────────────────────────┼─────────────────────┼───────────────────────────────────┤
  │ P2       │ BatchInsert response consistency (P8)  │ Medium              │ API inconsistency                 │
  ├──────────┼────────────────────────────────────────┼─────────────────────┼───────────────────────────────────┤
  │ P3       │ Transaction support (P7)               │ Large               │ Architectural gap                 │
  ├──────────┼────────────────────────────────────────┼─────────────────────┼───────────────────────────────────┤
  │ P3       │ Migration error recovery (P11)         │ Medium              │ Reliability                       │
  ├──────────┼────────────────────────────────────────┼─────────────────────┼───────────────────────────────────┤
  │ P3       │ Redis query limitations (P5)           │ Large               │ Performance trap                  │
  ├──────────┼────────────────────────────────────────┼─────────────────────┼───────────────────────────────────┤
  │ P4       │ True async bindings (P13)              │ Large               │ Performance under concurrency     │
  ├──────────┼────────────────────────────────────────┼─────────────────────┼───────────────────────────────────┤
  │ P4       │ FFI memory safety (P12)                │ Medium              │ Memory leaks                      │
  └──────────┴────────────────────────────────────────┴─────────────────────┴───────────────────────────────────┘

  The P0 items (sort determinism and schema registry thread safety) are bugs that should be fixed immediately. The P1 items (ID normalization, default limits, Engine.Close) are the
  highest-impact improvements for users adopting OmniQL across multiple databases.