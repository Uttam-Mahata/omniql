// Package mysql provides an OmniQL driver for MySQL databases.
// It translates OQL queries into standard SQL using the go-sql-driver/mysql driver.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	_ "github.com/go-sql-driver/mysql" // MySQL database driver
)

const driverName = "mysql"

// Driver is the OmniQL MySQL adapter.
type Driver struct {
	db *sql.DB
}

// New opens a MySQL database using the given DSN and returns a Driver wrapping it.
// DSN format: "user:password@tcp(host:port)/dbname?parseTime=true"
func New(dsn string) (*Driver, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql: open %q: %w", dsn, err)
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
		return nil, 0, fmt.Errorf("mysql: unsupported action %q", query.Action)
	}
}

func (d *Driver) find(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	where, args, err := buildWhere(query.Filter)
	if err != nil {
		return nil, 0, err
	}

	countSQL := "SELECT COUNT(*) FROM " + quote(query.Target)
	if where != "" {
		countSQL += " WHERE " + where
	}
	var total int64
	if err := d.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("mysql count: %w", err)
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
				return nil, 0, fmt.Errorf("mysql: %w", err)
			}
			if inc {
				cols = append(cols, quote(field))
			}
		}
		if len(cols) > 0 {
			sort.Strings(cols)
			fields = strings.Join(cols, ", ")
		}
	}

	selectSQL := "SELECT " + fields + " FROM " + quote(query.Target)
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
		return nil, 0, fmt.Errorf("mysql select: %w", err)
	}
	defer rows.Close()

	return scanRows(rows, total)
}

func (d *Driver) insert(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Document) == 0 {
		return nil, 0, fmt.Errorf("mysql: INSERT requires a non-empty document")
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
		return nil, 0, fmt.Errorf("mysql insert: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, 0, fmt.Errorf("mysql insert id: %w", err)
	}
	return []map[string]interface{}{{"id": id}}, 1, nil
}

// BatchInsert implements core.Driver.
func (d *Driver) BatchInsert(ctx context.Context, target string, docs []map[string]interface{}) ([]map[string]interface{}, error) {
	if len(docs) == 0 {
		return []map[string]interface{}{}, nil
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("mysql batch: begin tx: %w", err)
	}
	defer tx.Rollback()

	first := docs[0]
	cols := make([]string, 0, len(first))
	colOrder := make([]string, 0, len(first))
	placeholders := make([]string, 0, len(first))
	for col := range first {
		cols = append(cols, quote(col))
		colOrder = append(colOrder, col)
		placeholders = append(placeholders, "?")
	}

	stmtStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quote(target),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	stmt, err := tx.PrepareContext(ctx, stmtStr)
	if err != nil {
		return nil, fmt.Errorf("mysql batch: prepare: %w", err)
	}
	defer stmt.Close()

	for _, doc := range docs {
		vals := make([]interface{}, 0, len(first))
		for _, col := range colOrder {
			vals = append(vals, doc[col])
		}
		if _, err := stmt.ExecContext(ctx, vals...); err != nil {
			return nil, fmt.Errorf("mysql batch: exec: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("mysql batch: commit: %w", err)
	}

	return []map[string]interface{}{{"count": int64(len(docs))}}, nil
}

func (d *Driver) update(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Document) == 0 {
		return nil, 0, fmt.Errorf("mysql: UPDATE requires a non-empty document")
	}
	if len(query.Filter) == 0 {
		return nil, 0, fmt.Errorf("mysql: UPDATE requires a non-empty filter to prevent accidental bulk updates")
	}

	setClauses := make([]string, 0, len(query.Document))
	args := make([]interface{}, 0, len(query.Document))
	for col, val := range query.Document {
		setClauses = append(setClauses, quote(col)+" = ?")
		args = append(args, val)
	}

	where, whereArgs, err := buildWhere(query.Filter)
	if err != nil {
		return nil, 0, err
	}
	args = append(args, whereArgs...)

	stmt := "UPDATE " + quote(query.Target) + " SET " + strings.Join(setClauses, ", ")
	if where != "" {
		stmt += " WHERE " + where
	}

	result, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mysql update: %w", err)
	}
	affected, _ := result.RowsAffected()
	return []map[string]interface{}{}, affected, nil
}

func (d *Driver) delete(ctx context.Context, query core.OQLQuery) ([]map[string]interface{}, int64, error) {
	if len(query.Filter) == 0 {
		return nil, 0, fmt.Errorf("mysql: DELETE requires a non-empty filter to prevent accidental bulk deletes")
	}
	where, args, err := buildWhere(query.Filter)
	if err != nil {
		return nil, 0, err
	}

	stmt := "DELETE FROM " + quote(query.Target)
	if where != "" {
		stmt += " WHERE " + where
	}

	result, err := d.db.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mysql delete: %w", err)
	}
	affected, _ := result.RowsAffected()
	return []map[string]interface{}{}, affected, nil
}

