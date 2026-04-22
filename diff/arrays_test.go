package diff

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

func arrayType(items interface{}) interface{} {
	return map[string]interface{}{"type": "array", "items": items}
}

func mapType(values interface{}) interface{} {
	return map[string]interface{}{"type": "map", "values": values}
}

func TestDiffArrays(t *testing.T) {
	tests := []struct {
		name  string
		base  *model.Schema
		head  *model.Schema
		check func(t *testing.T, changes []model.Change)
	}{
		{
			name: "no change",
			base: schema(field("tags", arrayType("string"))),
			head: schema(field("tags", arrayType("string"))),
			check: func(t *testing.T, changes []model.Change) {
				if len(changes) != 0 {
					t.Errorf("expected no changes, got %v", changes)
				}
			},
		},
		{
			name: "items type changed",
			base: schema(field("ids", arrayType("string"))),
			head: schema(field("ids", arrayType("int"))),
			check: func(t *testing.T, changes []model.Change) {
				if len(changes) != 1 || changes[0].Severity != model.Breaking {
					t.Errorf("items type change should be BREAKING, got %v", changes)
				}
				if changes[0].Path != "fields.ids" {
					t.Errorf("path: got %q, want %q", changes[0].Path, "fields.ids")
				}
			},
		},
		{
			name: "items widened",
			base: schema(field("ids", arrayType("string"))),
			head: schema(field("ids", arrayType([]interface{}{"null", "string"}))),
			check: func(t *testing.T, changes []model.Change) {
				if len(changes) != 1 || changes[0].Severity != model.Safe {
					t.Errorf("items widening should be SAFE, got %v", changes)
				}
			},
		},
		{
			name: "items narrowed",
			base: schema(field("ids", arrayType([]interface{}{"null", "string"}))),
			head: schema(field("ids", arrayType("string"))),
			check: func(t *testing.T, changes []model.Change) {
				if len(changes) != 1 || changes[0].Severity != model.Breaking {
					t.Errorf("items narrowing should be BREAKING, got %v", changes)
				}
			},
		},
		{
			name: "field removed inside item record",
			base: schema(field("tags", arrayType(map[string]interface{}{
				"type": "record",
				"name": "Tag",
				"fields": []interface{}{
					strField("id"),
					strField("name"),
				},
			}))),
			head: schema(field("tags", arrayType(map[string]interface{}{
				"type": "record",
				"name": "Tag",
				"fields": []interface{}{
					strField("id"),
				},
			}))),
			check: func(t *testing.T, changes []model.Change) {
				if len(changes) != 1 || changes[0].Severity != model.Breaking {
					t.Errorf("field removed inside array item should be BREAKING, got %v", changes)
				}
				if changes[0].Path != "fields.tags.items.fields.name" {
					t.Errorf("path: got %q, want %q", changes[0].Path, "fields.tags.items.fields.name")
				}
			},
		},
		{
			name: "optional field added inside item record",
			base: schema(field("tags", arrayType(map[string]interface{}{
				"type": "record",
				"name": "Tag",
				"fields": []interface{}{
					strField("id"),
				},
			}))),
			head: schema(field("tags", arrayType(map[string]interface{}{
				"type": "record",
				"name": "Tag",
				"fields": []interface{}{
					strField("id"),
					map[string]interface{}{
						"name":    "label",
						"type":    []interface{}{"null", "string"},
						"default": nil,
					},
				},
			}))),
			check: func(t *testing.T, changes []model.Change) {
				if len(changes) != 1 || changes[0].Severity != model.Safe {
					t.Errorf("optional field added in array item should be SAFE, got %v", changes)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := diffFields(tt.base, tt.head, "", newCtx(tt.base, tt.head, model.ModeFull))
			tt.check(t, changes)
		})
	}
}

// TestDiffArray_NamedTypeChanged uses DiffSchemas (not diffFields) because named type
// resolution requires a top-level registry built from the full schema.
func TestDiffArray_NamedTypeChanged(t *testing.T) {
	base := &model.Schema{
		Type: "record",
		Name: "Event",
		Fields: []model.Field{
			{
				Name: "tagDef",
				Type: map[string]interface{}{
					"type": "record",
					"name": "Tag",
					"fields": []interface{}{
						strField("id"),
						strField("name"),
					},
				},
			},
			{
				Name: "tags",
				Type: arrayType("Tag"),
			},
		},
	}
	head := &model.Schema{
		Type: "record",
		Name: "Event",
		Fields: []model.Field{
			{
				Name: "tagDef",
				Type: map[string]interface{}{
					"type": "record",
					"name": "Tag",
					"fields": []interface{}{
						strField("id"),
					},
				},
			},
			{
				Name: "tags",
				Type: arrayType("Tag"),
			},
		},
	}
	result := DiffSchemas(base, head, model.ModeFull)
	if result.Level != model.LevelMajor {
		t.Errorf("breaking change inside array named item should be MAJOR, got %s. changes: %v",
			result.Level, result.Changes)
	}
}

func TestDiffMaps(t *testing.T) {
	tests := []struct {
		name  string
		base  *model.Schema
		head  *model.Schema
		check func(t *testing.T, changes []model.Change)
	}{
		{
			name: "no change",
			base: schema(field("meta", mapType("string"))),
			head: schema(field("meta", mapType("string"))),
			check: func(t *testing.T, changes []model.Change) {
				if len(changes) != 0 {
					t.Errorf("expected no changes, got %v", changes)
				}
			},
		},
		{
			name: "values type changed",
			base: schema(field("meta", mapType("string"))),
			head: schema(field("meta", mapType("int"))),
			check: func(t *testing.T, changes []model.Change) {
				if len(changes) != 1 || changes[0].Severity != model.Breaking {
					t.Errorf("map values type change should be BREAKING, got %v", changes)
				}
				if changes[0].Path != "fields.meta" {
					t.Errorf("path: got %q, want %q", changes[0].Path, "fields.meta")
				}
			},
		},
		{
			name: "values widened",
			base: schema(field("meta", mapType("string"))),
			head: schema(field("meta", mapType([]interface{}{"null", "string"}))),
			check: func(t *testing.T, changes []model.Change) {
				if len(changes) != 1 || changes[0].Severity != model.Safe {
					t.Errorf("map values widening should be SAFE, got %v", changes)
				}
			},
		},
		{
			name: "field removed inside value record",
			base: schema(field("meta", mapType(map[string]interface{}{
				"type": "record",
				"name": "Meta",
				"fields": []interface{}{
					strField("key"),
					strField("value"),
				},
			}))),
			head: schema(field("meta", mapType(map[string]interface{}{
				"type": "record",
				"name": "Meta",
				"fields": []interface{}{
					strField("key"),
				},
			}))),
			check: func(t *testing.T, changes []model.Change) {
				if len(changes) != 1 || changes[0].Severity != model.Breaking {
					t.Errorf("field removed inside map value record should be BREAKING, got %v", changes)
				}
				if changes[0].Path != "fields.meta.values.fields.value" {
					t.Errorf("path: got %q, want %q", changes[0].Path, "fields.meta.values.fields.value")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := diffFields(tt.base, tt.head, "", newCtx(tt.base, tt.head, model.ModeFull))
			tt.check(t, changes)
		})
	}
}

func TestTypePromotion(t *testing.T) {
	promotions := [][2]string{
		{"int", "long"}, {"int", "float"}, {"int", "double"},
		{"long", "float"}, {"long", "double"},
		{"float", "double"},
		{"string", "bytes"}, {"bytes", "string"},
	}
	for _, p := range promotions {
		if !isTypePromotion(p[0], p[1]) {
			t.Errorf("expected %s→%s to be a valid promotion", p[0], p[1])
		}
	}
	notPromotions := [][2]string{
		{"double", "float"}, {"long", "int"}, {"int", "string"},
	}
	for _, p := range notPromotions {
		if isTypePromotion(p[0], p[1]) {
			t.Errorf("expected %s→%s NOT to be a valid promotion", p[0], p[1])
		}
	}
}

func TestA02_ArrayItemsPromotion(t *testing.T) {
	tests := []struct {
		from, to string
	}{
		{"int", "long"},
		{"float", "double"},
		{"string", "bytes"},
	}
	for _, tt := range tests {
		base := schema(field("ids", arrayType(tt.from)))
		head := schema(field("ids", arrayType(tt.to)))
		changes := diffFields(base, head, "", newCtx(base, head, model.ModeFull))
		if len(changes) != 1 {
			t.Fatalf("A-02 %s→%s: expected 1 change, got %d: %v", tt.from, tt.to, len(changes), changes)
		}
		if changes[0].Rule != "A-02" {
			t.Errorf("A-02 %s→%s: expected rule A-02, got %q", tt.from, tt.to, changes[0].Rule)
		}
		if changes[0].Severity != model.Safe {
			t.Errorf("A-02 %s→%s: expected SAFE, got %s", tt.from, tt.to, changes[0].Severity)
		}
	}
}

func TestM02_MapValuesPromotion(t *testing.T) {
	base := schema(field("counts", mapType("int")))
	head := schema(field("counts", mapType("long")))
	changes := diffFields(base, head, "", newCtx(base, head, model.ModeFull))
	if len(changes) != 1 || changes[0].Rule != "M-02" {
		t.Fatalf("M-02: expected rule M-02, got %v", changes)
	}
	if changes[0].Severity != model.Safe {
		t.Errorf("M-02: expected SAFE, got %s", changes[0].Severity)
	}
}

func TestDiffArrayToMap(t *testing.T) {
	base := schema(field("data", arrayType("string")))
	head := schema(field("data", mapType("string")))
	changes := diffFields(base, head, "", newCtx(base, head, model.ModeFull))
	if len(changes) != 1 || changes[0].Severity != model.Breaking {
		t.Errorf("array → map should be BREAKING, got %v", changes)
	}
}
