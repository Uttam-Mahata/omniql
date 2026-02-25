/**
 * OmniQL Java Binding
 *
 * This JAR wraps the native OmniQL shared library (libomniql.so / omniql.dll)
 * via JNI and exposes a fluent Java API.
 *
 * Build requirements:
 *   - The native library must be built first:
 *       go build -buildmode=c-shared -o libomniql.so ../../pkg/ffi
 *   - Place libomniql.so (or the platform equivalent) on java.library.path.
 *
 * Example usage:
 *   OmniEngine engine = OmniEngine.create();
 *
 *   // 1. Register a driver and route a target to it.
 *   String driverName = engine.registerSQLiteDriver(":memory:");
 *   engine.route("users", driverName);
 *
 *   // 2. Execute a query.
 *   OmniResult result = engine.table("users")
 *       .find(Map.of("status", "active"))
 *       .limit(10)
 *       .execute();
 */
package io.omniql;

import com.google.gson.Gson;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * OmniEngine is the main entry-point for the OmniQL Java binding.
 * It holds a handle to a native Core Engine and routes OQL queries through it.
 */
public class OmniEngine implements AutoCloseable {

    static {
        NativeLoader.load();
    }

    // -------------------------------------------------------------------------
    // Native method declarations
    // -------------------------------------------------------------------------

    private static native int  nativeNewEngine();
    private static native void nativeFreeEngine(int handle);
    private static native String nativeExecute(int handle, String queryJson);
    private static native String nativeRegisterSchema(int handle, String schemaJson);
    private static native String nativeRoute(int handle, String target, String driverName);
    private static native String nativeRegisterSQLiteDriver(int handle, String dsn);
    private static native String nativeRegisterPostgresDriver(int handle, String connStr);
    private static native String nativeRegisterMongoDriver(int handle, String uri, String dbName);

    // -------------------------------------------------------------------------
    // Java API
    // -------------------------------------------------------------------------

    private final int handle;
    private final Gson gson = new Gson();

    private OmniEngine(int handle) {
        this.handle = handle;
    }

    /** Creates a new OmniEngine backed by a native Core Engine. */
    public static OmniEngine create() {
        return new OmniEngine(nativeNewEngine());
    }

    /**
     * Executes a raw OQL query and returns the OmniJSON result.
     *
     * @param query the OQL query object
     * @return OmniResult wrapping the JSON response
     */
    public OmniResult execute(OQLQuery query) {
        String json = gson.toJson(query);
        String responseJson = nativeExecute(handle, json);
        return gson.fromJson(responseJson, OmniResult.class);
    }

    /**
     * Registers a collection schema with the engine.
     *
     * @param schema the collection schema to register
     */
    public void registerSchema(CollectionSchema schema) {
        String json = gson.toJson(schema);
        nativeRegisterSchema(handle, json);
    }

    /** Begins a fluent query builder targeting the given table/collection. */
    public QueryBuilder table(String target) {
        return new QueryBuilder(this, target);
    }

    /**
     * Binds a collection/table target to a specific driver name.
     * Must be called after registering a driver.
     *
     * @param target     the collection or table name
     * @param driverName the driver name returned by a register*Driver call
     */
    public void route(String target, String driverName) {
        nativeRoute(handle, target, driverName);
    }

    /**
     * Registers a SQLite driver using the given DSN (file path or ":memory:").
     *
     * @param dsn the SQLite file path or ":memory:"
     * @return the driver name ("sqlite") to use with {@link #route}
     */
    public String registerSQLiteDriver(String dsn) {
        String json = nativeRegisterSQLiteDriver(handle, dsn);
        return extractDriverName(json, "sqlite");
    }

    /**
     * Registers a PostgreSQL driver using the given connection string.
     * e.g. "host=localhost user=pg password=pg dbname=mydb sslmode=disable"
     *
     * @param connStr the Postgres connection string
     * @return the driver name ("postgres") to use with {@link #route}
     */
    public String registerPostgresDriver(String connStr) {
        String json = nativeRegisterPostgresDriver(handle, connStr);
        return extractDriverName(json, "postgres");
    }

