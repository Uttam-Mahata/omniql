package sqlite_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	"github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

func TestEnsureTarget(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	drv := sqlite.NewFromDB(db)
	ctx := context.Background()

	schema := &core.CollectionSchema{
		Name: "users",
		Fields: map[string]core.FieldSchema{
			"username": {Type: core.FieldTypeString},
			"score":    {Type: core.FieldTypeInt},
		},
	}

	if err := drv.EnsureTarget(ctx, "users", schema); err != nil {
		t.Fatalf("EnsureTarget failed: %v", err)
	}

	// Verify table created
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='users'")
	if err != nil {
		t.Fatalf("query master failed: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("table users not found")
	}
}

func TestEnsureColumns(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	drv := sqlite.NewFromDB(db)
	ctx := context.Background()

	// Initial table
	_, err = db.Exec("CREATE TABLE items (id INTEGER PRIMARY KEY AUTOINCREMENT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Insert doc with new columns
	doc := map[string]interface{}{
		"name":  "Widget",
		"price": 10.5,
	}

	// insert() calls ensureColumns()
	_, _, err = drv.Execute(ctx, core.OQLQuery{
		Target:   "items",
		Action:   core.ActionInsert,
		Document: doc,
	})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// Verify columns added
	rows, err := db.Query("PRAGMA table_info(items)")
	if err != nil {
		t.Fatalf("pragma failed: %v", err)
	}
	defer rows.Close()

	cols := make(map[string]bool)
	var cid int
	var name, ctype string
	var notnull, pk int
	var dflt interface{}

	for rows.Next() {
		rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
		cols[name] = true
	}

	if !cols["name"] {
		t.Error("column 'name' missing")
	}
	if !cols["price"] {
		t.Error("column 'price' missing")
	}
}