// buildWhere translates an OQL Filter into a parameterised SQL WHERE clause.
// MySQL uses ? as the parameter placeholder.
func buildWhere(filter core.Filter) (string, []interface{}, error) {
	if len(filter) == 0 {
		return "", nil, nil
	}

	clauses := make([]string, 0, len(filter))
	args := make([]interface{}, 0, len(filter))

	for field, constraint := range filter {
		if field == "$or" || field == "$and" {
			list, ok := toSlice(constraint)
			if !ok {
				return "", nil, fmt.Errorf("mysql: %s requires a slice", field)
			}
			subClauses := make([]string, 0, len(list))
			for _, item := range list {
				subFilter, ok := item.(map[string]interface{})
				if !ok {
					return "", nil, fmt.Errorf("mysql: %s elements must be objects", field)
				}
				subWhere, subArgs, err := buildWhere(subFilter)
				if err != nil {
					return "", nil, err
				}
				if subWhere != "" {
					subClauses = append(subClauses, "("+subWhere+")")
					args = append(args, subArgs...)
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

		switch c := constraint.(type) {
		case map[string]interface{}:
			for op, val := range c {
				switch op {
				case "$eq":
					clauses = append(clauses, translateColumn(field)+" = ?")
					args = append(args, val)
				case "$ne":
					clauses = append(clauses, translateColumn(field)+" != ?")
					args = append(args, val)
				case "$lt":
					clauses = append(clauses, translateColumn(field)+" < ?")
					args = append(args, val)
				case "$lte":
					clauses = append(clauses, translateColumn(field)+" <= ?")
					args = append(args, val)
				case "$gt":
					clauses = append(clauses, translateColumn(field)+" > ?")
					args = append(args, val)
				case "$gte":
					clauses = append(clauses, translateColumn(field)+" >= ?")
					args = append(args, val)
				case "$in":
					if vals, ok := toSlice(val); ok {
						if len(vals) == 0 {
							return "", nil, fmt.Errorf("mysql: $in requires non-empty slice")
						}
						placeholders := make([]string, len(vals))
						for i, v := range vals {
							placeholders[i] = "?"
							args = append(args, v)
						}
						clauses = append(clauses, translateColumn(field)+" IN ("+strings.Join(placeholders, ", ")+")")
					} else {
						return "", nil, fmt.Errorf("mysql: $in requires a slice")
					}
				case "$nin":
					if vals, ok := toSlice(val); ok {
						if len(vals) == 0 {
							return "", nil, fmt.Errorf("mysql: $nin requires non-empty slice")
						}
						placeholders := make([]string, len(vals))
						for i, v := range vals {
							placeholders[i] = "?"
							args = append(args, v)
						}
						clauses = append(clauses, translateColumn(field)+" NOT IN ("+strings.Join(placeholders, ", ")+")")
					} else {
						return "", nil, fmt.Errorf("mysql: $nin requires a slice")
					}
				default:
					return "", nil, fmt.Errorf("mysql: unsupported operator %q", op)
				}
			}
		default:
			clauses = append(clauses, translateColumn(field)+" = ?")
			args = append(args, constraint)
		}
	}

	return strings.Join(clauses, " AND "), args, nil
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

// quote wraps an identifier in backticks (MySQL identifier quoting).
func quote(ident string) string {
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
}

// translateColumn converts a dot-notated field (e.g. "profile.name") into
// a MySQL JSON path expression using the ->> operator (MySQL 5.7+).
// If the field has no dots, it is simply quoted.
func translateColumn(field string) string {
	parts := strings.Split(field, ".")
	if len(parts) == 1 {
		return quote(field)
	}

	// First part is the column name; remaining parts form the JSON path.
	col := quote(parts[0])
	path := "$." + strings.Join(parts[1:], ".")
	path = strings.ReplaceAll(path, "'", "''")

	// ->> returns unquoted text value (equivalent to JSON_UNQUOTE(JSON_EXTRACT(...)))
	return fmt.Sprintf("%s->>'%s'", col, path)
}

// toSlice converts interface{} to []interface{} if possible.
func toSlice(v interface{}) ([]interface{}, bool) {
	s, ok := v.([]interface{})
	return s, ok
}

// fieldProjectionValue returns (true, nil) for inclusion (1), (false, nil) for
// unknown values, and (false, error) when an exclusion value (0) is detected.
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
