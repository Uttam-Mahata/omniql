// Package postgres provides an OmniQL driver for PostgreSQL databases.
// It translates OQL queries into standard SQL and executes them via the
// database/sql interface with the pgx driver.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	"github.com/Uttam-Mahata/omniql/pkg/drivers/sqlutil"
)

const driverName = "postgres"

// Driver is the OmniQL PostgreSQL adapter.
type Driver struct {
	db *sql.DB
}

// New creates a new PostgreSQL Driver using the provided *sql.DB connection.
// The caller is responsible for opening and configuring the connection.
func New(db *sql.DB) *Driver {
	return &Driver{db: db}
}

// Name satisfies core.Driver.
func (d *Driver) Name() string { return driverName }

// Ping satisfies core.Driver.
func (d *Driver) Ping(ctx context.Context) error { return d.db.PingContext(ctx) }

// Close satisfies core.Driver.
func (d *Driver) Close() error { return d.db.Close() }

// Execute translates an OQLQuery into a SQL statement, runs it, and returns
// the result rows as a slice of generic maps.
func (d *Driver) Execute(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	switch query.Action {
	case core.ActionFind, core.ActionCount:
		return d.find(ctx, query)
	case core.ActionInsert:
		return d.insert(ctx, query)
	case core.ActionUpdate:
		return d.update(ctx, query)
	case core.ActionDelete:
		return d.delete(ctx, query)
	default:
		return nil, 0, fmt.Errorf("postgres: unsupported action %q", query.Action)
	}
}

// find builds and executes a SELECT statement.
func (d *Driver) find(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	where, args, _, err := sqlutil.BuildWhere(query.Filter, sqlutil.Dollar, quote, translateColumn, 1)
	if err != nil {
		return nil, 0, err
	}

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", quote(query.Target))
	if where != "" {
		countSQL += " WHERE " + where
	}
	var total int64
	if err := d.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres count: %w", err)
	}

	if query.Action == core.ActionCount {
		return []map[string]interface{}{{"count": total}}, total, nil
	}

	fields := "*"
	if len(query.Options.Fields) > 0 {
		var cols []string
		for field, val := range query.Options.Fields {
			inc, err := fieldProjectionValue(val)
			if err != nil {
				return nil, 0, fmt.Errorf("postgres: %w", err)
			}
			if inc {
				cols = append(cols, quote(field))
			}
		}
		if len(cols) > 0 {
			sort.Strings(cols) // Sort for deterministic SQL
			fields = strings.Join(cols, ", ")
		}
	}

	selectSQL := fmt.Sprintf("SELECT %s FROM %s", fields, quote(query.Target))
	if where != "" {
		selectSQL += " WHERE " + where
	}

	if len(query.Options.Sort) > 0 {
		var sortClauses []string
		keys := make([]string, 0, len(query.Options.Sort))
		for k := range query.Options.Sort {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			val := query.Options.Sort[k]
			order := "ASC"
			if val == -1 {
				order = "DESC"
			}
			sortClauses = append(sortClauses, quote(k)+" "+order)
		}
		selectSQL += " ORDER BY " + strings.Join(sortClauses, ", ")
	}

	if query.Options.Limit > 0 {
		selectSQL += fmt.Sprintf(" LIMIT %d", query.Options.Limit)
	}
	if query.Options.Skip > 0 {
		selectSQL += fmt.Sprintf(" OFFSET %d", query.Options.Skip)
	}

	rows, err := d.db.QueryContext(ctx, selectSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres select: %w", err)
	}
	defer rows.Close()

	return scanRows(rows, total)
}

// insert builds and executes an INSERT statement.
func (d *Driver) insert(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Document) == 0 {
		return nil, 0, fmt.Errorf("postgres: INSERT requires a non-empty document")
	}

	cols := make([]string, 0, len(query.Document))
	placeholders := make([]string, 0, len(query.Document))
	vals := make([]interface{}, 0, len(query.Document))
	i := 1
	for col, val := range query.Document {
		cols = append(cols, quote(col))
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		vals = append(vals, val)
		i++
	}

	stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		quote(query.Target),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	rows, err := d.db.QueryContext(ctx, stmt, vals...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres insert: %w", err)
	}
	defer rows.Close()

	return scanRows(rows, 1)
}

