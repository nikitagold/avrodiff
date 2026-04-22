package diff

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

// addressType returns an inline record map with given fields, for use as a field type.
func addressType(fields ...map[string]interface{}) interface{} {
	fieldSlice := make([]interface{}, len(fields))
	for i, f := range fields {
		fieldSlice[i] = f
	}
	return map[string]interface{}{
		"type":   "record",
		"name":   "Address",
		"fields": fieldSlice,
	}
}

func strField(name string) map[string]interface{} {
	return map[string]interface{}{"name": name, "type": "string"}
}

// schemaWithNamedRef builds a schema where field "addr" uses a string reference "Address",
// and "def" defines the Address type inline.
func schemaWithNamedRef(addrDef interface{}) *model.Schema {
	return &model.Schema{
		Type: "record",
		Name: "Order",
		Fields: []model.Field{
			{Name: "def", Type: addrDef},
			{Name: "addr", Type: "Address"},
		},
	}
}

func TestNamedTypeReference_NoChange(t *testing.T) {
	addr := addressType(strField("city"), strField("street"))

	base := schemaWithNamedRef(addr)
	head := schemaWithNamedRef(addr)

	result := DiffSchemas(base, head, model.ModeFull)
	if len(result.Changes) != 0 {
		t.Errorf("expected no changes, got %v", result.Changes)
	}
}

func TestNamedTypeReference_FieldRemovedInDefinition(t *testing.T) {
	baseAddr := addressType(strField("city"), strField("street"))
	headAddr := addressType(strField("city")) // "street" removed

	base := schemaWithNamedRef(baseAddr)
	head := schemaWithNamedRef(headAddr)

	result := DiffSchemas(base, head, model.ModeFull)

	// Should report BREAKING for both "def.street" (inline) and "addr.street" (via reference)
	breaking := 0
	for _, c := range result.Changes {
		if c.Severity == model.Breaking {
			breaking++
		}
	}
	if breaking == 0 {
		t.Errorf("expected BREAKING changes, got none. changes: %v", result.Changes)
	}
	if result.Level != model.LevelMajor {
		t.Errorf("expected MAJOR, got %s", result.Level)
	}
}

func TestNamedTypeReference_FieldAddedWithDefault(t *testing.T) {
	baseAddr := addressType(strField("city"))
	headAddr := addressType(
		strField("city"),
		map[string]interface{}{"name": "zip", "type": []interface{}{"null", "string"}, "default": nil},
	)

	base := schemaWithNamedRef(baseAddr)
	head := schemaWithNamedRef(headAddr)

	result := DiffSchemas(base, head, model.ModeFull)

	for _, c := range result.Changes {
		if c.Severity == model.Breaking {
			t.Errorf("expected no BREAKING changes, got: %v", c)
		}
	}
	if result.Level != model.LevelMinor {
		t.Errorf("expected MINOR, got %s", result.Level)
	}
}

func TestNamedTypeReference_ReferenceVsInline_SameDefinition(t *testing.T) {
	addr := addressType(strField("city"))

	// base: both fields inline
	base := &model.Schema{
		Type: "record",
		Name: "Order",
		Fields: []model.Field{
			{Name: "def", Type: addr},
			{Name: "addr", Type: addr}, // inline, same definition
		},
	}
	// head: one uses reference, one inline — same definitions
	head := &model.Schema{
		Type: "record",
		Name: "Order",
		Fields: []model.Field{
			{Name: "def", Type: addr},
			{Name: "addr", Type: "Address"}, // now a reference
		},
	}

	result := DiffSchemas(base, head, model.ModeFull)
	if len(result.Changes) != 0 {
		t.Errorf("inline vs reference with same definition should produce no changes, got %v", result.Changes)
	}
}

func TestNamedTypeReference_EnumChanged(t *testing.T) {
	baseStatus := map[string]interface{}{
		"type":    "enum",
		"name":    "Status",
		"symbols": []interface{}{"ACTIVE", "INACTIVE"},
	}
	headStatus := map[string]interface{}{
		"type":    "enum",
		"name":    "Status",
		"symbols": []interface{}{"ACTIVE"}, // "INACTIVE" removed — BREAKING
	}

	base := &model.Schema{
		Type: "record", Name: "Event",
		Fields: []model.Field{
			{Name: "statusDef", Type: baseStatus},
			{Name: "status", Type: "Status"},
		},
	}
	head := &model.Schema{
		Type: "record", Name: "Event",
		Fields: []model.Field{
			{Name: "statusDef", Type: headStatus},
			{Name: "status", Type: "Status"},
		},
	}

	result := DiffSchemas(base, head, model.ModeFull)
	if result.Level != model.LevelMajor {
		t.Errorf("removing enum symbol via named reference should be MAJOR, got %s. changes: %v",
			result.Level, result.Changes)
	}
}

func TestBuildTypeRegistry(t *testing.T) {
	addr := addressType(strField("city"))
	s := &model.Schema{
		Type: "record", Name: "Order",
		Fields: []model.Field{
			{Name: "addr", Type: addr},
		},
	}
	reg := buildTypeRegistry(s)
	if _, ok := reg["Address"]; !ok {
		t.Error("expected 'Address' in registry")
	}
}

func TestBuildTypeRegistry_NestedNamed(t *testing.T) {
	inner := map[string]interface{}{
		"type":   "record",
		"name":   "Zip",
		"fields": []interface{}{strField("code")},
	}
	outer := map[string]interface{}{
		"type": "record",
		"name": "Address",
		"fields": []interface{}{
			map[string]interface{}{"name": "city", "type": "string"},
			map[string]interface{}{"name": "zip", "type": inner},
		},
	}
	s := &model.Schema{
		Type: "record", Name: "Order",
		Fields: []model.Field{
			{Name: "addr", Type: outer},
		},
	}
	reg := buildTypeRegistry(s)
	if _, ok := reg["Address"]; !ok {
		t.Error("expected 'Address' in registry")
	}
	if _, ok := reg["Zip"]; !ok {
		t.Error("expected 'Zip' in registry (nested inside Address)")
	}
}
