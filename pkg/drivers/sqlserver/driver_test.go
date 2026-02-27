package sqlserver_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	drvsqlserver "github.com/Uttam-Mahata/omniql/pkg/drivers/sqlserver"
	_ "github.com/microsoft/go-mssqldb"
)

func sqlserverDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("SQLSERVER_DSN")
	if dsn == "" {
		t.Skip("SQLSERVER_DSN not set; skipping SQL Server integration tests")
	}
	return dsn
}

func setupSQLServer(t *testing.T) (*drvsqlserver.Driver, func()) {
	t.Helper()
	dsn := sqlserverDSN(t)
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		t.Fatalf("sqlserver open: %v", err)
	}
	// Create test table.
	db.Exec(`DROP TABLE IF EXISTS products`)
	_, err = db.Exec(`CREATE TABLE products (
		id       INT IDENTITY(1,1) PRIMARY KEY,
		name     NVARCHAR(200) NOT NULL,
		price    FLOAT         NOT NULL DEFAULT 0,
		category NVARCHAR(100) NOT NULL DEFAULT '',
		stock    INT           NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("sqlserver create table: %v", err)
	}
	drv := drvsqlserver.NewFromDB(db)
	return drv, func() {
		db.Exec(`DROP TABLE IF EXISTS products`)
		db.Close()
	}
}

func TestSQLServerDriver_Name(t *testing.T) {
	sqlserverDSN(t)
	dsn := os.Getenv("SQLSERVER_DSN")
	drv, err := drvsqlserver.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer drv.Close()
	if drv.Name() != "sqlserver" {
		t.Errorf("expected name=sqlserver, got %q", drv.Name())
	}
}

func TestSQLServerDriver_InsertAndFind(t *testing.T) {
	drv, cleanup := setupSQLServer(t)
	defer cleanup()
	ctx := context.Background()

	_, affected, err := drv.Execute(ctx, core.OQLQuery{
		Target: "products",
		Action: core.ActionInsert,
		Document: map[string]interface{}{
			"name": "Laptop", "price": 999.99, "category": "electronics", "stock": 10,
		},
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if affected != 1 {
		t.Errorf("affected: want 1, got %d", affected)
	}

	_, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: "products",
		Action: core.ActionFind,
	})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if total != 1 {
		t.Errorf("total: want 1, got %d", total)
	}
}

func TestSQLServerDriver_Filter(t *testing.T) {
	drv, cleanup := setupSQLServer(t)
	defer cleanup()
	ctx := context.Background()

	seeds := []map[string]interface{}{
		{"name": "Laptop", "price": 999.99, "category": "electronics", "stock": 10},
		{"name": "Book", "price": 19.99, "category": "books", "stock": 100},
		{"name": "Phone", "price": 499.00, "category": "electronics", "stock": 25},
	}
	for _, doc := range seeds {
		if _, _, err := drv.Execute(ctx, core.OQLQuery{
			Target: "products", Action: core.ActionInsert, Document: doc,
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	_, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: "products",
		Action: core.ActionFind,
		Filter: core.Filter{"category": "electronics"},
	})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if total != 2 {
		t.Errorf("total: want 2, got %d", total)
	}
}

func TestSQLServerDriver_Update(t *testing.T) {
	drv, cleanup := setupSQLServer(t)
	defer cleanup()
	ctx := context.Background()

	drv.Execute(ctx, core.OQLQuery{
		Target:   "products",
		Action:   core.ActionInsert,
		Document: map[string]interface{}{"name": "Widget", "price": 5.00, "category": "misc", "stock": 1},
	})

	_, affected, err := drv.Execute(ctx, core.OQLQuery{
		Target:   "products",
		Action:   core.ActionUpdate,
		Filter:   core.Filter{"name": "Widget"},
		Document: map[string]interface{}{"price": 9.99},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if affected != 1 {
		t.Errorf("affected: want 1, got %d", affected)
	}
}

func TestSQLServerDriver_Delete(t *testing.T) {
	drv, cleanup := setupSQLServer(t)
	defer cleanup()
	ctx := context.Background()

	drv.Execute(ctx, core.OQLQuery{
		Target:   "products",
		Action:   core.ActionInsert,
		Document: map[string]interface{}{"name": "Trash", "price": 1.00, "category": "junk", "stock": 1},
	})

	_, affected, err := drv.Execute(ctx, core.OQLQuery{
		Target: "products",
		Action: core.ActionDelete,
		Filter: core.Filter{"name": "Trash"},
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if affected != 1 {
		t.Errorf("affected: want 1, got %d", affected)
	}
}

func TestSQLServerDriver_Count(t *testing.T) {
	drv, cleanup := setupSQLServer(t)
	defer cleanup()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		drv.Execute(ctx, core.OQLQuery{
			Target:   "products",
			Action:   core.ActionInsert,
			Document: map[string]interface{}{"name": "item", "price": float64(i), "category": "x", "stock": i},
		})
	}

	_, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: "products",
		Action: core.ActionCount,
	})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 3 {
		t.Errorf("total: want 3, got %d", total)
	}
}

func TestSQLServerDriver_Pagination(t *testing.T) {
	drv, cleanup := setupSQLServer(t)
	defer cleanup()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		drv.Execute(ctx, core.OQLQuery{
			Target:   "products",
			Action:   core.ActionInsert,
			Document: map[string]interface{}{"name": "item", "price": float64(i), "category": "x", "stock": i},
		})
	}

	rows, total, err := drv.Execute(ctx, core.OQLQuery{
		Target:  "products",
		Action:  core.ActionFind,
		Options: core.QueryOptions{Limit: 3, Skip: 2},
	})
	if err != nil {
		t.Fatalf("pagination: %v", err)
	}
	if total != 10 {
		t.Errorf("total: want 10, got %d", total)
	}
	if len(rows) != 3 {
		t.Errorf("rows: want 3, got %d", len(rows))
	}
}

func TestSQLServerDriver_BatchInsert(t *testing.T) {
	drv, cleanup := setupSQLServer(t)
	defer cleanup()
	ctx := context.Background()

	docs := []map[string]interface{}{
		{"name": "A", "price": 1.0, "category": "test", "stock": 1},
		{"name": "B", "price": 2.0, "category": "test", "stock": 2},
	}
	result, err := drv.BatchInsert(ctx, "products", docs)
	if err != nil {
		t.Fatalf("BatchInsert: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected result")
	}
}

func TestSQLServerDriver_UnsupportedAction(t *testing.T) {
	drv, cleanup := setupSQLServer(t)
	defer cleanup()

	_, _, err := drv.Execute(context.Background(), core.OQLQuery{
		Target: "products",
		Action: core.Action("MERGE"),
	})
	if err == nil {
		t.Fatal("expected error for unsupported action")
	}
}

func TestSQLServerDriver_UpdateRequiresFilter(t *testing.T) {
	drv, cleanup := setupSQLServer(t)
	defer cleanup()

	_, _, err := drv.Execute(context.Background(), core.OQLQuery{
		Target:   "products",
		Action:   core.ActionUpdate,
		Document: map[string]interface{}{"price": 1.0},
	})
	if err == nil {
		t.Fatal("expected error for UPDATE without filter")
	}
}

func TestSQLServerDriver_DeleteRequiresFilter(t *testing.T) {
	drv, cleanup := setupSQLServer(t)
	defer cleanup()

	_, _, err := drv.Execute(context.Background(), core.OQLQuery{
		Target: "products",
		Action: core.ActionDelete,
	})
	if err == nil {
		t.Fatal("expected error for DELETE without filter")
	}
}

func TestSQLServerDriver_Ping(t *testing.T) {
	drv, cleanup := setupSQLServer(t)
	defer cleanup()

	if err := drv.Ping(context.Background()); err != nil {
		t.Errorf("ping failed: %v", err)
	}
}
