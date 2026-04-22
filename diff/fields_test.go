package diff

import (
	"strings"
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

func schema(fields ...model.Field) *model.Schema {
	return &model.Schema{Type: "record", Name: "Test", Fields: fields}
}

func field(name string, typ interface{}) model.Field {
	return model.Field{Name: name, Type: typ}
}

func fieldWithDefault(name string, typ interface{}, def interface{}) model.Field {
	return model.Field{Name: name, Type: typ, Default: def, HasDefault: true}
}

func TestDiffFields(t *testing.T) {
	tests := []struct {
		name        string
		base        *model.Schema
		head        *model.Schema
		wantPath    string
		wantSev     model.Severity
		wantDescSub string // optional substring of Description to verify correct branch
	}{
		{
			name:     "field removed",
			base:     schema(field("email", "string")),
			head:     schema(),
			wantPath: "fields.email",
			wantSev:  model.Breaking,
		},
		{
			name:     "field added without default",
			base:     schema(),
			head:     schema(field("phone", "string")),
			wantPath: "fields.phone",
			wantSev:  model.Breaking,
		},
		{
			name:     "field added with default",
			base:     schema(),
			head:     schema(fieldWithDefault("phone", []interface{}{"null", "string"}, nil)),
			wantPath: "fields.phone",
			wantSev:  model.Safe,
		},
		{
			name:     "type changed",
			base:     schema(field("age", "int")),
			head:     schema(field("age", "string")),
			wantPath: "fields.age",
			wantSev:  model.Breaking,
		},
		{
			name:     "nullable widening safe",
			base:     schema(field("name", "string")),
			head:     schema(field("name", []interface{}{"null", "string"})),
			wantPath: "fields.name",
			wantSev:  model.Safe,
		},
		{
			name:     "nullable narrowing breaking",
			base:     schema(field("name", []interface{}{"null", "string"})),
			head:     schema(field("name", "string")),
			wantPath: "fields.name",
			wantSev:  model.Breaking,
		},
		{
			name: "no changes",
			base: schema(field("id", "string")),
			head: schema(field("id", "string")),
		},
		// FIX-1: nullable widening for complex (named) types
		{
			// inline record → ["null", "NamedType"] должен быть safe widening
			name: "nullable widening safe: inline record to nullable union",
			base: schema(field("address", map[string]interface{}{
				"type":   "record",
				"name":   "Address",
				"fields": []interface{}{map[string]interface{}{"name": "city", "type": "string"}},
			})),
			head:     schema(field("address", []interface{}{"null", "Address"})),
			wantPath: "fields.address",
			wantSev:  model.Safe,
		},
		{
			// ["null", "NamedType"] → inline record должен быть breaking (narrowing)
			name: "nullable narrowing breaking: nullable union to inline record",
			base: schema(field("address", []interface{}{"null", "Address"})),
			head: schema(field("address", map[string]interface{}{
				"type":   "record",
				"name":   "Address",
				"fields": []interface{}{map[string]interface{}{"name": "city", "type": "string"}},
			})),
			wantPath:    "fields.address",
			wantSev:     model.Breaking,
			wantDescSub: "narrowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := diffFields(tt.base, tt.head, "", newCtx(tt.base, tt.head, model.ModeFull))
			if tt.wantPath == "" {
				if len(changes) != 0 {
					t.Fatalf("expected no changes, got %v", changes)
				}
				return
			}
			if len(changes) == 0 {
				t.Fatal("expected changes, got none")
			}
			got := changes[0]
			if got.Path != tt.wantPath {
				t.Errorf("path: got %q, want %q", got.Path, tt.wantPath)
			}
			if got.Severity != tt.wantSev {
				t.Errorf("severity: got %q, want %q — %s", got.Severity, tt.wantSev, got.Reason)
			}
			if tt.wantDescSub != "" && !strings.Contains(got.Description, tt.wantDescSub) {
				t.Errorf("description: got %q, want it to contain %q", got.Description, tt.wantDescSub)
			}
		})
	}
}

func fieldWithAliases(name string, typ interface{}, aliases ...string) model.Field {
	return model.Field{Name: name, Type: typ, Aliases: aliases}
}

func TestFieldRules(t *testing.T) {
	tests := []struct {
		name     string
		base     *model.Schema
		head     *model.Schema
		wantRule string
		wantSev  model.Severity
	}{
		{
			name:     "F-07: alias removed → MAJOR",
			base:     schema(fieldWithAliases("id", "string", "userId")),
			head:     schema(field("id", "string")),
			wantRule: "F-07",
			wantSev:  model.Breaking,
		},
		{
			name:     "F-08: alias added → MINOR",
			base:     schema(field("id", "string")),
			head:     schema(fieldWithAliases("id", "string", "userId")),
			wantRule: "F-08",
			wantSev:  model.Safe,
		},
		{
			name:     "F-11: field order changed → MAJOR",
			base:     schema(field("a", "string"), field("b", "string")),
			head:     schema(field("b", "string"), field("a", "string")),
			wantRule: "F-11",
			wantSev:  model.Breaking,
		},
		{
			name:     "F-13: default added → MINOR",
			base:     schema(field("country", "string")),
			head:     schema(fieldWithDefault("country", "string", "US")),
			wantRule: "F-13",
			wantSev:  model.Safe,
		},
		{
			name:     "F-14: default removed → MAJOR",
			base:     schema(fieldWithDefault("country", "string", "US")),
			head:     schema(field("country", "string")),
			wantRule: "F-14",
			wantSev:  model.Breaking,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := diffFields(tt.base, tt.head, "", newCtx(tt.base, tt.head, model.ModeFull))
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

func TestDiffSameSchema(t *testing.T) {
	base := schema(
		field("id", "string"),
		fieldWithDefault("age", "int", 0),
	)
	result := DiffSchemas(base, base, model.ModeFull)
	if len(result.Changes) != 0 {
		t.Errorf("same schema should produce no changes, got %v", result.Changes)
	}
	if result.Level != model.LevelNone {
		t.Errorf("expected NONE, got %s", result.Level)
	}
}
