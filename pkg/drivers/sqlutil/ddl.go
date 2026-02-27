package sqlutil

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Uttam-Mahata/omniql/pkg/core"
)

// TypeMapper maps OmniQL FieldType to a native SQL column type string.
type TypeMapper func(core.FieldType) string

// SQLiteTypes maps FieldType to SQLite column types.
func SQLiteTypes(ft core.FieldType) string {
	switch ft {
	case core.FieldTypeString:
		return "TEXT"
	case core.FieldTypeInt:
		return "INTEGER"
	case core.FieldTypeFloat:
		return "REAL"
	case core.FieldTypeBool:
		return "INTEGER" // SQLite has no BOOLEAN
	case core.FieldTypeArray:
		return "TEXT" // stored as JSON string
	case core.FieldTypeObject:
		return "TEXT" // stored as JSON string
	default:
		return "TEXT"
	}
}

// PostgresTypes maps FieldType to PostgreSQL column types.
func PostgresTypes(ft core.FieldType) string {
	switch ft {
	case core.FieldTypeString:
		return "TEXT"
	case core.FieldTypeInt:
		return "BIGINT"
	case core.FieldTypeFloat:
		return "DOUBLE PRECISION"
	case core.FieldTypeBool:
		return "BOOLEAN"
	case core.FieldTypeArray:
		return "JSONB"
	case core.FieldTypeObject:
		return "JSONB"
	default:
		return "TEXT"
	}
}

// MySQLTypes maps FieldType to MySQL column types.
func MySQLTypes(ft core.FieldType) string {
	switch ft {
	case core.FieldTypeString:
		return "TEXT"
	case core.FieldTypeInt:
		return "BIGINT"
	case core.FieldTypeFloat:
		return "DOUBLE"
	case core.FieldTypeBool:
		return "BOOLEAN"
	case core.FieldTypeArray:
		return "JSON"
	case core.FieldTypeObject:
		return "JSON"
	default:
		return "TEXT"
	}
}

// SQLServerTypes maps FieldType to SQL Server column types.
func SQLServerTypes(ft core.FieldType) string {
	switch ft {
	case core.FieldTypeString:
		return "NVARCHAR(MAX)"
	case core.FieldTypeInt:
		return "BIGINT"
	case core.FieldTypeFloat:
		return "FLOAT"
	case core.FieldTypeBool:
		return "BIT"
	case core.FieldTypeArray:
		return "NVARCHAR(MAX)" // stored as JSON string
	case core.FieldTypeObject:
		return "NVARCHAR(MAX)" // stored as JSON string
	default:
		return "NVARCHAR(MAX)"
	}
}

// AutoIDColumn returns the DDL fragment for an auto-incrementing primary key.
func AutoIDColumn(dialect string, quoteIdent func(string) string) string {
	switch dialect {
	case "sqlite":
		return quoteIdent("id") + " INTEGER PRIMARY KEY AUTOINCREMENT"
	case "postgres":
		return quoteIdent("id") + " BIGSERIAL PRIMARY KEY"
	case "mysql":
		return quoteIdent("id") + " BIGINT AUTO_INCREMENT PRIMARY KEY"
	case "sqlserver":
		return quoteIdent("id") + " BIGINT IDENTITY(1,1) PRIMARY KEY"
	default:
		return quoteIdent("id") + " INTEGER PRIMARY KEY"
	}
}

// BuildCreateTable generates a CREATE TABLE IF NOT EXISTS statement.
func BuildCreateTable(
	target string,
	schema *core.CollectionSchema,
	dialect string,
	quoteIdent func(string) string,
	typeMapper TypeMapper,
) string {
	var cols []string

	// Add auto-ID column if schema doesn't define "id"
	if _, hasID := schema.Fields["id"]; !hasID {
		cols = append(cols, AutoIDColumn(dialect, quoteIdent))
	}

	// Sort field names for deterministic output
	keys := make([]string, 0, len(schema.Fields))
	for k := range schema.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		field := schema.Fields[name]
		if name == "id" {
			// If schema defines "id", make it the primary key
			cols = append(cols, quoteIdent(name)+" "+typeMapper(field.Type)+" PRIMARY KEY")
			continue
		}
		colDef := quoteIdent(name) + " " + typeMapper(field.Type)
		if field.Required {
			colDef += " NOT NULL"
		}
		cols = append(cols, colDef)
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)",
		quoteIdent(target), strings.Join(cols, ",\n  "))
}
