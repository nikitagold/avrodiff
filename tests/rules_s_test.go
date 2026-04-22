package avrodiff_test

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

// TestRulesS covers schema-level rules S-01..S-04.
// S-02..S-04 are lint rules that only activate when the base schema has a "version" field.
func TestRulesS(t *testing.T) {
	runRuleCases(t, []ruleCase{
		{
			name:      "S-01: root schema type changed (record → enum)",
			wantRule:  "S-01",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": []
			}`,
			head: `{
				"type": "enum",
				"name": "User",
				"symbols": ["A", "B"]
			}`,
		},
		{
			name:      "S-02: version field missing in head schema",
			wantRule:  "S-02",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"version": "1.0.0",
				"fields": []
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": []
			}`,
		},
		{
			// MINOR change (field added with default) but only PATCH bumped → S-03
			name:      "S-03: version bump too small for the change level",
			wantRule:  "S-03",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"version": "1.0.0",
				"fields": [{"name": "id", "type": "string"}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"version": "1.0.1",
				"fields": [
					{"name": "id", "type": "string"},
					{"name": "email", "type": "string", "default": ""}
				]
			}`,
		},
		{
			name:      "S-04: version decreased",
			wantRule:  "S-04",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"version": "2.0.0",
				"fields": []
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"version": "1.9.0",
				"fields": []
			}`,
		},
	})
}
