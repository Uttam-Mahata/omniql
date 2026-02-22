package core

import "context"

// FieldType enumerates the supported schema field types.
type FieldType string

const (
	FieldTypeString  FieldType = "string"
	FieldTypeInt     FieldType = "int"
	FieldTypeFloat   FieldType = "float"
	FieldTypeBool    FieldType = "bool"
	FieldTypeArray   FieldType = "array"
	FieldTypeObject  FieldType = "object"
	FieldTypeAny     FieldType = "any"
)

// FieldSchema describes a single field within a collection schema.
type FieldSchema struct {
	Type     FieldType `json:"type"`
	Required bool      `json:"required,omitempty"`
}

// CollectionSchema describes the shape of a target collection / table / index.
type CollectionSchema struct {
	// Name is the collection / table identifier that OQL queries reference.
	Name   string                 `json:"name"`
	Fields map[string]FieldSchema `json:"fields"`
}

// Driver is the interface every database adapter must satisfy.
//
// A Driver is responsible for:
//  1. Translating an OQLQuery into the native database command (AST Translation).
//  2. Executing the native command against the database.
//  3. Returning raw results that the engine will normalise into OmniJSON.
type Driver interface {
	// Name returns the unique identifier of the driver (e.g. "postgres", "mongo").
	Name() string

	// Execute receives a validated OQLQuery and returns raw result rows together
	// with the total count of matching records and any error that occurred.
	Execute(ctx context.Context, query OQLQuery) (rows []map[string]interface{}, total int64, err error)

	// BatchInsert inserts multiple records in a single operation.
	BatchInsert(ctx context.Context, target string, docs []map[string]interface{}) (rows []map[string]interface{}, err error)

	// Ping verifies that the underlying database connection is healthy.
	Ping(ctx context.Context) error

	// Close releases all resources held by the driver.
	Close() error
}
