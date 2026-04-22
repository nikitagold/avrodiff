package avrodiff_test

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

// TestRulesU covers union rules U-01..U-05.
func TestRulesU(t *testing.T) {
	runRuleCases(t, []ruleCase{
		{
			name:      "U-01: type removed from union",
			wantRule:  "U-01",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "value", "type": ["null", "string", "int"]}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "value", "type": ["null", "string"]}]
			}`,
		},
		{
			name:      "U-02: type added to union",
			wantRule:  "U-02",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "value", "type": ["null", "string"]}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "value", "type": ["null", "string", "int"]}]
			}`,
		},
		{
			name:      "U-03: union member order changed",
			wantRule:  "U-03",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "value", "type": ["string", "int"]}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "value", "type": ["int", "string"]}]
			}`,
		},
		{
			name:      "U-04: null moved from first position",
			wantRule:  "U-04",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "value", "type": ["null", "string"]}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "value", "type": ["string", "null"]}]
			}`,
		},
		{
			// U-05: breaking change inside a union member.
			// The inner record loses a required field (F-01), which triggers U-05.
			name:      "U-05: incompatible change inside union member",
			wantRule:  "U-05",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "payload", "type": ["null", {
					"type": "record",
					"name": "Address",
					"fields": [{"name": "street", "type": "string"}]
				}]}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "payload", "type": ["null", {
					"type": "record",
					"name": "Address",
					"fields": []
				}]}]
			}`,
		},
	})
}
