package diff

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

// TestCompatMode verifies that Severity changes correctly across BACKWARD / FORWARD / FULL / NONE.
func TestCompatMode(t *testing.T) {
	tests := []struct {
		name     string
		base     *model.Schema
		head     *model.Schema
		mode     model.CompatMode
		wantSev  model.Severity
		wantPath string
	}{
		// --- Field removal ---
		{
			name:     "field removed / BACKWARD → SAFE",
			base:     schema(field("email", "string")),
			head:     schema(),
			mode:     model.ModeBackward,
			wantSev:  model.Safe,
			wantPath: "fields.email",
		},
		{
			name:     "field removed / FORWARD → BREAKING",
			base:     schema(field("email", "string")),
			head:     schema(),
			mode:     model.ModeForward,
			wantSev:  model.Breaking,
			wantPath: "fields.email",
		},
		{
			name:     "field removed / FULL → BREAKING",
			base:     schema(field("email", "string")),
			head:     schema(),
			mode:     model.ModeFull,
			wantSev:  model.Breaking,
			wantPath: "fields.email",
		},
		{
			name:     "field removed / NONE → SAFE",
			base:     schema(field("email", "string")),
			head:     schema(),
			mode:     model.ModeNone,
			wantSev:  model.Safe,
			wantPath: "fields.email",
		},
		// Field removal with default: SAFE in ALL modes
		{
			name:     "field removed with default / FULL → SAFE",
			base:     schema(fieldWithDefault("email", "string", "unknown")),
			head:     schema(),
			mode:     model.ModeFull,
			wantSev:  model.Safe,
			wantPath: "fields.email",
		},
		{
			name:     "field removed with default / FORWARD → SAFE",
			base:     schema(fieldWithDefault("email", "string", "unknown")),
			head:     schema(),
			mode:     model.ModeForward,
			wantSev:  model.Safe,
			wantPath: "fields.email",
		},

		// --- Field addition without default ---
		{
			name:     "field added no default / BACKWARD → BREAKING",
			base:     schema(),
			head:     schema(field("phone", "string")),
			mode:     model.ModeBackward,
			wantSev:  model.Breaking,
			wantPath: "fields.phone",
		},
		{
			name:     "field added no default / FORWARD → SAFE",
			base:     schema(),
			head:     schema(field("phone", "string")),
			mode:     model.ModeForward,
			wantSev:  model.Safe,
			wantPath: "fields.phone",
		},
		{
			name:     "field added no default / FULL → BREAKING",
			base:     schema(),
			head:     schema(field("phone", "string")),
			mode:     model.ModeFull,
			wantSev:  model.Breaking,
			wantPath: "fields.phone",
		},

		// --- Enum symbol addition ---
		{
			name:     "enum symbol added / BACKWARD → SAFE",
			base:     schema(field("status", enumSchema("S", "A", "B"))),
			head:     schema(field("status", enumSchema("S", "A", "B", "C"))),
			mode:     model.ModeBackward,
			wantSev:  model.Safe,
			wantPath: "fields.status",
		},
		{
			name:     "enum symbol added / FORWARD → BREAKING",
			base:     schema(field("status", enumSchema("S", "A", "B"))),
			head:     schema(field("status", enumSchema("S", "A", "B", "C"))),
			mode:     model.ModeForward,
			wantSev:  model.Breaking,
			wantPath: "fields.status",
		},
		{
			name:     "enum symbol added / FULL → BREAKING",
			base:     schema(field("status", enumSchema("S", "A", "B"))),
			head:     schema(field("status", enumSchema("S", "A", "B", "C"))),
			mode:     model.ModeFull,
			wantSev:  model.Breaking,
			wantPath: "fields.status",
		},

		// --- Enum symbol removal ---
		{
			name:     "enum symbol removed / BACKWARD → BREAKING",
			base:     schema(field("status", enumSchema("S", "A", "B", "C"))),
			head:     schema(field("status", enumSchema("S", "A", "B"))),
			mode:     model.ModeBackward,
			wantSev:  model.Breaking,
			wantPath: "fields.status",
		},
		{
			name:     "enum symbol removed / FORWARD → SAFE",
			base:     schema(field("status", enumSchema("S", "A", "B", "C"))),
			head:     schema(field("status", enumSchema("S", "A", "B"))),
			mode:     model.ModeForward,
			wantSev:  model.Safe,
			wantPath: "fields.status",
		},

		// --- Union type addition ---
		{
			name:     "union type added / BACKWARD → SAFE",
			base:     schema(field("v", []interface{}{"null", "string"})),
			head:     schema(field("v", []interface{}{"null", "string", "int"})),
			mode:     model.ModeBackward,
			wantSev:  model.Safe,
			wantPath: "fields.v",
		},
		{
			name:     "union type added / FORWARD → BREAKING",
			base:     schema(field("v", []interface{}{"null", "string"})),
			head:     schema(field("v", []interface{}{"null", "string", "int"})),
			mode:     model.ModeForward,
			wantSev:  model.Breaking,
			wantPath: "fields.v",
		},

		// --- AffectedModes are set ---
		{
			// field removed no default → AffectedModes should include FORWARD and FULL
			name:     "field removed / AffectedModes populated",
			base:     schema(field("email", "string")),
			head:     schema(),
			mode:     model.ModeBackward, // SAFE for this mode, but AffectedModes != []
			wantSev:  model.Safe,
			wantPath: "fields.email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DiffSchemas(tt.base, tt.head, tt.mode)
			var got *model.Change
			for i := range result.Changes {
				if result.Changes[i].Path == tt.wantPath {
					got = &result.Changes[i]
					break
				}
			}
			if got == nil {
				t.Fatalf("no change at path %q. changes: %v", tt.wantPath, result.Changes)
			}
			if got.Severity != tt.wantSev {
				t.Errorf("severity: got %q, want %q [mode=%s]\n  reason: %s\n  affected: %v",
					got.Severity, tt.wantSev, tt.mode, got.Reason, got.AffectedModes)
			}
		})
	}
}

func TestCompatMode_AffectedModesPopulated(t *testing.T) {
	// Field removed without default: AffectedModes should be [FORWARD, FULL]
	base := schema(field("email", "string"))
	head := schema()

	result := DiffSchemas(base, head, model.ModeBackward) // SAFE for BACKWARD

	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result.Changes))
	}
	c := result.Changes[0]
	if c.Severity != model.Safe {
		t.Errorf("expected SAFE for BACKWARD, got %s", c.Severity)
	}

	hasForward, hasFull := false, false
	for _, m := range c.AffectedModes {
		if m == model.ModeForward {
			hasForward = true
		}
		if m == model.ModeFull {
			hasFull = true
		}
	}
	if !hasForward || !hasFull {
		t.Errorf("expected AffectedModes to contain FORWARD and FULL, got %v", c.AffectedModes)
	}
}
