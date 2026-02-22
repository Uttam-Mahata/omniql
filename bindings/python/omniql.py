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
    from omniql import OmniEngine, Query, Filter

    async def main():
        engine = OmniEngine()
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
    void   OmniQL_Free(char *ptr);
"""
)

_LIB_SEARCH_PATHS = [
    os.path.join(os.path.dirname(__file__), "libomniql.so"),
    os.path.join(os.path.dirname(__file__), "omniql.dll"),
    "libomniql.so",
    "omniql.dll",
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

    # ------------------------------------------------------------------
    # Asyncio API
    # ------------------------------------------------------------------

    async def execute(self, query: Query) -> OmniResult:
        """Execute a query asynchronously (runs in a thread-pool executor)."""
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(None, self.execute_sync, query)

    async def register_schema(self, schema: dict) -> None:
        """Register a schema asynchronously."""
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, self.register_schema_sync, schema)
