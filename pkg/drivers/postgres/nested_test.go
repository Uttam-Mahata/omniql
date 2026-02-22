package postgres

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

	if !strings.Contains(clause, `"profile"->>'name'`) {
		t.Errorf("expected clause to contain %q, got %q", `"profile"->>'name'`, clause)
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

	// Expected: "data"->'user'->>'id' = $1
	if !strings.Contains(clause, `"data"->'user'->>'id'`) {
		t.Errorf("expected clause to contain %q, got %q", `"data"->'user'->>'id'`, clause)
	}
}
