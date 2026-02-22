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
 *   OmniResult result = engine.table("users")
 *       .find(Filter.eq("status", "active"))
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
        System.loadLibrary("omniql");
    }

    // -------------------------------------------------------------------------
    // Native method declarations
    // -------------------------------------------------------------------------

    private static native int  nativeNewEngine();
    private static native void nativeFreeEngine(int handle);
    private static native String nativeExecute(int handle, String queryJson);
    private static native String nativeRegisterSchema(int handle, String schemaJson);

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

        public QueryBuilder limit(int n) {
            this.query.options.limit = n;
            return this;
        }

        public QueryBuilder skip(int n) {
            this.query.options.skip = n;
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
