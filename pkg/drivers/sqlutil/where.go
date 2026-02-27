// Package sqlutil provides shared SQL utilities for OmniQL SQL-based drivers.
// The primary export is BuildWhere, which translates an OQL Filter map into a
// parameterised WHERE clause that is compatible with any SQL dialect.
package sqlutil

import (
	"fmt"
	"strings"
)

// PlaceholderFn generates a parameter placeholder for the nth parameter (1-based).
// Examples:
//   - Question returns "?" for all n        — MySQL, SQLite
//   - Dollar  returns "$n"                  — PostgreSQL
//   - AtParam returns "@pn"                 — SQL Server
type PlaceholderFn func(n int) string

// Question always returns "?" — suitable for MySQL and SQLite.
func Question(_ int) string { return "?" }

// Dollar returns "$n" — suitable for PostgreSQL.
func Dollar(n int) string { return fmt.Sprintf("$%d", n) }

// AtParam returns "@pn" — suitable for SQL Server (go-mssqldb).
func AtParam(n int) string { return fmt.Sprintf("@p%d", n) }

// BuildWhere builds a parameterised WHERE clause from an OQL filter map.
//
// Parameters:
//   - filter:       the OQL filter (operator keys start with "$")
//   - placeholder:  a PlaceholderFn that produces the parameter token for index n
//   - quoteIdent:   wraps an identifier in driver-appropriate quotes (backtick, double-quote…)
//   - translateCol: converts a (possibly dot-notated) field name to a SQL expression;
//     if nil, quoteIdent is used for simple fields and an error returned for dotted fields
//   - startIdx:     first parameter index (normally 1; >1 for Postgres/SQLServer UPDATE)
//
// Returns the WHERE clause string (empty if filter is empty), the argument slice,
// the next unused parameter index, and any error encountered.
func BuildWhere(
	filter map[string]interface{},
	placeholder PlaceholderFn,
	quoteIdent func(string) string,
	translateCol func(string) string,
	startIdx int,
) (clause string, args []interface{}, nextIdx int, err error) {
	if len(filter) == 0 {
		return "", nil, startIdx, nil
	}

	if translateCol == nil {
		translateCol = quoteIdent
	}

	clauses := make([]string, 0, len(filter))
	args = make([]interface{}, 0, len(filter))
	i := startIdx

	for field, constraint := range filter {
		if field == "$or" || field == "$and" {
			list, ok := toSlice(constraint)
			if !ok {
				return "", nil, i, fmt.Errorf("sqlutil: %s requires a slice", field)
			}
			subClauses := make([]string, 0, len(list))
			for _, item := range list {
				subFilter, ok := item.(map[string]interface{})
				if !ok {
					return "", nil, i, fmt.Errorf("sqlutil: %s elements must be objects", field)
				}
				subWhere, subArgs, nextI, serr := BuildWhere(subFilter, placeholder, quoteIdent, translateCol, i)
				if serr != nil {
					return "", nil, i, serr
				}
				if subWhere != "" {
					subClauses = append(subClauses, "("+subWhere+")")
					args = append(args, subArgs...)
					i = nextI
				}
			}
			op := " OR "
			if field == "$and" {
				op = " AND "
			}
			if len(subClauses) > 0 {
				clauses = append(clauses, "("+strings.Join(subClauses, op)+")")
			}
			continue
		}

		col := translateCol(field)

		switch c := constraint.(type) {
		case map[string]interface{}:
			for op, val := range c {
				switch op {
				case "$eq":
					clauses = append(clauses, col+" = "+placeholder(i))
					args = append(args, val)
					i++
				case "$ne":
					clauses = append(clauses, col+" != "+placeholder(i))
					args = append(args, val)
					i++
				case "$lt":
					clauses = append(clauses, col+" < "+placeholder(i))
					args = append(args, val)
					i++
				case "$lte":
					clauses = append(clauses, col+" <= "+placeholder(i))
					args = append(args, val)
					i++
				case "$gt":
					clauses = append(clauses, col+" > "+placeholder(i))
					args = append(args, val)
					i++
				case "$gte":
					clauses = append(clauses, col+" >= "+placeholder(i))
					args = append(args, val)
					i++
				case "$in":
					vals, ok := toSlice(val)
					if !ok {
						return "", nil, i, fmt.Errorf("sqlutil: $in requires a slice")
					}
					if len(vals) == 0 {
						return "", nil, i, fmt.Errorf("sqlutil: $in requires non-empty slice")
					}
					ph := make([]string, len(vals))
					for j, v := range vals {
						ph[j] = placeholder(i)
						args = append(args, v)
						i++
					}
					clauses = append(clauses, col+" IN ("+strings.Join(ph, ", ")+")")
				case "$nin":
					vals, ok := toSlice(val)
					if !ok {
						return "", nil, i, fmt.Errorf("sqlutil: $nin requires a slice")
					}
					if len(vals) == 0 {
						return "", nil, i, fmt.Errorf("sqlutil: $nin requires non-empty slice")
					}
					ph := make([]string, len(vals))
					for j, v := range vals {
						ph[j] = placeholder(i)
						args = append(args, v)
						i++
					}
					clauses = append(clauses, col+" NOT IN ("+strings.Join(ph, ", ")+")")
				default:
					return "", nil, i, fmt.Errorf("sqlutil: unsupported operator %q", op)
				}
			}
		default:
			// Bare value: equality.
			clauses = append(clauses, col+" = "+placeholder(i))
			args = append(args, constraint)
			i++
		}
	}

	return strings.Join(clauses, " AND "), args, i, nil
}

// toSlice converts interface{} to []interface{} if possible.
func toSlice(v interface{}) ([]interface{}, bool) {
	s, ok := v.([]interface{})
	return s, ok
}
