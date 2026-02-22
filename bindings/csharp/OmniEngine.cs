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
//   var result = await engine
//       .Table("users")
//       .Find(new Filter { { "status", new { _eq = "active" } } })
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

        [DllImport(LibName, EntryPoint = "OmniQL_Free")]
        internal static extern void Free(IntPtr ptr);
    }

    // -------------------------------------------------------------------------
    // Data transfer objects
    // -------------------------------------------------------------------------

    public class OQLQuery
    {
        public string Target   { get; set; } = string.Empty;
        public string Action   { get; set; } = "FIND";
        public Dictionary<string, object>? Filter   { get; set; }
        public Dictionary<string, object>? Document { get; set; }
        public OQLOptions Options { get; set; } = new();
    }

    public class OQLOptions
    {
        public int Limit { get; set; }
        public int Skip  { get; set; }
    }

    public class OmniResult
    {
        public List<Dictionary<string, JsonElement>> Data { get; set; } = new();
        public OmniMeta Meta  { get; set; } = new();
        public OmniError? Error { get; set; }
    }

    public class OmniMeta
    {
        public long   Total    { get; set; }
        public int    Returned { get; set; }
        public string Driver   { get; set; } = string.Empty;
        public string Target   { get; set; } = string.Empty;
    }

    public class OmniError
    {
        public string Code    { get; set; } = string.Empty;
        public string Message { get; set; } = string.Empty;
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

        public QueryBuilder Limit(int n)  { _query.Options.Limit = n; return this; }
        public QueryBuilder Skip(int n)   { _query.Options.Skip  = n; return this; }

        public Task<OmniResult> ExecuteAsync() => _engine.ExecuteAsync(_query);
    }
}
