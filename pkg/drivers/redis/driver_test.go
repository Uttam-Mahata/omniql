package redis_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	drvredis "github.com/Uttam-Mahata/omniql/pkg/drivers/redis"
	goredis "github.com/redis/go-redis/v9"
)

func redisClient(t *testing.T) *goredis.Client {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set; skipping Redis integration tests")
	}
	opts, err := goredis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse redis url: %v", err)
	}
	return goredis.NewClient(opts)
}

func redisDriver(t *testing.T) (*drvredis.Driver, func()) {
	t.Helper()
	client := redisClient(t)
	ctx := context.Background()

	// Flush test namespace to start clean.
	keys, _ := client.Keys(ctx, "testproducts:*").Result()
	if len(keys) > 0 {
		client.Del(ctx, keys...)
	}

	drv := drvredis.NewFromClient(client)
	return drv, func() {
		keys, _ := client.Keys(ctx, "testproducts:*").Result()
		if len(keys) > 0 {
			client.Del(ctx, keys...)
		}
		client.Close()
	}
}

func TestRedisDriver_Name(t *testing.T) {
	drv, cleanup := redisDriver(t)
	defer cleanup()
	if drv.Name() != "redis" {
		t.Errorf("expected name=redis, got %q", drv.Name())
	}
}

func TestRedisDriver_Ping(t *testing.T) {
	drv, cleanup := redisDriver(t)
	defer cleanup()
	if err := drv.Ping(context.Background()); err != nil {
		t.Errorf("ping failed: %v", err)
	}
}

func TestRedisDriver_InsertAndFind(t *testing.T) {
	drv, cleanup := redisDriver(t)
	defer cleanup()
	ctx := context.Background()

	_, affected, err := drv.Execute(ctx, core.OQLQuery{
		Target: "testproducts",
		Action: core.ActionInsert,
		Document: map[string]interface{}{
			"id": "laptop", "name": "Laptop", "price": 999.99,
		},
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if affected != 1 {
		t.Errorf("affected: want 1, got %d", affected)
	}

	rows, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: "testproducts",
		Action: core.ActionFind,
		Filter: core.Filter{"id": "laptop"},
	})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if total != 1 {
		t.Errorf("total: want 1, got %d", total)
	}
	if len(rows) != 1 {
		t.Fatalf("rows: want 1, got %d", len(rows))
	}
	if rows[0]["name"] != "Laptop" {
		t.Errorf("name: want Laptop, got %v", rows[0]["name"])
	}
}

func TestRedisDriver_FindAll(t *testing.T) {
	drv, cleanup := redisDriver(t)
	defer cleanup()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		drv.Execute(ctx, core.OQLQuery{
			Target: "testproducts",
			Action: core.ActionInsert,
			Document: map[string]interface{}{
				"id": fmt.Sprintf("item%d", i), "name": fmt.Sprintf("Item%d", i),
			},
		})
	}

	rows, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: "testproducts",
		Action: core.ActionFind,
	})
	if err != nil {
		t.Fatalf("find all: %v", err)
	}
	if total != 3 {
		t.Errorf("total: want 3, got %d", total)
	}
	if len(rows) != 3 {
		t.Errorf("rows: want 3, got %d", len(rows))
	}
}

