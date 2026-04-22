package avrodiff_test

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

// TestRulesL covers logical type rules L-01..L-07.
func TestRulesL(t *testing.T) {
	runRuleCases(t, []ruleCase{
		{
			name:      "L-01: logicalType annotation added",
			wantRule:  "L-01",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "created_at", "type": "int"}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "created_at", "type": {"type": "int", "logicalType": "date"}}]
			}`,
		},
		{
			name:      "L-02: logicalType annotation removed",
			wantRule:  "L-02",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "created_at", "type": {"type": "int", "logicalType": "date"}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "created_at", "type": "int"}]
			}`,
		},
		{
			name:      "L-03: logicalType changed (date → time-millis)",
			wantRule:  "L-03",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "ts", "type": {"type": "int", "logicalType": "date"}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "ts", "type": {"type": "int", "logicalType": "time-millis"}}]
			}`,
		},
		{
			name:      "L-04: decimal precision changed",
			wantRule:  "L-04",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "amount", "type": {
					"type": "bytes", "logicalType": "decimal",
					"precision": 10, "scale": 2
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "amount", "type": {
					"type": "bytes", "logicalType": "decimal",
					"precision": 20, "scale": 2
				}}]
			}`,
		},
		{
			name:      "L-05: decimal scale decreased",
			wantRule:  "L-05",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "amount", "type": {
					"type": "bytes", "logicalType": "decimal",
					"precision": 10, "scale": 4
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "amount", "type": {
					"type": "bytes", "logicalType": "decimal",
					"precision": 10, "scale": 2
				}}]
			}`,
		},
		{
			name:      "L-06: decimal scale increased",
			wantRule:  "L-06",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "amount", "type": {
					"type": "bytes", "logicalType": "decimal",
					"precision": 10, "scale": 2
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "amount", "type": {
					"type": "bytes", "logicalType": "decimal",
					"precision": 10, "scale": 4
				}}]
			}`,
		},
		{
			name:      "L-07: underlying type changed, same logicalType",
			wantRule:  "L-07",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "created_at", "type": {"type": "int", "logicalType": "date"}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "created_at", "type": {"type": "long", "logicalType": "date"}}]
			}`,
		},
	})
}
