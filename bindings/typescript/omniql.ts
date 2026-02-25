/**
 * OmniQL TypeScript / Node.js Binding
 * =====================================
 *
 * A robust Node.js binding built with a native C++ addon to handle signal
 * masking and FFI stability.
 */

import * as path from 'path';
import * as fs from 'fs';

// ---------------------------------------------------------------------------
// TypeScript types (same as before)
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

// eslint-disable-next-line @typescript-eslint/no-var-requires
const bridge = require('bindings')('omniql_bridge');

const LIB_SEARCH_PATHS = [
  path.join(__dirname, 'libomniql.so'),
  path.join(__dirname, 'libomniql.dylib'),
  path.join(__dirname, 'omniql.dll'),
  path.join(process.cwd(), 'libomniql.so'),
  path.join(process.cwd(), 'libomniql.dylib'),
  path.join(process.cwd(), 'omniql.dll'),
  './libomniql.so',
  'libomniql.so',
];

function findLibPath(): string {
  for (const p of LIB_SEARCH_PATHS) {
    if (fs.existsSync(p)) return p;
  }
  throw new Error('OmniQL native library not found. Please ensure libomniql.so/dylib/dll is present.');
}

// Initialize the bridge by loading the shared library.
// The C++ bridge handles SIGURG masking internally.
bridge.loadLib(findLibPath());

// ---------------------------------------------------------------------------
// OmniEngine class
// ---------------------------------------------------------------------------

export class OmniEngine {
  private readonly handle: number;

  constructor() {
    this.handle = bridge.newEngine();
  }

  async execute<T = Record<string, unknown>>(query: Query): Promise<OmniResult<T>> {
    const queryWithDefaults: Query = { action: 'FIND', ...query };
    const json = JSON.stringify(queryWithDefaults);
    const rawResponse = bridge.execute(this.handle, json);
    return JSON.parse(rawResponse) as OmniResult<T>;
  }

  registerSchema(schema: CollectionSchema): void {
    bridge.registerSchema(this.handle, JSON.stringify(schema));
  }

  route(target: string, driverName: string): void {
    bridge.route(this.handle, target, driverName);
  }

  registerSQLiteDriver(dsn: string): string {
    const raw: string = bridge.registerSQLiteDriver(this.handle, dsn);
    return (JSON.parse(raw) as { driver: string }).driver ?? 'sqlite';
  }

  registerPostgresDriver(connStr: string): string {
    const raw: string = bridge.registerPostgresDriver(this.handle, connStr);
    return (JSON.parse(raw) as { driver: string }).driver ?? 'postgres';
  }

  registerMongoDriver(uri: string, dbName: string): string {
    const raw: string = bridge.registerMongoDriver(this.handle, uri, dbName);
    return (JSON.parse(raw) as { driver: string }).driver ?? 'mongo';
  }

  async batchInsert<T = Record<string, unknown>>(
    target: string,
    docs: Record<string, unknown>[],
  ): Promise<OmniResult<T>> {
    const payload = { target, action: 'BATCH_INSERT' as const, documents: docs };
    const json = JSON.stringify(payload);
    const rawResponse = bridge.execute(this.handle, json);
    return JSON.parse(rawResponse) as OmniResult<T>;
  }

  close(): void {
    bridge.freeEngine(this.handle);
  }
}
