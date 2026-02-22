// Package ffi provides a C-compatible Foreign Function Interface that exports
// the OmniQL Core Engine to language bindings (Java via JNI, Python via CFFI,
// C# via P/Invoke, TypeScript via N-API).
//
// Build as a shared library with:
//
//	go build -buildmode=c-shared -o libomniql.so ./pkg/ffi
package ffi

/*
#include <stdlib.h>
*/
import "C"
import (
	"context"
	"encoding/json"
	"sync"
	"unsafe"

	"github.com/Uttam-Mahata/omniql/pkg/core"
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

	result, err := engine.Execute(context.Background(), query)
	if err != nil {
		return errorJSON("ENGINE_ERROR", err.Error())
	}

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

// main is required for c-shared build mode.
func main() {}
