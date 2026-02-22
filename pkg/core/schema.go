package core

import (
	"errors"
	"fmt"
)

// ErrUnknownTarget is returned when a query targets a collection that has no
// registered schema.
var ErrUnknownTarget = errors.New("unknown target: no schema registered")

// SchemaRegistry holds all registered collection schemas and exposes the
// validation logic used by the Core Engine before query execution.
type SchemaRegistry struct {
	schemas map[string]CollectionSchema
}

// NewSchemaRegistry creates an empty SchemaRegistry.
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{schemas: make(map[string]CollectionSchema)}
}

// Register adds or replaces a CollectionSchema in the registry.
func (r *SchemaRegistry) Register(schema CollectionSchema) {
	r.schemas[schema.Name] = schema
}

// Get returns the schema for the given target name.
func (r *SchemaRegistry) Get(target string) (CollectionSchema, bool) {
	s, ok := r.schemas[target]
	return s, ok
}

// Validate checks that the query's target is registered and that any fields
// referenced in the filter and document are declared in the schema.
// When no schema is registered for the target the query is allowed through
// (open / schema-less mode), unless strict mode is enabled.
func (r *SchemaRegistry) Validate(query OQLQuery, strict bool) error {
	schema, ok := r.schemas[query.Target]
	if !ok {
		if strict {
			return fmt.Errorf("%w: %q", ErrUnknownTarget, query.Target)
		}
		return nil
	}

	for field := range query.Filter {
		if _, declared := schema.Fields[field]; !declared {
			return fmt.Errorf("unknown filter field %q for target %q", field, query.Target)
		}
	}

	for field := range query.Document {
		fs, declared := schema.Fields[field]
		if !declared {
			return fmt.Errorf("unknown document field %q for target %q", field, query.Target)
		}
		if fs.Required && query.Document[field] == nil {
			return fmt.Errorf("required field %q is nil for target %q", field, query.Target)
		}
	}

	return nil
}
