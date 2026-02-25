// OmniQL C# Binding
// =================
//
// A NuGet package that uses P/Invoke to call the native OmniQL shared library
// and exposes a LINQ-style API for .NET applications.
//
// Build the native library first:
//   go build -buildmode=c-shared -o omniql.dll ../../pkg/ffi
//
// Example usage:
//   using OmniQL;
//
//   using var engine = new OmniEngine();
//
//   // 1. Register a driver (SQLite shown; use RegisterPostgresDriver or RegisterMongoDriver for others).
//   string driverName = engine.RegisterSQLiteDriver(":memory:");
//
//   // 2. Route a collection target to the driver.
//   engine.Route("users", driverName);
//
//   // 3. Execute a query using the fluent builder.
//   var result = await engine
//       .Table("users")
//       .Find(new Dictionary<string, object> { { "status", "active" } })
//       .Limit(10)
//       .ExecuteAsync();
//
//   foreach (var row in result.Data)
//       Console.WriteLine(row["name"]);

using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Threading.Tasks;

namespace OmniQL
{
    // -------------------------------------------------------------------------
    // Native interop layer
    // -------------------------------------------------------------------------

    internal static class Native
    {
        private const string LibName = "omniql";

        [DllImport(LibName, EntryPoint = "OmniQL_NewEngine")]
        internal static extern int NewEngine();

        [DllImport(LibName, EntryPoint = "OmniQL_FreeEngine")]
        internal static extern void FreeEngine(int handle);

        [DllImport(LibName, EntryPoint = "OmniQL_Execute", CharSet = CharSet.Ansi)]
        internal static extern IntPtr Execute(int handle, string queryJson);

        [DllImport(LibName, EntryPoint = "OmniQL_RegisterSchema", CharSet = CharSet.Ansi)]
        internal static extern IntPtr RegisterSchema(int handle, string schemaJson);

        [DllImport(LibName, EntryPoint = "OmniQL_Route", CharSet = CharSet.Ansi)]
        internal static extern IntPtr Route(int handle, string target, string driverName);

        [DllImport(LibName, EntryPoint = "OmniQL_RegisterSQLiteDriver", CharSet = CharSet.Ansi)]
        internal static extern IntPtr RegisterSQLiteDriver(int handle, string dsn);

        [DllImport(LibName, EntryPoint = "OmniQL_RegisterPostgresDriver", CharSet = CharSet.Ansi)]
        internal static extern IntPtr RegisterPostgresDriver(int handle, string connStr);

        [DllImport(LibName, EntryPoint = "OmniQL_RegisterMongoDriver", CharSet = CharSet.Ansi)]
        internal static extern IntPtr RegisterMongoDriver(int handle, string uri, string dbName);

        [DllImport(LibName, EntryPoint = "OmniQL_RegisterMySQLDriver", CharSet = CharSet.Ansi)]
        internal static extern IntPtr RegisterMySQLDriver(int handle, string dsn);

        [DllImport(LibName, EntryPoint = "OmniQL_RegisterSQLServerDriver", CharSet = CharSet.Ansi)]
        internal static extern IntPtr RegisterSQLServerDriver(int handle, string dsn);

        [DllImport(LibName, EntryPoint = "OmniQL_RegisterRedisDriver", CharSet = CharSet.Ansi)]
        internal static extern IntPtr RegisterRedisDriver(int handle, string url);

        [DllImport(LibName, EntryPoint = "OmniQL_RegisterElasticsearchDriver", CharSet = CharSet.Ansi)]
        internal static extern IntPtr RegisterElasticsearchDriver(int handle, string addr);

        [DllImport(LibName, EntryPoint = "OmniQL_Free")]
        internal static extern void Free(IntPtr ptr);
    }

    // -------------------------------------------------------------------------
    // Data transfer objects
    // -------------------------------------------------------------------------

    public class OQLQuery
    {
        [JsonPropertyName("target")]   public string Target   { get; set; } = string.Empty;
        [JsonPropertyName("action")]   public string Action   { get; set; } = "FIND";
        [JsonPropertyName("filter")]   public Dictionary<string, object>? Filter   { get; set; }
        [JsonPropertyName("document")] public Dictionary<string, object>? Document { get; set; }
        [JsonPropertyName("options")]  public OQLOptions Options { get; set; } = new();
    }

