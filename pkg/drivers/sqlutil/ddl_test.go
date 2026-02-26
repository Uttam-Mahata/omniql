package sqlutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Uttam-Mahata/omniql/pkg/core"
)

func TestBuildCreateTable(t *testing.T) {
	schema := &core.CollectionSchema{
		Name: "users",
		Fields: map[string]core.FieldSchema{
			"name": {Type: core.FieldTypeString},
			"age":  {Type: core.FieldTypeInt},
		},
	}

	quoteIdent := func(s string) string { return fmt.Sprintf(`"%s"`, s) }

	// Test Postgres (no ID in schema -> auto-generated ID)
	ddl := BuildCreateTable("users", schema, "postgres", quoteIdent, PostgresTypes)
	if !strings.Contains(ddl, `"id" BIGSERIAL PRIMARY KEY`) {
		t.Errorf("postgres DDL missing auto ID: %s", ddl)
	}
	if !strings.Contains(ddl, `"name" TEXT`) {
		t.Errorf("postgres DDL missing name column: %s", ddl)
	}
	if !strings.Contains(ddl, `"age" BIGINT`) {
		t.Errorf("postgres DDL missing age column: %s", ddl)
	}

	// Test SQLite (ID in schema)
	schemaWithID := &core.CollectionSchema{
		Name: "items",
		Fields: map[string]core.FieldSchema{
			"id":    {Type: core.FieldTypeInt},
			"price": {Type: core.FieldTypeFloat},
		},
	}
	ddl = BuildCreateTable("items", schemaWithID, "sqlite", quoteIdent, SQLiteTypes)
	if !strings.Contains(ddl, `"id" INTEGER PRIMARY KEY`) {
		t.Errorf("sqlite DDL missing ID primary key: %s", ddl)
	}
	if !strings.Contains(ddl, `"price" REAL`) {
		t.Errorf("sqlite DDL missing price column: %s", ddl)
	}
    if strings.Contains(ddl, "AUTOINCREMENT") {
        // If ID is explicit, we don't force AUTOINCREMENT unless schema implies it,
        // but currently we just use typeMapper(FieldTypeInt) + PRIMARY KEY.
        // So it should be just INTEGER PRIMARY KEY.
    }
}

func TestAutoIDColumn(t *testing.T) {
	quoteIdent := func(s string) string { return fmt.Sprintf("`%s`", s) }

	tests := []struct {
		dialect string
		want    string
	}{
		{"sqlite", "`id` INTEGER PRIMARY KEY AUTOINCREMENT"},
		{"postgres", "`id` BIGSERIAL PRIMARY KEY"},
		{"mysql", "`id` BIGINT AUTO_INCREMENT PRIMARY KEY"},
		{"sqlserver", "`id` BIGINT IDENTITY(1,1) PRIMARY KEY"},
		{"oracle", "`id` INTEGER PRIMARY KEY"}, // default
	}

	for _, tt := range tests {
		got := AutoIDColumn(tt.dialect, quoteIdent)
		if got != tt.want {
			t.Errorf("AutoIDColumn(%q) = %q, want %q", tt.dialect, got, tt.want)
		}
	}
}
