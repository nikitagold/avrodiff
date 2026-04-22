package avrodiff_test

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

// TestRulesA covers array rules A-01..A-03.
func TestRulesA(t *testing.T) {
	runRuleCases(t, []ruleCase{
		{
			name:      "A-01: array item type changed (incompatible)",
			wantRule:  "A-01",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "tags", "type": {"type": "array", "items": "string"}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "tags", "type": {"type": "array", "items": "int"}}]
			}`,
		},
		{
			name:      "A-02: array item type promoted (int → long)",
			wantRule:  "A-02",
			wantLevel: model.LevelMinor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "scores", "type": {"type": "array", "items": "int"}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "scores", "type": {"type": "array", "items": "long"}}]
			}`,
		},
		{
			name:      "A-03: array replaced by a different type",
			wantRule:  "A-03",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "tags", "type": {"type": "array", "items": "string"}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "tags", "type": "string"}]
			}`,
		},
	})
}
