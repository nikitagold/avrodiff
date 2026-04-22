package avrodiff_test

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

// TestRulesF covers field-level rules F-01..F-15.
func TestRulesF(t *testing.T) {
	runRuleCases(t, []ruleCase{
		{
			name:      "F-01: field removed (no default)",
			wantRule:  "F-01",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "email", "type": "string"}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": []
			}`,
		},
		{
			name:      "F-02: field removed (had default)",
			wantRule:  "F-02",
			wantLevel: model.LevelMinor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "email", "type": "string", "default": ""}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": []
			}`,
		},
		{
			name:      "F-03: field added without default",
			wantRule:  "F-03",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": []
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "email", "type": "string"}]
			}`,
		},
		{
			name:      "F-04: field added with default",
			wantRule:  "F-04",
			wantLevel: model.LevelMinor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": []
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "email", "type": "string", "default": ""}]
			}`,
		},
		{
			// F-05: field renamed without alias cannot be distinguished from
			// delete+add — avrodiff emits F-01 (removed) + F-03 (added).
			name:      "F-05: field renamed without alias (detected as F-01 + F-03)",
			wantRule:  "F-01",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "email", "type": "string"}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "mail", "type": "string"}]
			}`,
		},
		{
			name:      "F-06: field renamed, old name kept as alias",
			wantRule:  "F-06",
			wantLevel: model.LevelMinor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "email", "type": "string"}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "mail", "type": "string", "aliases": ["email"]}]
			}`,
		},
		{
			name:      "F-07: field alias removed",
			wantRule:  "F-07",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "email", "type": "string", "aliases": ["mail"]}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "email", "type": "string"}]
			}`,
		},
		{
			name:      "F-08: field alias added",
			wantRule:  "F-08",
			wantLevel: model.LevelMinor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "email", "type": "string"}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "email", "type": "string", "aliases": ["mail"]}]
			}`,
		},
		{
			name:      "F-09: field type changed (incompatible)",
			wantRule:  "F-09",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "age", "type": "int"}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "age", "type": "string"}]
			}`,
		},
		{
			name:      "F-10: field type promoted (int → long)",
			wantRule:  "F-10",
			wantLevel: model.LevelMinor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "age", "type": "int"}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "age", "type": "long"}]
			}`,
		},
		{
			name:      "F-11: field order changed",
			wantRule:  "F-11",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [
					{"name": "a", "type": "string"},
					{"name": "b", "type": "string"}
				]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [
					{"name": "b", "type": "string"},
					{"name": "a", "type": "string"}
				]
			}`,
		},
		{
			name:      "F-12: default value changed",
			wantRule:  "F-12",
			wantLevel: model.LevelPatch,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "age", "type": "int", "default": 0}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "age", "type": "int", "default": 42}]
			}`,
		},
		{
			name:      "F-13: default added to field",
			wantRule:  "F-13",
			wantLevel: model.LevelMinor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "age", "type": "int"}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "age", "type": "int", "default": 0}]
			}`,
		},
		{
			name:      "F-14: default removed from field",
			wantRule:  "F-14",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "age", "type": "int", "default": 0}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "age", "type": "int"}]
			}`,
		},
		{
			name:      "F-15: field doc changed",
			wantRule:  "F-15",
			wantLevel: model.LevelPatch,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "age", "type": "int", "doc": "The user age."}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "age", "type": "int", "doc": "Age in years."}]
			}`,
		},
	})
}