// BatchInsert implements core.Driver.
func (d *Driver) BatchInsert(ctx context.Context, target string, docs []map[string]interface{}) ([]map[string]interface{}, error) {
	if len(docs) == 0 {
		return []map[string]interface{}{}, nil
	}

	first := docs[0]
	// Sort column names to guarantee consistent ordering between statement and values.
	colOrder := make([]string, 0, len(first))
	for col := range first {
		colOrder = append(colOrder, col)
	}
	sort.Strings(colOrder)

	cols := make([]string, 0, len(colOrder))
	for _, col := range colOrder {
		cols = append(cols, quote(col))
	}

	var values []string
	var args []interface{}
	pIdx := 1
	for _, doc := range docs {
		var rowPlaceholders []string
		for _, col := range colOrder {
			rowPlaceholders = append(rowPlaceholders, fmt.Sprintf("$%d", pIdx))
			args = append(args, doc[col])
			pIdx++
		}
		values = append(values, "("+strings.Join(rowPlaceholders, ", ")+")")
	}

	stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		quote(target),
		strings.Join(cols, ", "),
		strings.Join(values, ", "),
	)

	_, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres batch: %w", err)
	}

	return []map[string]interface{}{{"count": int64(len(docs))}}, nil
}

// update builds and executes an UPDATE statement.
func (d *Driver) update(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Document) == 0 {
		return nil, 0, fmt.Errorf("postgres: UPDATE requires a non-empty document")
	}
	if len(query.Filter) == 0 {
		return nil, 0, fmt.Errorf("postgres: UPDATE requires a non-empty filter to prevent accidental bulk updates")
	}

	setClauses := make([]string, 0, len(query.Document))
	args := make([]interface{}, 0, len(query.Document))
	i := 1
	for col, val := range query.Document {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quote(col), i))
		args = append(args, val)
		i++
	}

	where, whereArgs, _, err := sqlutil.BuildWhere(query.Filter, sqlutil.Dollar, quote, translateColumn, i)
	if err != nil {
		return nil, 0, err
	}
	args = append(args, whereArgs...)

	stmt := fmt.Sprintf("UPDATE %s SET %s", quote(query.Target), strings.Join(setClauses, ", "))
	if where != "" {
		stmt += " WHERE " + where
	}

	result, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres update: %w", err)
	}
	affected, _ := result.RowsAffected()
	return []map[string]interface{}{}, affected, nil
}

// delete builds and executes a DELETE statement.
func (d *Driver) delete(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Filter) == 0 {
		return nil, 0, fmt.Errorf("postgres: DELETE requires a non-empty filter to prevent accidental bulk deletes")
	}
	where, args, _, err := sqlutil.BuildWhere(query.Filter, sqlutil.Dollar, quote, translateColumn, 1)
	if err != nil {
		return nil, 0, err
	}

	stmt := fmt.Sprintf("DELETE FROM %s", quote(query.Target))
	if where != "" {
		stmt += " WHERE " + where
	}

	result, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres delete: %w", err)
	}
	affected, _ := result.RowsAffected()
	return []map[string]interface{}{}, affected, nil
}

// scanRows converts *sql.Rows into a slice of generic maps.
func scanRows(rows *sql.Rows, total int64) ([]map[string]interface{}, int64, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, 0, err
	}

	var result []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, 0, err
		}
		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			row[col] = vals[i]
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

// buildWhere is a package-level wrapper for internal tests.
func buildWhere(filter core.Filter) (string, []interface{}, error) {
	clause, args, _, err := sqlutil.BuildWhere(filter, sqlutil.Dollar, quote, translateColumn, 1)
	return clause, args, err
}

// quote wraps an identifier in double-quotes to prevent SQL injection and
// handle reserved keywords.
func quote(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// translateColumn converts a dot-notated field (e.g. "profile.name") into
// a PostgreSQL JSON path expression (e.g. "profile"->>'name').
// If the field has no dots, it is simply quoted.
func translateColumn(field string) string {
	parts := strings.Split(field, ".")
	if len(parts) == 1 {
		return quote(field)
	}

	// Start with the column name
	expr := quote(parts[0])

	// Iterate over path segments.
	// Use -> for intermediate steps (returns jsonb)
	// Use ->> for the final step (returns text)
	for i, part := range parts[1:] {
		// Use single quotes for JSON keys
		key := strings.ReplaceAll(part, "'", "''")
		if i == len(parts)-2 {
			// Last part: ->>
			expr = fmt.Sprintf("%s->>'%s'", expr, key)
		} else {
			// Intermediate part: ->
			expr = fmt.Sprintf("%s->'%s'", expr, key)
		}
	}
	return expr
}

// fieldProjectionValue returns (true, nil) for inclusion (1), (false, nil) for
// unknown values, and (false, error) when an exclusion value (0) is detected.
// SQL drivers support inclusion-only projection.
func fieldProjectionValue(val interface{}) (include bool, err error) {
	var n float64
	switch v := val.(type) {
	case int:
		n = float64(v)
	case float64:
		n = v
	default:
		return false, nil
	}
	if n == 1 {
		return true, nil
	}
	if n == 0 {
		return false, fmt.Errorf("field exclusion (value=0) is not supported; use inclusion (value=1) instead")
	}
	return false, nil
}
