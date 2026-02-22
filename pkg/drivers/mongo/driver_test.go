package mongo

import (
	"context"
	"testing"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestMongoDriver(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("InsertAndFind", func(mt *mtest.T) {
		d := New(mt.Client, "testdb")

		// 1. Insert
		mt.AddMockResponses(mtest.CreateSuccessResponse())

		doc := map[string]interface{}{"name": "Alice", "age": 30}
		res, count, err := d.Execute(context.Background(), core.OQLQuery{
			Target:   "users",
			Action:   core.ActionInsert,
			Document: doc,
		})
		if err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count 1, got %d", count)
		}
		if len(res) != 1 || res[0]["_id"] == nil {
			t.Errorf("Expected _id in result, got %v", res)
		}

		// 2. Find
		// Mock CountDocuments response (aggregate with $count)
		mt.AddMockResponses(mtest.CreateCursorResponse(
			0,
			"testdb.users",
			mtest.FirstBatch,
			bson.D{{"n", int32(1)}},
		))

		// Mock Find response
		objID := primitive.NewObjectID()
		mt.AddMockResponses(mtest.CreateCursorResponse(
			0,
			"testdb.users",
			mtest.FirstBatch,
			bson.D{{"_id", objID}, {"name", "Alice"}, {"age", 30}},
		))

		res, count, err = d.Execute(context.Background(), core.OQLQuery{
			Target: "users",
			Action: core.ActionFind,
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count 1, got %d", count)
		}
		if len(res) != 1 {
			t.Fatalf("Expected 1 row, got %d", len(res))
		}
		if res[0]["name"] != "Alice" {
			t.Errorf("Expected name Alice, got %v", res[0]["name"])
		}
	})

	mt.Run("FindWithFilter", func(mt *mtest.T) {
		d := New(mt.Client, "testdb")

		// Mock CountDocuments response
		mt.AddMockResponses(mtest.CreateCursorResponse(
			0,
			"testdb.products",
			mtest.FirstBatch,
			bson.D{{"n", int32(1)}},
		))

		// Mock Find response
		mt.AddMockResponses(mtest.CreateCursorResponse(
			0,
			"testdb.products",
			mtest.FirstBatch,
			bson.D{{"_id", primitive.NewObjectID()}, {"price", 100}},
		))

		res, count, err := d.Execute(context.Background(), core.OQLQuery{
			Target: "products",
			Action: core.ActionFind,
			Filter: core.Filter{"price": map[string]interface{}{"$lt": 200}},
		})
		if err != nil {
			t.Fatalf("FindWithFilter failed: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count 1, got %d", count)
		}
		if len(res) != 1 {
			t.Errorf("Expected 1 row, got %d", len(res))
		}
	})

	mt.Run("FindWithSort", func(mt *mtest.T) {
		d := New(mt.Client, "testdb")

		// Mock CountDocuments response
		mt.AddMockResponses(mtest.CreateCursorResponse(
			0,
			"testdb.products",
			mtest.FirstBatch,
			bson.D{{"n", int32(2)}},
		))

		// Mock Find response
		mt.AddMockResponses(mtest.CreateCursorResponse(
			0,
			"testdb.products",
			mtest.FirstBatch,
			bson.D{{"_id", primitive.NewObjectID()}, {"price", 20}},
			bson.D{{"_id", primitive.NewObjectID()}, {"price", 10}},
		))

		res, _, err := d.Execute(context.Background(), core.OQLQuery{
			Target: "products",
			Action: core.ActionFind,
			Options: core.QueryOptions{
				Sort: map[string]int{"price": -1},
			},
		})
		if err != nil {
			t.Fatalf("FindWithSort failed: %v", err)
		}
		if len(res) != 2 {
			t.Errorf("Expected 2 rows, got %d", len(res))
		}
	})

	mt.Run("FindWithProjection", func(mt *mtest.T) {
		d := New(mt.Client, "testdb")

		// Mock CountDocuments response
		mt.AddMockResponses(mtest.CreateCursorResponse(
			0,
			"testdb.users",
			mtest.FirstBatch,
			bson.D{{"n", int32(1)}},
		))

		// Mock Find response
		mt.AddMockResponses(mtest.CreateCursorResponse(
			0,
			"testdb.users",
			mtest.FirstBatch,
			bson.D{{"_id", primitive.NewObjectID()}, {"name", "Bob"}},
		))

		res, _, err := d.Execute(context.Background(), core.OQLQuery{
			Target: "users",
			Action: core.ActionFind,
			Options: core.QueryOptions{
				Fields: map[string]interface{}{"name": 1},
			},
		})
		if err != nil {
			t.Fatalf("FindWithProjection failed: %v", err)
		}
		if len(res) != 1 {
			t.Errorf("Expected 1 row, got %d", len(res))
		}
	})

	mt.Run("Count", func(mt *mtest.T) {
		d := New(mt.Client, "testdb")

		// Mock CountDocuments response
		mt.AddMockResponses(mtest.CreateCursorResponse(
			0,
			"testdb.users",
			mtest.FirstBatch,
			bson.D{{"n", int32(5)}},
		))

		res, count, err := d.Execute(context.Background(), core.OQLQuery{
			Target: "users",
			Action: core.ActionCount,
		})
		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}
		if count != 5 {
			t.Errorf("Expected count 5, got %d", count)
		}
		if len(res) != 1 || res[0]["count"] != int64(5) {
			t.Errorf("Expected result [{'count': 5}], got %v", res)
		}
	})

	mt.Run("Update", func(mt *mtest.T) {
		d := New(mt.Client, "testdb")

		// Mock UpdateMany response
		mt.AddMockResponses(bson.D{
			{"ok", 1},
			{"n", int32(1)},
			{"nModified", int32(1)},
		})

		res, count, err := d.Execute(context.Background(), core.OQLQuery{
			Target:   "users",
			Action:   core.ActionUpdate,
			Filter:   core.Filter{"name": "Alice"},
			Document: map[string]interface{}{"age": 31},
		})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count 1, got %d", count)
		}
		if len(res) != 0 {
			t.Errorf("Expected empty result for update, got %v", res)
		}
	})

	mt.Run("Delete", func(mt *mtest.T) {
		d := New(mt.Client, "testdb")

		// Mock DeleteMany response
		mt.AddMockResponses(bson.D{
			{"ok", 1},
			{"n", int32(1)},
		})

		res, count, err := d.Execute(context.Background(), core.OQLQuery{
			Target: "users",
			Action: core.ActionDelete,
			Filter: core.Filter{"name": "Alice"},
		})
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected count 1, got %d", count)
		}
		if len(res) != 0 {
			t.Errorf("Expected empty result for delete, got %v", res)
		}
	})
}
