package core

import (
	"context"
	"fmt"
	"testing"
)

type MockDriver struct {
	name string
	data map[string][]map[string]interface{}
}

func NewMockDriver(name string) *MockDriver {
	return &MockDriver{name: name, data: make(map[string][]map[string]interface{})}
}

func (d *MockDriver) Name() string { return d.name }
func (d *MockDriver) Ping(ctx context.Context) error { return nil }
func (d *MockDriver) Close() error { return nil }

func (d *MockDriver) Execute(ctx context.Context, query OQLQuery) ([]map[string]interface{}, int64, error) {
	if query.Action == ActionFind {
		rows := d.data[query.Target]
		skip := query.Options.Skip
		limit := query.Options.Limit

		if skip >= len(rows) {
			return []map[string]interface{}{}, int64(len(rows)), nil
		}
		res := rows[skip:]
		if limit > 0 && limit < len(res) {
			res = res[:limit]
		}
		return res, int64(len(rows)), nil
	}
	if query.Action == ActionInsert {
		d.data[query.Target] = append(d.data[query.Target], query.Document)
		return []map[string]interface{}{{"id": 1}}, 1, nil
	}
	return nil, 0, fmt.Errorf("unsupported action")
}

func (d *MockDriver) BatchInsert(ctx context.Context, target string, docs []map[string]interface{}) ([]map[string]interface{}, error) {
	d.data[target] = append(d.data[target], docs...)
	return []map[string]interface{}{{"count": int64(len(docs))}}, nil
}

func (d *MockDriver) ListTargets(_ context.Context) ([]string, error) {
	targets := make([]string, 0, len(d.data))
	for t := range d.data {
		targets = append(targets, t)
	}
	return targets, nil
}

func (d *MockDriver) EnsureTarget(ctx context.Context, target string, schema *CollectionSchema) error {
	// No-op for mock
	return nil
}

func TestMigrate(t *testing.T) {
	src := NewMockDriver("source_db")
	dest := NewMockDriver("dest_db")

	// Seed source
	for i := 0; i < 25; i++ {
		src.data["users"] = append(src.data["users"], map[string]interface{}{
			"id":   i,
			"name": fmt.Sprintf("user_%d", i),
		})
	}

	e := NewEngine()
	e.RegisterDriver(src)
	e.RegisterDriver(dest)
	e.Route("users", "source_db")
	e.Route("users_migrated", "dest_db")

	// Migrate with batch size 10
	res, err := e.Migrate(context.Background(), "users", "users_migrated", MigrateOptions{BatchSize: 10})
	if err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	if res.RecordsRead != 25 {
		t.Errorf("expected 25 read, got %d", res.RecordsRead)
	}
	if res.RecordsWritten != 25 {
		t.Errorf("expected 25 written, got %d", res.RecordsWritten)
	}
	if res.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", res.Errors)
	}

	// Verify destination data
	if len(dest.data["users_migrated"]) != 25 {
		t.Errorf("expected 25 records in dest, got %d", len(dest.data["users_migrated"]))
	}
}

func TestMigrateDryRun(t *testing.T) {
	src := NewMockDriver("source_db_dry")
	dest := NewMockDriver("dest_db_dry")

	// Seed source
	for i := 0; i < 5; i++ {
		src.data["items"] = append(src.data["items"], map[string]interface{}{"id": i})
	}

	e := NewEngine()
	e.RegisterDriver(src)
	e.RegisterDriver(dest)
	e.Route("items", "source_db_dry")
	e.Route("items_migrated", "dest_db_dry")

	res, err := e.Migrate(context.Background(), "items", "items_migrated", MigrateOptions{BatchSize: 2, DryRun: true})
	if err != nil {
		t.Fatalf("migrate dry run failed: %v", err)
	}

	if res.RecordsRead != 5 {
		t.Errorf("expected 5 read, got %d", res.RecordsRead)
	}
	if res.RecordsWritten != 0 {
		t.Errorf("expected 0 written (dry run), got %d", res.RecordsWritten)
	}

	if len(dest.data["items_migrated"]) != 0 {
		t.Errorf("expected 0 records in dest, got %d", len(dest.data["items_migrated"]))
	}
}

func TestMigrateAll(t *testing.T) {
	src := NewMockDriver("source_db_all")
	dest := NewMockDriver("dest_db_all")

	// Seed source with multiple tables
	src.data["table1"] = []map[string]interface{}{{"id": 1}, {"id": 2}}
	src.data["table2"] = []map[string]interface{}{{"val": "a"}, {"val": "b"}, {"val": "c"}}

	e := NewEngine()
	e.RegisterDriver(src)
	e.RegisterDriver(dest)

	// MigrateAll
	res, err := e.MigrateAll(context.Background(), "source_db_all", "dest_db_all", MigrateOptions{BatchSize: 2})
	if err != nil {
		t.Fatalf("migrate all failed: %v", err)
	}

	// table1 (2 recs) + table2 (3 recs) = 5 total
	if res.RecordsRead != 5 {
		t.Errorf("expected 5 read, got %d", res.RecordsRead)
	}
	if res.RecordsWritten != 5 {
		t.Errorf("expected 5 written, got %d", res.RecordsWritten)
	}

	// Verify destination has data
	if len(dest.data["table1"]) != 2 {
		t.Errorf("expected 2 records in table1, got %d", len(dest.data["table1"]))
	}
	if len(dest.data["table2"]) != 3 {
		t.Errorf("expected 3 records in table2, got %d", len(dest.data["table2"]))
	}
}
