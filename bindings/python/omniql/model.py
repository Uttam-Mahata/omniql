from __future__ import annotations
from typing import Any, ClassVar, Optional, Dict, List, Set

from omniql import OmniEngine


class Field:
    """Descriptor for OmniModel fields."""

    def __init__(self, field_type: str, required: bool = False, default: Any = None):
        self.field_type = field_type
        self.required = required
        self.default = default
        self.name: str = ""

    def __set_name__(self, owner: Any, name: str) -> None:
        self.name = name

    def __get__(self, obj: Any, objtype: Any = None) -> Any:
        if obj is None:
            return self
        return obj._data.get(self.name, self.default)

    def __set__(self, obj: Any, value: Any) -> None:
        obj._data[self.name] = value
        obj._dirty.add(self.name)


class StringField(Field):
    def __init__(self, **kwargs: Any): super().__init__("string", **kwargs)

class IntField(Field):
    def __init__(self, **kwargs: Any): super().__init__("int", **kwargs)

class FloatField(Field):
    def __init__(self, **kwargs: Any): super().__init__("float", **kwargs)

class BoolField(Field):
    def __init__(self, **kwargs: Any): super().__init__("bool", **kwargs)


class ModelMeta(type):
    """Metaclass that collects Field descriptors and builds schema."""
    def __new__(mcs, name: str, bases: tuple, namespace: dict) -> Any:
        fields = {}
        for key, val in namespace.items():
            if isinstance(val, Field):
                val.name = key
                fields[key] = val
        cls = super().__new__(mcs, name, bases, namespace)
        cls._fields = fields
        return cls


class QuerySet:
    """Class-level query interface for OmniModel."""

    def __init__(self, model_cls: type[OmniModel]):
        self._model_cls = model_cls

    async def find_many(self, filter: dict | None = None, **options: Any) -> list[OmniModel]:
        engine = self._model_cls._engine
        if not engine:
            raise RuntimeError(f"{self._model_cls.__name__} not configured")
        target = self._model_cls.__target__
        docs = await engine.find_many(target, filter or {}, options)
        return [self._model_cls._from_doc(doc) for doc in docs]

    async def find_first(self, filter: dict | None = None) -> OmniModel | None:
        engine = self._model_cls._engine
        if not engine:
            raise RuntimeError(f"{self._model_cls.__name__} not configured")
        target = self._model_cls.__target__
        doc = await engine.find_first(target, filter or {})
        return self._model_cls._from_doc(doc) if doc else None

    async def count(self, filter: dict | None = None) -> int:
        engine = self._model_cls._engine
        if not engine:
            raise RuntimeError(f"{self._model_cls.__name__} not configured")
        return await engine.count(self._model_cls.__target__, filter or {})

    async def delete_many(self, filter: dict) -> Any:
        engine = self._model_cls._engine
        if not engine:
            raise RuntimeError(f"{self._model_cls.__name__} not configured")
        return await engine.delete_many(self._model_cls.__target__, filter)

    async def update_many(self, filter: dict, update: dict) -> Any:
        engine = self._model_cls._engine
        if not engine:
            raise RuntimeError(f"{self._model_cls.__name__} not configured")
        return await engine.update_many(self._model_cls.__target__, filter, update)

    # Synchronous variants
    def find_many_sync(self, filter: dict | None = None, **options: Any) -> list[OmniModel]:
        engine = self._model_cls._engine
        if not engine:
            raise RuntimeError(f"{self._model_cls.__name__} not configured")
        target = self._model_cls.__target__
        docs = engine.find_many_sync(target, filter or {}, options)
        return [self._model_cls._from_doc(doc) for doc in docs]

    def find_first_sync(self, filter: dict | None = None) -> OmniModel | None:
        engine = self._model_cls._engine
        if not engine:
            raise RuntimeError(f"{self._model_cls.__name__} not configured")
        target = self._model_cls.__target__
        doc = engine.find_first_sync(target, filter or {})
        return self._model_cls._from_doc(doc) if doc else None

    def count_sync(self, filter: dict | None = None) -> int:
        engine = self._model_cls._engine
        if not engine:
            raise RuntimeError(f"{self._model_cls.__name__} not configured")
        return engine.count_sync(self._model_cls.__target__, filter or {})


