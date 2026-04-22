package avrodiff_test

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

// TestRulesM covers map rules M-01..M-03.
func TestRulesM(t *testing.T) {
	runRuleCases(t, []ruleCase{
		{
			name:      "M-01: map value type changed (incompatible)",
			wantRule:  "M-01",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "meta", "type": {"type": "map", "values": "string"}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "meta", "type": {"type": "map", "values": "int"}}]
			}`,
		},
		{
			name:      "M-02: map value type promoted (int → long)",
			wantRule:  "M-02",
			wantLevel: model.LevelMinor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "counts", "type": {"type": "map", "values": "int"}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "counts", "type": {"type": "map", "values": "long"}}]
			}`,
		},
		{
			name:      "M-03: map replaced by a different type",
			wantRule:  "M-03",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "meta", "type": {"type": "map", "values": "string"}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "meta", "type": "string"}]
			}`,
		},
	})
}
