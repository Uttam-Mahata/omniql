OmniQL is designed to be a **multi-model abstraction layer**, meaning it isn't limited to just traditional tables. By **v1.0.0**, it is slated to support nearly every major category of data storage.

Here is the breakdown of the database types OmniQL supports or has on the roadmap:

### 1. Relational (SQL)

These are your standard structured databases. OmniQL translates OQL into standard SQL queries.

* **Current (v0.3.0):** PostgreSQL, SQLite.
* **Roadmap:** MySQL, MariaDB, SQL Server, **BigQuery**, and Oracle.

### 2. Document (NoSQL)

These store data as JSON-like documents. OQL is natively very similar to these formats.

* **Current (v0.3.0):** MongoDB.
* **Roadmap:** CouchDB, DynamoDB, and Firestore.

### 3. Key-Value & Cache

Used for high-speed data retrieval. OmniQL maps `FIND/INSERT` to `GET/SET` operations.

* **Roadmap:** Redis, Dragonfly, Memcached, and Valkey.

### 4. Search & Vector Engines

These are critical for AI and full-text search. OmniQL will handle the complex filtering and similarity scores.

* **Roadmap:** Elasticsearch, Meilisearch, Pinecone, and Weaviate.

### 5. Time-Series & Graph

Specialized databases for sequential data or relationship mapping.

* **Roadmap:** InfluxDB, TimescaleDB, Neo4j, and SurrealDB.

---

### Summary Table

| Category | Storage Style | Primary Use Case |
| --- | --- | --- |
| **Relational** | Tables/Rows | Structured business data, finance. |
| **Document** | JSON/BSON | Content management, flexible schemas. |
| **Key-Value** | Simple Pairs | Caching, session management. |
| **Search/Vector** | Indices/Embeddings | AI Agents, RAG, full-text search. |
| **Time-Series** | Metrics/Logs | IoT data, monitoring, stock prices. |
| **Graph** | Nodes/Edges | Social networks, fraud detection. |

---

### Why this breadth matters

By supporting all these types, OmniQL ensures that an **AI Agent** or a **cross-platform app** doesn't need to know the underlying tech. It treats "Project Data" the same whether it's a row in Postgres or a document in MongoDB.

