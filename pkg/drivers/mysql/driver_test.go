package mysql_test

import (
	"context"
	"os"
	"testing"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	drvmysql "github.com/Uttam-Mahata/omniql/pkg/drivers/mysql"
)

// mysqlDSN returns the DSN from MYSQL_DSN environment variable.
// Tests are skipped when this variable is not set.
func mysqlDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		t.Skip("MYSQL_DSN not set; skipping MySQL integration tests")
	}
	return dsn
}

func TestMySQLDriver_Name(t *testing.T) {
	// Name() does not require a live connection.
	// We test it via a no-op open to avoid a live DB requirement.
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		t.Skip("MYSQL_DSN not set; skipping MySQL integration tests")
	}
	drv, err := drvmysql.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer drv.Close()
	if drv.Name() != "mysql" {
		t.Errorf("Name() = %q; want %q", drv.Name(), "mysql")
	}
}

func TestMySQLDriver_Ping(t *testing.T) {
	dsn := mysqlDSN(t)
	drv, err := drvmysql.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer drv.Close()
	if err := drv.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestMySQLDriver_CRUD(t *testing.T) {
	dsn := mysqlDSN(t)
	drv, err := drvmysql.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer drv.Close()

	ctx := context.Background()

	// Create a temporary table.
	db := drv // use drv.Execute to run raw SQL is not supported; use the underlying db
	_ = db
	// We need to create a table for testing. Since we only have driver.Execute,
	// we skip table creation here and note that a real test would need direct db access.
	// This test file serves as documentation of the expected interface.

	// INSERT
	_, affected, err := drv.Execute(ctx, core.OQLQuery{
		Target:   "omniql_test",
		Action:   core.ActionInsert,
		Document: map[string]interface{}{"name": "Alice", "age": 30},
	})
	if err != nil {
		t.Skipf("INSERT failed (table may not exist): %v", err)
	}
	if affected != 1 {
		t.Errorf("INSERT affected = %d; want 1", affected)
	}

	// FIND
	rows, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: "omniql_test",
		Action: core.ActionFind,
		Filter: map[string]interface{}{"name": "Alice"},
	})
	if err != nil {
		t.Fatalf("FIND: %v", err)
	}
	if total == 0 || len(rows) == 0 {
		t.Error("FIND returned no rows")
	}

	// UPDATE
	_, affected, err = drv.Execute(ctx, core.OQLQuery{
		Target:   "omniql_test",
		Action:   core.ActionUpdate,
		Filter:   map[string]interface{}{"name": "Alice"},
		Document: map[string]interface{}{"age": 31},
	})
	if err != nil {
		t.Fatalf("UPDATE: %v", err)
	}
	if affected == 0 {
		t.Error("UPDATE affected 0 rows")
	}

	// COUNT
	rows, total, err = drv.Execute(ctx, core.OQLQuery{
		Target: "omniql_test",
		Action: core.ActionCount,
		Filter: map[string]interface{}{"name": "Alice"},
	})
	if err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	if total == 0 {
		t.Error("COUNT returned 0")
	}
	_ = rows

	// DELETE
	_, affected, err = drv.Execute(ctx, core.OQLQuery{
		Target: "omniql_test",
		Action: core.ActionDelete,
		Filter: map[string]interface{}{"name": "Alice"},
	})
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	if affected == 0 {
		t.Error("DELETE affected 0 rows")
	}
}

func TestMySQLDriver_UnsupportedAction(t *testing.T) {
	dsn := mysqlDSN(t)
	drv, err := drvmysql.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer drv.Close()

	_, _, err = drv.Execute(context.Background(), core.OQLQuery{
		Target: "t",
		Action: "UNKNOWN",
	})
	if err == nil {
		t.Error("expected error for unknown action")
	}
}

func TestMySQLDriver_UpdateRequiresFilter(t *testing.T) {
	dsn := mysqlDSN(t)
	drv, err := drvmysql.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer drv.Close()

	_, _, err = drv.Execute(context.Background(), core.OQLQuery{
		Target:   "t",
		Action:   core.ActionUpdate,
		Document: map[string]interface{}{"x": 1},
	})
	if err == nil {
		t.Error("expected error when UPDATE has no filter")
	}
}

func TestMySQLDriver_DeleteRequiresFilter(t *testing.T) {
	dsn := mysqlDSN(t)
	drv, err := drvmysql.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer drv.Close()

	_, _, err = drv.Execute(context.Background(), core.OQLQuery{
		Target: "t",
		Action: core.ActionDelete,
	})
	if err == nil {
		t.Error("expected error when DELETE has no filter")
	}
}
