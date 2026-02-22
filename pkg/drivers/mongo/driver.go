// Package mongo provides an OmniQL driver for MongoDB.
// It translates OQL queries into MongoDB filter documents and executes them
// via the official mongo-driver.
package mongo

import (
	"context"
	"fmt"
	"sort"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const driverName = "mongo"

// Driver is the OmniQL MongoDB adapter.
type Driver struct {
	client *mongo.Client
	dbName string
}

// New creates a new MongoDB Driver.
func New(client *mongo.Client, dbName string) *Driver {
	return &Driver{client: client, dbName: dbName}
}

// Name satisfies core.Driver.
func (d *Driver) Name() string { return driverName }

// Ping satisfies core.Driver.
func (d *Driver) Ping(ctx context.Context) error {
	return d.client.Ping(ctx, nil)
}

// Close satisfies core.Driver.
func (d *Driver) Close() error {
	return d.client.Disconnect(context.Background())
}

// Execute translates an OQLQuery into MongoDB operations and returns results.
func (d *Driver) Execute(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	coll := d.client.Database(d.dbName).Collection(query.Target)

	switch query.Action {
	case core.ActionFind, core.ActionCount:
		return d.find(ctx, coll, query)
	case core.ActionInsert:
		return d.insert(ctx, coll, query)
	case core.ActionUpdate:
		return d.update(ctx, coll, query)
	case core.ActionDelete:
		return d.delete(ctx, coll, query)
	default:
		return nil, 0, fmt.Errorf("mongo: unsupported action %q", query.Action)
	}
}

func (d *Driver) find(ctx context.Context, coll *mongo.Collection, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	filter, err := buildFilter(query.Filter)
	if err != nil {
		return nil, 0, err
	}

	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("mongo count: %w", err)
	}

	if query.Action == core.ActionCount {
		return []map[string]interface{}{{"count": total}}, total, nil
	}

	opts := options.Find()
	if query.Options.Limit > 0 {
		opts.SetLimit(int64(query.Options.Limit))
	}
	if query.Options.Skip > 0 {
		opts.SetSkip(int64(query.Options.Skip))
	}
	if len(query.Options.Sort) > 0 {
		keys := make([]string, 0, len(query.Options.Sort))
		for k := range query.Options.Sort {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		sortDoc := bson.D{}
		for _, k := range keys {
			sortDoc = append(sortDoc, bson.E{Key: k, Value: query.Options.Sort[k]})
		}
		opts.SetSort(sortDoc)
	}
	if len(query.Options.Fields) > 0 {
		opts.SetProjection(bson.M(query.Options.Fields))
	}

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("mongo find: %w", err)
	}
	defer cursor.Close(ctx)

	var rows []map[string]interface{}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, 0, fmt.Errorf("mongo decode: %w", err)
	}
	return rows, total, nil
}

func (d *Driver) insert(ctx context.Context, coll *mongo.Collection, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Document) == 0 {
		return nil, 0, fmt.Errorf("mongo: INSERT requires a non-empty document")
	}
	insertResult, err := coll.InsertOne(ctx, query.Document)
	if err != nil {
		return nil, 0, fmt.Errorf("mongo insert: %w", err)
	}
	return []map[string]interface{}{{"_id": insertResult.InsertedID}}, 1, nil
}

func (d *Driver) update(ctx context.Context, coll *mongo.Collection, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Document) == 0 {
		return nil, 0, fmt.Errorf("mongo: UPDATE requires a non-empty document")
	}
	filter, err := buildFilter(query.Filter)
	if err != nil {
		return nil, 0, err
	}
	update := bson.M{"$set": query.Document}

	result, err := coll.UpdateMany(ctx, filter, update)
	if err != nil {
		return nil, 0, fmt.Errorf("mongo update: %w", err)
	}
	return []map[string]interface{}{}, result.ModifiedCount, nil
}

func (d *Driver) delete(ctx context.Context, coll *mongo.Collection, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	filter, err := buildFilter(query.Filter)
	if err != nil {
		return nil, 0, err
	}
	result, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("mongo delete: %w", err)
	}
	return []map[string]interface{}{}, result.DeletedCount, nil
}

// buildFilter converts an OQL Filter into a MongoDB bson.M filter document.
// OQL operators ($in, $lt, $gt, etc.) map directly to MongoDB operators.
func buildFilter(filter core.Filter) (bson.M, error) {
	if len(filter) == 0 {
		return bson.M{}, nil
	}
	doc := make(bson.M, len(filter))
	for field, constraint := range filter {
		// Handle logical operators
		if field == "$or" || field == "$and" {
			list, ok := toSlice(constraint)
			if !ok {
				return nil, fmt.Errorf("mongo: %s requires a slice", field)
			}

			subFilters := make(bson.A, 0, len(list))
			for _, item := range list {
				subFilter, ok := item.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("mongo: %s elements must be objects", field)
				}
				// Recurse
				subDoc, err := buildFilter(subFilter)
				if err != nil {
					return nil, err
				}
				subFilters = append(subFilters, subDoc)
			}
			doc[field] = subFilters
			continue
		}

		switch c := constraint.(type) {
		case map[string]interface{}:
			// Validate $in / $nin usage
			for op, val := range c {
				if op == "$in" || op == "$nin" {
					if vals, ok := toSlice(val); ok {
						if len(vals) == 0 {
							return nil, fmt.Errorf("mongo: %s requires non-empty slice", op)
						}
					} else {
						return nil, fmt.Errorf("mongo: %s requires a slice", op)
					}
				}
			}
			// OQL operators align with MongoDB operators – pass them through directly.
			doc[field] = bson.M(c)
		default:
			doc[field] = constraint
		}
	}
	return doc, nil
}

// toSlice converts an interface{} to []interface{} if possible.
func toSlice(v interface{}) ([]interface{}, bool) {
	switch s := v.(type) {
	case []interface{}:
		return s, true
	default:
		return nil, false
	}
}
