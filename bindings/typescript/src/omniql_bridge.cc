#include <napi.h>
#include <uv.h>
#include <dlfcn.h>
#include <signal.h>
#include <iostream>
#include <vector>

// Define function pointers for the Go exported functions
typedef int (*OmniQL_NewEngine_t)();
typedef void (*OmniQL_FreeEngine_t)(int);
typedef char* (*OmniQL_Execute_t)(int, char*);
typedef char* (*OmniQL_RegisterSchema_t)(int, char*);
typedef char* (*OmniQL_Route_t)(int, char*, char*);
typedef char* (*OmniQL_RegisterSQLiteDriver_t)(int, char*);
typedef char* (*OmniQL_RegisterPostgresDriver_t)(int, char*);
typedef char* (*OmniQL_RegisterMongoDriver_t)(int, char*, char*);
typedef void (*OmniQL_Free_t)(char*);

// Global function pointers
OmniQL_NewEngine_t ptr_NewEngine = nullptr;
OmniQL_FreeEngine_t ptr_FreeEngine = nullptr;
OmniQL_Execute_t ptr_Execute = nullptr;
OmniQL_RegisterSchema_t ptr_RegisterSchema = nullptr;
OmniQL_Route_t ptr_Route = nullptr;
OmniQL_RegisterSQLiteDriver_t ptr_RegisterSQLiteDriver = nullptr;
OmniQL_RegisterPostgresDriver_t ptr_RegisterPostgresDriver = nullptr;
OmniQL_RegisterMongoDriver_t ptr_RegisterMongoDriver = nullptr;
OmniQL_Free_t ptr_Free = nullptr;

void* lib_handle = nullptr;

// Helper to load the library and symbols
void LoadLib(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();
    
    if (info.Length() < 1 || !info[0].IsString()) {
        Napi::TypeError::New(env, "String expected for library path").ThrowAsJavaScriptException();
        return;
    }
    
    std::string libPath = info[0].As<Napi::String>().Utf8Value();

    // CRITICAL FIX: Ignore SIGURG before loading Go.
    // Go uses SIGURG for preemptive scheduling. Node.js/libuv doesn't like unexpected signals.
    signal(SIGURG, SIG_IGN);

    // Open the shared library
    // RTLD_GLOBAL is important if the Go runtime needs to see symbols from other loaded libs,
    // or if we load multiple plugins. RTLD_NOW ensures we fail fast if symbols are missing.
    lib_handle = dlopen(libPath.c_str(), RTLD_NOW | RTLD_GLOBAL);
    
    if (!lib_handle) {
        std::string err = "Failed to load library: " + std::string(dlerror());
        Napi::Error::New(env, err).ThrowAsJavaScriptException();
        return;
    }

    // Load symbols
    ptr_NewEngine = (OmniQL_NewEngine_t)dlsym(lib_handle, "OmniQL_NewEngine");
    ptr_FreeEngine = (OmniQL_FreeEngine_t)dlsym(lib_handle, "OmniQL_FreeEngine");
    ptr_Execute = (OmniQL_Execute_t)dlsym(lib_handle, "OmniQL_Execute");
    ptr_RegisterSchema = (OmniQL_RegisterSchema_t)dlsym(lib_handle, "OmniQL_RegisterSchema");
    ptr_Route = (OmniQL_Route_t)dlsym(lib_handle, "OmniQL_Route");
    ptr_RegisterSQLiteDriver = (OmniQL_RegisterSQLiteDriver_t)dlsym(lib_handle, "OmniQL_RegisterSQLiteDriver");
    ptr_RegisterPostgresDriver = (OmniQL_RegisterPostgresDriver_t)dlsym(lib_handle, "OmniQL_RegisterPostgresDriver");
    ptr_RegisterMongoDriver = (OmniQL_RegisterMongoDriver_t)dlsym(lib_handle, "OmniQL_RegisterMongoDriver");
    ptr_Free = (OmniQL_Free_t)dlsym(lib_handle, "OmniQL_Free");

    if (!ptr_NewEngine) {
        Napi::Error::New(env, "Failed to load symbols from library").ThrowAsJavaScriptException();
        return;
    }
}

Napi::Number NewEngine(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();
    if (!ptr_NewEngine) return Napi::Number::New(env, 0);
    return Napi::Number::New(env, ptr_NewEngine());
}

