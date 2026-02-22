/**
 * OmniQL TypeScript / Node.js Binding
 * =====================================
 *
 * A modern, low-friction Node.js binding built with Koffi that wraps the
 * OmniQL shared library.
 *
 * Example usage:
 *   import { OmniEngine } from 'omniql';
 *
 *   const engine = new OmniEngine();
 *   const driver = engine.registerSQLiteDriver(':memory:');
 *   engine.route('data', driver);
 *   const result = await engine.execute({ target: 'data', action: 'FIND' });
 *   engine.close();
 */

import * as koffi from 'koffi';
import * as path from 'path';
import * as fs from 'fs';

// ---------------------------------------------------------------------------
// TypeScript types
// ---------------------------------------------------------------------------

export type Action = 'FIND' | 'INSERT' | 'UPDATE' | 'DELETE' | 'COUNT';

export interface Filter {
  [field: string]: unknown;
}

export interface QueryOptions {
  limit?: number;
  skip?: number;
  sort?: Record<string, 1 | -1>;
  fields?: Record<string, unknown>;
}

export interface Query {
  target: string;
  action?: Action;
  filter?: Filter;
  document?: Record<string, unknown>;
  options?: QueryOptions;
}

export interface OmniMeta {
  total: number;
  returned: number;
  driver: string;
  target: string;
}

export interface OmniError {
  code: string;
  message: string;
}

export interface OmniResult<T = Record<string, unknown>> {
  data: T[];
  meta: OmniMeta;
  error?: OmniError;
}

export interface CollectionSchema {
  name: string;
  fields: Record<string, { type: string; required?: boolean }>;
}

// ---------------------------------------------------------------------------
// Native library loader
// ---------------------------------------------------------------------------

const LIB_SEARCH_PATHS = [
  path.join(__dirname, 'libomniql.so'),
  path.join(__dirname, 'libomniql.dylib'),
  path.join(__dirname, 'omniql.dll'),
  path.join(process.cwd(), 'libomniql.so'),
  path.join(process.cwd(), 'libomniql.dylib'),
  path.join(process.cwd(), 'omniql.dll'),
];

function findLibPath(): string {
  for (const p of LIB_SEARCH_PATHS) {
    if (fs.existsSync(p)) return p;
  }
  throw new Error(
    'OmniQL native library not found. ' +
    'Please ensure libomniql.so/dylib/dll is present in the package or CWD.',
  );
}

const lib = koffi.load(findLibPath());

// Function definitions
const OmniQL_NewEngine = lib.func('int OmniQL_NewEngine()');
const OmniQL_FreeEngine = lib.func('void OmniQL_FreeEngine(int)');
const OmniQL_Execute = lib.func('char *OmniQL_Execute(int, const char *)');
const OmniQL_RegisterSchema = lib.func('char *OmniQL_RegisterSchema(int, const char *)');
const OmniQL_Route = lib.func('char *OmniQL_Route(int, const char *, const char *)');
const OmniQL_RegisterSQLiteDriver = lib.func('char *OmniQL_RegisterSQLiteDriver(int, const char *)');
const OmniQL_RegisterPostgresDriver = lib.func('char *OmniQL_RegisterPostgresDriver(int, const char *)');
const OmniQL_RegisterMongoDriver = lib.func('char *OmniQL_RegisterMongoDriver(int, const char *, const char *)');
const OmniQL_Free = lib.func('void OmniQL_Free(void *)');

// ---------------------------------------------------------------------------
// OmniEngine class
// ---------------------------------------------------------------------------

export class OmniEngine {
  private readonly handle: number;

  constructor() {
    this.handle = OmniQL_NewEngine();
  }

  /**
   * Internal helper to call native functions that return a JSON string
   * and need to be freed.
   */
  private callNative(fn: (...args: any[]) => any, ...args: any[]): string {
    const raw = fn(this.handle, ...args);
    if (!raw) return '{}';
    try {
      return koffi.decode(raw, 'char *') as string;
    } finally {
      OmniQL_Free(raw);
    }
  }

  /**
   * Executes an OQL query and returns the OmniJSON result.
   */
  async execute<T = Record<string, unknown>>(query: Query): Promise<OmniResult<T>> {
    const queryWithDefaults: Query = { action: 'FIND', ...query };
    const json = JSON.stringify(queryWithDefaults);

    return new Promise((resolve, reject) => {
      try {
        const rawResponse = this.callNative(OmniQL_Execute, json);
        resolve(JSON.parse(rawResponse) as OmniResult<T>);
      } catch (err) {
        reject(err);
      }
    });
  }

  /**
   * Registers a collection schema with the engine.
   */
  registerSchema(schema: CollectionSchema): void {
    this.callNative(OmniQL_RegisterSchema, JSON.stringify(schema));
  }

  /**
   * Binds a collection/table target name to a driver name.
   */
  route(target: string, driverName: string): void {
    this.callNative(OmniQL_Route, target, driverName);
  }

  /**
   * Registers a SQLite driver.
   */
  registerSQLiteDriver(dsn: string): string {
    const raw: string = this.callNative(OmniQL_RegisterSQLiteDriver, dsn);
    return (JSON.parse(raw) as { driver: string }).driver ?? 'sqlite';
  }

  /**
   * Registers a PostgreSQL driver.
   */
  registerPostgresDriver(connStr: string): string {
    const raw: string = this.callNative(OmniQL_RegisterPostgresDriver, connStr);
    return (JSON.parse(raw) as { driver: string }).driver ?? 'postgres';
  }

  /**
   * Registers a MongoDB driver.
   */
  registerMongoDriver(uri: string, dbName: string): string {
    const raw: string = this.callNative(OmniQL_RegisterMongoDriver, uri, dbName);
    return (JSON.parse(raw) as { driver: string }).driver ?? 'mongo';
  }

  /**
   * Releases the native engine handle.
   */
  close(): void {
    OmniQL_FreeEngine(this.handle);
  }
}
