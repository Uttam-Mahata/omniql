"""
OmniQL Python Binding
=====================

A native extension built with CFFI that wraps the OmniQL shared library
(``libomniql.so`` / ``omniql.dll``) and exposes an asyncio-compatible API.

Build the native library first::

    go build -buildmode=c-shared -o libomniql.so ../../pkg/ffi

Then install the Python package::

    pip install .

Example usage::

    import asyncio
    from omniql import OmniEngine, Query

    async def main():
        engine = OmniEngine()

        # 1. Register a driver and bind a target to it.
        driver_name = await engine.register_sqlite_driver(":memory:")
        await engine.route("analytics_data", driver_name)

        # 2. Execute a query.
        result = await engine.execute(
            Query(
                target="analytics_data",
                action="FIND",
                filter={
                    "category": {"$in": ["electronics", "books"]},
                    "price": {"$lt": 500},
                },
                options={"limit": 20},
            )
        )
        print(result.data)

    asyncio.run(main())
"""

from __future__ import annotations

import asyncio
import json
import os
import threading
from dataclasses import dataclass, field, asdict
from typing import Any, Dict, List, Optional

import cffi  # type: ignore

# ---------------------------------------------------------------------------
# Load the native library via CFFI
# ---------------------------------------------------------------------------

_ffi = cffi.FFI()
_ffi.cdef(
    """
    int    OmniQL_NewEngine(void);
    void   OmniQL_FreeEngine(int handle);
    char * OmniQL_Execute(int handle, const char *queryJson);
    char * OmniQL_RegisterSchema(int handle, const char *schemaJson);
    char * OmniQL_Route(int handle, const char *target, const char *driverName);
    char * OmniQL_RegisterSQLiteDriver(int handle, const char *dsn);
    char * OmniQL_RegisterPostgresDriver(int handle, const char *connStr);
    char * OmniQL_RegisterMongoDriver(int handle, const char *uri, const char *dbName);
    char * OmniQL_RegisterMySQLDriver(int handle, const char *dsn);
    char * OmniQL_RegisterSQLServerDriver(int handle, const char *dsn);
    char * OmniQL_RegisterRedisDriver(int handle, const char *url);
    char * OmniQL_RegisterElasticsearchDriver(int handle, const char *addr);
    void   OmniQL_Free(char *ptr);
"""
)

_LIB_SEARCH_PATHS = [
    os.path.join(os.path.dirname(__file__), "libomniql.so"),
    os.path.join(os.path.dirname(__file__), "omniql.dll"),
    os.path.join(os.path.dirname(__file__), "libomniql.dylib"),
    "libomniql.so",
    "omniql.dll",
    "libomniql.dylib",
]


def _load_lib():
    for path in _LIB_SEARCH_PATHS:
        if os.path.exists(path):
            return _ffi.dlopen(path)
    raise OSError(
        "OmniQL native library not found. "
        "Build it with: go build -buildmode=c-shared -o libomniql.so ../../pkg/ffi"
    )


_lib = _load_lib()


# ---------------------------------------------------------------------------
# Python data classes
# ---------------------------------------------------------------------------


@dataclass
class QueryOptions:
    limit: int = 0
    skip: int = 0
    sort: Dict[str, int] = field(default_factory=dict)
    fields: Dict[str, Any] = field(default_factory=dict)


@dataclass
class Query:
    """An OQL query object."""

    target: str
    action: str = "FIND"
    filter: Dict[str, Any] = field(default_factory=dict)
    document: Dict[str, Any] = field(default_factory=dict)
    options: QueryOptions = field(default_factory=QueryOptions)

    def to_dict(self) -> dict:
        return asdict(self)


@dataclass
class OmniMeta:
    total: int = 0
    returned: int = 0
    driver: str = ""
    target: str = ""


@dataclass
class OmniError:
    code: str = ""
    message: str = ""


