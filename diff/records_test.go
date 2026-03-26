package diff

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

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
		base, err := model.ReadSchema("../testdata/order-base.avsc")
		if err != nil {
			t.Fatal(err)
		}
		result := DiffSchemas(base, base, model.ModeFull)
		if len(result.Changes) != 0 {
			t.Errorf("same schema should produce no changes, got %v", result.Changes)
		}
	})
}
