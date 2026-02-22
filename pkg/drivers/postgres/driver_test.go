package postgres

import (
	"testing"

	"github.com/Uttam-Mahata/omniql/pkg/core"
)

func TestBuildWhere_Empty(t *testing.T) {
	clause, args := buildWhere(core.Filter{})
	if clause != "" {
		t.Errorf("expected empty clause, got %q", clause)
	}
	if len(args) != 0 {
		t.Errorf("expected no args, got %v", args)
	}
}

func TestBuildWhere_BareValue(t *testing.T) {
	clause, args := buildWhere(core.Filter{"status": "active"})
	if clause != `"status" = $1` {
		t.Errorf("unexpected clause: %q", clause)
	}
	if len(args) != 1 || args[0] != "active" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestBuildWhere_Operators(t *testing.T) {
	tests := []struct {
		name     string
		filter   core.Filter
		wantArgs int
	}{
		{"$eq",  core.Filter{"price": map[string]interface{}{"$eq": 100}}, 1},
		{"$ne",  core.Filter{"price": map[string]interface{}{"$ne": 100}}, 1},
		{"$lt",  core.Filter{"price": map[string]interface{}{"$lt": 500}}, 1},
		{"$lte", core.Filter{"price": map[string]interface{}{"$lte": 500}}, 1},
		{"$gt",  core.Filter{"price": map[string]interface{}{"$gt": 10}}, 1},
		{"$gte", core.Filter{"price": map[string]interface{}{"$gte": 10}}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, args := buildWhere(tc.filter)
			if len(args) != tc.wantArgs {
				t.Errorf("expected %d args, got %d", tc.wantArgs, len(args))
			}
		})
	}
}

func TestBuildWhere_InOperator(t *testing.T) {
	filter := core.Filter{
		"category": map[string]interface{}{
			"$in": []interface{}{"electronics", "books"},
		},
	}
	clause, args := buildWhere(filter)
	if clause == "" {
		t.Fatal("expected non-empty clause")
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args for $in, got %d", len(args))
	}
}

func TestBuildWhere_NinOperator(t *testing.T) {
	filter := core.Filter{
		"category": map[string]interface{}{
			"$nin": []interface{}{"spam", "junk"},
		},
	}
	_, args := buildWhere(filter)
	if len(args) != 2 {
		t.Errorf("expected 2 args for $nin, got %d", len(args))
	}
}

func TestQuote(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"table", `"table"`},
		{"my table", `"my table"`},
		{`col"name`, `"col""name"`},
	}
	for _, tc := range tests {
		got := quote(tc.in)
		if got != tc.want {
			t.Errorf("quote(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
