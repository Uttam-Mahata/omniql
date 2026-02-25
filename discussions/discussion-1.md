
## 🛠️ How the Engine Works (v0.4.0 – v0.9.0)

In these versions, the **OmniQL Core Engine** evolves from a simple router into a sophisticated **Abstract Syntax Tree (AST) Transformer**.

### 1. The Multi-Model AST

The engine will maintain a central "OQL AST." When you send a JSON query, the engine performs the following:

* **Normalization:** It converts your OQL JSON into an internal Go struct tree.
* **Capability Check:** It checks if the target driver (e.g., Redis vs. Postgres) supports the requested action.
* **Dialect Translation:** The driver takes the "Clean AST" and translates it into the database's "Messy Dialect" (SQL strings, BSON, or Key-Value commands).

### 2. Standardizing "OmniJSON"

One of the hardest parts of this phase is ensuring the output is identical regardless of the source.

* **The Problem:** Postgres returns rows; MongoDB returns documents; Redis returns strings.
* **The Solution:** The `Normalization` step (Step 5 in your diagram) will use a **Schema Mapper**. It will force all outputs into a standard JSON array of objects, ensuring your Python/C# code doesn't have to change if you swap databases.

---

## 📅 Version-by-Version Implementation

| Version | Focus | Technical Goal |
| --- | --- | --- |
| **v0.4.0** | **Logical Complexity** | Add `$or`, `$and`, and `$not` to the Postgres and Mongo drivers. |
| **v0.5.0** | **Key-Value Support** | Add **Redis**. Map OQL `FIND` with a single ID filter to a Redis `GET`. |
| **v0.6.0** | **Relational Parity** | Add **MySQL and SQL Server**. Standardize how different SQL flavors handle "Limit/Offset." |
| **v0.7.0** | **Search & Full-Text** | Add **Elasticsearch/Meilisearch**. Map OQL filters to "Match" and "Fuzzy" queries. |
| **v0.8.0** | **Vector & AI Data** | Add **Vector DBs** (Pinecone/Weaviate). Support a new `$near` operator in OQL. |
| **v0.9.0** | **The Final Polish** | Implement **Field Projection** (selecting only certain columns) and **Sorting** across all 10+ drivers. |

---

## 🚀 How This Helps You

By the time you hit **v0.9.0**, you have achieved **"The Great Decoupling"**:

1. **Developer Freedom:** A developer can write a query once in Python. If the company moves from an on-premise Postgres to a cloud-native DynamoDB, the **application code remains exactly the same.**
2. **Platform Stability:** The FFI layer will be battle-tested against 15+ different data types (Dates, UUIDs, JSONB, Vectors).
3. **The Perfect Setup for MCP:** Because the engine is now "feature complete," your **v1.0.0 MCP Server** will be the most powerful AI data tool on the market. It won't just talk to "some" databases; it will talk to **every** database.

## 💡 Summary of the Work

In this phase, you are moving away from writing **code** and moving toward writing **specs**. You are defining exactly how an "Insert" should behave across 20 different technologies.

