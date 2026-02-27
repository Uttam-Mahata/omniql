// Package ffi provides a C-compatible Foreign Function Interface that exports
// the OmniQL Core Engine to language bindings (Java via JNI, Python via CFFI,
// C# via P/Invoke, TypeScript via N-API).
//
// Build as a shared library with:
//
//	go build -buildmode=c-shared -o libomniql.so ./pkg/ffi
package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"unsafe"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	drvels "github.com/Uttam-Mahata/omniql/pkg/drivers/elasticsearch"
	drvmongo "github.com/Uttam-Mahata/omniql/pkg/drivers/mongo"
	drvmysql "github.com/Uttam-Mahata/omniql/pkg/drivers/mysql"
	drvpostgres "github.com/Uttam-Mahata/omniql/pkg/drivers/postgres"
	drvredis "github.com/Uttam-Mahata/omniql/pkg/drivers/redis"
	drvsqlite "github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite"
	drvsqlserver "github.com/Uttam-Mahata/omniql/pkg/drivers/sqlserver"
	_ "github.com/go-sql-driver/mysql"            // MySQL database/sql driver
	_ "github.com/lib/pq"                         // Postgres database/sql driver
	_ "github.com/microsoft/go-mssqldb"           // SQL Server database/sql driver
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// engineRegistry holds Engine instances keyed by an opaque integer handle so
// that multiple engines can coexist within the same process.
var (
	mu       sync.Mutex
	engines  = map[C.int]*core.Engine{}
	nextID   C.int
)

// OmniQL_NewEngine creates a new Core Engine and returns an opaque integer
// handle that must be passed to every subsequent API call.
//
//export OmniQL_NewEngine
func OmniQL_NewEngine() C.int {
	mu.Lock()
	defer mu.Unlock()
	nextID++
	engines[nextID] = core.NewEngine()
	return nextID
}

// OmniQL_FreeEngine releases the Engine identified by handle.
//
//export OmniQL_FreeEngine
func OmniQL_FreeEngine(handle C.int) {
	mu.Lock()
	defer mu.Unlock()
	delete(engines, handle)
}

// OmniQL_Execute receives a JSON-encoded OQLQuery, executes it through the
// engine identified by handle, and returns a JSON-encoded OmniJSON response.
//
// The caller is responsible for freeing the returned C string with
// OmniQL_Free.
//
//export OmniQL_Execute
func OmniQL_Execute(handle C.int, queryJSON *C.char) *C.char {
	engine := getEngine(handle)
	if engine == nil {
		return errorJSON("INVALID_HANDLE", "unknown engine handle")
	}

	var query core.OQLQuery
	if err := json.Unmarshal([]byte(C.GoString(queryJSON)), &query); err != nil {
		return errorJSON("PARSE_ERROR", err.Error())
	}

	result, _ := engine.Execute(context.Background(), query)

	data, err := json.Marshal(result)
	if err != nil {
		return errorJSON("MARSHAL_ERROR", err.Error())
	}
	return C.CString(string(data))
}

// OmniQL_Free releases a C string previously returned by OmniQL_Execute.
//
//export OmniQL_Free
func OmniQL_Free(ptr *C.char) {
	C.free(unsafe.Pointer(ptr))
}

// OmniQL_RegisterSchema registers a JSON-encoded CollectionSchema with the
// engine identified by handle.  Returns an empty string on success or a
// JSON-encoded error.
//
// The caller is responsible for freeing the returned C string with
// OmniQL_Free.
//
//export OmniQL_RegisterSchema
func OmniQL_RegisterSchema(handle C.int, schemaJSON *C.char) *C.char {
	engine := getEngine(handle)
	if engine == nil {
		return errorJSON("INVALID_HANDLE", "unknown engine handle")
	}

	var schema core.CollectionSchema
	if err := json.Unmarshal([]byte(C.GoString(schemaJSON)), &schema); err != nil {
		return errorJSON("PARSE_ERROR", err.Error())
	}
	engine.RegisterSchema(schema)
	return C.CString("")
}

