package diff

import (
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
		name     string
		base     *model.Schema
		head     *model.Schema
		wantPath string
		wantSev  model.Severity
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
		})
	}
}

func TestDiffFromFiles(t *testing.T) {
	base, err := model.ReadSchema("../testdata/user-base.avsc")
	if err != nil {
		t.Fatal(err)
	}
	// Same file = no changes
	result := DiffSchemas(base, base, model.ModeFull)
	if len(result.Changes) != 0 {
		t.Errorf("same schema should produce no changes, got %v", result.Changes)
	}
	if result.Level != model.LevelNone {
		t.Errorf("expected NONE, got %s", result.Level)
	}
}
