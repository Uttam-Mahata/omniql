// Package redis provides an OmniQL driver for Redis key-value stores.
// It maps OQL queries to Redis commands (GET, SET, SCAN, DEL, EXPIRE).
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	goredis "github.com/redis/go-redis/v9"
)

const driverName = "redis"

// Driver is the OmniQL Redis adapter.
type Driver struct {
	client *goredis.Client
}

// New creates a Redis driver from a standard Redis URL.
// URL format: "redis://:password@host:6379/0"
func New(url string) (*Driver, error) {
	opts, err := goredis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis: parse url: %w", err)
	}
	return &Driver{client: goredis.NewClient(opts)}, nil
}

// NewFromClient creates a Driver from an existing *goredis.Client.
func NewFromClient(client *goredis.Client) *Driver {
	return &Driver{client: client}
}

// Name satisfies core.Driver.
func (d *Driver) Name() string { return driverName }

// Ping satisfies core.Driver.
func (d *Driver) Ping(ctx context.Context) error {
	return d.client.Ping(ctx).Err()
}

// Close satisfies core.Driver.
func (d *Driver) Close() error {
	return d.client.Close()
}

// EnsureTarget satisfies SchemaAwareDriver.
func (d *Driver) EnsureTarget(ctx context.Context, target string, schema *core.CollectionSchema) error {
	return nil
}

// Execute dispatches the OQL query to the appropriate Redis operation.
func (d *Driver) Execute(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	switch query.Action {
	case core.ActionFind:
		return d.find(ctx, query)
	case core.ActionCount:
		return d.count(ctx, query)
	case core.ActionInsert:
		return d.insert(ctx, query)
	case core.ActionUpdate:
		return d.update(ctx, query)
	case core.ActionDelete:
		return d.delete(ctx, query)
	default:
		return nil, 0, fmt.Errorf("redis: unsupported action %q", query.Action)
	}
}

// BatchInsert implements core.Driver by inserting multiple documents via SET.
func (d *Driver) BatchInsert(ctx context.Context, target string, docs []map[string]interface{}) ([]map[string]interface{}, error) {
	if len(docs) == 0 {
		return []map[string]interface{}{}, nil
	}

	pipe := d.client.Pipeline()
	for _, doc := range docs {
		id, key, err := docKey(target, doc)
		if err != nil {
			return nil, fmt.Errorf("redis batch: %w", err)
		}
		_ = id

		data, err := json.Marshal(doc)
		if err != nil {
			return nil, fmt.Errorf("redis batch: marshal: %w", err)
		}

		ttl := extractTTL(doc)
		pipe.Set(ctx, key, data, ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("redis batch: exec: %w", err)
	}

	return []map[string]interface{}{{"count": int64(len(docs))}}, nil
}

func (d *Driver) find(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	// If filter has exactly one field "id" (or "_id"), use direct GET.
	if id, ok := extractIDFromFilter(query.Filter); ok {
		key := keyFor(query.Target, fmt.Sprintf("%v", id))
		val, err := d.client.Get(ctx, key).Result()
		if err == goredis.Nil {
			return []map[string]interface{}{}, 0, nil
		}
		if err != nil {
			return nil, 0, fmt.Errorf("redis get: %w", err)
		}

		var doc map[string]interface{}
		if err := json.Unmarshal([]byte(val), &doc); err != nil {
			return nil, 0, fmt.Errorf("redis get: unmarshal: %w", err)
		}
		return []map[string]interface{}{doc}, 1, nil
	}

	// No id filter: SCAN all keys matching <target>:* then GET each.
	docs, err := d.scanAll(ctx, query.Target)
	if err != nil {
		return nil, 0, err
	}

	// Apply in-memory filter for non-id fields.
	if len(query.Filter) > 0 {
		docs = filterDocs(docs, query.Filter)
	}

	total := int64(len(docs))

	// Apply sort (in-memory, best-effort for single field).
	if len(query.Options.Sort) > 0 {
		sortDocs(docs, query.Options.Sort)
	}

	// Apply pagination.
	if query.Options.Skip > 0 {
		if int(query.Options.Skip) >= len(docs) {
			docs = nil
		} else {
			docs = docs[query.Options.Skip:]
		}
	}
	if query.Options.Limit > 0 && int(query.Options.Limit) < len(docs) {
		docs = docs[:query.Options.Limit]
	}

	return docs, total, nil
}

func (d *Driver) count(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	docs, err := d.scanAll(ctx, query.Target)
	if err != nil {
		return nil, 0, err
	}

	if len(query.Filter) > 0 {
		docs = filterDocs(docs, query.Filter)
	}

	total := int64(len(docs))
	return []map[string]interface{}{{"count": total}}, total, nil
}

func (d *Driver) insert(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Document) == 0 {
		return nil, 0, fmt.Errorf("redis: INSERT requires a non-empty document")
	}

	_, key, err := docKey(query.Target, query.Document)
	if err != nil {
		return nil, 0, fmt.Errorf("redis insert: %w", err)
	}

	data, err := json.Marshal(query.Document)
	if err != nil {
		return nil, 0, fmt.Errorf("redis insert: marshal: %w", err)
	}

	ttl := extractTTL(query.Document)
	if err := d.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return nil, 0, fmt.Errorf("redis set: %w", err)
	}

	return []map[string]interface{}{{"id": query.Document["id"]}}, 1, nil
}

