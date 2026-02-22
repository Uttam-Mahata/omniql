# OmniQL for .NET

A standardized, multi-database query language (OQL) for .NET. OmniQL abstracts away database-specific complexities, allowing you to use a single query format for PostgreSQL, MongoDB, and SQLite.

## Installation

```bash
dotnet add package OmniQL
```

## Usage

```csharp
using OmniQL;

using var engine = new OmniEngine();

// 1. Register a driver
string driverName = engine.RegisterSQLiteDriver(":memory:");

// 2. Route a collection/table target to the driver
engine.Route("users", driverName);

// 3. Execute a query using the fluent builder
var result = await engine.Table("users")
    .Find(new Dictionary<string, object> { { "status", "active" } })
    .Limit(10)
    .ExecuteAsync();

foreach (var row in result.Data)
{
    Console.WriteLine(row["username"]);
}
```

## Supported Platforms

- Windows (x64)
- Linux (x64)
- macOS (x64)

The package includes pre-compiled native binaries for the OmniQL Core Engine.