// OmniQL_Route binds a collection/table target name to a driver name within
// the engine identified by handle.  Call this after registering a driver so
// the engine knows which driver to use for each target.
//
// Returns an empty C string on success or a JSON-encoded error on failure.
// The caller is responsible for freeing the returned C string with OmniQL_Free.
//
//export OmniQL_Route
func OmniQL_Route(handle C.int, target *C.char, driverName *C.char) *C.char {
	engine := getEngine(handle)
	if engine == nil {
		return errorJSON("INVALID_HANDLE", "unknown engine handle")
	}
	engine.Route(C.GoString(target), C.GoString(driverName))
	return C.CString("")
}

// OmniQL_RegisterSQLiteDriver creates a SQLite driver using the given DSN
// (file path or ":memory:"), registers it with the engine, and returns its
// driver name ("sqlite") in a JSON string so the caller can use it with
// OmniQL_Route.
//
// Returns JSON {"driver":"sqlite"} on success or a JSON-encoded error on failure.
// The caller is responsible for freeing the returned C string with OmniQL_Free.
//
//export OmniQL_RegisterSQLiteDriver
func OmniQL_RegisterSQLiteDriver(handle C.int, dsn *C.char) *C.char {
	engine := getEngine(handle)
	if engine == nil {
		return errorJSON("INVALID_HANDLE", "unknown engine handle")
	}
	drv, err := drvsqlite.New(C.GoString(dsn))
	if err != nil {
		return errorJSON("DRIVER_INIT_ERROR", fmt.Sprintf("sqlite: %v", err))
	}
	engine.RegisterDriver(drv)
	data, _ := json.Marshal(map[string]string{"driver": drv.Name()})
	return C.CString(string(data))
}

// OmniQL_RegisterPostgresDriver opens a Postgres connection using the
// provided connection string (e.g. "postgres://user:pass@host/db?sslmode=disable"),
// wraps it in the OmniQL Postgres driver, registers it with the engine, and
// returns {"driver":"postgres"} on success.
//
// Returns JSON {"driver":"postgres"} on success or a JSON-encoded error on failure.
// The caller is responsible for freeing the returned C string with OmniQL_Free.
//
//export OmniQL_RegisterPostgresDriver
func OmniQL_RegisterPostgresDriver(handle C.int, connStr *C.char) *C.char {
	engine := getEngine(handle)
	if engine == nil {
		return errorJSON("INVALID_HANDLE", "unknown engine handle")
	}
	db, err := sql.Open("postgres", C.GoString(connStr))
	if err != nil {
		return errorJSON("DRIVER_INIT_ERROR", fmt.Sprintf("postgres open: %v", err))
	}
	drv := drvpostgres.New(db)
	engine.RegisterDriver(drv)
	data, _ := json.Marshal(map[string]string{"driver": drv.Name()})
	return C.CString(string(data))
}

// OmniQL_RegisterMongoDriver connects to MongoDB using the provided URI
// (e.g. "mongodb://localhost:27017"), selects the given database name,
// registers the driver with the engine, and returns {"driver":"mongo"} on success.
//
// Returns JSON {"driver":"mongo"} on success or a JSON-encoded error on failure.
// The caller is responsible for freeing the returned C string with OmniQL_Free.
//
//export OmniQL_RegisterMongoDriver
func OmniQL_RegisterMongoDriver(handle C.int, uri *C.char, dbName *C.char) *C.char {
	engine := getEngine(handle)
	if engine == nil {
		return errorJSON("INVALID_HANDLE", "unknown engine handle")
	}
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(C.GoString(uri)))
	if err != nil {
		return errorJSON("DRIVER_INIT_ERROR", fmt.Sprintf("mongo connect: %v", err))
	}
	drv := drvmongo.New(client, C.GoString(dbName))
	engine.RegisterDriver(drv)
	data, _ := json.Marshal(map[string]string{"driver": drv.Name()})
	return C.CString(string(data))
}

// OmniQL_RegisterMySQLDriver opens a MySQL connection using the provided DSN
// (e.g. "user:password@tcp(host:port)/dbname?parseTime=true"), registers the
// driver with the engine, and returns {"driver":"mysql"} on success.
//
// Returns JSON {"driver":"mysql"} on success or a JSON-encoded error on failure.
// The caller is responsible for freeing the returned C string with OmniQL_Free.
//
//export OmniQL_RegisterMySQLDriver
func OmniQL_RegisterMySQLDriver(handle C.int, dsn *C.char) *C.char {
	engine := getEngine(handle)
	if engine == nil {
		return errorJSON("INVALID_HANDLE", "unknown engine handle")
	}
	drv, err := drvmysql.New(C.GoString(dsn))
	if err != nil {
		return errorJSON("DRIVER_INIT_ERROR", fmt.Sprintf("mysql: %v", err))
	}
	engine.RegisterDriver(drv)
	data, _ := json.Marshal(map[string]string{"driver": drv.Name()})
	return C.CString(string(data))
}