func (d *Driver) update(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	id, ok := extractIDFromFilter(query.Filter)
	if !ok {
		return nil, 0, fmt.Errorf("redis: UPDATE requires a filter with 'id' field")
	}

	key := keyFor(query.Target, fmt.Sprintf("%v", id))

	// GET existing document.
	val, err := d.client.Get(ctx, key).Result()
	if err == goredis.Nil {
		return []map[string]interface{}{}, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("redis update get: %w", err)
	}

	var existing map[string]interface{}
	if err := json.Unmarshal([]byte(val), &existing); err != nil {
		return nil, 0, fmt.Errorf("redis update: unmarshal: %w", err)
	}

	// Merge update fields.
	for k, v := range query.Document {
		existing[k] = v
	}

	data, err := json.Marshal(existing)
	if err != nil {
		return nil, 0, fmt.Errorf("redis update: marshal: %w", err)
	}

	ttl := extractTTL(existing)
	if err := d.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return nil, 0, fmt.Errorf("redis update set: %w", err)
	}

	return []map[string]interface{}{}, 1, nil
}

func (d *Driver) delete(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	id, ok := extractIDFromFilter(query.Filter)
	if !ok {
		return nil, 0, fmt.Errorf("redis: DELETE requires a filter with 'id' field")
	}

	key := keyFor(query.Target, fmt.Sprintf("%v", id))
	n, err := d.client.Del(ctx, key).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("redis del: %w", err)
	}
	return []map[string]interface{}{}, n, nil
}

// scanAll returns all documents stored under <target>:* keys.
func (d *Driver) scanAll(ctx context.Context, target string) ([]map[string]interface{}, error) {
	pattern := target + ":*"
	var docs []map[string]interface{}

	var cursor uint64
	for {
		keys, nextCursor, err := d.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan: %w", err)
		}

		for _, k := range keys {
			val, err := d.client.Get(ctx, k).Result()
			if err == goredis.Nil {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("redis get %q: %w", k, err)
			}
			var doc map[string]interface{}
			if err := json.Unmarshal([]byte(val), &doc); err != nil {
				continue // skip malformed entries
			}
			docs = append(docs, doc)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return docs, nil
}

// keyFor returns the Redis key for a given target and id.
func keyFor(target, id string) string {
	return target + ":" + id
}

// docKey derives a key from the document's "id" (or "_id") field.
// Returns an error if neither field is present.
func docKey(target string, doc map[string]interface{}) (interface{}, string, error) {
	for _, field := range []string{"id", "_id"} {
		if v, ok := doc[field]; ok {
			return v, keyFor(target, fmt.Sprintf("%v", v)), nil
		}
	}
	return nil, "", fmt.Errorf("document must contain an 'id' or '_id' field for Redis storage")
}

// extractIDFromFilter returns the id value if the filter contains exactly
// an "id" or "_id" equality constraint (bare value or $eq).
func extractIDFromFilter(filter core.Filter) (interface{}, bool) {
	for _, field := range []string{"id", "_id"} {
		v, ok := filter[field]
		if !ok {
			continue
		}
		// Bare value equality.
		if m, isMap := v.(map[string]interface{}); isMap {
			if eq, hasEq := m["$eq"]; hasEq {
				return eq, true
			}
			continue
		}
		return v, true
	}
	return nil, false
}

// extractTTL extracts a _ttl duration (in seconds as int or float64) from the document.
// Returns 0 (no expiry) if the field is absent or invalid.
func extractTTL(doc map[string]interface{}) time.Duration {
	v, ok := doc["_ttl"]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return time.Duration(n) * time.Second
	case int:
		return time.Duration(n) * time.Second
	case int64:
		return time.Duration(n) * time.Second
	}
	return 0
}

// filterDocs applies simple in-memory filtering for non-id fields.
// Only bare value equality and $eq are supported for non-SQL drivers.
func filterDocs(docs []map[string]interface{}, filter core.Filter) []map[string]interface{} {
	var result []map[string]interface{}
outer:
	for _, doc := range docs {
		for field, constraint := range filter {
			if field == "id" || field == "_id" {
				continue // already used for key lookup
			}
			docVal, exists := doc[field]
			if !exists {
				continue outer
			}
			if m, isMap := constraint.(map[string]interface{}); isMap {
				if eq, hasEq := m["$eq"]; hasEq {
					if fmt.Sprintf("%v", docVal) != fmt.Sprintf("%v", eq) {
						continue outer
					}
				}
				// Complex operators not supported; skip
				continue
			}
			// Bare value equality.
			if fmt.Sprintf("%v", docVal) != fmt.Sprintf("%v", constraint) {
				continue outer
			}
		}
		result = append(result, doc)
	}
	return result
}

// sortDocs performs a simple in-memory sort on a single field.
func sortDocs(docs []map[string]interface{}, sortSpec map[string]int) {
	// Pick the first sort key alphabetically for determinism.
	if len(sortSpec) == 0 {
		return
	}
	keys := make([]string, 0, len(sortSpec))
	for k := range sortSpec {
		keys = append(keys, k)
	}
	// Use first key only for simplicity.
	key := keys[0]
	dir := sortSpec[key]

	// Simple bubble-pass; good enough for small Redis result sets.
	n := len(docs)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			a := fmt.Sprintf("%v", docs[j][key])
			b := fmt.Sprintf("%v", docs[j+1][key])
			less := strings.Compare(a, b) < 0
			if (dir == 1 && !less) || (dir == -1 && less) {
				docs[j], docs[j+1] = docs[j+1], docs[j]
			}
		}
	}
}