    /**
     * Registers a MongoDB driver using the given URI and database name.
     * e.g. uri = "mongodb://localhost:27017", dbName = "mydb"
     *
     * @param uri    the MongoDB connection URI
     * @param dbName the database name to use
     * @return the driver name ("mongo") to use with {@link #route}
     */
    public String registerMongoDriver(String uri, String dbName) {
        String json = nativeRegisterMongoDriver(handle, uri, dbName);
        return extractDriverName(json, "mongo");
    }

    /**
     * Inserts multiple documents in a single BATCH_INSERT operation.
     *
     * @param target the collection or table name
     * @param docs   the list of documents to insert
     * @return OmniResult with the insertion outcome
     */
    public OmniResult batchInsert(String target, List<Map<String, Object>> docs) {
        Map<String, Object> queryMap = new HashMap<>();
        queryMap.put("target", target);
        queryMap.put("action", "BATCH_INSERT");
        queryMap.put("documents", docs);
        String json = gson.toJson(queryMap);
        String responseJson = nativeExecute(handle, json);
        return gson.fromJson(responseJson, OmniResult.class);
    }

    private String extractDriverName(String json, String fallback) {
        // Simple extraction without a full JSON parse dependency.
        // json is of the form {"driver":"sqlite"} or a JSON error.
        if (json != null && json.contains("\"driver\"")) {
            int start = json.indexOf("\"driver\"") + 9; // skip "driver":
            start = json.indexOf('"', start) + 1;
            int end = json.indexOf('"', start);
            if (start > 0 && end > start) {
                return json.substring(start, end);
            }
        }
        return fallback;
    }

    @Override
    public void close() {
        nativeFreeEngine(handle);
    }

    // -------------------------------------------------------------------------
    // Inner classes
    // -------------------------------------------------------------------------

    /** Fluent query builder. */
    public static class QueryBuilder {
        private final OmniEngine engine;
        private final OQLQuery query;

        QueryBuilder(OmniEngine engine, String target) {
            this.engine = engine;
            this.query  = new OQLQuery();
            this.query.target = target;
            this.query.action = "FIND";
            this.query.filter = new HashMap<>();
            this.query.options = new OQLQuery.Options();
        }

        public QueryBuilder find(Map<String, Object> filter) {
            this.query.action = "FIND";
            this.query.filter = filter;
            return this;
        }

        public QueryBuilder insert(Map<String, Object> document) {
            this.query.action   = "INSERT";
            this.query.document = document;
            return this;
        }

        public QueryBuilder update(Map<String, Object> document) {
            this.query.action   = "UPDATE";
            this.query.document = document;
            return this;
        }

        public QueryBuilder delete(Map<String, Object> filter) {
            this.query.action = "DELETE";
            this.query.filter = filter;
            return this;
        }

        public QueryBuilder count(Map<String, Object> filter) {
            this.query.action = "COUNT";
            this.query.filter = filter;
            return this;
        }

        public OmniResult batchInsert(List<Map<String, Object>> docs) {
            return engine.batchInsert(this.query.target, docs);
        }

        public QueryBuilder limit(int n) {
            this.query.options.limit = n;
            return this;
        }

        public QueryBuilder skip(int n) {
            this.query.options.skip = n;
            return this;
        }

        public QueryBuilder sort(Map<String, Integer> sort) {
            this.query.options.sort = sort;
            return this;
        }

        public QueryBuilder fields(Map<String, Object> fields) {
            this.query.options.fields = fields;
            return this;
        }

        public OmniResult execute() {
            return engine.execute(query);
        }
    }

    /** POJO for an OQL query. */
    public static class OQLQuery {
        public String target;
        public String action;
        public Map<String, Object> filter;
        public Map<String, Object> document;
        public Options options = new Options();

        public static class Options {
            public int limit;
            public int skip;
            public Map<String, Integer> sort;
            public Map<String, Object> fields;
        }
    }

    /** POJO for the OmniJSON response. */
    public static class OmniResult {
        public List<Map<String, Object>> data;
        public Meta meta;
        public OmniError error;

        public static class Meta {
            public long total;
            public int  returned;
            public String driver;
            public String target;
        }

        public static class OmniError {
            public String code;
            public String message;
        }
    }

    /** POJO for a collection schema. */
    public static class CollectionSchema {
        public String name;
        public Map<String, FieldSchema> fields;

        public static class FieldSchema {
            public String type;
            public boolean required;
        }
    }
}