// getEngine retrieves the engine by handle (thread-safe).
func getEngine(handle C.int) *core.Engine {
	mu.Lock()
	defer mu.Unlock()
	return engines[handle]
}

// errorJSON serializes an error into a JSON OmniJSON response.
func errorJSON(code, message string) *C.char {
	resp := core.OmniJSON{
		Data: []map[string]interface{}{},
		Meta: core.OmniMeta{},
		Error: &core.OmniError{
			Code:    code,
			Message: message,
		},
	}
	data, _ := json.Marshal(resp)
	return C.CString(string(data))
}

// OmniQL_RegisterSQLServerDriver opens a SQL Server connection using the
// provided DSN (e.g. "sqlserver://sa:Pass@localhost:1433?database=test"),
// registers the driver with the engine, and returns {"driver":"sqlserver"} on success.
//
// Returns JSON {"driver":"sqlserver"} on success or a JSON-encoded error on failure.
// The caller is responsible for freeing the returned C string with OmniQL_Free.
//
//export OmniQL_RegisterSQLServerDriver
func OmniQL_RegisterSQLServerDriver(handle C.int, dsn *C.char) *C.char {
	engine := getEngine(handle)
	if engine == nil {
		return errorJSON("INVALID_HANDLE", "unknown engine handle")
	}
	drv, err := drvsqlserver.New(C.GoString(dsn))
	if err != nil {
		return errorJSON("DRIVER_INIT_ERROR", fmt.Sprintf("sqlserver: %v", err))
	}
	engine.RegisterDriver(drv)
	data, _ := json.Marshal(map[string]string{"driver": drv.Name()})
	return C.CString(string(data))
}

// OmniQL_RegisterRedisDriver creates a Redis driver using the provided URL
// (e.g. "redis://:password@host:6379/0"), registers it with the engine, and
// returns {"driver":"redis"} on success.
//
// Returns JSON {"driver":"redis"} on success or a JSON-encoded error on failure.
// The caller is responsible for freeing the returned C string with OmniQL_Free.
//
//export OmniQL_RegisterRedisDriver
func OmniQL_RegisterRedisDriver(handle C.int, url *C.char) *C.char {
	engine := getEngine(handle)
	if engine == nil {
		return errorJSON("INVALID_HANDLE", "unknown engine handle")
	}
	drv, err := drvredis.New(C.GoString(url))
	if err != nil {
		return errorJSON("DRIVER_INIT_ERROR", fmt.Sprintf("redis: %v", err))
	}
	engine.RegisterDriver(drv)
	data, _ := json.Marshal(map[string]string{"driver": drv.Name()})
	return C.CString(string(data))
}

// OmniQL_RegisterElasticsearchDriver creates an Elasticsearch driver using the
// provided address (e.g. "http://localhost:9200"), registers it with the engine,
// and returns {"driver":"elasticsearch"} on success.
//
// Returns JSON {"driver":"elasticsearch"} on success or a JSON-encoded error on failure.
// The caller is responsible for freeing the returned C string with OmniQL_Free.
//
//export OmniQL_RegisterElasticsearchDriver
func OmniQL_RegisterElasticsearchDriver(handle C.int, addr *C.char) *C.char {
	engine := getEngine(handle)
	if engine == nil {
		return errorJSON("INVALID_HANDLE", "unknown engine handle")
	}
	drv, err := drvels.New(C.GoString(addr))
	if err != nil {
		return errorJSON("DRIVER_INIT_ERROR", fmt.Sprintf("elasticsearch: %v", err))
	}
	engine.RegisterDriver(drv)
	data, _ := json.Marshal(map[string]string{"driver": drv.Name()})
	return C.CString(string(data))
}

// main is required for c-shared build mode.
func main() {}