    public class OQLOptions
    {
        [JsonPropertyName("limit")] public int Limit { get; set; }
        [JsonPropertyName("skip")]  public int Skip  { get; set; }
        [JsonPropertyName("sort")]   public Dictionary<string, int>? Sort   { get; set; }
        [JsonPropertyName("fields")] public Dictionary<string, object>? Fields { get; set; }
    }

    public class OmniResult
    {
        [JsonPropertyName("data")]  public List<Dictionary<string, JsonElement>> Data { get; set; } = new();
        [JsonPropertyName("meta")]  public OmniMeta Meta  { get; set; } = new();
        [JsonPropertyName("error")] public OmniError? Error { get; set; }
    }

    public class OmniMeta
    {
        [JsonPropertyName("total")]    public long   Total    { get; set; }
        [JsonPropertyName("returned")] public int    Returned { get; set; }
        [JsonPropertyName("driver")]   public string Driver   { get; set; } = string.Empty;
        [JsonPropertyName("target")]   public string Target   { get; set; } = string.Empty;
    }

    public class OmniError
    {
        [JsonPropertyName("code")]    public string Code    { get; set; } = string.Empty;
        [JsonPropertyName("message")] public string Message { get; set; } = string.Empty;
    }

    // -------------------------------------------------------------------------
    // Main engine class
    // -------------------------------------------------------------------------

    /// <summary>
    /// OmniEngine wraps the native OmniQL Core Engine via P/Invoke and exposes
    /// a LINQ-friendly fluent API.
    /// </summary>
    public sealed class OmniEngine : IDisposable
    {
        private readonly int _handle;
        private bool _disposed;

        /// <summary>Creates a new OmniEngine backed by a native Core Engine.</summary>
        public OmniEngine()
        {
            _handle = Native.NewEngine();
        }

        /// <summary>Executes a raw OQL query asynchronously.</summary>
        public Task<OmniResult> ExecuteAsync(OQLQuery query)
        {
            return Task.Run(() =>
            {
                string queryJson = JsonSerializer.Serialize(query);
                IntPtr ptr = Native.Execute(_handle, queryJson);
                try
                {
                    string json = Marshal.PtrToStringAnsi(ptr) ?? "{}";
                    return JsonSerializer.Deserialize<OmniResult>(json) ?? new OmniResult();
                }
                finally
                {
                    Native.Free(ptr);
                }
            });
        }

        /// <summary>Registers a collection schema with the engine.</summary>
        public void RegisterSchema(object schema)
        {
            string json = JsonSerializer.Serialize(schema);
            IntPtr ptr = Native.RegisterSchema(_handle, json);
            Native.Free(ptr);
        }

        /// <summary>
        /// Registers the SQLite driver using the given DSN (file path or ":memory:").
        /// Returns the driver name ("sqlite") to use with <see cref="Route"/>.
        /// </summary>
        public string RegisterSQLiteDriver(string dsn)
        {
            IntPtr ptr = Native.RegisterSQLiteDriver(_handle, dsn);
            try
            {
                string json = Marshal.PtrToStringAnsi(ptr) ?? "{}";
                var result = JsonSerializer.Deserialize<Dictionary<string, string>>(json);
                return result != null && result.TryGetValue("driver", out var name) ? name : "sqlite";
            }
            finally { Native.Free(ptr); }
        }

        /// <summary>
        /// Registers the PostgreSQL driver using the given connection string
        /// (e.g. "host=localhost user=pg password=pg dbname=mydb sslmode=disable").
        /// Returns the driver name ("postgres") to use with <see cref="Route"/>.
        /// </summary>
        public string RegisterPostgresDriver(string connStr)
        {
            IntPtr ptr = Native.RegisterPostgresDriver(_handle, connStr);
            try
            {
                string json = Marshal.PtrToStringAnsi(ptr) ?? "{}";
                var result = JsonSerializer.Deserialize<Dictionary<string, string>>(json);
                return result != null && result.TryGetValue("driver", out var name) ? name : "postgres";
            }
            finally { Native.Free(ptr); }
        }