func TestRedisDriver_Update(t *testing.T) {
	drv, cleanup := redisDriver(t)
	defer cleanup()
	ctx := context.Background()

	drv.Execute(ctx, core.OQLQuery{
		Target:   "testproducts",
		Action:   core.ActionInsert,
		Document: map[string]interface{}{"id": "w1", "name": "Widget", "price": 5.0},
	})

	_, affected, err := drv.Execute(ctx, core.OQLQuery{
		Target:   "testproducts",
		Action:   core.ActionUpdate,
		Filter:   core.Filter{"id": "w1"},
		Document: map[string]interface{}{"price": 9.99},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if affected != 1 {
		t.Errorf("affected: want 1, got %d", affected)
	}

	rows, _, err := drv.Execute(ctx, core.OQLQuery{
		Target: "testproducts",
		Action: core.ActionFind,
		Filter: core.Filter{"id": "w1"},
	})
	if err != nil {
		t.Fatalf("find after update: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows: want 1, got %d", len(rows))
	}
	if rows[0]["price"] != 9.99 {
		t.Errorf("price: want 9.99, got %v", rows[0]["price"])
	}
}

func TestRedisDriver_Delete(t *testing.T) {
	drv, cleanup := redisDriver(t)
	defer cleanup()
	ctx := context.Background()

	drv.Execute(ctx, core.OQLQuery{
		Target:   "testproducts",
		Action:   core.ActionInsert,
		Document: map[string]interface{}{"id": "trash", "name": "Trash"},
	})

	_, affected, err := drv.Execute(ctx, core.OQLQuery{
		Target: "testproducts",
		Action: core.ActionDelete,
		Filter: core.Filter{"id": "trash"},
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if affected != 1 {
		t.Errorf("affected: want 1, got %d", affected)
	}

	_, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: "testproducts",
		Action: core.ActionFind,
		Filter: core.Filter{"id": "trash"},
	})
	if err != nil {
		t.Fatalf("find after delete: %v", err)
	}
	if total != 0 {
		t.Errorf("total: want 0, got %d", total)
	}
}

func TestRedisDriver_Count(t *testing.T) {
	drv, cleanup := redisDriver(t)
	defer cleanup()
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		drv.Execute(ctx, core.OQLQuery{
			Target: "testproducts",
			Action: core.ActionInsert,
			Document: map[string]interface{}{
				"id": fmt.Sprintf("c%d", i), "name": "item",
			},
		})
	}

	_, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: "testproducts",
		Action: core.ActionCount,
	})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 4 {
		t.Errorf("total: want 4, got %d", total)
	}
}

func TestRedisDriver_BatchInsert(t *testing.T) {
	drv, cleanup := redisDriver(t)
	defer cleanup()
	ctx := context.Background()

	docs := []map[string]interface{}{
		{"id": "b1", "name": "A", "price": 1.0},
		{"id": "b2", "name": "B", "price": 2.0},
	}
	result, err := drv.BatchInsert(ctx, "testproducts", docs)
	if err != nil {
		t.Fatalf("BatchInsert: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected result")
	}

	_, total, err := drv.Execute(ctx, core.OQLQuery{
		Target: "testproducts",
		Action: core.ActionFind,
	})
	if err != nil {
		t.Fatalf("find after batch: %v", err)
	}
	if total != 2 {
		t.Errorf("total after batch: want 2, got %d", total)
	}
}

func TestRedisDriver_UnsupportedAction(t *testing.T) {
	drv, cleanup := redisDriver(t)
	defer cleanup()

	_, _, err := drv.Execute(context.Background(), core.OQLQuery{
		Target: "testproducts",
		Action: core.Action("MERGE"),
	})
	if err == nil {
		t.Fatal("expected error for unsupported action")
	}
}

func TestRedisDriver_InsertRequiresDocument(t *testing.T) {
	drv, cleanup := redisDriver(t)
	defer cleanup()

	_, _, err := drv.Execute(context.Background(), core.OQLQuery{
		Target: "testproducts",
		Action: core.ActionInsert,
	})
	if err == nil {
		t.Fatal("expected error for INSERT without document")
	}
}

func TestRedisDriver_UpdateRequiresIDFilter(t *testing.T) {
	drv, cleanup := redisDriver(t)
	defer cleanup()

	_, _, err := drv.Execute(context.Background(), core.OQLQuery{
		Target:   "testproducts",
		Action:   core.ActionUpdate,
		Filter:   core.Filter{"name": "foo"},
		Document: map[string]interface{}{"price": 1.0},
	})
	if err == nil {
		t.Fatal("expected error for UPDATE without id filter")
	}
}
