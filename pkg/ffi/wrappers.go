package main

// This file provides Go-typed wrappers around the C-typed FFI export
// functions.  They are used internally by the test suite so that test files
// do not need to import "C" (which is forbidden in test files of packages that
// use //export).
//
// These wrappers are NOT part of the public C ABI; they exist solely to make
// the package testable from Go.

/*
#include <stdlib.h>
*/
import "C"
import (
	"unsafe"

	"github.com/Uttam-Mahata/omniql/pkg/core"
)

// goNewEngine wraps OmniQL_NewEngine and returns a plain Go int handle.
func goNewEngine() int {
	return int(OmniQL_NewEngine())
}

// goFreeEngine wraps OmniQL_FreeEngine.
func goFreeEngine(handle int) {
	OmniQL_FreeEngine(C.int(handle))
}

// goExecute wraps OmniQL_Execute: accepts and returns plain Go strings.
func goExecute(handle int, queryJSON string) string {
	cs := C.CString(queryJSON)
	defer C.free(unsafe.Pointer(cs))
	result := OmniQL_Execute(C.int(handle), cs)
	s := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return s
}

// goRegisterSchema wraps OmniQL_RegisterSchema.
func goRegisterSchema(handle int, schemaJSON string) string {
	cs := C.CString(schemaJSON)
	defer C.free(unsafe.Pointer(cs))
	result := OmniQL_RegisterSchema(C.int(handle), cs)
	s := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return s
}

// goRoute wraps OmniQL_Route.
func goRoute(handle int, target, driverName string) string {
	ct := C.CString(target)
	defer C.free(unsafe.Pointer(ct))
	cd := C.CString(driverName)
	defer C.free(unsafe.Pointer(cd))
	result := OmniQL_Route(C.int(handle), ct, cd)
	s := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return s
}

// goRegisterSQLiteDriver wraps OmniQL_RegisterSQLiteDriver.
func goRegisterSQLiteDriver(handle int, dsn string) string {
	cs := C.CString(dsn)
	defer C.free(unsafe.Pointer(cs))
	result := OmniQL_RegisterSQLiteDriver(C.int(handle), cs)
	s := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return s
}

// goRegisterMySQLDriver wraps OmniQL_RegisterMySQLDriver.
func goRegisterMySQLDriver(handle int, dsn string) string {
	cs := C.CString(dsn)
	defer C.free(unsafe.Pointer(cs))
	result := OmniQL_RegisterMySQLDriver(C.int(handle), cs)
	s := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return s
}

// goRegisterSQLServerDriver wraps OmniQL_RegisterSQLServerDriver.
func goRegisterSQLServerDriver(handle int, dsn string) string {
	cs := C.CString(dsn)
	defer C.free(unsafe.Pointer(cs))
	result := OmniQL_RegisterSQLServerDriver(C.int(handle), cs)
	s := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return s
}

// goRegisterRedisDriver wraps OmniQL_RegisterRedisDriver.
func goRegisterRedisDriver(handle int, url string) string {
	cs := C.CString(url)
	defer C.free(unsafe.Pointer(cs))
	result := OmniQL_RegisterRedisDriver(C.int(handle), cs)
	s := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return s
}

// goRegisterElasticsearchDriver wraps OmniQL_RegisterElasticsearchDriver.
func goRegisterElasticsearchDriver(handle int, addr string) string {
	cs := C.CString(addr)
	defer C.free(unsafe.Pointer(cs))
	result := OmniQL_RegisterElasticsearchDriver(C.int(handle), cs)
	s := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return s
}

// goGetEngine is a test-helper wrapper around getEngine that accepts a plain
// Go int rather than a C.int.
func goGetEngine(handle int) *core.Engine {
	return getEngine(C.int(handle))
}
