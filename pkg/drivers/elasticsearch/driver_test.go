package elasticsearch_test

import (
	"context"
	"os"
	"testing"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	drvels "github.com/Uttam-Mahata/omniql/pkg/drivers/elasticsearch"
)

func requireAddr(t *testing.T) string {
	t.Helper()
	addr := os.Getenv("ELASTICSEARCH_ADDR")
	if addr == "" {
		t.Skip("ELASTICSEARCH_ADDR not set; skipping Elasticsearch integration tests")
	}
	return addr
}

func TestElasticsearchPing(t *testing.T) {
	addr := requireAddr(t)
	drv, err := drvels.New(addr)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer drv.Close()

	if err := drv.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestElasticsearchInsertAndFind(t *testing.T) {
	addr := requireAddr(t)
	drv, err := drvels.New(addr)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer drv.Close()

	ctx := context.Background()
	target := "omniql_test_insert"

	// Insert
	insertQ := core.OQLQuery{
		Target: target,
		Action: core.ActionInsert,
		Document: map[string]interface{}{
			"name":  "Alice",
			"score": 42,
		},
	}
	rows, n, err := drv.Execute(ctx, insertQ)
	if err != nil {
		t.Fatalf("INSERT: %v", err)
	}
	if n != 1 {
		t.Fatalf("INSERT: expected 1 row, got %d", n)
	}
	t.Logf("Inserted: %v", rows)
}

func TestElasticsearchCount(t *testing.T) {
	addr := requireAddr(t)
	drv, err := drvels.New(addr)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer drv.Close()

	ctx := context.Background()
	target := "omniql_test_count"

	// Insert a document first
	_, _, err = drv.Execute(ctx, core.OQLQuery{
		Target:   target,
		Action:   core.ActionInsert,
		Document: map[string]interface{}{"name": "Bob", "value": 10},
	})
	if err != nil {
		t.Fatalf("INSERT for count test: %v", err)
	}

	// Count
	rows, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: target,
		Action: core.ActionCount,
		Filter: core.Filter{},
	})
	if err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	if total < 1 {
		t.Fatalf("COUNT: expected >= 1, got %d", total)
	}
	t.Logf("COUNT result: %v (total=%d)", rows, total)
}

func TestElasticsearchBatchInsert(t *testing.T) {
	addr := requireAddr(t)
	drv, err := drvels.New(addr)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer drv.Close()

	ctx := context.Background()
	target := "omniql_test_batch"

	docs := []map[string]interface{}{
		{"name": "item1", "value": 1},
		{"name": "item2", "value": 2},
		{"name": "item3", "value": 3},
	}

	rows, err := drv.BatchInsert(ctx, target, docs)
	if err != nil {
		t.Fatalf("BatchInsert: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("BatchInsert: expected at least one result row")
	}
	t.Logf("BatchInsert result: %v", rows)
}

func TestElasticsearchName(t *testing.T) {
	addr := requireAddr(t)
	drv, err := drvels.New(addr)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if drv.Name() != "elasticsearch" {
		t.Fatalf("Name: expected %q, got %q", "elasticsearch", drv.Name())
	}
}
