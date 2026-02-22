package sqlite_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	"github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite"
)

// setupDB creates an in-memory SQLite database with a test table.
func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE products (
		id    INTEGER PRIMARY KEY AUTOINCREMENT,
		name  TEXT    NOT NULL,
		price REAL    NOT NULL,
		category TEXT
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func TestSQLiteDriver_InsertAndFind(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	drv := sqlite.NewFromDB(db)
	ctx := context.Background()

	// Insert a row.
	_, affected, err := drv.Execute(ctx, core.OQLQuery{
		Target: "products",
		Action: core.ActionInsert,
		Document: map[string]interface{}{
			"name":     "Laptop",
			"price":    999.99,
			"category": "electronics",
		},
	})
	if err != nil {
		t.Fatalf("insert error: %v", err)
	}
	if affected != 1 {
		t.Errorf("expected 1 affected row, got %d", affected)
	}

	// Find all rows.
	rows, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: "products",
		Action: core.ActionFind,
	})
	if err != nil {
		t.Fatalf("find error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1, got %d", total)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["name"] != "Laptop" {
		t.Errorf("expected name=Laptop, got %v", rows[0]["name"])
	}
}

func TestSQLiteDriver_FindWithFilter(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	drv := sqlite.NewFromDB(db)
	ctx := context.Background()

	// Seed some rows.
	seeds := []map[string]interface{}{
		{"name": "Laptop", "price": 999.99, "category": "electronics"},
		{"name": "Book", "price": 19.99, "category": "books"},
		{"name": "Phone", "price": 499.00, "category": "electronics"},
	}
	for _, doc := range seeds {
		if _, _, err := drv.Execute(ctx, core.OQLQuery{
			Target:   "products",
			Action:   core.ActionInsert,
			Document: doc,
		}); err != nil {
			t.Fatalf("seed error: %v", err)
		}
	}

	// Find with $lt filter.
	rows, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: "products",
		Action: core.ActionFind,
		Filter: core.Filter{
			"price": map[string]interface{}{"$lt": 500},
		},
	})
	if err != nil {
		t.Fatalf("find error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total=2, got %d", total)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
}

func TestSQLiteDriver_FindWithLimit(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	drv := sqlite.NewFromDB(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if _, _, err := drv.Execute(ctx, core.OQLQuery{
			Target:   "products",
			Action:   core.ActionInsert,
			Document: map[string]interface{}{"name": "item", "price": float64(i * 10), "category": "misc"},
		}); err != nil {
			t.Fatal(err)
		}
	}

	rows, total, err := drv.Execute(ctx, core.OQLQuery{
		Target:  "products",
		Action:  core.ActionFind,
		Options: core.QueryOptions{Limit: 3},
	})
	if err != nil {
		t.Fatalf("find error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total=5, got %d", total)
	}
	if len(rows) != 3 {
		t.Errorf("expected 3 rows (limit), got %d", len(rows))
	}
}

func TestSQLiteDriver_Update(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	drv := sqlite.NewFromDB(db)
	ctx := context.Background()

	if _, _, err := drv.Execute(ctx, core.OQLQuery{
		Target:   "products",
		Action:   core.ActionInsert,
		Document: map[string]interface{}{"name": "Widget", "price": 5.00, "category": "misc"},
	}); err != nil {
		t.Fatal(err)
	}

	_, affected, err := drv.Execute(ctx, core.OQLQuery{
		Target:   "products",
		Action:   core.ActionUpdate,
		Filter:   core.Filter{"name": "Widget"},
		Document: map[string]interface{}{"price": 9.99},
	})
	if err != nil {
		t.Fatalf("update error: %v", err)
	}
	if affected != 1 {
		t.Errorf("expected 1 affected, got %d", affected)
	}
}

func TestSQLiteDriver_Delete(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	drv := sqlite.NewFromDB(db)
	ctx := context.Background()

	if _, _, err := drv.Execute(ctx, core.OQLQuery{
		Target:   "products",
		Action:   core.ActionInsert,
		Document: map[string]interface{}{"name": "Trash", "price": 1.00, "category": "junk"},
	}); err != nil {
		t.Fatal(err)
	}

	_, affected, err := drv.Execute(ctx, core.OQLQuery{
		Target: "products",
		Action: core.ActionDelete,
		Filter: core.Filter{"name": "Trash"},
	})
	if err != nil {
		t.Fatalf("delete error: %v", err)
	}
	if affected != 1 {
		t.Errorf("expected 1 affected, got %d", affected)
	}

	rows, _, err := drv.Execute(ctx, core.OQLQuery{
		Target: "products",
		Action: core.ActionFind,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows after delete, got %d", len(rows))
	}
}

func TestSQLiteDriver_Count(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	drv := sqlite.NewFromDB(db)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		if _, _, err := drv.Execute(ctx, core.OQLQuery{
			Target:   "products",
			Action:   core.ActionInsert,
			Document: map[string]interface{}{"name": "item", "price": float64(i), "category": "x"},
		}); err != nil {
			t.Fatal(err)
		}
	}

	rows, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: "products",
		Action: core.ActionCount,
	})
	if err != nil {
		t.Fatalf("count error: %v", err)
	}
	if total != 4 {
		t.Errorf("expected total=4, got %d", total)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row from COUNT, got %d", len(rows))
	}
}

func TestSQLiteDriver_Ping(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	drv := sqlite.NewFromDB(db)
	if err := drv.Ping(context.Background()); err != nil {
		t.Errorf("ping failed: %v", err)
	}
}

func TestSQLiteDriver_Name(t *testing.T) {
	db := setupDB(t)
	defer db.Close()
	drv := sqlite.NewFromDB(db)
	if drv.Name() != "sqlite" {
		t.Errorf("expected name=sqlite, got %q", drv.Name())
	}
}

func TestSQLiteDriver_UnsupportedAction(t *testing.T) {
	db := setupDB(t)
	defer db.Close()

	drv := sqlite.NewFromDB(db)
	_, _, err := drv.Execute(context.Background(), core.OQLQuery{
		Target: "products",
		Action: core.Action("MERGE"),
	})
	if err == nil {
		t.Fatal("expected error for unsupported action")
	}
}
