package main

/*
#cgo CFLAGS: -I/opt/java/jdk-25+36/include -I/opt/java/jdk-25+36/include/linux
#include <jni.h>
#include <stdlib.h>

// Helper wrappers to call JNIEnv function pointers from Go.
static jstring jni_new_string(JNIEnv *env, const char *str) {
    return (*env)->NewStringUTF(env, str);
}
static const char *jni_get_string(JNIEnv *env, jstring s) {
    return (*env)->GetStringUTFChars(env, s, NULL);
}
static void jni_release_string(JNIEnv *env, jstring s, const char *cs) {
    (*env)->ReleaseStringUTFChars(env, s, cs);
}
*/
import "C"
import "unsafe"

//export Java_io_omniql_OmniEngine_nativeNewEngine
func Java_io_omniql_OmniEngine_nativeNewEngine(env *C.JNIEnv, class C.jclass) C.jint {
	return C.jint(OmniQL_NewEngine())
}

//export Java_io_omniql_OmniEngine_nativeFreeEngine
func Java_io_omniql_OmniEngine_nativeFreeEngine(env *C.JNIEnv, class C.jclass, handle C.jint) {
	OmniQL_FreeEngine(C.int(handle))
}

//export Java_io_omniql_OmniEngine_nativeExecute
func Java_io_omniql_OmniEngine_nativeExecute(env *C.JNIEnv, class C.jclass, handle C.jint, queryJSON C.jstring) C.jstring {
	cQuery := C.jni_get_string(env, queryJSON)
	defer C.jni_release_string(env, queryJSON, cQuery)

	result := OmniQL_Execute(C.int(handle), cQuery)
	defer C.free(unsafe.Pointer(result))

	return C.jni_new_string(env, result)
}

//export Java_io_omniql_OmniEngine_nativeRegisterSchema
func Java_io_omniql_OmniEngine_nativeRegisterSchema(env *C.JNIEnv, class C.jclass, handle C.jint, schemaJSON C.jstring) C.jstring {
	cSchema := C.jni_get_string(env, schemaJSON)
	defer C.jni_release_string(env, schemaJSON, cSchema)

	result := OmniQL_RegisterSchema(C.int(handle), cSchema)
	defer C.free(unsafe.Pointer(result))

	return C.jni_new_string(env, result)
}

//export Java_io_omniql_OmniEngine_nativeRoute
func Java_io_omniql_OmniEngine_nativeRoute(env *C.JNIEnv, class C.jclass, handle C.jint, target C.jstring, driverName C.jstring) C.jstring {
	cTarget := C.jni_get_string(env, target)
	defer C.jni_release_string(env, target, cTarget)

	cDriver := C.jni_get_string(env, driverName)
	defer C.jni_release_string(env, driverName, cDriver)

	result := OmniQL_Route(C.int(handle), cTarget, cDriver)
	defer C.free(unsafe.Pointer(result))

	return C.jni_new_string(env, result)
}

//export Java_io_omniql_OmniEngine_nativeRegisterSQLiteDriver
func Java_io_omniql_OmniEngine_nativeRegisterSQLiteDriver(env *C.JNIEnv, class C.jclass, handle C.jint, dsn C.jstring) C.jstring {
	cDSN := C.jni_get_string(env, dsn)
	defer C.jni_release_string(env, dsn, cDSN)

	result := OmniQL_RegisterSQLiteDriver(C.int(handle), cDSN)
	defer C.free(unsafe.Pointer(result))

	return C.jni_new_string(env, result)
}

//export Java_io_omniql_OmniEngine_nativeRegisterPostgresDriver
func Java_io_omniql_OmniEngine_nativeRegisterPostgresDriver(env *C.JNIEnv, class C.jclass, handle C.jint, connStr C.jstring) C.jstring {
	cConn := C.jni_get_string(env, connStr)
	defer C.jni_release_string(env, connStr, cConn)

	result := OmniQL_RegisterPostgresDriver(C.int(handle), cConn)
	defer C.free(unsafe.Pointer(result))

	return C.jni_new_string(env, result)
}

//export Java_io_omniql_OmniEngine_nativeRegisterMongoDriver
func Java_io_omniql_OmniEngine_nativeRegisterMongoDriver(env *C.JNIEnv, class C.jclass, handle C.jint, uri C.jstring, dbName C.jstring) C.jstring {
	cURI := C.jni_get_string(env, uri)
	defer C.jni_release_string(env, uri, cURI)

	cDB := C.jni_get_string(env, dbName)
	defer C.jni_release_string(env, dbName, cDB)

	result := OmniQL_RegisterMongoDriver(C.int(handle), cURI, cDB)
	defer C.free(unsafe.Pointer(result))

	return C.jni_new_string(env, result)
}
