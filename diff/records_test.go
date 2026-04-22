package diff

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

func recordSchema(name, namespace string, aliases []string, fields ...model.Field) model.Schema {
	return model.Schema{Type: "record", Name: name, Namespace: namespace, Aliases: aliases, Fields: fields}
}

func TestRecordRules(t *testing.T) {
	tests := []struct {
		name     string
		base     model.Schema
		head     model.Schema
		wantRule string
		wantSev  model.Severity
	}{
		{
			name:     "R-02: rename without alias → MAJOR",
			base:     recordSchema("User", "", nil),
			head:     recordSchema("UserV2", "", nil),
			wantRule: "R-02",
			wantSev:  model.Breaking,
		},
		{
			name:     "R-03: rename with alias covering old name → MINOR",
			base:     recordSchema("User", "", nil),
			head:     recordSchema("UserV2", "", []string{"User"}),
			wantRule: "R-03",
			wantSev:  model.Safe,
		},
		{
			name:     "R-04: namespace changed → MAJOR",
			base:     recordSchema("User", "com.example", nil),
			head:     recordSchema("User", "com.example.v2", nil),
			wantRule: "R-04",
			wantSev:  model.Breaking,
		},
		{
			name:     "R-05: alias removed → MAJOR",
			base:     recordSchema("User", "", []string{"LegacyUser"}),
			head:     recordSchema("User", "", nil),
			wantRule: "R-05",
			wantSev:  model.Breaking,
		},
		{
			name:     "R-06: alias added → MINOR",
			base:     recordSchema("User", "", nil),
			head:     recordSchema("User", "", []string{"LegacyUser"}),
			wantRule: "R-06",
			wantSev:  model.Safe,
		},
		{
			name:     "R-07: doc changed → PATCH",
			base:     model.Schema{Type: "record", Name: "User", Doc: "old doc"},
			head:     model.Schema{Type: "record", Name: "User", Doc: "new doc"},
			wantRule: "R-07",
			wantSev:  model.Cosmetic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &DiffContext{
				BaseTypes: buildTypeRegistry(&model.Schema{Type: "record", Name: tt.base.Name, Fields: tt.base.Fields}),
				HeadTypes: buildTypeRegistry(&model.Schema{Type: "record", Name: tt.head.Name, Fields: tt.head.Fields}),
				Mode:      model.ModeFull,
			}
			changes := diffRecord(tt.base, tt.head, "", ctx)
			var found *model.Change
			for i := range changes {
				if changes[i].Rule == tt.wantRule {
					found = &changes[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("rule %s not found in changes: %v", tt.wantRule, changes)
			}
			if found.Severity != tt.wantSev {
				t.Errorf("severity: got %q, want %q — %s", found.Severity, tt.wantSev, found.Reason)
			}
		})
	}
}

func TestDiffNestedRecord(t *testing.T) {
	t.Run("nested field removed", func(t *testing.T) {
		base := schema(model.Field{
			Name: "address",
			Type: map[string]interface{}{
				"type": "record",
				"name": "Address",
				"fields": []interface{}{
					map[string]interface{}{"name": "city", "type": "string"},
					map[string]interface{}{"name": "street", "type": "string"},
				},
			},
		})
		head := schema(model.Field{
			Name: "address",
			Type: map[string]interface{}{
				"type": "record",
				"name": "Address",
				"fields": []interface{}{
					map[string]interface{}{"name": "city", "type": "string"},
					// "street" removed
				},
			},
		})

		changes := diffFields(base, head, "", newCtx(base, head, model.ModeFull))
		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d: %v", len(changes), changes)
		}
		c := changes[0]
		if c.Path != "fields.address.fields.street" {
			t.Errorf("path: got %q, want %q", c.Path, "fields.address.fields.street")
		}
		if c.Severity != model.Breaking {
			t.Errorf("severity: got %q, want BREAKING", c.Severity)
		}
	})

	t.Run("nested field added with default", func(t *testing.T) {
		base := schema(model.Field{
			Name: "address",
			Type: map[string]interface{}{
				"type":   "record",
				"name":   "Address",
				"fields": []interface{}{map[string]interface{}{"name": "city", "type": "string"}},
			},
		})
		head := schema(model.Field{
			Name: "address",
			Type: map[string]interface{}{
				"type": "record",
				"name": "Address",
				"fields": []interface{}{
					map[string]interface{}{"name": "city", "type": "string"},
					map[string]interface{}{"name": "zip", "type": []interface{}{"null", "string"}, "default": nil},
				},
			},
		})

		changes := diffFields(base, head, "", newCtx(base, head, model.ModeFull))
		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d: %v", len(changes), changes)
		}
		c := changes[0]
		if c.Path != "fields.address.fields.zip" {
			t.Errorf("path: got %q, want %q", c.Path, "fields.address.fields.zip")
		}
		if c.Severity != model.Safe {
			t.Errorf("severity: got %q, want SAFE", c.Severity)
		}
	})

	t.Run("no changes in nested record", func(t *testing.T) {
		base := schema(model.Field{
			Name: "address",
			Type: map[string]interface{}{
				"type": "record",
				"name": "Address",
				"fields": []interface{}{
					map[string]interface{}{"name": "city", "type": "string"},
					map[string]interface{}{"name": "street", "type": "string"},
				},
			},
		})
		result := DiffSchemas(base, base, model.ModeFull)
		if len(result.Changes) != 0 {
			t.Errorf("same schema should produce no changes, got %v", result.Changes)
		}
	})
}
