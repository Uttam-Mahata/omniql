package core_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Uttam-Mahata/omniql/pkg/core"
)

// ---------------------------------------------------------------------------
// Mock driver used across tests
// ---------------------------------------------------------------------------

type mockDriver struct {
	name    string
	rows    []map[string]interface{}
	total   int64
	execErr error
	pingErr error
}

func (m *mockDriver) Name() string { return m.name }
func (m *mockDriver) Ping(_ context.Context) error {
	return m.pingErr
}
func (m *mockDriver) Close() error { return nil }
func (m *mockDriver) Execute(_ context.Context, _ core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if m.execErr != nil {
		return nil, 0, m.execErr
	}
	return m.rows, m.total, nil
}
func (m *mockDriver) BatchInsert(_ context.Context, _ string, docs []map[string]interface{}) ([]map[string]interface{}, error) {
	return []map[string]interface{}{{"count": int64(len(docs))}}, nil
}

// ---------------------------------------------------------------------------
// Engine tests
// ---------------------------------------------------------------------------

func TestEngine_Execute_NoDrivers(t *testing.T) {
	engine := core.NewEngine()
	result, err := engine.Execute(context.Background(), core.OQLQuery{
		Target: "users",
		Action: core.ActionFind,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error == nil {
		t.Fatal("expected an error in result, got nil")
	}
}

func TestEngine_Execute_WithDriver(t *testing.T) {
	engine := core.NewEngine()
	mock := &mockDriver{
		name:  "mock",
		rows:  []map[string]interface{}{{"id": 1, "name": "Alice"}},
		total: 1,
	}
	engine.RegisterDriver(mock)

	result, err := engine.Execute(context.Background(), core.OQLQuery{
		Target: "users",
		Action: core.ActionFind,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("unexpected result error: %v", result.Error)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result.Data))
	}
	if result.Meta.Driver != "mock" {
		t.Errorf("expected driver=mock, got %q", result.Meta.Driver)
	}
}

func TestEngine_Execute_DriverError(t *testing.T) {
	engine := core.NewEngine()
	mock := &mockDriver{
		name:    "mock",
		execErr: errors.New("connection refused"),
	}
	engine.RegisterDriver(mock)

	result, err := engine.Execute(context.Background(), core.OQLQuery{
		Target: "users",
		Action: core.ActionFind,
	})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Error == nil {
		t.Fatal("expected error in OmniJSON result")
	}
	if result.Error.Code != "EXECUTION_ERROR" {
		t.Errorf("expected code EXECUTION_ERROR, got %q", result.Error.Code)
	}
}

func TestEngine_Route(t *testing.T) {
	engine := core.NewEngine()

	mockA := &mockDriver{name: "driverA", rows: []map[string]interface{}{{"src": "A"}}, total: 1}
	mockB := &mockDriver{name: "driverB", rows: []map[string]interface{}{{"src": "B"}}, total: 1}
	engine.RegisterDriver(mockA)
	engine.RegisterDriver(mockB)

	// Route "orders" to driverB explicitly.
	engine.Route("orders", "driverB")

	result, err := engine.Execute(context.Background(), core.OQLQuery{
		Target: "orders",
		Action: core.ActionFind,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("unexpected result error: %v", result.Error)
	}
	if result.Meta.Driver != "driverB" {
		t.Errorf("expected driverB, got %q", result.Meta.Driver)
	}
}

func TestEngine_RegisterDriver_Overwrite(t *testing.T) {
	engine := core.NewEngine()
	mock1 := &mockDriver{name: "mydb", rows: []map[string]interface{}{{"v": 1}}, total: 1}
	mock2 := &mockDriver{name: "mydb", rows: []map[string]interface{}{{"v": 2}}, total: 1}

	engine.RegisterDriver(mock1)
	engine.RegisterDriver(mock2) // overwrites

	result, err := engine.Execute(context.Background(), core.OQLQuery{Target: "t", Action: core.ActionFind})
	if err != nil {
		t.Fatal(err)
	}
	if result.Data[0]["v"] != 2 {
		t.Errorf("expected v=2 (from mock2), got %v", result.Data[0]["v"])
	}
}

func TestEngine_Drivers(t *testing.T) {
	engine := core.NewEngine()
	engine.RegisterDriver(&mockDriver{name: "pg"})
	engine.RegisterDriver(&mockDriver{name: "mongo"})

	names := engine.Drivers()
	if len(names) != 2 {
		t.Errorf("expected 2 drivers, got %d", len(names))
	}
}

// ---------------------------------------------------------------------------
// Schema tests
// ---------------------------------------------------------------------------

func TestSchemaRegistry_Validate_NoSchema(t *testing.T) {
	reg := core.NewSchemaRegistry()
	err := reg.Validate(core.OQLQuery{Target: "items", Action: core.ActionFind}, false)
	if err != nil {
		t.Errorf("expected no error in non-strict mode, got %v", err)
	}
}

func TestSchemaRegistry_Validate_StrictMode(t *testing.T) {
	reg := core.NewSchemaRegistry()
	err := reg.Validate(core.OQLQuery{Target: "items", Action: core.ActionFind}, true)
	if err == nil {
		t.Fatal("expected error in strict mode with no schema registered")
	}
	if !errors.Is(err, core.ErrUnknownTarget) {
		t.Errorf("expected ErrUnknownTarget, got %v", err)
	}
}

func TestSchemaRegistry_Validate_UnknownField(t *testing.T) {
	reg := core.NewSchemaRegistry()
	reg.Register(core.CollectionSchema{
		Name: "products",
		Fields: map[string]core.FieldSchema{
			"name":  {Type: core.FieldTypeString},
			"price": {Type: core.FieldTypeFloat},
		},
	})

	err := reg.Validate(core.OQLQuery{
		Target: "products",
		Action: core.ActionFind,
		Filter: core.Filter{"nonexistent": "value"},
	}, false)
	if err == nil {
		t.Fatal("expected error for unknown filter field")
	}
}

func TestSchemaRegistry_Validate_ValidQuery(t *testing.T) {
	reg := core.NewSchemaRegistry()
	reg.Register(core.CollectionSchema{
		Name: "products",
		Fields: map[string]core.FieldSchema{
			"name":  {Type: core.FieldTypeString},
			"price": {Type: core.FieldTypeFloat},
		},
	})

	err := reg.Validate(core.OQLQuery{
		Target: "products",
		Action: core.ActionFind,
		Filter: core.Filter{
			"price": map[string]interface{}{"$lt": 500},
		},
	}, false)
	if err != nil {
		t.Errorf("expected no validation error, got %v", err)
	}
}

func TestEngine_WithStrictSchema(t *testing.T) {
	engine := core.NewEngine(core.WithStrictSchema())
	engine.RegisterDriver(&mockDriver{name: "mock", rows: nil, total: 0})

	result, err := engine.Execute(context.Background(), core.OQLQuery{
		Target: "nonexistent",
		Action: core.ActionFind,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Error == nil || result.Error.Code != "SCHEMA_ERROR" {
		t.Errorf("expected SCHEMA_ERROR, got %v", result.Error)
	}
}

// ---------------------------------------------------------------------------
// OmniJSON / OmniError tests
// ---------------------------------------------------------------------------

func TestOmniError_Error(t *testing.T) {
	e := &core.OmniError{Code: "NOT_FOUND", Message: "record not found"}
	want := "NOT_FOUND: record not found"
	if e.Error() != want {
		t.Errorf("got %q, want %q", e.Error(), want)
	}
}