void FreeEngine(const Napi::CallbackInfo& info) {
    if (!ptr_FreeEngine) return;
    int handle = info[0].As<Napi::Number>().Int32Value();
    ptr_FreeEngine(handle);
}

Napi::String Execute(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();
    int handle = info[0].As<Napi::Number>().Int32Value();
    std::string query = info[1].As<Napi::String>().Utf8Value();
    
    // Go expects a non-const char* (though it treats it as const usually),
    // but our typedef says char*. We cast to be safe.
    char* res = ptr_Execute(handle, const_cast<char*>(query.c_str()));
    
    std::string resultStr = "{}";
    if (res) {
        resultStr = std::string(res);
        ptr_Free(res);
    }
    return Napi::String::New(env, resultStr);
}

Napi::String RegisterMongoDriver(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();
    int handle = info[0].As<Napi::Number>().Int32Value();
    std::string uri = info[1].As<Napi::String>().Utf8Value();
    std::string dbName = info[2].As<Napi::String>().Utf8Value();
    
    char* res = ptr_RegisterMongoDriver(handle, const_cast<char*>(uri.c_str()), const_cast<char*>(dbName.c_str()));
    
    std::string resultStr = "{}";
    if (res) {
        resultStr = std::string(res);
        ptr_Free(res);
    }
    return Napi::String::New(env, resultStr);
}

Napi::String RegisterSQLiteDriver(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();
    int handle = info[0].As<Napi::Number>().Int32Value();
    std::string dsn = info[1].As<Napi::String>().Utf8Value();
    
    char* res = ptr_RegisterSQLiteDriver(handle, const_cast<char*>(dsn.c_str()));
    
    std::string resultStr = "{}";
    if (res) {
        resultStr = std::string(res);
        ptr_Free(res);
    }
    return Napi::String::New(env, resultStr);
}

Napi::String RegisterPostgresDriver(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();
    int handle = info[0].As<Napi::Number>().Int32Value();
    std::string conn = info[1].As<Napi::String>().Utf8Value();
    
    char* res = ptr_RegisterPostgresDriver(handle, const_cast<char*>(conn.c_str()));
    
    std::string resultStr = "{}";
    if (res) {
        resultStr = std::string(res);
        ptr_Free(res);
    }
    return Napi::String::New(env, resultStr);
}

Napi::String RegisterSchema(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();
    int handle = info[0].As<Napi::Number>().Int32Value();
    std::string schema = info[1].As<Napi::String>().Utf8Value();
    
    char* res = ptr_RegisterSchema(handle, const_cast<char*>(schema.c_str()));
    
    std::string resultStr = "{}";
    if (res) {
        resultStr = std::string(res);
        ptr_Free(res);
    }
    return Napi::String::New(env, resultStr);
}

Napi::String Route(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();
    int handle = info[0].As<Napi::Number>().Int32Value();
    std::string target = info[1].As<Napi::String>().Utf8Value();
    std::string driver = info[2].As<Napi::String>().Utf8Value();
    
    char* res = ptr_Route(handle, const_cast<char*>(target.c_str()), const_cast<char*>(driver.c_str()));
    
    std::string resultStr = "{}";
    if (res) {
        resultStr = std::string(res);
        ptr_Free(res);
    }
    return Napi::String::New(env, resultStr);
}

Napi::Object Init(Napi::Env env, Napi::Object exports) {
    exports.Set("loadLib", Napi::Function::New(env, LoadLib));
    exports.Set("newEngine", Napi::Function::New(env, NewEngine));
    exports.Set("freeEngine", Napi::Function::New(env, FreeEngine));
    exports.Set("execute", Napi::Function::New(env, Execute));
    exports.Set("registerMongoDriver", Napi::Function::New(env, RegisterMongoDriver));
    exports.Set("registerSQLiteDriver", Napi::Function::New(env, RegisterSQLiteDriver));
    exports.Set("registerPostgresDriver", Napi::Function::New(env, RegisterPostgresDriver));
    exports.Set("registerSchema", Napi::Function::New(env, RegisterSchema));
    exports.Set("route", Napi::Function::New(env, Route));
    return exports;
}

NODE_API_MODULE(omniql_bridge, Init)