@dataclass
class OmniResult:
    """The standardised OmniJSON response."""

    data: List[Dict[str, Any]] = field(default_factory=list)
    meta: OmniMeta = field(default_factory=OmniMeta)
    error: Optional[OmniError] = None

    @classmethod
    def from_dict(cls, d: dict) -> "OmniResult":
        meta = OmniMeta(**d.get("meta", {}))
        err_data = d.get("error")
        err = OmniError(**err_data) if err_data else None
        return cls(data=d.get("data", []), meta=meta, error=err)


# ---------------------------------------------------------------------------
# OmniEngine
# ---------------------------------------------------------------------------


class OmniEngine:
    """High-level wrapper around the native OmniQL Core Engine.

    The engine is thread-safe and supports asyncio via
    :meth:`execute` (async) or :meth:`execute_sync` (synchronous).
    """

    def __init__(self) -> None:
        self._handle: int = _lib.OmniQL_NewEngine()
        self._lock = threading.Lock()

    def __del__(self) -> None:
        try:
            _lib.OmniQL_FreeEngine(self._handle)
        except Exception:
            pass

    # ------------------------------------------------------------------
    # Synchronous API
    # ------------------------------------------------------------------

    def execute_sync(self, query: Query) -> OmniResult:
        """Execute a query synchronously and return the result."""
        query_json = json.dumps(query.to_dict()).encode("utf-8")
        with self._lock:
            raw = _lib.OmniQL_Execute(self._handle, query_json)
        try:
            response = json.loads(_ffi.string(raw).decode("utf-8"))
        finally:
            _lib.OmniQL_Free(raw)
        return OmniResult.from_dict(response)

    def register_schema_sync(self, schema: dict) -> None:
        """Register a collection schema synchronously."""
        schema_json = json.dumps(schema).encode("utf-8")
        with self._lock:
            raw = _lib.OmniQL_RegisterSchema(self._handle, schema_json)
        _lib.OmniQL_Free(raw)

    def route_sync(self, target: str, driver_name: str) -> None:
        """Bind a target collection/table to a driver name synchronously."""
        with self._lock:
            raw = _lib.OmniQL_Route(self._handle, target.encode(), driver_name.encode())
        _lib.OmniQL_Free(raw)

    def register_sqlite_driver_sync(self, dsn: str) -> str:
        """Register a SQLite driver synchronously.  Returns the driver name."""
        with self._lock:
            raw = _lib.OmniQL_RegisterSQLiteDriver(self._handle, dsn.encode())
        try:
            result = json.loads(_ffi.string(raw).decode())
            return result.get("driver", "sqlite")
        finally:
            _lib.OmniQL_Free(raw)

    def register_postgres_driver_sync(self, conn_str: str) -> str:
        """Register a PostgreSQL driver synchronously.  Returns the driver name."""
        with self._lock:
            raw = _lib.OmniQL_RegisterPostgresDriver(self._handle, conn_str.encode())
        try:
            result = json.loads(_ffi.string(raw).decode())
            return result.get("driver", "postgres")
        finally:
            _lib.OmniQL_Free(raw)

    def register_mongo_driver_sync(self, uri: str, db_name: str) -> str:
        """Register a MongoDB driver synchronously.  Returns the driver name."""
        with self._lock:
            raw = _lib.OmniQL_RegisterMongoDriver(self._handle, uri.encode(), db_name.encode())
        try:
            result = json.loads(_ffi.string(raw).decode())
            return result.get("driver", "mongo")
        finally:
            _lib.OmniQL_Free(raw)

    def register_mysql_driver_sync(self, dsn: str) -> str:
        """Register a MySQL driver synchronously.  Returns the driver name."""
        with self._lock:
            raw = _lib.OmniQL_RegisterMySQLDriver(self._handle, dsn.encode())
        try:
            result = json.loads(_ffi.string(raw).decode())
            return result.get("driver", "mysql")
        finally:
            _lib.OmniQL_Free(raw)

    def register_sqlserver_driver_sync(self, dsn: str) -> str:
        """Register a SQL Server driver synchronously.  Returns the driver name."""
        with self._lock:
            raw = _lib.OmniQL_RegisterSQLServerDriver(self._handle, dsn.encode())
        try:
            result = json.loads(_ffi.string(raw).decode())
            return result.get("driver", "sqlserver")
        finally:
            _lib.OmniQL_Free(raw)

    def register_redis_driver_sync(self, url: str) -> str:
        """Register a Redis driver synchronously.  Returns the driver name."""
        with self._lock:
            raw = _lib.OmniQL_RegisterRedisDriver(self._handle, url.encode())
        try:
            result = json.loads(_ffi.string(raw).decode())
            return result.get("driver", "redis")
        finally:
            _lib.OmniQL_Free(raw)

    def register_elasticsearch_driver_sync(self, addr: str) -> str:
        """Register an Elasticsearch driver synchronously.  Returns the driver name."""
        with self._lock:
            raw = _lib.OmniQL_RegisterElasticsearchDriver(self._handle, addr.encode())
        try:
            result = json.loads(_ffi.string(raw).decode())
            return result.get("driver", "elasticsearch")
        finally:
            _lib.OmniQL_Free(raw)

    # ------------------------------------------------------------------
    # Asyncio API
    # ------------------------------------------------------------------

    def _get_loop(self) -> asyncio.AbstractEventLoop:
        try:
            return asyncio.get_running_loop()
        except RuntimeError:
            return asyncio.get_event_loop()

    async def execute(self, query: Query) -> OmniResult:
        """Execute a query asynchronously (runs in a thread-pool executor)."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.execute_sync, query)

    async def register_schema(self, schema: dict) -> None:
        """Register a schema asynchronously."""
        loop = self._get_loop()
        await loop.run_in_executor(None, self.register_schema_sync, schema)

    async def route(self, target: str, driver_name: str) -> None:
        """Bind a target collection/table to a driver name asynchronously."""
        loop = self._get_loop()
        await loop.run_in_executor(None, self.route_sync, target, driver_name)

    async def register_sqlite_driver(self, dsn: str) -> str:
        """Register a SQLite driver asynchronously.  Returns the driver name."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.register_sqlite_driver_sync, dsn)

    async def register_postgres_driver(self, conn_str: str) -> str:
        """Register a PostgreSQL driver asynchronously.  Returns the driver name."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.register_postgres_driver_sync, conn_str)

    async def register_mongo_driver(self, uri: str, db_name: str) -> str:
        """Register a MongoDB driver asynchronously.  Returns the driver name."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.register_mongo_driver_sync, uri, db_name)

    async def register_mysql_driver(self, dsn: str) -> str:
        """Register a MySQL driver asynchronously.  Returns the driver name."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.register_mysql_driver_sync, dsn)

    async def register_sqlserver_driver(self, dsn: str) -> str:
        """Register a SQL Server driver asynchronously.  Returns the driver name."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.register_sqlserver_driver_sync, dsn)

    async def register_redis_driver(self, url: str) -> str:
        """Register a Redis driver asynchronously.  Returns the driver name."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.register_redis_driver_sync, url)

    async def register_elasticsearch_driver(self, addr: str) -> str:
        """Register an Elasticsearch driver asynchronously.  Returns the driver name."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.register_elasticsearch_driver_sync, addr)

    def batch_insert_sync(self, target: str, docs: List[Dict[str, Any]]) -> OmniResult:
        """Insert multiple documents synchronously via BATCH_INSERT action."""
        query = Query(
            target=target,
            action="BATCH_INSERT",
        )
        d = query.to_dict()
        d["documents"] = docs
        query_json = json.dumps(d).encode("utf-8")
        with self._lock:
            raw = _lib.OmniQL_Execute(self._handle, query_json)
        try:
            response = json.loads(_ffi.string(raw).decode("utf-8"))
        finally:
            _lib.OmniQL_Free(raw)
        return OmniResult.from_dict(response)

    async def batch_insert(self, target: str, docs: List[Dict[str, Any]]) -> OmniResult:
        """Insert multiple documents asynchronously via BATCH_INSERT action."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.batch_insert_sync, target, docs)

    # ------------------------------------------------------------------
    # Terminal convenience methods (sync)
    # ------------------------------------------------------------------

    def find_many_sync(
        self,
        target: str,
        filter: Optional[Dict[str, Any]] = None,
        options: Optional[Dict[str, Any]] = None,
    ) -> List[Dict[str, Any]]:
        """Return all matching documents synchronously."""
        opts = QueryOptions(**options) if options else QueryOptions()
        result = self.execute_sync(Query(target=target, action="FIND", filter=filter or {}, options=opts))
        if result.error:
            raise RuntimeError(f"find_many error: {result.error.code}: {result.error.message}")
        return result.data

    def find_first_sync(
        self,
        target: str,
        filter: Optional[Dict[str, Any]] = None,
    ) -> Optional[Dict[str, Any]]:
        """Return the first matching document synchronously, or None."""
        opts = QueryOptions(limit=1)
        result = self.execute_sync(Query(target=target, action="FIND", filter=filter or {}, options=opts))
        if result.error:
            raise RuntimeError(f"find_first error: {result.error.code}: {result.error.message}")
        return result.data[0] if result.data else None

    def count_sync(self, target: str, filter: Optional[Dict[str, Any]] = None) -> int:
        """Return the count of matching documents synchronously."""
        result = self.execute_sync(Query(target=target, action="COUNT", filter=filter or {}))
        if result.error:
            raise RuntimeError(f"count error: {result.error.code}: {result.error.message}")
        return int(result.meta.total)

    def insert_one_sync(self, target: str, doc: Dict[str, Any]) -> Dict[str, Any]:
        """Insert a single document synchronously and return the result row."""
        result = self.execute_sync(Query(target=target, action="INSERT", document=doc))
        if result.error:
            raise RuntimeError(f"insert_one error: {result.error.code}: {result.error.message}")
        return result.data[0] if result.data else {}

    def update_many_sync(
        self,
        target: str,
        filter: Dict[str, Any],
        update: Dict[str, Any],
    ) -> OmniResult:
        """Update matching documents synchronously."""
        result = self.execute_sync(Query(target=target, action="UPDATE", filter=filter, document=update))
        if result.error:
            raise RuntimeError(f"update_many error: {result.error.code}: {result.error.message}")
        return result

    def delete_many_sync(self, target: str, filter: Dict[str, Any]) -> OmniResult:
        """Delete matching documents synchronously."""
        result = self.execute_sync(Query(target=target, action="DELETE", filter=filter))
        if result.error:
            raise RuntimeError(f"delete_many error: {result.error.code}: {result.error.message}")
        return result

    # ------------------------------------------------------------------
    # Terminal convenience methods (async)
    # ------------------------------------------------------------------

    async def find_many(
        self,
        target: str,
        filter: Optional[Dict[str, Any]] = None,
        options: Optional[Dict[str, Any]] = None,
    ) -> List[Dict[str, Any]]:
        """Return all matching documents asynchronously."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.find_many_sync, target, filter, options)

    async def find_first(
        self,
        target: str,
        filter: Optional[Dict[str, Any]] = None,
    ) -> Optional[Dict[str, Any]]:
        """Return the first matching document asynchronously, or None."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.find_first_sync, target, filter)

    async def count(self, target: str, filter: Optional[Dict[str, Any]] = None) -> int:
        """Return the count of matching documents asynchronously."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.count_sync, target, filter)

    async def insert_one(self, target: str, doc: Dict[str, Any]) -> Dict[str, Any]:
        """Insert a single document asynchronously."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.insert_one_sync, target, doc)

    async def update_many(
        self,
        target: str,
        filter: Dict[str, Any],
        update: Dict[str, Any],
    ) -> OmniResult:
        """Update matching documents asynchronously."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.update_many_sync, target, filter, update)

    async def delete_many(self, target: str, filter: Dict[str, Any]) -> OmniResult:
        """Delete matching documents asynchronously."""
        loop = self._get_loop()
        return await loop.run_in_executor(None, self.delete_many_sync, target, filter)
from .model import OmniModel, Field, StringField, IntField, FloatField, BoolField
