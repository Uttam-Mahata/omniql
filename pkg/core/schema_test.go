package core

import (
	"testing"
)

func TestInferSchema(t *testing.T) {
	doc := map[string]interface{}{
		"title":    "SQLi",
		"points":   100, // int
		"weight":   75.5,
		"active":   true,
		"tags":     []interface{}{"web", "security"},
		"metadata": map[string]interface{}{"author": "alice"},
	}

	schema := InferSchema("challenges", doc)

	if schema.Name != "challenges" {
		t.Errorf("expected Name 'challenges', got %q", schema.Name)
	}

	expectedFields := map[string]FieldSchema{
		"title":    {Type: FieldTypeString},
		"points":   {Type: FieldTypeInt},
		"weight":   {Type: FieldTypeFloat},
		"active":   {Type: FieldTypeBool},
		"tags":     {Type: FieldTypeArray},
		"metadata": {Type: FieldTypeObject},
	}

	if len(schema.Fields) != len(expectedFields) {
		t.Errorf("expected %d fields, got %d", len(expectedFields), len(schema.Fields))
	}

	for k, expected := range expectedFields {
		actual, ok := schema.Fields[k]
		if !ok {
			t.Errorf("missing field %q", k)
			continue
		}
		if actual.Type != expected.Type {
			t.Errorf("field %q: expected type %s, got %s", k, expected.Type, actual.Type)
		}
	}
}

func TestInferFieldType(t *testing.T) {
	tests := []struct {
		val  interface{}
		want FieldType
	}{
		{"hello", FieldTypeString},
		{int(123), FieldTypeInt},
		{int64(123), FieldTypeInt},
		{float64(12.34), FieldTypeFloat},
		{true, FieldTypeBool},
		{[]interface{}{}, FieldTypeArray},
		{map[string]interface{}{}, FieldTypeObject},
		{nil, FieldTypeAny},
	}

	for _, tt := range tests {
		got := InferFieldType(tt.val)
		if got != tt.want {
			t.Errorf("InferFieldType(%v) = %s, want %s", tt.val, got, tt.want)
		}
	}
}
