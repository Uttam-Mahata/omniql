// Package postgres provides an OmniQL driver for PostgreSQL databases.
// It translates OQL queries into standard SQL and executes them via the
// database/sql interface with the pgx driver.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Uttam-Mahata/omniql/pkg/core"
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
	where, args := buildWhere(query.Filter)

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

	selectSQL := fmt.Sprintf("SELECT * FROM %s", quote(query.Target))
	if where != "" {
		selectSQL += " WHERE " + where
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

	stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quote(query.Target),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	result, err := d.db.ExecContext(ctx, stmt, vals...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres insert: %w", err)
	}
	affected, _ := result.RowsAffected()
	return []map[string]interface{}{}, affected, nil
}

// update builds and executes an UPDATE statement.
func (d *Driver) update(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Document) == 0 {
		return nil, 0, fmt.Errorf("postgres: UPDATE requires a non-empty document")
	}

	setClauses := make([]string, 0, len(query.Document))
	args := make([]interface{}, 0, len(query.Document))
	i := 1
	for col, val := range query.Document {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quote(col), i))
		args = append(args, val)
		i++
	}

	where, whereArgs := buildWhereFrom(query.Filter, i)
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
	where, args := buildWhere(query.Filter)

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

// buildWhere translates an OQL Filter into a parameterised SQL WHERE clause.
// Parameters are numbered from $1.
func buildWhere(filter core.Filter) (string, []interface{}) {
	return buildWhereFrom(filter, 1)
}

// buildWhereFrom is like buildWhere but starts parameter numbering at startIdx.
func buildWhereFrom(filter core.Filter, startIdx int) (string, []interface{}) {
	if len(filter) == 0 {
		return "", nil
	}

	clauses := make([]string, 0, len(filter))
	args := make([]interface{}, 0, len(filter))
	i := startIdx

	for field, constraint := range filter {
		switch c := constraint.(type) {
		case map[string]interface{}:
			for op, val := range c {
				switch op {
				case "$eq":
					clauses = append(clauses, fmt.Sprintf("%s = $%d", quote(field), i))
					args = append(args, val)
					i++
				case "$ne":
					clauses = append(clauses, fmt.Sprintf("%s != $%d", quote(field), i))
					args = append(args, val)
					i++
				case "$lt":
					clauses = append(clauses, fmt.Sprintf("%s < $%d", quote(field), i))
					args = append(args, val)
					i++
				case "$lte":
					clauses = append(clauses, fmt.Sprintf("%s <= $%d", quote(field), i))
					args = append(args, val)
					i++
				case "$gt":
					clauses = append(clauses, fmt.Sprintf("%s > $%d", quote(field), i))
					args = append(args, val)
					i++
				case "$gte":
					clauses = append(clauses, fmt.Sprintf("%s >= $%d", quote(field), i))
					args = append(args, val)
					i++
				case "$in":
					if vals, ok := toSlice(val); ok {
						placeholders := make([]string, len(vals))
						for j, v := range vals {
							placeholders[j] = fmt.Sprintf("$%d", i)
							args = append(args, v)
							i++
						}
						clauses = append(clauses, fmt.Sprintf("%s IN (%s)", quote(field), strings.Join(placeholders, ", ")))
					}
				case "$nin":
					if vals, ok := toSlice(val); ok {
						placeholders := make([]string, len(vals))
						for j, v := range vals {
							placeholders[j] = fmt.Sprintf("$%d", i)
							args = append(args, v)
							i++
						}
						clauses = append(clauses, fmt.Sprintf("%s NOT IN (%s)", quote(field), strings.Join(placeholders, ", ")))
					}
				}
			}
		default:
			// Bare value: treat as equality.
			clauses = append(clauses, fmt.Sprintf("%s = $%d", quote(field), i))
			args = append(args, constraint)
			i++
		}
	}

	return strings.Join(clauses, " AND "), args
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

// quote wraps an identifier in double-quotes to prevent SQL injection and
// handle reserved keywords.
func quote(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
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
