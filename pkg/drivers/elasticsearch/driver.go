// Package elasticsearch provides an OmniQL driver for Elasticsearch.
// It maps OQL queries to Elasticsearch REST API calls using the Query DSL.
package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
)

const driverName = "elasticsearch"

// Driver is the OmniQL Elasticsearch adapter.
type Driver struct {
	client *elasticsearch8.Client
}

// New creates an Elasticsearch driver from an address (e.g. "http://localhost:9200").
func New(addr string) (*Driver, error) {
	cfg := elasticsearch8.Config{
		Addresses: []string{addr},
	}
	client, err := elasticsearch8.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: create client: %w", err)
	}
	return &Driver{client: client}, nil
}

// NewFromClient creates a Driver from an existing *elasticsearch8.Client.
func NewFromClient(client *elasticsearch8.Client) *Driver {
	return &Driver{client: client}
}

// Name satisfies core.Driver.
func (d *Driver) Name() string { return driverName }

// Ping satisfies core.Driver.
func (d *Driver) Ping(ctx context.Context) error {
	res, err := d.client.Ping(d.client.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("elasticsearch ping: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("elasticsearch ping: status %s", res.Status())
	}
	return nil
}

// Close satisfies core.Driver. Elasticsearch HTTP client has no explicit close.
func (d *Driver) Close() error { return nil }

// ListTargets returns an error as Elasticsearch target listing is not implemented.
func (d *Driver) ListTargets(ctx context.Context) ([]string, error) {
	return nil, fmt.Errorf("elasticsearch: ListTargets not supported")
}

// EnsureTarget satisfies SchemaAwareDriver.
func (d *Driver) EnsureTarget(ctx context.Context, target string, schema *core.CollectionSchema) error {
	// Check if index exists
	res, err := d.client.Indices.Exists([]string{target}, d.client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("elasticsearch check index: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		return nil // exists
	}

	// Create index
	body := map[string]interface{}{}
	if schema != nil {
		props := make(map[string]interface{})
		for name, field := range schema.Fields {
			props[name] = map[string]interface{}{"type": esType(field.Type)}
		}
		body["mappings"] = map[string]interface{}{
			"properties": props,
		}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resCreate, err := d.client.Indices.Create(
		target,
		d.client.Indices.Create.WithContext(ctx),
		d.client.Indices.Create.WithBody(bytes.NewReader(data)),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch create index: %w", err)
	}
	defer resCreate.Body.Close()

	if resCreate.IsError() {
		return fmt.Errorf("elasticsearch create index: %s", resCreate.Status())
	}

	return nil
}

func esType(ft core.FieldType) string {
	switch ft {
	case core.FieldTypeString:
		return "text"
	case core.FieldTypeInt:
		return "long"
	case core.FieldTypeFloat:
		return "double"
	case core.FieldTypeBool:
		return "boolean"
	case core.FieldTypeArray:
		return "nested"
	case core.FieldTypeObject:
		return "object"
	default:
		return "text"
	}
}

// Execute dispatches the OQL query to the appropriate Elasticsearch operation.
func (d *Driver) Execute(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	switch query.Action {
	case core.ActionFind:
		return d.find(ctx, query)
	case core.ActionCount:
		return d.count(ctx, query)
	case core.ActionInsert:
		return d.insert(ctx, query)
	case core.ActionUpdate:
		return d.updateByQuery(ctx, query)
	case core.ActionDelete:
		return d.deleteByQuery(ctx, query)
	default:
		return nil, 0, fmt.Errorf("elasticsearch: unsupported action %q", query.Action)
	}
}

// BatchInsert implements core.Driver using the Elasticsearch Bulk API.
func (d *Driver) BatchInsert(ctx context.Context, target string, docs []map[string]interface{}) ([]map[string]interface{}, error) {
	if len(docs) == 0 {
		return []map[string]interface{}{}, nil
	}

	var buf bytes.Buffer
	for _, doc := range docs {
		// Action line
		meta := map[string]interface{}{"index": map[string]interface{}{"_index": target}}
		actionLine, err := json.Marshal(meta)
		if err != nil {
			return nil, fmt.Errorf("elasticsearch bulk: marshal action: %w", err)
		}
		buf.Write(actionLine)
		buf.WriteByte('\n')

		// Document line
		docLine, err := json.Marshal(doc)
		if err != nil {
			return nil, fmt.Errorf("elasticsearch bulk: marshal doc: %w", err)
		}
		buf.Write(docLine)
		buf.WriteByte('\n')
	}

	res, err := d.client.Bulk(
		bytes.NewReader(buf.Bytes()),
		d.client.Bulk.WithContext(ctx),
		d.client.Bulk.WithIndex(target),
	)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch bulk: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch bulk: response error: %s", res.Status())
	}

	return []map[string]interface{}{{"count": int64(len(docs))}}, nil
}

// find executes a search query against Elasticsearch.
func (d *Driver) find(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	body := map[string]interface{}{
		"query": buildQueryDSL(query.Filter),
	}

	// Pagination
	if query.Options.Skip > 0 {
		body["from"] = query.Options.Skip
	}
	if query.Options.Limit > 0 {
		body["size"] = query.Options.Limit
	}

	// Sort
	if len(query.Options.Sort) > 0 {
		sortArr := buildSort(query.Options.Sort)
		body["sort"] = sortArr
	}

	// Fields projection (inclusion only)
	if len(query.Options.Fields) > 0 {
		includes := make([]string, 0, len(query.Options.Fields))
		for field, v := range query.Options.Fields {
			if isInclude(v) {
				includes = append(includes, field)
			}
		}
		sort.Strings(includes)
		if len(includes) > 0 {
			body["_source"] = map[string]interface{}{"includes": includes}
		}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("elasticsearch find: marshal: %w", err)
	}

	res, err := d.client.Search(
		d.client.Search.WithContext(ctx),
		d.client.Search.WithIndex(query.Target),
		d.client.Search.WithBody(bytes.NewReader(data)),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("elasticsearch search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("elasticsearch search: response error: %s", res.Status())
	}

	var result struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("elasticsearch search: decode: %w", err)
	}

	docs := make([]map[string]interface{}, 0, len(result.Hits.Hits))
	for _, h := range result.Hits.Hits {
		docs = append(docs, h.Source)
	}

	return docs, result.Hits.Total.Value, nil
}

// count executes a count query against Elasticsearch.
func (d *Driver) count(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	body := map[string]interface{}{
		"query": buildQueryDSL(query.Filter),
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("elasticsearch count: marshal: %w", err)
	}

	res, err := d.client.Count(
		d.client.Count.WithContext(ctx),
		d.client.Count.WithIndex(query.Target),
		d.client.Count.WithBody(bytes.NewReader(data)),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("elasticsearch count: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("elasticsearch count: response error: %s", res.Status())
	}

	var result struct {
		Count int64 `json:"count"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("elasticsearch count: decode: %w", err)
	}

	return []map[string]interface{}{{"count": result.Count}}, result.Count, nil
}

// insert indexes a single document into Elasticsearch.
func (d *Driver) insert(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Document) == 0 {
		return nil, 0, fmt.Errorf("elasticsearch: INSERT requires a non-empty document")
	}

	data, err := json.Marshal(query.Document)
	if err != nil {
		return nil, 0, fmt.Errorf("elasticsearch insert: marshal: %w", err)
	}

	res, err := d.client.Index(
		query.Target,
		bytes.NewReader(data),
		d.client.Index.WithContext(ctx),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("elasticsearch index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("elasticsearch index: response error: %s", res.Status())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("elasticsearch index: decode: %w", err)
	}

	doc := map[string]interface{}{}
	if id, ok := result["_id"]; ok {
		doc["_id"] = id
	}
	return []map[string]interface{}{doc}, 1, nil
}

// updateByQuery updates documents matching the filter using a Painless script.
func (d *Driver) updateByQuery(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Document) == 0 {
		return nil, 0, fmt.Errorf("elasticsearch: UPDATE requires a non-empty document")
	}

	// Build deterministic Painless script from sorted document keys.
	keys := make([]string, 0, len(query.Document))
	for k := range query.Document {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	scriptParts := make([]string, 0, len(keys))
	for _, k := range keys {
		scriptParts = append(scriptParts, fmt.Sprintf("ctx._source['%s']=params['%s']", k, k))
	}
	scriptSource := strings.Join(scriptParts, "; ")

	body := map[string]interface{}{
		"query": buildQueryDSL(query.Filter),
		"script": map[string]interface{}{
			"source": scriptSource,
			"lang":   "painless",
			"params": query.Document,
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("elasticsearch update: marshal: %w", err)
	}

	res, err := d.client.UpdateByQuery(
		[]string{query.Target},
		d.client.UpdateByQuery.WithContext(ctx),
		d.client.UpdateByQuery.WithBody(bytes.NewReader(data)),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("elasticsearch update_by_query: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("elasticsearch update_by_query: response error: %s", res.Status())
	}

	var result struct {
		Updated int64 `json:"updated"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("elasticsearch update_by_query: decode: %w", err)
	}

	return []map[string]interface{}{}, result.Updated, nil
}

// deleteByQuery deletes documents matching the filter.
func (d *Driver) deleteByQuery(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	body := map[string]interface{}{
		"query": buildQueryDSL(query.Filter),
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("elasticsearch delete: marshal: %w", err)
	}

	res, err := d.client.DeleteByQuery(
		[]string{query.Target},
		bytes.NewReader(data),
		d.client.DeleteByQuery.WithContext(ctx),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("elasticsearch delete_by_query: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("elasticsearch delete_by_query: response error: %s", res.Status())
	}

	var result struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("elasticsearch delete_by_query: decode: %w", err)
	}

	return []map[string]interface{}{}, result.Deleted, nil
}

// buildQueryDSL translates an OQL filter map into an Elasticsearch Query DSL map.
func buildQueryDSL(filter core.Filter) map[string]interface{} {
	if len(filter) == 0 {
		return map[string]interface{}{"match_all": map[string]interface{}{}}
	}

	// Handle $or / $and at the top level
	if v, ok := filter["$or"]; ok {
		return buildLogical("should", v)
	}
	if v, ok := filter["$and"]; ok {
		return buildLogical("must", v)
	}

	// Multiple top-level fields → bool.must
	clauses := make([]map[string]interface{}, 0, len(filter))
	for field, constraint := range filter {
		clause := buildFieldClause(field, constraint)
		clauses = append(clauses, clause)
	}

	if len(clauses) == 1 {
		return clauses[0]
	}

	return map[string]interface{}{
		"bool": map[string]interface{}{"must": clauses},
	}
}

// buildLogical builds a bool should/must clause from an $or/$and value.
func buildLogical(boolKey string, v interface{}) map[string]interface{} {
	arr, ok := v.([]interface{})
	if !ok {
		return map[string]interface{}{"match_all": map[string]interface{}{}}
	}

	clauses := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			clauses = append(clauses, buildQueryDSL(core.Filter(m)))
		}
	}

	return map[string]interface{}{
		"bool": map[string]interface{}{boolKey: clauses},
	}
}

// buildFieldClause converts a single OQL field constraint into a Query DSL clause.
func buildFieldClause(field string, constraint interface{}) map[string]interface{} {
	m, isMap := constraint.(map[string]interface{})
	if !isMap {
		// Bare value → term query
		return map[string]interface{}{"term": map[string]interface{}{field: constraint}}
	}

	// Collect range operators first
	rangeOps := map[string]interface{}{}
	mustNot := make([]map[string]interface{}, 0)

	for op, val := range m {
		switch op {
		case "$eq":
			return map[string]interface{}{"term": map[string]interface{}{field: val}}
		case "$ne":
			mustNot = append(mustNot, map[string]interface{}{"term": map[string]interface{}{field: val}})
		case "$lt":
			rangeOps["lt"] = val
		case "$lte":
			rangeOps["lte"] = val
		case "$gt":
			rangeOps["gt"] = val
		case "$gte":
			rangeOps["gte"] = val
		case "$in":
			return map[string]interface{}{"terms": map[string]interface{}{field: val}}
		case "$nin":
			return map[string]interface{}{
				"bool": map[string]interface{}{
					"must_not": []map[string]interface{}{
						{"terms": map[string]interface{}{field: val}},
					},
				},
			}
		}
	}

	if len(rangeOps) > 0 {
		return map[string]interface{}{"range": map[string]interface{}{field: rangeOps}}
	}

	if len(mustNot) > 0 {
		return map[string]interface{}{
			"bool": map[string]interface{}{"must_not": mustNot},
		}
	}

	return map[string]interface{}{"match_all": map[string]interface{}{}}
}

// buildSort converts an OQL sort map to an Elasticsearch sort array.
func buildSort(sortSpec map[string]int) []map[string]interface{} {
	// Sort keys for determinism
	keys := make([]string, 0, len(sortSpec))
	for k := range sortSpec {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	arr := make([]map[string]interface{}, 0, len(keys))
	for _, k := range keys {
		order := "asc"
		if sortSpec[k] == -1 {
			order = "desc"
		}
		arr = append(arr, map[string]interface{}{k: map[string]interface{}{"order": order}})
	}
	return arr
}

// isInclude returns true if the field projection value represents inclusion (1 or true).
func isInclude(v interface{}) bool {
	switch n := v.(type) {
	case int:
		return n == 1
	case float64:
		return n == 1
	case bool:
		return n
	}
	return false
}
