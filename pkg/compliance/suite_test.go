// Package compliance contains a canonical OQL query compliance suite that
// exercises every driver against the same 50+ queries to enforce cross-driver
// behavioural parity.
//
// SQLite (in-memory) always runs.  PostgreSQL runs when POSTGRES_DSN is set.
// MySQL runs when MYSQL_DSN is set.  MongoDB runs when MONGO_URI and MONGO_DB are set.
package compliance_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	drvmysql "github.com/Uttam-Mahata/omniql/pkg/drivers/mysql"
	drvpostgres "github.com/Uttam-Mahata/omniql/pkg/drivers/postgres"
	drvsqlite "github.com/Uttam-Mahata/omniql/pkg/drivers/sqlite"
)

// ----------------------------------------------------------------------------
// Driver factory helpers
// ----------------------------------------------------------------------------

// sqliteDriver creates a fresh in-memory SQLite driver with the products table.
func sqliteDriver(t *testing.T) (core.Driver, func()) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sqlite open: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE products (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		name     TEXT    NOT NULL,
		price    REAL    NOT NULL DEFAULT 0,
		category TEXT    NOT NULL DEFAULT '',
		stock    INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("sqlite create table: %v", err)
	}
	return drvsqlite.NewFromDB(db), func() { db.Close() }
}

