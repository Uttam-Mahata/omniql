package postgres

import (
	"strings"
	"testing"

	"github.com/Uttam-Mahata/omniql/pkg/core"
)

func TestBuildWhere_Empty(t *testing.T) {
	clause, args, err := buildWhere(core.Filter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clause != "" {
		t.Errorf("expected empty clause, got %q", clause)
	}
	if len(args) != 0 {
		t.Errorf("expected no args, got %v", args)
	}
}

func TestBuildWhere_BareValue(t *testing.T) {
	clause, args, err := buildWhere(core.Filter{"status": "active"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
		{"$eq", core.Filter{"price": map[string]interface{}{"$eq": 100}}, 1},
		{"$ne", core.Filter{"price": map[string]interface{}{"$ne": 100}}, 1},
		{"$lt", core.Filter{"price": map[string]interface{}{"$lt": 500}}, 1},
		{"$lte", core.Filter{"price": map[string]interface{}{"$lte": 500}}, 1},
		{"$gt", core.Filter{"price": map[string]interface{}{"$gt": 10}}, 1},
		{"$gte", core.Filter{"price": map[string]interface{}{"$gte": 10}}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, args, err := buildWhere(tc.filter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
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
	clause, args, err := buildWhere(filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	_, args, err := buildWhere(filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args for $nin, got %d", len(args))
	}
}

func TestBuildWhere_LogicalOps(t *testing.T) {
	filter := core.Filter{
		"$or": []interface{}{
			map[string]interface{}{"status": "active"},
			map[string]interface{}{"age": map[string]interface{}{"$gt": 18}},
		},
	}
	clause, args, err := buildWhere(filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expected clause: ("status" = $1) OR ("age" > $2)
	// Or similar structure
	if !strings.Contains(clause, " OR ") {
		t.Errorf("expected OR clause, got %q", clause)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}
}

func TestBuildWhere_ErrorCases(t *testing.T) {
	tests := []struct {
		name   string
		filter core.Filter
		errMsg string
	}{
		{
			"empty $in",
			core.Filter{"f": map[string]interface{}{"$in": []interface{}{}}},
			"requires non-empty slice",
		},
		{
			"unknown operator",
			core.Filter{"f": map[string]interface{}{"$regex": ".*"}},
			"unsupported operator",
		},
		{
			"invalid $or",
			core.Filter{"$or": "not-a-slice"},
			"requires a slice",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := buildWhere(tc.filter)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("expected error containing %q, got %v", tc.errMsg, err)
			}
		})
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