        /// <summary>
        /// Registers the MongoDB driver using the given URI and database name.
        /// Returns the driver name ("mongo") to use with <see cref="Route"/>.
        /// </summary>
        public string RegisterMongoDriver(string uri, string dbName)
        {
            IntPtr ptr = Native.RegisterMongoDriver(_handle, uri, dbName);
            try
            {
                string json = Marshal.PtrToStringAnsi(ptr) ?? "{}";
                var result = JsonSerializer.Deserialize<Dictionary<string, string>>(json);
                return result != null && result.TryGetValue("driver", out var name) ? name : "mongo";
            }
            finally { Native.Free(ptr); }
        }

        /// <summary>
        /// Registers the MySQL driver using the given DSN.
        /// Returns the driver name ("mysql") to use with <see cref="Route"/>.
        /// </summary>
        public string RegisterMySQLDriver(string dsn)
        {
            IntPtr ptr = Native.RegisterMySQLDriver(_handle, dsn);
            try
            {
                string json = Marshal.PtrToStringAnsi(ptr) ?? "{}";
                var result = JsonSerializer.Deserialize<Dictionary<string, string>>(json);
                return result != null && result.TryGetValue("driver", out var name) ? name : "mysql";
            }
            finally { Native.Free(ptr); }
        }

        /// <summary>
        /// Registers the SQL Server driver using the given DSN.
        /// Returns the driver name ("sqlserver") to use with <see cref="Route"/>.
        /// </summary>
        public string RegisterSQLServerDriver(string dsn)
        {
            IntPtr ptr = Native.RegisterSQLServerDriver(_handle, dsn);
            try
            {
                string json = Marshal.PtrToStringAnsi(ptr) ?? "{}";
                var result = JsonSerializer.Deserialize<Dictionary<string, string>>(json);
                return result != null && result.TryGetValue("driver", out var name) ? name : "sqlserver";
            }
            finally { Native.Free(ptr); }
        }

        /// <summary>
        /// Registers the Redis driver using the given URL.
        /// Returns the driver name ("redis") to use with <see cref="Route"/>.
        /// </summary>
        public string RegisterRedisDriver(string url)
        {
            IntPtr ptr = Native.RegisterRedisDriver(_handle, url);
            try
            {
                string json = Marshal.PtrToStringAnsi(ptr) ?? "{}";
                var result = JsonSerializer.Deserialize<Dictionary<string, string>>(json);
                return result != null && result.TryGetValue("driver", out var name) ? name : "redis";
            }
            finally { Native.Free(ptr); }
        }

        /// <summary>
        /// Registers the Elasticsearch driver using the given address.
        /// Returns the driver name ("elasticsearch") to use with <see cref="Route"/>.
        /// </summary>
        public string RegisterElasticsearchDriver(string addr)
        {
            IntPtr ptr = Native.RegisterElasticsearchDriver(_handle, addr);
            try
            {
                string json = Marshal.PtrToStringAnsi(ptr) ?? "{}";
                var result = JsonSerializer.Deserialize<Dictionary<string, string>>(json);
                return result != null && result.TryGetValue("driver", out var name) ? name : "elasticsearch";
            }
            finally { Native.Free(ptr); }
        }

        /// <summary>
        /// Binds a collection/table <paramref name="target"/> name to a <paramref name="driverName"/>
        /// so the engine routes queries for that target to the correct driver.
        /// </summary>
        public void Route(string target, string driverName)
        {
            IntPtr ptr = Native.Route(_handle, target, driverName);
            Native.Free(ptr);
        }

        /// <summary>
        /// Inserts multiple documents in a single BATCH_INSERT operation asynchronously.
        /// </summary>
        public Task<OmniResult> BatchInsertAsync(string target, IEnumerable<Dictionary<string, object>> docs)
        {
            return Task.Run(() =>
            {
                var queryMap = new Dictionary<string, object>
                {
                    ["target"]    = target,
                    ["action"]    = "BATCH_INSERT",
                    ["documents"] = docs,
                };
                string queryJson = JsonSerializer.Serialize(queryMap);
                IntPtr ptr = Native.Execute(_handle, queryJson);
                try
                {
                    string json = Marshal.PtrToStringAnsi(ptr) ?? "{}";
                    return JsonSerializer.Deserialize<OmniResult>(json) ?? new OmniResult();
                }
                finally
                {
                    Native.Free(ptr);
                }
            });
        }

        // -------------------------------------------------------------------------
        // Terminal convenience methods
        // -------------------------------------------------------------------------