class OmniModel(metaclass=ModelMeta):
    """Active Record base class for OmniQL."""

    __target__: ClassVar[str] = ""
    _engine: ClassVar[OmniEngine | None] = None
    _fields: ClassVar[dict[str, Field]] = {}
    objects: ClassVar[QuerySet]

    def __init_subclass__(cls, **kwargs: Any) -> None:
        super().__init_subclass__(**kwargs)
        cls.objects = QuerySet(cls)

    def __init__(self, **kwargs: Any):
        self._data: dict[str, Any] = {}
        self._dirty: set[str] = set()
        self._persisted: bool = False
        for name, field in self._fields.items():
            if name in kwargs:
                self._data[name] = kwargs[name]
            elif field.default is not None:
                self._data[name] = field.default

    @classmethod
    def configure(cls, engine: OmniEngine) -> None:
        """Bind model to an engine and auto-register schema."""
        cls._engine = engine
        schema = {"name": cls.__target__, "fields": {"id": {"type": "any", "required": False}, "_id": {"type": "any", "required": False}}}
        for name, field in cls._fields.items():
            schema["fields"][name] = {"type": field.field_type, "required": field.required}
        engine.register_schema_sync(schema)

    @classmethod
    def _from_doc(cls, doc: dict) -> OmniModel:
        instance = cls.__new__(cls)
        instance._data = dict(doc)
        instance._dirty = set()
        instance._persisted = True
        return instance

    def to_dict(self) -> dict:
        return dict(self._data)

    async def save(self) -> None:
        if self._engine is None:
            raise RuntimeError(f"{type(self).__name__} not configured")
        for name, field in self._fields.items():
            if field.required and name not in self._data:
                raise ValueError(f"Required field '{name}' is missing")
        if self._persisted:
            if not self._dirty:
                return
            update_doc = {k: self._data[k] for k in self._dirty if k in self._data}
            id_val = self._data.get("id") or self._data.get("_id")
            if id_val is None:
                raise ValueError("Cannot update without id or _id")
            id_key = "id" if "id" in self._data else "_id"
            await self._engine.update_many(self.__target__, {id_key: id_val}, update_doc)
        else:
            result = await self._engine.insert_one(self.__target__, self._data)
            if result and isinstance(result, dict):
                self._data.update(result)
            self._persisted = True
        self._dirty.clear()

    async def delete(self) -> None:
        if self._engine is None:
            raise RuntimeError(f"{type(self).__name__} not configured")
        if not self._persisted:
            raise ValueError("Cannot delete a record that hasn't been saved")
        id_val = self._data.get("id") or self._data.get("_id")
        if id_val is None:
            raise ValueError("Cannot delete without id or _id")
        id_key = "id" if "id" in self._data else "_id"
        await self._engine.delete_many(self.__target__, {id_key: id_val})
        self._persisted = False

    async def refresh(self) -> None:
        if self._engine is None:
            raise RuntimeError(f"{type(self).__name__} not configured")
        id_val = self._data.get("id") or self._data.get("_id")
        if id_val is None:
            raise ValueError("Cannot refresh without id or _id")
        id_key = "id" if "id" in self._data else "_id"
        doc = await self._engine.find_first(self.__target__, {id_key: id_val})
        if doc is not None:
            self._data = dict(doc)
            self._dirty.clear()

    def __repr__(self) -> str:
        cls_name = type(self).__name__
        fields_str = ", ".join(f"{k}={v!r}" for k, v in self._data.items())
        return f"{cls_name}({fields_str})"
