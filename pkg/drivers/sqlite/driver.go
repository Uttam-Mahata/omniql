// Package sqlite provides an OmniQL driver for SQLite databases.
// It translates OQL queries into standard SQL using the mattn/go-sqlite3 driver.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	_ "github.com/mattn/go-sqlite3" // SQLite database driver
)

const driverName = "sqlite"

// Driver is the OmniQL SQLite adapter.
type Driver struct {
	db *sql.DB
}

// New opens a SQLite database at the given DSN (file path or ":memory:") and
// returns a Driver wrapping it.
func New(dsn string) (*Driver, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %q: %w", dsn, err)
	}
	return &Driver{db: db}, nil
}

// NewFromDB creates a Driver from an existing *sql.DB.
func NewFromDB(db *sql.DB) *Driver {
	return &Driver{db: db}
}

// Name satisfies core.Driver.
func (d *Driver) Name() string { return driverName }

// Ping satisfies core.Driver.
func (d *Driver) Ping(ctx context.Context) error { return d.db.PingContext(ctx) }

// Close satisfies core.Driver.
func (d *Driver) Close() error { return d.db.Close() }

// Execute translates an OQLQuery into SQL, runs it, and returns the results.
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
		return nil, 0, fmt.Errorf("sqlite: unsupported action %q", query.Action)
	}
}

func (d *Driver) find(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	where, args := buildWhere(query.Filter)

	countSQL := "SELECT COUNT(*) FROM " + quote(query.Target)
	if where != "" {
		countSQL += " WHERE " + where
	}
	var total int64
	if err := d.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("sqlite count: %w", err)
	}

	if query.Action == core.ActionCount {
		return []map[string]interface{}{{"count": total}}, total, nil
	}

	selectSQL := "SELECT * FROM " + quote(query.Target)
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
		return nil, 0, fmt.Errorf("sqlite select: %w", err)
	}
	defer rows.Close()

	return scanRows(rows, total)
}

func (d *Driver) insert(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Document) == 0 {
		return nil, 0, fmt.Errorf("sqlite: INSERT requires a non-empty document")
	}

	cols := make([]string, 0, len(query.Document))
	placeholders := make([]string, 0, len(query.Document))
	vals := make([]interface{}, 0, len(query.Document))
	for col, val := range query.Document {
		cols = append(cols, quote(col))
		placeholders = append(placeholders, "?")
		vals = append(vals, val)
	}

	stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quote(query.Target),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	result, err := d.db.ExecContext(ctx, stmt, vals...)
	if err != nil {
		return nil, 0, fmt.Errorf("sqlite insert: %w", err)
	}
	affected, _ := result.RowsAffected()
	return []map[string]interface{}{}, affected, nil
}

func (d *Driver) update(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Document) == 0 {
		return nil, 0, fmt.Errorf("sqlite: UPDATE requires a non-empty document")
	}

	setClauses := make([]string, 0, len(query.Document))
	args := make([]interface{}, 0, len(query.Document))
	for col, val := range query.Document {
		setClauses = append(setClauses, quote(col)+" = ?")
		args = append(args, val)
	}

	where, whereArgs := buildWhere(query.Filter)
	args = append(args, whereArgs...)

	stmt := "UPDATE " + quote(query.Target) + " SET " + strings.Join(setClauses, ", ")
	if where != "" {
		stmt += " WHERE " + where
	}

	result, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("sqlite update: %w", err)
	}
	affected, _ := result.RowsAffected()
	return []map[string]interface{}{}, affected, nil
}

func (d *Driver) delete(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	where, args := buildWhere(query.Filter)

	stmt := "DELETE FROM " + quote(query.Target)
	if where != "" {
		stmt += " WHERE " + where
	}

	result, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("sqlite delete: %w", err)
	}
	affected, _ := result.RowsAffected()
	return []map[string]interface{}{}, affected, nil
}

// buildWhere translates an OQL Filter into a parameterised SQL WHERE clause.
// SQLite uses ? as the parameter placeholder.
func buildWhere(filter core.Filter) (string, []interface{}) {
	if len(filter) == 0 {
		return "", nil
	}

	clauses := make([]string, 0, len(filter))
	args := make([]interface{}, 0, len(filter))

	for field, constraint := range filter {
		switch c := constraint.(type) {
		case map[string]interface{}:
			for op, val := range c {
				switch op {
				case "$eq":
					clauses = append(clauses, quote(field)+" = ?")
					args = append(args, val)
				case "$ne":
					clauses = append(clauses, quote(field)+" != ?")
					args = append(args, val)
				case "$lt":
					clauses = append(clauses, quote(field)+" < ?")
					args = append(args, val)
				case "$lte":
					clauses = append(clauses, quote(field)+" <= ?")
					args = append(args, val)
				case "$gt":
					clauses = append(clauses, quote(field)+" > ?")
					args = append(args, val)
				case "$gte":
					clauses = append(clauses, quote(field)+" >= ?")
					args = append(args, val)
				case "$in":
					if vals, ok := toSlice(val); ok {
						placeholders := make([]string, len(vals))
						for i, v := range vals {
							placeholders[i] = "?"
							args = append(args, v)
						}
						clauses = append(clauses, quote(field)+" IN ("+strings.Join(placeholders, ", ")+")")
					}
				case "$nin":
					if vals, ok := toSlice(val); ok {
						placeholders := make([]string, len(vals))
						for i, v := range vals {
							placeholders[i] = "?"
							args = append(args, v)
						}
						clauses = append(clauses, quote(field)+" NOT IN ("+strings.Join(placeholders, ", ")+")")
					}
				}
			}
		default:
			clauses = append(clauses, quote(field)+" = ?")
			args = append(args, constraint)
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

// quote wraps an identifier in double-quotes.
func quote(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// toSlice converts interface{} to []interface{} if possible.
func toSlice(v interface{}) ([]interface{}, bool) {
	s, ok := v.([]interface{})
	return s, ok
}
