/**
 * OmniQL TypeScript / Node.js Binding
 * =====================================
 *
 * A Node.js addon using N-API that wraps the OmniQL shared library and exposes
 * a fully type-safe, Promise-based API with generated TypeScript definitions.
 *
 * Build the native library first:
 *   go build -buildmode=c-shared -o libomniql.so ../../pkg/ffi
 *
 * Install:
 *   npm install
 *
 * Example usage:
 *   import { OmniEngine, Query } from 'omniql';
 *
 *   const engine = new OmniEngine();
 *
 *   // 1. Register a driver and route a target to it.
 *   const driverName = engine.registerSQLiteDriver(':memory:');
 *   engine.route('analytics_data', driverName);
 *
 *   // 2. Execute a query.
 *   const result = await engine.execute({
 *     target: 'analytics_data',
 *     action: 'FIND',
 *     filter: {
 *       category: { $in: ['electronics', 'books'] },
 *       price:    { $lt: 500 },
 *     },
 *     options: { limit: 20 },
 *   });
 *
 *   console.log(result.data);
 *   engine.close();
 */

import * as ffi from 'ffi-napi';
import * as ref from 'ref-napi';
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
  path.join(__dirname, 'omniql.dll'),
  'libomniql',
];

function loadLib(): ReturnType<typeof ffi.Library> {
  for (const p of LIB_SEARCH_PATHS) {
    if (fs.existsSync(p)) {
      return ffi.Library(p, {
        OmniQL_NewEngine:              ['int',    []],
        OmniQL_FreeEngine:             ['void',   ['int']],
        OmniQL_Execute:                ['string', ['int', 'string']],
        OmniQL_RegisterSchema:         ['string', ['int', 'string']],
        OmniQL_Route:                  ['string', ['int', 'string', 'string']],
        OmniQL_RegisterSQLiteDriver:   ['string', ['int', 'string']],
        OmniQL_RegisterPostgresDriver: ['string', ['int', 'string']],
        OmniQL_RegisterMongoDriver:    ['string', ['int', 'string', 'string']],
        OmniQL_Free:                   ['void',   ['pointer']],
      });
    }
  }
  throw new Error(
    'OmniQL native library not found. ' +
    'Build it with: go build -buildmode=c-shared -o libomniql.so ../../pkg/ffi',
  );
}

const lib = loadLib();

// ---------------------------------------------------------------------------
// OmniEngine class
// ---------------------------------------------------------------------------

/**
 * OmniEngine is the main entry-point for the OmniQL TypeScript binding.
 *
 * All `execute` calls are async and safe to use with `await`.
 */
export class OmniEngine {
  private readonly handle: number;

  constructor() {
    this.handle = lib.OmniQL_NewEngine();
  }

  /**
   * Executes an OQL query and returns the OmniJSON result.
   */
  async execute<T = Record<string, unknown>>(query: Query): Promise<OmniResult<T>> {
    const queryWithDefaults: Query = { action: 'FIND', ...query };
    const json = JSON.stringify(queryWithDefaults);

    return new Promise((resolve, reject) => {
      try {
        const raw: string = lib.OmniQL_Execute(this.handle, json);
        resolve(JSON.parse(raw) as OmniResult<T>);
      } catch (err) {
        reject(err);
      }
    });
  }

  /**
   * Registers a collection schema with the engine.
   */
  registerSchema(schema: CollectionSchema): void {
    lib.OmniQL_RegisterSchema(this.handle, JSON.stringify(schema));
  }

  /**
   * Binds a collection/table target name to a driver name.
   * Must be called after registering a driver.
   *
   * @param target     The collection or table name.
   * @param driverName The driver name returned by a registerXxxDriver call.
   */
  route(target: string, driverName: string): void {
    lib.OmniQL_Route(this.handle, target, driverName);
  }

  /**
   * Registers a SQLite driver using the given DSN (file path or ":memory:").
   * Returns the driver name ("sqlite") to use with {@link route}.
   */
  registerSQLiteDriver(dsn: string): string {
    const raw: string = lib.OmniQL_RegisterSQLiteDriver(this.handle, dsn);
    return (JSON.parse(raw) as { driver: string }).driver ?? 'sqlite';
  }

  /**
   * Registers a PostgreSQL driver using the given connection string.
   * e.g. "host=localhost user=pg password=pg dbname=mydb sslmode=disable"
   * Returns the driver name ("postgres") to use with {@link route}.
   */
  registerPostgresDriver(connStr: string): string {
    const raw: string = lib.OmniQL_RegisterPostgresDriver(this.handle, connStr);
    return (JSON.parse(raw) as { driver: string }).driver ?? 'postgres';
  }

  /**
   * Registers a MongoDB driver using the given URI and database name.
   * e.g. uri = "mongodb://localhost:27017", dbName = "mydb"
   * Returns the driver name ("mongo") to use with {@link route}.
   */
  registerMongoDriver(uri: string, dbName: string): string {
    const raw: string = lib.OmniQL_RegisterMongoDriver(this.handle, uri, dbName);
    return (JSON.parse(raw) as { driver: string }).driver ?? 'mongo';
  }

  /**
   * Releases the native engine handle.  Call when done.
   */
  close(): void {
    lib.OmniQL_FreeEngine(this.handle);
  }
}