        /// <summary>Returns all documents matching the given filter asynchronously.</summary>
        public Task<List<Dictionary<string, JsonElement>>> FindManyAsync(
            string target,
            Dictionary<string, object>? filter = null)
        {
            var q = new OQLQuery { Target = target, Action = "FIND", Filter = filter };
            return ExecuteAsync(q).ContinueWith(t => t.Result.Data);
        }

        /// <summary>Returns the first matching document asynchronously, or null.</summary>
        public Task<Dictionary<string, JsonElement>?> FindFirstAsync(
            string target,
            Dictionary<string, object>? filter = null)
        {
            var q = new OQLQuery { Target = target, Action = "FIND", Filter = filter, Options = new OQLOptions { Limit = 1 } };
            return ExecuteAsync(q).ContinueWith(t =>
                t.Result.Data.Count > 0 ? t.Result.Data[0] : (Dictionary<string, JsonElement>?)null);
        }

        /// <summary>Returns the count of matching documents asynchronously.</summary>
        public Task<long> CountAsync(
            string target,
            Dictionary<string, object>? filter = null)
        {
            var q = new OQLQuery { Target = target, Action = "COUNT", Filter = filter };
            return ExecuteAsync(q).ContinueWith(t => t.Result.Meta.Total);
        }

        /// <summary>Inserts a single document asynchronously.</summary>
        public Task<OmniResult> InsertOneAsync(string target, Dictionary<string, object> document)
        {
            var q = new OQLQuery { Target = target, Action = "INSERT", Document = document };
            return ExecuteAsync(q);
        }

        /// <summary>Updates all documents matching the filter asynchronously.</summary>
        public Task<OmniResult> UpdateManyAsync(
            string target,
            Dictionary<string, object> filter,
            Dictionary<string, object> update)
        {
            var q = new OQLQuery { Target = target, Action = "UPDATE", Filter = filter, Document = update };
            return ExecuteAsync(q);
        }

        /// <summary>Deletes all documents matching the filter asynchronously.</summary>
        public Task<OmniResult> DeleteManyAsync(
            string target,
            Dictionary<string, object> filter)
        {
            var q = new OQLQuery { Target = target, Action = "DELETE", Filter = filter };
            return ExecuteAsync(q);
        }

        /// <summary>Returns a fluent query builder for the given target.</summary>
        public QueryBuilder Table(string target) => new QueryBuilder(this, target);

        public void Dispose()
        {
            if (!_disposed)
            {
                Native.FreeEngine(_handle);
                _disposed = true;
            }
        }
    }

    // -------------------------------------------------------------------------
    // Fluent query builder
    // -------------------------------------------------------------------------

    /// <summary>Provides a LINQ-style fluent interface for building OQL queries.</summary>
    public sealed class QueryBuilder
    {
        private readonly OmniEngine _engine;
        private readonly OQLQuery   _query;

        internal QueryBuilder(OmniEngine engine, string target)
        {
            _engine = engine;
            _query  = new OQLQuery { Target = target };
        }

        public QueryBuilder Find(Dictionary<string, object> filter)
        {
            _query.Action = "FIND";
            _query.Filter = filter;
            return this;
        }

        public QueryBuilder Insert(Dictionary<string, object> document)
        {
            _query.Action   = "INSERT";
            _query.Document = document;
            return this;
        }

        public QueryBuilder Update(Dictionary<string, object> document)
        {
            _query.Action   = "UPDATE";
            _query.Document = document;
            return this;
        }

        public QueryBuilder Delete(Dictionary<string, object> filter)
        {
            _query.Action = "DELETE";
            _query.Filter = filter;
            return this;
        }

        public QueryBuilder Count(Dictionary<string, object>? filter = null)
        {
            _query.Action = "COUNT";
            _query.Filter = filter;
            return this;
        }

        public QueryBuilder Limit(int n)  { _query.Options.Limit = n; return this; }
        public QueryBuilder Skip(int n)   { _query.Options.Skip  = n; return this; }

        public QueryBuilder Sort(Dictionary<string, int> sort) { _query.Options.Sort = sort; return this; }
        public QueryBuilder Fields(Dictionary<string, object> fields) { _query.Options.Fields = fields; return this; }

        public Task<OmniResult> ExecuteAsync() => _engine.ExecuteAsync(_query);

        public Task<OmniResult> BatchInsertAsync(IEnumerable<Dictionary<string, object>> docs)
            => _engine.BatchInsertAsync(_query.Target, docs);
    }
}
