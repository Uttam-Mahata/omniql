// Package core contains the OmniQL Core Engine – the central "Traffic
// Controller" that routes OQL queries to the appropriate pluggable driver and
// normalises the results into the standardised OmniJSON format.
package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrDriverNotFound is returned when no driver is registered for the requested target.
var ErrDriverNotFound = errors.New("no driver registered for target")

// ErrNoDrivers is returned when Execute is called but no drivers have been registered.
var ErrNoDrivers = errors.New("no drivers registered")

// Engine is the OmniQL Core Engine.
//
// Responsibilities (in order):
//  1. Receive an OQLQuery from a language binding.
//  2. Validate the query against the SchemaRegistry.
//  3. Select the target Driver.
//  4. Hand the query to the Driver for AST translation and execution.
//  5. Normalise raw results into OmniJSON and return them to the caller.
type Engine struct {
	mu            sync.RWMutex
	drivers       map[string]Driver
	routes        map[string]string // target name -> driver name
	registry      *SchemaRegistry
	strict        bool
	defaultDriver string // name of the first driver registered (deterministic fallback)
	ensuredTargets map[string]bool
	autoSchema     bool
}

// EngineOption is a functional option for configuring the Engine.
type EngineOption func(*Engine)

// WithStrictSchema enables strict schema validation: queries against
// unregistered targets are rejected.
func WithStrictSchema() EngineOption {
	return func(e *Engine) { e.strict = true }
}

// WithAutoSchema controls whether the engine auto-creates targets.
// Enabled by default. Disable for production environments where schema
// changes should be explicit.
func WithAutoSchema(enabled bool) EngineOption {
	return func(e *Engine) { e.autoSchema = enabled }
}

// NewEngine creates a new Core Engine with the given options.
func NewEngine(opts ...EngineOption) *Engine {
	e := &Engine{
		drivers:        make(map[string]Driver),
		routes:         make(map[string]string),
		registry:       NewSchemaRegistry(),
		ensuredTargets: make(map[string]bool),
		autoSchema:     true,
	}
	for _, o := range opts {
		o(e)
	}
	return e
}

// RegisterDriver adds a Driver to the engine.  If a driver with the same name
// already exists it is replaced.  The first driver registered becomes the
// default fallback for targets with no explicit Route.
func (e *Engine) RegisterDriver(d Driver) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.defaultDriver == "" {
		e.defaultDriver = d.Name()
	}
	e.drivers[d.Name()] = d
}

// Route binds a collection/table target name to a specific driver name.
// If no explicit route exists for a target, the engine falls back to the
// first registered driver (deterministic).
func (e *Engine) Route(target, driverName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.routes[target] = driverName
}

// RegisterSchema registers a CollectionSchema in the engine's SchemaRegistry.
func (e *Engine) RegisterSchema(schema CollectionSchema) {
	e.registry.Register(schema)
}

// Execute is the primary entry-point.  It orchestrates the full pipeline:
// validate → select driver → execute → normalise.
func (e *Engine) Execute(ctx context.Context, query OQLQuery) (*OmniJSON, error) {
	if err := e.registry.Validate(query, e.strict); err != nil {
		return errorResponse(err, "SCHEMA_ERROR"), nil
	}

	driver, err := e.selectDriver(query.Target)
	if err != nil {
		return errorResponse(err, "DRIVER_ERROR"), nil
	}

	// Auto-ensure target on first write operation
	if e.autoSchema && isWriteAction(query.Action) {
		e.mu.Lock()
		ensured := e.ensuredTargets[query.Target]
		e.mu.Unlock()

		if !ensured {
			if sad, ok := driver.(SchemaAwareDriver); ok {
				schema := e.inferOrGetSchema(query)
				if err := sad.EnsureTarget(ctx, query.Target, schema); err != nil {
					return errorResponse(err, "EXECUTION_ERROR"), nil
				}
				e.mu.Lock()
				e.ensuredTargets[query.Target] = true
				e.mu.Unlock()
			}
		}
	}

	var rows []map[string]interface{}
	var total int64

	if query.Action == ActionBatchInsert {
		rows, err = driver.BatchInsert(ctx, query.Target, query.Documents)
		total = int64(len(rows))
	} else {
		rows, total, err = driver.Execute(ctx, query)
	}

	if err != nil {
		return errorResponse(err, "EXECUTION_ERROR"), nil
	}

	return &OmniJSON{
		Data: rows,
		Meta: OmniMeta{
			Total:    total,
			Returned: len(rows),
			Driver:   driver.Name(),
			Target:   query.Target,
		},
	}, nil
}

// selectDriver returns the driver that should handle the given target.
// It checks the explicit route table first; if no explicit route exists it
// falls back to the first registered driver.
func (e *Engine) selectDriver(target string) (Driver, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if name, ok := e.routes[target]; ok {
		d, found := e.drivers[name]
		if !found {
			return nil, fmt.Errorf("%w: %q (routed to %q)", ErrDriverNotFound, target, name)
		}
		return d, nil
	}

	// Fallback: use the first registered driver (deterministic).
	if e.defaultDriver != "" {
		if d, ok := e.drivers[e.defaultDriver]; ok {
			return d, nil
		}
	}

	return nil, fmt.Errorf("%w: %q", ErrNoDrivers, target)
}

// GetDriver returns the driver with the given name.
func (e *Engine) GetDriver(name string) (Driver, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	d, ok := e.drivers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrDriverNotFound, name)
	}
	return d, nil
}

// Drivers returns a snapshot of all registered driver names.
func (e *Engine) Drivers() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	names := make([]string, 0, len(e.drivers))
	for n := range e.drivers {
		names = append(names, n)
	}
	return names
}

// Routes returns a snapshot of all explicit target→driver route mappings.
func (e *Engine) Routes() map[string]string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	snapshot := make(map[string]string, len(e.routes))
	for k, v := range e.routes {
		snapshot[k] = v
	}
	return snapshot
}

func isWriteAction(action Action) bool {
	return action == ActionInsert || action == ActionBatchInsert ||
		action == ActionUpdate || action == ActionDelete
}

// inferOrGetSchema returns the registered schema for a target, or infers
// one from the query's document/documents.
func (e *Engine) inferOrGetSchema(query OQLQuery) *CollectionSchema {
	// Check registered schemas first
	if schema, ok := e.registry.Get(query.Target); ok {
		return &schema
	}

	// Infer from document
	if len(query.Document) > 0 {
		s := InferSchema(query.Target, query.Document)
		return &s
	}
	if len(query.Documents) > 0 {
		// Merge fields from all documents
		merged := make(map[string]interface{})
		for _, doc := range query.Documents {
			for k, v := range doc {
				merged[k] = v
			}
		}
		s := InferSchema(query.Target, merged)
		return &s
	}

	return nil
}

func errorResponse(err error, code string) *OmniJSON {
	return &OmniJSON{
		Data: []map[string]interface{}{},
		Meta: OmniMeta{Total: 0, Returned: 0},
		Error: &OmniError{
			Code:    code,
			Message: err.Error(),
		},
	}
}
