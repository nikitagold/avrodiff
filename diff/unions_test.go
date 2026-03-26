package diff

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

func union(types ...interface{}) []interface{} {
	return types
}

func TestDiffUnions(t *testing.T) {
	tests := []struct {
		name    string
		mode    model.CompatMode
		base    []interface{}
		head    []interface{}
		wantN   int
		wantSev model.Severity
	}{
		{
			name:  "no changes",
			mode:  model.ModeFull,
			base:  union("null", "string"),
			head:  union("null", "string"),
			wantN: 0,
		},
		{
			// FULL: type removed → BREAKING (old data may contain it)
			name:    "type removed full",
			mode:    model.ModeFull,
			base:    union("null", "string", "int"),
			head:    union("null", "string"),
			wantN:   1,
			wantSev: model.Breaking,
		},
		{
			// FORWARD: type removed → SAFE (new data won't contain removed type)
			name:    "type removed forward",
			mode:    model.ModeForward,
			base:    union("null", "string", "int"),
			head:    union("null", "string"),
			wantN:   1,
			wantSev: model.Safe,
		},
		{
			// FULL: type added → BREAKING (old readers don't know the new type)
			name:    "type added full",
			mode:    model.ModeFull,
			base:    union("null", "string"),
			head:    union("null", "string", "int"),
			wantN:   1,
			wantSev: model.Breaking,
		},
		{
			// BACKWARD: type added → SAFE (old data doesn't contain the new type)
			name:    "type added backward",
			mode:    model.ModeBackward,
			base:    union("null", "string"),
			head:    union("null", "string", "int"),
			wantN:   1,
			wantSev: model.Safe,
		},
		{
			name:    "order changed",
			mode:    model.ModeFull,
			base:    union("null", "string"),
			head:    union("string", "null"),
			wantN:   1,
			wantSev: model.Breaking,
		},
		{
			name:    "type swapped",
			mode:    model.ModeFull,
			base:    union("null", "string"),
			head:    union("null", "int"),
			wantN:   2, // removed "string" + added "int"
			wantSev: model.Breaking,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := diffUnions(tt.base, tt.head, "fields.value", minCtx(tt.mode))
			if len(changes) != tt.wantN {
				t.Fatalf("expected %d changes, got %d: %v", tt.wantN, len(changes), changes)
			}
			if tt.wantN > 0 && changes[0].Severity != tt.wantSev {
				t.Errorf("severity: got %q, want %q — %s", changes[0].Severity, tt.wantSev, changes[0].Reason)
			}
		})
	}
}

func TestDiffUnionInField(t *testing.T) {
	base := schema(model.Field{Name: "value", Type: union("null", "string")})
	head := schema(model.Field{Name: "value", Type: union("null", "string", "int")})

	// FULL mode: type added is BREAKING
	changes := diffFields(base, head, "", newCtx(base, head, model.ModeFull))
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %v", len(changes), changes)
	}
	if changes[0].Severity != model.Breaking {
		t.Errorf("adding type to union in FULL mode should be BREAKING, got %s", changes[0].Severity)
	}
}

func TestUnionTypeKey(t *testing.T) {
	tests := []struct {
		input interface{}
		want  string
	}{
		{"string", "string"},
		{"null", "null"},
		{map[string]interface{}{"type": "record", "name": "Address"}, "record.Address"},
		{map[string]interface{}{"type": "enum", "name": "Status"}, "enum.Status"},
		{map[string]interface{}{"type": "array", "items": "string"}, "array"},
	}
	for _, tt := range tests {
		got := unionTypeKey(tt.input)
		if got != tt.want {
			t.Errorf("unionTypeKey(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
