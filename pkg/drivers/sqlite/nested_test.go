package sqlite

import (
	"strings"
	"testing"

	"github.com/Uttam-Mahata/omniql/pkg/core"
)

func TestBuildWhere_NestedField(t *testing.T) {
	filter := core.Filter{"profile.name": "Uttam"}
	clause, args, err := buildWhere(filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: json_extract("profile", '$.name') = ?
	expected := `json_extract("profile", '$.name')`

	if !strings.Contains(clause, expected) {
		t.Errorf("expected clause to contain %q, got %q", expected, clause)
	}
	if len(args) != 1 || args[0] != "Uttam" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestBuildWhere_NestedField_Deep(t *testing.T) {
	filter := core.Filter{"data.user.id": 123}
	clause, _, err := buildWhere(filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: json_extract("data", '$.user.id')
	expected := `json_extract("data", '$.user.id')`
	if !strings.Contains(clause, expected) {
		t.Errorf("expected clause to contain %q, got %q", expected, clause)
	}
}
