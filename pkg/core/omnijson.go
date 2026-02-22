package core

// OmniJSON is the standardized response format returned by the Core Engine
// after a driver executes a query.  All driver-specific response shapes are
// normalised into this structure before being handed back to the caller.
type OmniJSON struct {
	// Data contains the result rows/documents as a slice of generic maps.
	Data []map[string]interface{} `json:"data"`

	// Meta carries optional metadata about the result set.
	Meta OmniMeta `json:"meta"`

	// Error is non-nil when the operation failed.
	Error *OmniError `json:"error,omitempty"`
}

// OmniMeta holds metadata about a query result.
type OmniMeta struct {
	// Total is the total number of matching records (may be -1 if unknown).
	Total int64 `json:"total"`

	// Returned is the number of records actually returned in this response.
	Returned int `json:"returned"`

	// Driver is the name of the driver that executed the query.
	Driver string `json:"driver"`

	// Target is the collection/table/index that was queried.
	Target string `json:"target"`
}

// OmniError encapsulates a structured error returned from the engine or a driver.
type OmniError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error satisfies the built-in error interface.
func (e *OmniError) Error() string {
	return e.Code + ": " + e.Message
}
