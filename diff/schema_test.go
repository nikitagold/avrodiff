package diff

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

func schemaWithVersion(version string, fields ...model.Field) *model.Schema {
	return &model.Schema{Type: "record", Name: "Test", Version: version, Fields: fields}
}

func TestSchemaRules(t *testing.T) {
	tests := []struct {
		name     string
		base     *model.Schema
		head     *model.Schema
		wantRule string
		wantSev  model.Severity
	}{
		{
			name:     "S-01: root type changed → MAJOR",
			base:     &model.Schema{Type: "record", Name: "X"},
			head:     &model.Schema{Type: "enum", Name: "X"},
			wantRule: "S-01",
			wantSev:  model.Breaking,
		},
		{
			name:     "S-02: version removed → MAJOR lint",
			base:     schemaWithVersion("1.0.0"),
			head:     &model.Schema{Type: "record", Name: "Test"},
			wantRule: "S-02",
			wantSev:  model.Breaking,
		},
		{
			name: "S-03: version under-bumped (MINOR change, only patch bumped) → MAJOR lint",
			base: schemaWithVersion("1.0.0", field("a", "string")),
			head: schemaWithVersion("1.0.1", field("a", "string"),
				fieldWithDefault("b", "string", "x")), // MINOR: field added with default
			wantRule: "S-03",
			wantSev:  model.Breaking,
		},
		{
			name: "S-04: version decreased → MAJOR lint",
			base: schemaWithVersion("2.0.0"),
			head: schemaWithVersion("1.9.0"),
			wantRule: "S-04",
			wantSev:  model.Breaking,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DiffSchemas(tt.base, tt.head, model.ModeFull)
			var found *model.Change
			for i := range result.Changes {
				if result.Changes[i].Rule == tt.wantRule {
					found = &result.Changes[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("rule %s not found in changes: %v", tt.wantRule, result.Changes)
			}
			if found.Severity != tt.wantSev {
				t.Errorf("severity: got %q, want %q", found.Severity, tt.wantSev)
			}
			if result.Level != model.LevelMajor {
				t.Errorf("level: got %s, want MAJOR", result.Level)
			}
		})
	}
}

func TestSchemaVersionNoChecksWithoutBaseVersion(t *testing.T) {
	// Schemas without version field → no lint checks fired
	base := schema(field("a", "string"))
	head := schema(field("a", "string"))
	result := DiffSchemas(base, head, model.ModeFull)
	for _, c := range result.Changes {
		if c.Rule == "S-02" || c.Rule == "S-03" || c.Rule == "S-04" {
			t.Errorf("lint rule %s should not fire when base has no version", c.Rule)
		}
	}
}

func TestSchemaVersionCorrectBump(t *testing.T) {
	// MINOR change + MINOR bump → no S-03
	base := schemaWithVersion("1.0.0", field("a", "string"))
	head := schemaWithVersion("1.1.0", field("a", "string"),
		fieldWithDefault("b", "string", "x"))
	result := DiffSchemas(base, head, model.ModeFull)
	for _, c := range result.Changes {
		if c.Rule == "S-03" {
			t.Errorf("S-03 should not fire for correct MINOR bump: %v", c)
		}
	}
}

func TestSemverParsing(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		major   int
		minor   int
		patch   int
	}{
		{"1.2.3", false, 1, 2, 3},
		{"0.0.0", false, 0, 0, 0},
		{"10.20.30", false, 10, 20, 30},
		{"1.2", true, 0, 0, 0},
		{"1.2.x", true, 0, 0, 0},
		{"", true, 0, 0, 0},
	}
	for _, tt := range tests {
		v, err := parseSemver(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseSemver(%q): expected error, got %v", tt.input, v)
			}
		} else {
			if err != nil {
				t.Errorf("parseSemver(%q): unexpected error: %v", tt.input, err)
			}
			if v.major != tt.major || v.minor != tt.minor || v.patch != tt.patch {
				t.Errorf("parseSemver(%q) = %v, want {%d,%d,%d}", tt.input, v, tt.major, tt.minor, tt.patch)
			}
		}
	}
}
