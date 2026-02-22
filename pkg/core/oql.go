// Package core defines the OmniQL protocol types, driver interface, schema
// resolver and the central execution engine.
package core

// Action represents the type of operation requested in an OQL query.
type Action string

const (
	ActionFind   Action = "FIND"
	ActionInsert Action = "INSERT"
	ActionUpdate Action = "UPDATE"
	ActionDelete Action = "DELETE"
	ActionCount  Action = "COUNT"
)

// Filter is a map of field names to constraint expressions.
// Constraints follow MongoDB-style operators (e.g. {"$in": [...], "$lt": 500}).
type Filter map[string]interface{}

// QueryOptions carries optional modifiers for a query (pagination, projection, sort).
type QueryOptions struct {
	Limit  int                    `json:"limit,omitempty"`
	Skip   int                    `json:"skip,omitempty"`
	Sort   map[string]int         `json:"sort,omitempty"`
	Fields map[string]interface{} `json:"fields,omitempty"`
}

// OQLQuery is the canonical, database-agnostic query object sent to the Core Engine.
//
// Example:
//
//	{
//	  "target":  "analytics_data",
//	  "action":  "FIND",
//	  "filter":  { "category": {"$in": ["electronics","books"]}, "price": {"$lt": 500} },
//	  "options": { "limit": 20 }
//	}
type OQLQuery struct {
	// Target is the name of the collection, table, index or key-prefix.
	Target string `json:"target"`

	// Action is the operation to perform.
	Action Action `json:"action"`

	// Filter contains the constraint expressions used to select records.
	Filter Filter `json:"filter,omitempty"`

	// Document is the data payload for INSERT / UPDATE operations.
	Document map[string]interface{} `json:"document,omitempty"`

	// Options holds optional modifiers (limit, skip, sort, projection).
	Options QueryOptions `json:"options,omitempty"`
}
