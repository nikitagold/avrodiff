package diff

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

func decimal(underlying string, precision, scale float64) interface{} {
	return map[string]interface{}{
		"type":        underlying,
		"logicalType": "decimal",
		"precision":   precision,
		"scale":       scale,
	}
}

func logicalType(underlying, lt string) interface{} {
	return map[string]interface{}{
		"type":        underlying,
		"logicalType": lt,
	}
}

func TestLogicalTypeRules(t *testing.T) {
	ctx := minCtx(model.ModeFull)

	tests := []struct {
		name     string
		base     interface{}
		head     interface{}
		wantRule string
		wantSev  model.Severity
	}{
		{
			name:     "L-01: logicalType added",
			base:     "bytes",
			head:     logicalType("bytes", "decimal"),
			wantRule: "L-01",
			wantSev:  model.Breaking,
		},
		{
			name:     "L-02: logicalType removed",
			base:     logicalType("bytes", "decimal"),
			head:     "bytes",
			wantRule: "L-02",
			wantSev:  model.Breaking,
		},
		{
			name:     "L-03: logicalType changed",
			base:     logicalType("int", "date"),
			head:     logicalType("long", "timestamp-millis"),
			wantRule: "L-03",
			wantSev:  model.Breaking,
		},
		{
			name:     "L-04: decimal precision changed",
			base:     decimal("bytes", 10, 2),
			head:     decimal("bytes", 20, 2),
			wantRule: "L-04",
			wantSev:  model.Breaking,
		},
		{
			name:     "L-05: decimal scale decreased",
			base:     decimal("bytes", 10, 4),
			head:     decimal("bytes", 10, 2),
			wantRule: "L-05",
			wantSev:  model.Breaking,
		},
		{
			name:     "L-06: decimal scale increased",
			base:     decimal("bytes", 10, 2),
			head:     decimal("bytes", 10, 4),
			wantRule: "L-06",
			wantSev:  model.Breaking,
		},
		{
			name:     "L-07: underlying type changed, logicalType same",
			base:     decimal("bytes", 10, 2),
			head:     decimal("fixed", 10, 2),
			wantRule: "L-07",
			wantSev:  model.Breaking,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, handled := diffLogicalTypes(tt.base, tt.head, "fields.x", ctx)
			if !handled {
				t.Fatal("expected handled=true")
			}
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
				t.Errorf("severity: got %q, want %q", found.Severity, tt.wantSev)
			}
		})
	}
}

func TestLogicalTypeNotHandled(t *testing.T) {
	ctx := minCtx(model.ModeFull)
	// Neither side has logicalType → not handled (falls through to F-09)
	_, handled := diffLogicalTypes("int", "string", "fields.x", ctx)
	if handled {
		t.Error("expected handled=false when neither type has logicalType")
	}
}

func TestLogicalTypeInField(t *testing.T) {
	base := schema(field("amount", decimal("bytes", 10, 2)))
	head := schema(field("amount", decimal("bytes", 20, 2)))
	result := DiffSchemas(base, head, model.ModeFull)

	found := false
	for _, c := range result.Changes {
		if c.Rule == "L-04" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected L-04 in DiffSchemas result, got: %v", result.Changes)
	}
	if result.Level != model.LevelMajor {
		t.Errorf("level: got %s, want MAJOR", result.Level)
	}
}