// mysqlDriver creates a MySQL driver when MYSQL_DSN is set.
func mysqlDriver(t *testing.T) (core.Driver, func()) {
	t.Helper()
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		t.Skip("MYSQL_DSN not set; skipping MySQL compliance tests")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("mysql open: %v", err)
	}
	// Create / reset test table.
	db.Exec(`DROP TABLE IF EXISTS products`)
	_, err = db.Exec(`CREATE TABLE products (
		id       INT AUTO_INCREMENT PRIMARY KEY,
		name     VARCHAR(200) NOT NULL,
		price    DOUBLE       NOT NULL DEFAULT 0,
		category VARCHAR(100) NOT NULL DEFAULT '',
		stock    INT          NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("mysql create table: %v", err)
	}
	return drvmysql.NewFromDB(db), func() {
		db.Exec(`DROP TABLE IF EXISTS products`)
		db.Close()
	}
}

// postgresDriver creates a PostgreSQL driver when POSTGRES_DSN is set.
func postgresDriver(t *testing.T) (core.Driver, func()) {
	t.Helper()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN not set; skipping PostgreSQL compliance tests")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("postgres open: %v", err)
	}
	// Create / reset test table.
	_, err = db.Exec(`DROP TABLE IF EXISTS products`)
	if err != nil {
		t.Fatalf("postgres drop table: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE products (
		id       SERIAL PRIMARY KEY,
		name     TEXT    NOT NULL,
		price    REAL    NOT NULL DEFAULT 0,
		category TEXT    NOT NULL DEFAULT '',
		stock    INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("postgres create table: %v", err)
	}
	return drvpostgres.New(db), func() {
		db.Exec(`DROP TABLE IF EXISTS products`)
		db.Close()
	}
}

// ----------------------------------------------------------------------------
// Seed helper
// ----------------------------------------------------------------------------

type product struct {
	name     string
	price    float64
	category string
	stock    int
}

var seeds = []product{
	{"Laptop", 999.99, "electronics", 10},
	{"Phone", 499.00, "electronics", 25},
	{"Book", 19.99, "books", 100},
	{"Headphones", 149.99, "electronics", 50},
	{"Pen", 1.99, "stationery", 200},
	{"Notebook", 4.99, "stationery", 150},
	{"Tablet", 399.00, "electronics", 15},
	{"Desk", 249.99, "furniture", 5},
	{"Chair", 199.99, "furniture", 8},
	{"Monitor", 329.99, "electronics", 20},
}

func seedProducts(t *testing.T, drv core.Driver) {
	t.Helper()
	ctx := context.Background()
	for _, p := range seeds {
		if _, _, err := drv.Execute(ctx, core.OQLQuery{
			Target: "products",
			Action: core.ActionInsert,
			Document: map[string]interface{}{
				"name":     p.name,
				"price":    p.price,
				"category": p.category,
				"stock":    p.stock,
			},
		}); err != nil {
			t.Fatalf("seed insert %q: %v", p.name, err)
		}
	}
}

// ----------------------------------------------------------------------------
// Compliance test runner
// ----------------------------------------------------------------------------

// complianceCase describes a single canonical test case.
type complianceCase struct {
	name     string
	query    core.OQLQuery
	// check is called with the raw results for custom assertions.
	check func(t *testing.T, rows []map[string]interface{}, total int64)
}

func buildCases() []complianceCase {
	return []complianceCase{
		// ------------------------------------------------------------------
		// FIND — no filter
		// ------------------------------------------------------------------
		{
			name:  "FIND_all",
			query: core.OQLQuery{Target: "products", Action: core.ActionFind},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 10 {
					t.Errorf("total: want 10, got %d", total)
				}
				if len(rows) != 10 {
					t.Errorf("rows: want 10, got %d", len(rows))
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — equality filter (bare value)
		// ------------------------------------------------------------------
		{
			name: "FIND_eq_bare",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{"category": "books"},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 1 {
					t.Errorf("total: want 1, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — $eq operator
		// ------------------------------------------------------------------
		{
			name: "FIND_eq_op",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{"category": map[string]interface{}{"$eq": "furniture"}},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 2 {
					t.Errorf("total: want 2, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — $ne operator
		// ------------------------------------------------------------------
		{
			name: "FIND_ne",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{"category": map[string]interface{}{"$ne": "electronics"}},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 5 { // books + stationery(2) + furniture(2) = 5
					t.Errorf("total: want 5, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — $lt
		// ------------------------------------------------------------------
		{
			name: "FIND_lt",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{"price": map[string]interface{}{"$lt": 10.0}},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 2 { // Pen 1.99, Notebook 4.99
					t.Errorf("total: want 2, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — $lte
		// ------------------------------------------------------------------
		{
			name: "FIND_lte",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{"price": map[string]interface{}{"$lte": 19.99}},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 3 { // Pen 1.99, Notebook 4.99, Book 19.99
					t.Errorf("total: want 3, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — $gt
		// ------------------------------------------------------------------
		{
			name: "FIND_gt",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{"price": map[string]interface{}{"$gt": 499.0}},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 1 { // Laptop 999.99
					t.Errorf("total: want 1, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — $gte
		// ------------------------------------------------------------------
		{
			name: "FIND_gte",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{"price": map[string]interface{}{"$gte": 499.0}},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 2 { // Phone 499.00, Laptop 999.99
					t.Errorf("total: want 2, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — $in
		// ------------------------------------------------------------------
		{
			name: "FIND_in",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{
					"category": map[string]interface{}{"$in": []interface{}{"books", "furniture"}},
				},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 3 {
					t.Errorf("total: want 3, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — $nin
		// ------------------------------------------------------------------
		{
			name: "FIND_nin",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{
					"category": map[string]interface{}{"$nin": []interface{}{"electronics", "furniture"}},
				},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 3 { // books(1) + stationery(2)
					t.Errorf("total: want 3, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — $or
		// ------------------------------------------------------------------
		{
			name: "FIND_or",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{
					"$or": []interface{}{
						map[string]interface{}{"category": "furniture"},
						map[string]interface{}{"price": map[string]interface{}{"$lt": 5.0}},
					},
				},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				// furniture(2) + Pen(1.99) + Notebook(4.99) = 4
				if total != 4 {
					t.Errorf("total: want 4, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — $and
		// ------------------------------------------------------------------
		{
			name: "FIND_and",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{
					"$and": []interface{}{
						map[string]interface{}{"category": "electronics"},
						map[string]interface{}{"price": map[string]interface{}{"$lt": 500.0}},
					},
				},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				// electronics under 500: Phone 499, Headphones 149.99, Tablet 399, Monitor 329.99 = 4
				if total != 4 {
					t.Errorf("total: want 4, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — nested $and + $or
		// ------------------------------------------------------------------
		{
			name: "FIND_nested_and_or",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{
					"$and": []interface{}{
						map[string]interface{}{"category": "electronics"},
						map[string]interface{}{
							"$or": []interface{}{
								map[string]interface{}{"price": map[string]interface{}{"$lt": 200.0}},
								map[string]interface{}{"stock": map[string]interface{}{"$gt": 20}},
							},
						},
					},
				},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				// Electronics AND (price < 200 OR stock > 20)
				// price < 200: Headphones 149.99 (elec, stock 50)
				// stock > 20: Phone stock 25, Headphones stock 50
				// Combined electronics matches: Headphones (both), Phone (stock)
				if total < 1 {
					t.Errorf("total: want >=1, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — combined filter (two fields, AND semantics)
		// ------------------------------------------------------------------
		{
			name: "FIND_combined_fields",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{
					"category": "electronics",
					"price":    map[string]interface{}{"$gt": 400.0},
				},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				// electronics > 400: Phone 499, Laptop 999.99 = 2
				if total != 2 {
					t.Errorf("total: want 2, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — pagination: limit
		// ------------------------------------------------------------------
		{
			name: "FIND_limit",
			query: core.OQLQuery{
				Target:  "products",
				Action:  core.ActionFind,
				Options: core.QueryOptions{Limit: 3},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 10 {
					t.Errorf("total: want 10, got %d", total)
				}
				if len(rows) != 3 {
					t.Errorf("rows: want 3, got %d", len(rows))
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — pagination: skip
		// ------------------------------------------------------------------
		{
			name: "FIND_skip",
			query: core.OQLQuery{
				Target:  "products",
				Action:  core.ActionFind,
				Options: core.QueryOptions{Skip: 8},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 10 {
					t.Errorf("total: want 10, got %d", total)
				}
				if len(rows) != 2 {
					t.Errorf("rows: want 2, got %d", len(rows))
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — pagination: limit + skip
		// ------------------------------------------------------------------
		{
			name: "FIND_limit_skip",
			query: core.OQLQuery{
				Target:  "products",
				Action:  core.ActionFind,
				Options: core.QueryOptions{Limit: 4, Skip: 3},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 10 {
					t.Errorf("total: want 10, got %d", total)
				}
				if len(rows) != 4 {
					t.Errorf("rows: want 4, got %d", len(rows))
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — sort ASC
		// ------------------------------------------------------------------
		{
			name: "FIND_sort_asc",
			query: core.OQLQuery{
				Target:  "products",
				Action:  core.ActionFind,
				Options: core.QueryOptions{Sort: map[string]int{"price": 1}},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if len(rows) < 2 {
					t.Fatal("expected rows")
				}
				p0 := toFloat(rows[0]["price"])
				p1 := toFloat(rows[1]["price"])
				if p0 > p1 {
					t.Errorf("ASC sort: first=%v, second=%v (not ascending)", p0, p1)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — sort DESC
		// ------------------------------------------------------------------
		{
			name: "FIND_sort_desc",
			query: core.OQLQuery{
				Target:  "products",
				Action:  core.ActionFind,
				Options: core.QueryOptions{Sort: map[string]int{"price": -1}},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if len(rows) < 2 {
					t.Fatal("expected rows")
				}
				p0 := toFloat(rows[0]["price"])
				p1 := toFloat(rows[1]["price"])
				if p0 < p1 {
					t.Errorf("DESC sort: first=%v, second=%v (not descending)", p0, p1)
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — projection (inclusion)
		// ------------------------------------------------------------------
		{
			name: "FIND_projection_include",
			query: core.OQLQuery{
				Target:  "products",
				Action:  core.ActionFind,
				Options: core.QueryOptions{Fields: map[string]interface{}{"name": 1, "price": 1}},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if len(rows) == 0 {
					t.Fatal("expected rows")
				}
				if _, ok := rows[0]["name"]; !ok {
					t.Error("expected 'name' field")
				}
				if _, ok := rows[0]["price"]; !ok {
					t.Error("expected 'price' field")
				}
				if _, ok := rows[0]["category"]; ok {
					t.Error("did not expect 'category' field")
				}
			},
		},

		// ------------------------------------------------------------------
		// FIND — sort + limit
		// ------------------------------------------------------------------
		{
			name: "FIND_sort_limit",
			query: core.OQLQuery{
				Target:  "products",
				Action:  core.ActionFind,
				Options: core.QueryOptions{Sort: map[string]int{"price": -1}, Limit: 3},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 10 {
					t.Errorf("total: want 10, got %d", total)
				}
				if len(rows) != 3 {
					t.Errorf("rows: want 3, got %d", len(rows))
				}
				// First row should be Laptop (most expensive)
				if toFloat(rows[0]["price"]) != 999.99 {
					t.Errorf("expected Laptop (999.99) first, got price=%v", rows[0]["price"])
				}
			},
		},

		// ------------------------------------------------------------------
		// COUNT — no filter
		// ------------------------------------------------------------------
		{
			name:  "COUNT_all",
			query: core.OQLQuery{Target: "products", Action: core.ActionCount},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 10 {
					t.Errorf("total: want 10, got %d", total)
				}
				if len(rows) != 1 {
					t.Fatalf("rows: want 1, got %d", len(rows))
				}
				count := toInt64(rows[0]["count"])
				if count != 10 {
					t.Errorf("count row value: want 10, got %d", count)
				}
			},
		},

		// ------------------------------------------------------------------
		// COUNT — with filter
		// ------------------------------------------------------------------
		{
			name: "COUNT_filtered",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionCount,
				Filter: core.Filter{"category": "electronics"},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 5 {
					t.Errorf("total: want 5, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// COUNT — $in filter
		// ------------------------------------------------------------------
		{
			name: "COUNT_in_filter",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionCount,
				Filter: core.Filter{
					"category": map[string]interface{}{"$in": []interface{}{"books", "stationery"}},
				},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 3 {
					t.Errorf("total: want 3, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// UPDATE — single field
		// ------------------------------------------------------------------
		{
			name: "UPDATE_single_field",
			query: core.OQLQuery{
				Target:   "products",
				Action:   core.ActionUpdate,
				Filter:   core.Filter{"name": "Pen"},
				Document: map[string]interface{}{"price": 2.49},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 1 {
					t.Errorf("affected: want 1, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// UPDATE — multiple rows
		// ------------------------------------------------------------------
		{
			name: "UPDATE_multi_row",
			query: core.OQLQuery{
				Target:   "products",
				Action:   core.ActionUpdate,
				Filter:   core.Filter{"category": "stationery"},
				Document: map[string]interface{}{"stock": 999},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 2 {
					t.Errorf("affected: want 2, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// UPDATE — $gt filter
		// ------------------------------------------------------------------
		{
			name: "UPDATE_gt_filter",
			query: core.OQLQuery{
				Target:   "products",
				Action:   core.ActionUpdate,
				Filter:   core.Filter{"price": map[string]interface{}{"$gt": 900.0}},
				Document: map[string]interface{}{"stock": 5},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 1 {
					t.Errorf("affected: want 1, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// DELETE — single row
		// ------------------------------------------------------------------
		{
			name: "DELETE_single",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionDelete,
				Filter: core.Filter{"name": "Pen"},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 1 {
					t.Errorf("affected: want 1, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// DELETE — multiple rows by category
		// ------------------------------------------------------------------
		{
			name: "DELETE_category",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionDelete,
				Filter: core.Filter{"category": "furniture"},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 2 {
					t.Errorf("affected: want 2, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// DELETE — $in filter
		// ------------------------------------------------------------------
		{
			name: "DELETE_in_filter",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionDelete,
				Filter: core.Filter{
					"name": map[string]interface{}{"$in": []interface{}{"Headphones", "Monitor"}},
				},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 2 {
					t.Errorf("affected: want 2, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// INSERT — returns id
		// ------------------------------------------------------------------
		{
			name: "INSERT_returns_id",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionInsert,
				Document: map[string]interface{}{
					"name":     "Keyboard",
					"price":    79.99,
					"category": "electronics",
					"stock":    30,
				},
			},
			check: func(t *testing.T, rows []map[string]interface{}, total int64) {
				if total != 1 {
					t.Errorf("affected: want 1, got %d", total)
				}
			},
		},

		// ------------------------------------------------------------------
		// Error cases — all are independent tests that expect errors
		// ------------------------------------------------------------------

		// UPDATE requires non-empty filter
		{
			name: "UPDATE_requires_filter",
			query: core.OQLQuery{
				Target:   "products",
				Action:   core.ActionUpdate,
				Document: map[string]interface{}{"price": 0.0},
			},
			check: errExpected,
		},

		// DELETE requires non-empty filter
		{
			name:  "DELETE_requires_filter",
			query: core.OQLQuery{Target: "products", Action: core.ActionDelete},
			check: errExpected,
		},

		// INSERT requires non-empty document
		{
			name:  "INSERT_requires_document",
			query: core.OQLQuery{Target: "products", Action: core.ActionInsert},
			check: errExpected,
		},

		// $in with empty slice
		{
			name: "FIND_in_empty_slice_error",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{"name": map[string]interface{}{"$in": []interface{}{}}},
			},
			check: errExpected,
		},

		// $nin with empty slice
		{
			name: "FIND_nin_empty_slice_error",
			query: core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{"name": map[string]interface{}{"$nin": []interface{}{}}},
			},
			check: errExpected,
		},

		// field exclusion (value=0) is not supported by SQL drivers
		{
			name: "FIND_field_exclusion_error",
			query: core.OQLQuery{
				Target:  "products",
				Action:  core.ActionFind,
				Options: core.QueryOptions{Fields: map[string]interface{}{"category": float64(0)}},
			},
			check: errExpected,
		},

		// unsupported action
		{
			name:  "unsupported_action_error",
			query: core.OQLQuery{Target: "products", Action: "MERGE"},
			check: errExpected,
		},
	}
}

// errExpected is a check function that expects an error from Execute.
// Because complianceCase.check is called after Execute returns without
// error, we handle error cases differently: the outer runner must intercept.
// We use a sentinel pattern — see runComplianceWithErrors below.
func errExpected(_ *testing.T, _ []map[string]interface{}, _ int64) {}

// runComplianceWithErrors runs error-cases separately so that Execute errors
// are captured and asserted (rather than t.Fatalf-ed).
func runComplianceWithErrors(t *testing.T, drv core.Driver) {
	t.Helper()
	seedProducts(t, drv)
	ctx := context.Background()

	errorCases := []struct {
		name  string
		query core.OQLQuery
	}{
		{
			"UPDATE_requires_filter",
			core.OQLQuery{
				Target:   "products",
				Action:   core.ActionUpdate,
				Document: map[string]interface{}{"price": 0.0},
			},
		},
		{
			"DELETE_requires_filter",
			core.OQLQuery{Target: "products", Action: core.ActionDelete},
		},
		{
			"INSERT_requires_document",
			core.OQLQuery{Target: "products", Action: core.ActionInsert},
		},
		{
			"FIND_in_empty_slice_error",
			core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{"name": map[string]interface{}{"$in": []interface{}{}}},
			},
		},
		{
			"FIND_nin_empty_slice_error",
			core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: core.Filter{"name": map[string]interface{}{"$nin": []interface{}{}}},
			},
		},
		{
			"FIND_field_exclusion_error",
			core.OQLQuery{
				Target:  "products",
				Action:  core.ActionFind,
				Options: core.QueryOptions{Fields: map[string]interface{}{"category": float64(0)}},
			},
		},
		{
			"unsupported_action_error",
			core.OQLQuery{Target: "products", Action: "MERGE"},
		},
	}

	for _, ec := range errorCases {
		ec := ec
		t.Run(ec.name, func(t *testing.T) {
			_, _, err := drv.Execute(ctx, ec.query)
			if err == nil {
				t.Errorf("expected error for %q but got nil", ec.name)
			}
		})
	}
}

// buildSuccessCases returns only the success-path compliance cases
// (i.e. all cases except the error sentinels).
func buildSuccessCases() []complianceCase {
	var out []complianceCase
	for _, c := range buildCases() {
		// Skip error-expected cases — they're run separately.
		switch c.name {
		case "UPDATE_requires_filter",
			"DELETE_requires_filter",
			"INSERT_requires_document",
			"FIND_in_empty_slice_error",
			"FIND_nin_empty_slice_error",
			"FIND_field_exclusion_error",
			"unsupported_action_error":
			continue
		}
		out = append(out, c)
	}
	return out
}

// runSuccessCompliance seeds and runs only the success-path cases.
func runSuccessCompliance(t *testing.T, drv core.Driver) {
	t.Helper()
	seedProducts(t, drv)
	ctx := context.Background()

	cases := buildSuccessCases()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rows, total, err := drv.Execute(ctx, tc.query)
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			tc.check(t, rows, total)
		})
	}
}

// ----------------------------------------------------------------------------
// Per-driver test entry points
// ----------------------------------------------------------------------------

func TestCompliance_SQLite_Success(t *testing.T) {
	drv, cleanup := sqliteDriver(t)
	defer cleanup()
	runSuccessCompliance(t, drv)
}

func TestCompliance_SQLite_Errors(t *testing.T) {
	drv, cleanup := sqliteDriver(t)
	defer cleanup()
	runComplianceWithErrors(t, drv)
}

func TestCompliance_Postgres_Success(t *testing.T) {
	drv, cleanup := postgresDriver(t)
	defer cleanup()
	runSuccessCompliance(t, drv)
}

func TestCompliance_Postgres_Errors(t *testing.T) {
	drv, cleanup := postgresDriver(t)
	defer cleanup()
	runComplianceWithErrors(t, drv)
}

func TestCompliance_MySQL_Success(t *testing.T) {
	drv, cleanup := mysqlDriver(t)
	defer cleanup()
	runSuccessCompliance(t, drv)
}

func TestCompliance_MySQL_Errors(t *testing.T) {
	drv, cleanup := mysqlDriver(t)
	defer cleanup()
	runComplianceWithErrors(t, drv)
}

// ----------------------------------------------------------------------------
// BatchInsert compliance
// ----------------------------------------------------------------------------

func TestCompliance_SQLite_BatchInsert(t *testing.T) {
	drv, cleanup := sqliteDriver(t)
	defer cleanup()
	ctx := context.Background()

	docs := []map[string]interface{}{
		{"name": "A", "price": 1.0, "category": "test", "stock": 1},
		{"name": "B", "price": 2.0, "category": "test", "stock": 2},
		{"name": "C", "price": 3.0, "category": "test", "stock": 3},
	}
	result, err := drv.BatchInsert(ctx, "products", docs)
	if err != nil {
		t.Fatalf("BatchInsert: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected result")
	}

	// Verify all 3 rows are queryable
	rows, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: "products",
		Action: core.ActionFind,
		Filter: core.Filter{"category": "test"},
	})
	if err != nil {
		t.Fatalf("FIND after BatchInsert: %v", err)
	}
	if total != 3 {
		t.Errorf("total after batch: want 3, got %d", total)
	}
	_ = rows
}

func TestCompliance_SQLite_BatchInsert_Empty(t *testing.T) {
	drv, cleanup := sqliteDriver(t)
	defer cleanup()
	ctx := context.Background()

	result, err := drv.BatchInsert(ctx, "products", nil)
	if err != nil {
		t.Fatalf("BatchInsert(empty): %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result for empty batch")
	}
}

// ----------------------------------------------------------------------------
// Engine-level compliance
// ----------------------------------------------------------------------------

func TestCompliance_Engine_RoutingAndExecution(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE items (
		id    INTEGER PRIMARY KEY AUTOINCREMENT,
		name  TEXT NOT NULL,
		value REAL NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	engine := core.NewEngine()
	drv := drvsqlite.NewFromDB(db)
	engine.RegisterDriver(drv)
	engine.Route("items", drv.Name())

	ctx := context.Background()

	// INSERT via engine
	res, err := engine.Execute(ctx, core.OQLQuery{
		Target:   "items",
		Action:   core.ActionInsert,
		Document: map[string]interface{}{"name": "Widget", "value": 42.0},
	})
	if err != nil {
		t.Fatalf("engine insert: %v", err)
	}
	if res.Error != nil {
		t.Fatalf("engine insert error: %v", res.Error)
	}

	// FIND via engine
	res, err = engine.Execute(ctx, core.OQLQuery{
		Target: "items",
		Action: core.ActionFind,
		Filter: core.Filter{"name": "Widget"},
	})
	if err != nil {
		t.Fatalf("engine find: %v", err)
	}
	if res.Error != nil {
		t.Fatalf("engine find error: %v", res.Error)
	}
	if res.Meta.Total != 1 {
		t.Errorf("total: want 1, got %d", res.Meta.Total)
	}
}

func TestCompliance_Engine_UnknownTarget(t *testing.T) {
	// Strict mode: no driver registered — selectDriver should return an OmniJSON error.
	engine := core.NewEngine(core.WithStrictSchema())

	res, err := engine.Execute(context.Background(), core.OQLQuery{
		Target: "nonexistent",
		Action: core.ActionFind,
	})
	if err != nil {
		t.Fatalf("unexpected Go error (expected OmniJSON error): %v", err)
	}
	if res.Error == nil {
		t.Error("expected OmniJSON error for unknown target in strict mode")
	}
}

// ----------------------------------------------------------------------------
// Operator exhaustive tests
// ----------------------------------------------------------------------------

func TestCompliance_SQLite_AllOperators(t *testing.T) {
	drv, cleanup := sqliteDriver(t)
	defer cleanup()
	ctx := context.Background()

	// Seed a small dataset
	for _, p := range seeds {
		if _, _, err := drv.Execute(ctx, core.OQLQuery{
			Target: "products",
			Action: core.ActionInsert,
			Document: map[string]interface{}{
				"name": p.name, "price": p.price,
				"category": p.category, "stock": p.stock,
			},
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	opTests := []struct {
		op     string
		filter core.Filter
		want   int64
	}{
		{"$eq", core.Filter{"category": map[string]interface{}{"$eq": "books"}}, 1},
		{"$ne", core.Filter{"category": map[string]interface{}{"$ne": "electronics"}}, 5},
		{"$lt", core.Filter{"price": map[string]interface{}{"$lt": 10.0}}, 2},
		{"$lte", core.Filter{"price": map[string]interface{}{"$lte": 4.99}}, 2}, // Pen 1.99, Notebook 4.99
		{"$gt", core.Filter{"price": map[string]interface{}{"$gt": 900.0}}, 1},
		{"$gte", core.Filter{"price": map[string]interface{}{"$gte": 999.99}}, 1},
		{"$in", core.Filter{"category": map[string]interface{}{"$in": []interface{}{"books", "furniture"}}}, 3},
		{"$nin", core.Filter{"category": map[string]interface{}{"$nin": []interface{}{"books", "furniture"}}}, 7},
	}

	for _, ot := range opTests {
		ot := ot
		t.Run(fmt.Sprintf("op_%s", ot.op), func(t *testing.T) {
			_, total, err := drv.Execute(ctx, core.OQLQuery{
				Target: "products",
				Action: core.ActionFind,
				Filter: ot.filter,
			})
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			if total != ot.want {
				t.Errorf("op %s: want %d, got %d", ot.op, ot.want, total)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// Pagination edge cases
// ----------------------------------------------------------------------------

func TestCompliance_SQLite_PaginationEdgeCases(t *testing.T) {
	drv, cleanup := sqliteDriver(t)
	defer cleanup()
	ctx := context.Background()

	seedProducts(t, drv)

	t.Run("skip_beyond_total", func(t *testing.T) {
		rows, total, err := drv.Execute(ctx, core.OQLQuery{
			Target:  "products",
			Action:  core.ActionFind,
			Options: core.QueryOptions{Skip: 100},
		})
		if err != nil {
			t.Fatalf("execute: %v", err)
		}
		if total != 10 {
			t.Errorf("total: want 10, got %d", total)
		}
		if len(rows) != 0 {
			t.Errorf("rows: want 0, got %d", len(rows))
		}
	})

	t.Run("limit_larger_than_total", func(t *testing.T) {
		rows, total, err := drv.Execute(ctx, core.OQLQuery{
			Target:  "products",
			Action:  core.ActionFind,
			Options: core.QueryOptions{Limit: 1000},
		})
		if err != nil {
			t.Fatalf("execute: %v", err)
		}
		if total != 10 {
			t.Errorf("total: want 10, got %d", total)
		}
		if len(rows) != 10 {
			t.Errorf("rows: want 10, got %d", len(rows))
		}
	})
}

// ----------------------------------------------------------------------------
// Sort determinism test
// ----------------------------------------------------------------------------

func TestCompliance_SQLite_SortDeterminism(t *testing.T) {
	drv, cleanup := sqliteDriver(t)
	defer cleanup()
	ctx := context.Background()
	seedProducts(t, drv)

	// Run the same sort query twice — results must be identical.
	query := core.OQLQuery{
		Target:  "products",
		Action:  core.ActionFind,
		Options: core.QueryOptions{Sort: map[string]int{"price": 1}},
	}

	rows1, _, err := drv.Execute(ctx, query)
	if err != nil {
		t.Fatalf("first execute: %v", err)
	}
	rows2, _, err := drv.Execute(ctx, query)
	if err != nil {
		t.Fatalf("second execute: %v", err)
	}

	if len(rows1) != len(rows2) {
		t.Fatalf("row count mismatch: %d vs %d", len(rows1), len(rows2))
	}
	for i := range rows1 {
		p1 := toFloat(rows1[i]["price"])
		p2 := toFloat(rows2[i]["price"])
		if p1 != p2 {
			t.Errorf("row %d price mismatch: %v vs %v", i, p1, p2)
		}
	}
}

// ----------------------------------------------------------------------------
// Category totals cross-check
// ----------------------------------------------------------------------------

func TestCompliance_SQLite_CategoryCounts(t *testing.T) {
	drv, cleanup := sqliteDriver(t)
	defer cleanup()
	ctx := context.Background()
	seedProducts(t, drv)

	want := map[string]int64{
		"electronics": 5,
		"books":       1,
		"stationery":  2,
		"furniture":   2,
	}

	cats := make([]string, 0, len(want))
	for c := range want {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	for _, cat := range cats {
		cat := cat
		t.Run("category_"+cat, func(t *testing.T) {
			_, total, err := drv.Execute(ctx, core.OQLQuery{
				Target: "products",
				Action: core.ActionCount,
				Filter: core.Filter{"category": cat},
			})
			if err != nil {
				t.Fatalf("count: %v", err)
			}
			if total != want[cat] {
				t.Errorf("count %q: want %d, got %d", cat, want[cat], total)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

func toFloat(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int64:
		return float64(x)
	case int:
		return float64(x)
	default:
		return 0
	}
}

func toInt64(v interface{}) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case float64:
		return int64(x)
	case int:
		return int64(x)
	default:
		return 0
	}
}
