package diff

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

func enumSchema(name string, symbols ...string) interface{} {
	return map[string]interface{}{
		"type":    "enum",
		"name":    name,
		"symbols": toIfaceSlice(symbols),
	}
}

func toIfaceSlice(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func TestDiffEnums(t *testing.T) {
	tests := []struct {
		name    string
		mode    model.CompatMode
		base    model.EnumSchema
		head    model.EnumSchema
		wantN   int
		wantSev model.Severity
	}{
		{
			name:  "no changes",
			mode:  model.ModeFull,
			base:  model.EnumSchema{Symbols: []string{"A", "B"}},
			head:  model.EnumSchema{Symbols: []string{"A", "B"}},
			wantN: 0,
		},
		{
			// FULL mode: symbol added is BREAKING (old readers don't know the new index)
			name:    "symbol added full",
			mode:    model.ModeFull,
			base:    model.EnumSchema{Symbols: []string{"A", "B"}},
			head:    model.EnumSchema{Symbols: []string{"A", "B", "C"}},
			wantN:   1,
			wantSev: model.Breaking,
		},
		{
			// BACKWARD mode: symbol added is SAFE (old data doesn't contain new symbol)
			name:    "symbol added backward",
			mode:    model.ModeBackward,
			base:    model.EnumSchema{Symbols: []string{"A", "B"}},
			head:    model.EnumSchema{Symbols: []string{"A", "B", "C"}},
			wantN:   1,
			wantSev: model.Safe,
		},
		{
			// FULL mode: symbol removed is BREAKING (old data contains this value)
			name:    "symbol removed full",
			mode:    model.ModeFull,
			base:    model.EnumSchema{Symbols: []string{"A", "B", "C"}},
			head:    model.EnumSchema{Symbols: []string{"A", "B"}},
			wantN:   1,
			wantSev: model.Breaking,
		},
		{
			// FORWARD mode: symbol removed is SAFE (new data won't contain removed symbol)
			name:    "symbol removed forward",
			mode:    model.ModeForward,
			base:    model.EnumSchema{Symbols: []string{"A", "B", "C"}},
			head:    model.EnumSchema{Symbols: []string{"A", "B"}},
			wantN:   1,
			wantSev: model.Safe,
		},
		{
			name:    "symbol order changed",
			mode:    model.ModeFull,
			base:    model.EnumSchema{Symbols: []string{"A", "B"}},
			head:    model.EnumSchema{Symbols: []string{"B", "A"}},
			wantN:   1,
			wantSev: model.Breaking,
		},
		{
			// NONE mode: no changes are BREAKING
			name:    "symbol removed none mode",
			mode:    model.ModeNone,
			base:    model.EnumSchema{Symbols: []string{"A", "B"}},
			head:    model.EnumSchema{Symbols: []string{"A"}},
			wantN:   1,
			wantSev: model.Safe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := diffEnums(tt.base, tt.head, "fields.status", minCtx(tt.mode))
			if len(changes) != tt.wantN {
				t.Fatalf("expected %d changes, got %d: %v", tt.wantN, len(changes), changes)
			}
			if tt.wantN > 0 && changes[0].Severity != tt.wantSev {
				t.Errorf("severity: got %q, want %q — %s", changes[0].Severity, tt.wantSev, changes[0].Reason)
			}
		})
	}
}

func TestDiffEnumInField_Full(t *testing.T) {
	base := schema(model.Field{Name: "status", Type: enumSchema("Status", "ACTIVE", "INACTIVE")})
	head := schema(model.Field{Name: "status", Type: enumSchema("Status", "ACTIVE", "INACTIVE", "PENDING")})

	// FULL mode: symbol added is BREAKING
	changes := diffFields(base, head, "", newCtx(base, head, model.ModeFull))
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %v", len(changes), changes)
	}
	if changes[0].Severity != model.Breaking {
		t.Errorf("adding enum symbol in FULL mode should be BREAKING, got %s", changes[0].Severity)
	}
}

func TestDiffEnumInField_Backward(t *testing.T) {
	base := schema(model.Field{Name: "status", Type: enumSchema("Status", "ACTIVE", "INACTIVE")})
	head := schema(model.Field{Name: "status", Type: enumSchema("Status", "ACTIVE", "INACTIVE", "PENDING")})

	// BACKWARD mode: symbol added is SAFE
	changes := diffFields(base, head, "", newCtx(base, head, model.ModeBackward))
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %v", len(changes), changes)
	}
	if changes[0].Severity != model.Safe {
		t.Errorf("adding enum symbol in BACKWARD mode should be SAFE, got %s", changes[0].Severity)
	}
	if len(changes[0].AffectedModes) == 0 {
		t.Error("expected AffectedModes to be set (FORWARD, FULL)")
	}
}
