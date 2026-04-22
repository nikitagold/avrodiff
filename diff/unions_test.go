package diff

import (
	"strings"
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

// FIX-2: union ↔ single type должен маршрутизироваться через diffUnions.
func TestDiffUnionToSingleType(t *testing.T) {
	tests := []struct {
		name        string
		mode        model.CompatMode
		base        *model.Schema
		head        *model.Schema
		wantN       int
		wantSev     model.Severity
		wantDescSub string // подстрока в Description — проверяем что не "type changed"
	}{
		{
			// ["null","string","int"] → "string": удалены null и int
			// BACKWARD breaking (старые данные могли содержать null/int)
			name:        "multi-union narrowing to single type: backward breaking",
			mode:        model.ModeBackward,
			base:        schema(field("value", union("null", "string", "int"))),
			head:        schema(field("value", "string")),
			wantN:       2, // "null" removed + "int" removed
			wantSev:     model.Breaking,
			wantDescSub: "removed",
		},
		{
			// ["null","string","int"] → "string": в FORWARD mode удаление типов — SAFE
			// (новый продюсер пишет только string, старый консьюмер справляется)
			name:    "multi-union narrowing to single type: forward safe",
			mode:    model.ModeForward,
			base:    schema(field("value", union("null", "string", "int"))),
			head:    schema(field("value", "string")),
			wantN:   2,
			wantSev: model.Safe,
		},
		{
			// "string" → ["null","string","int"]: добавлены null и int
			// FORWARD breaking (старый консьюмер не знает null и int)
			name:        "single type widening to multi-union: forward breaking",
			mode:        model.ModeForward,
			base:        schema(field("value", "string")),
			head:        schema(field("value", union("null", "string", "int"))),
			wantN:       2, // "null" added + "int" added
			wantSev:     model.Breaking,
			wantDescSub: "added",
		},
		{
			// "string" → ["null","string","int"]: в BACKWARD mode добавление типов — SAFE
			name:    "single type widening to multi-union: backward safe",
			mode:    model.ModeBackward,
			base:    schema(field("value", "string")),
			head:    schema(field("value", union("null", "string", "int"))),
			wantN:   2,
			wantSev: model.Safe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := diffFields(tt.base, tt.head, "", newCtx(tt.base, tt.head, tt.mode))
			if len(changes) != tt.wantN {
				t.Fatalf("expected %d changes, got %d: %v", tt.wantN, len(changes), changes)
			}
			if tt.wantN > 0 {
				if changes[0].Severity != tt.wantSev {
					t.Errorf("severity: got %q, want %q — %s", changes[0].Severity, tt.wantSev, changes[0].Reason)
				}
				if tt.wantDescSub != "" && !strings.Contains(changes[0].Description, tt.wantDescSub) {
					t.Errorf("description: got %q, want it to contain %q", changes[0].Description, tt.wantDescSub)
				}
			}
		})
	}
}

func TestUnionRules(t *testing.T) {
	ctx := minCtx(model.ModeFull)

	t.Run("U-04: null moved to non-first position", func(t *testing.T) {
		changes := diffUnions(
			union("null", "string"),
			union("string", "null"),
			"fields.x", ctx,
		)
		var found *model.Change
		for i := range changes {
			if changes[i].Rule == "U-04" {
				found = &changes[i]
				break
			}
		}
		if found == nil {
			t.Fatalf("U-04 not found in changes: %v", changes)
		}
		if found.Severity != model.Breaking {
			t.Errorf("severity: got %q, want BREAKING", found.Severity)
		}
		// U-03 should NOT also fire (U-04 is more specific)
		for _, c := range changes {
			if c.Rule == "U-03" {
				t.Errorf("U-03 should not fire when U-04 already covers the null reorder")
			}
		}
	})

	t.Run("U-03: non-null reorder still fires", func(t *testing.T) {
		changes := diffUnions(
			union("null", "string", "int"),
			union("null", "int", "string"),
			"fields.x", ctx,
		)
		var found *model.Change
		for i := range changes {
			if changes[i].Rule == "U-03" {
				found = &changes[i]
				break
			}
		}
		if found == nil {
			t.Fatalf("U-03 not found in changes: %v", changes)
		}
	})

	t.Run("U-05: field removed inside union record member", func(t *testing.T) {
		baseRec := map[string]interface{}{
			"type":   "record",
			"name":   "Payload",
			"fields": []interface{}{strField("id"), strField("name")},
		}
		headRec := map[string]interface{}{
			"type":   "record",
			"name":   "Payload",
			"fields": []interface{}{strField("id")}, // "name" removed
		}
		base := schema(field("data", union("null", baseRec)))
		head := schema(field("data", union("null", headRec)))
		result := DiffSchemas(base, head, model.ModeFull)
		found := false
		for _, c := range result.Changes {
			if c.Rule == "F-01" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected F-01 (field removed inside union member), changes: %v", result.Changes)
		}
		if result.Level != model.LevelMajor {
			t.Errorf("level: got %s, want MAJOR", result.Level)
		}
	})
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
		// FIX-8: array/map включают items/values в ключ
		{map[string]interface{}{"type": "array", "items": "string"}, "array.string"},
		{map[string]interface{}{"type": "array", "items": "int"}, "array.int"},
	}
	for _, tt := range tests {
		got := unionTypeKey(tt.input)
		if got != tt.want {
			t.Errorf("unionTypeKey(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestUnionTypeKeyNoCollision(t *testing.T) {
	// FIX-8: два array с разными items в union должны давать разные ключи
	arrayString := map[string]interface{}{"type": "array", "items": "string"}
	arrayInt := map[string]interface{}{"type": "array", "items": "int"}

	keyString := unionTypeKey(arrayString)
	keyInt := unionTypeKey(arrayInt)

	if keyString == keyInt {
		t.Errorf("different array types got same key %q — collision", keyString)
	}
}
