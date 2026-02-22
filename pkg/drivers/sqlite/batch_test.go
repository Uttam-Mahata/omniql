package sqlite

import (
	"context"
	"testing"

	"github.com/Uttam-Mahata/omniql/pkg/core"
)

func TestSQLiteDriver_BatchInsert(t *testing.T) {
	drv, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}
	defer drv.Close()

	ctx := context.Background()
	_, err = drv.db.ExecContext(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	docs := []map[string]interface{}{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35},
	}

	rows, err := drv.BatchInsert(ctx, "users", docs)
	if err != nil {
		t.Fatalf("BatchInsert failed: %v", err)
	}

	if len(rows) != 1 || rows[0]["count"] != int64(3) {
		t.Errorf("expected count 3, got %v", rows)
	}

	// Verify data
	query := core.OQLQuery{
		Target: "users",
		Action: core.ActionFind,
	}
	results, total, err := drv.find(ctx, query)
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}

	if total != 3 {
		t.Errorf("expected 3 records, got %d", total)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 result rows, got %d", len(results))
	}
}
