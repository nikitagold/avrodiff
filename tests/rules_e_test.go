package avrodiff_test

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

// TestRulesE covers enum rules E-01..E-11.
func TestRulesE(t *testing.T) {
	runRuleCases(t, []ruleCase{
		{
			name:      "E-01: enum symbol removed",
			wantRule:  "E-01",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE", "INACTIVE"]
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE"]
				}}]
			}`,
		},
		{
			name:      "E-02: enum symbol added (no enum default)",
			wantRule:  "E-02",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE"]
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE", "INACTIVE"]
				}}]
			}`,
		},
		{
			name:      "E-03: enum symbol added (enum has default)",
			wantRule:  "E-03",
			wantLevel: model.LevelMinor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE"], "default": "ACTIVE"
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE", "INACTIVE"], "default": "ACTIVE"
				}}]
			}`,
		},
		{
			name:      "E-04: enum symbols reordered",
			wantRule:  "E-04",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE", "INACTIVE"]
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["INACTIVE", "ACTIVE"]
				}}]
			}`,
		},
		{
			name:      "E-05: enum renamed without alias",
			wantRule:  "E-05",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE"]
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "UserStatus",
					"symbols": ["ACTIVE"]
				}}]
			}`,
		},
		{
			name:      "E-06: enum renamed, old name kept as alias",
			wantRule:  "E-06",
			wantLevel: model.LevelMinor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE"]
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "UserStatus",
					"aliases": ["Status"],
					"symbols": ["ACTIVE"]
				}}]
			}`,
		},
		{
			name:      "E-07: enum alias removed",
			wantRule:  "E-07",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"aliases": ["UserStatus"],
					"symbols": ["ACTIVE"]
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE"]
				}}]
			}`,
		},
		{
			name:      "E-08: enum alias added",
			wantRule:  "E-08",
			wantLevel: model.LevelMinor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE"]
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"aliases": ["UserStatus"],
					"symbols": ["ACTIVE"]
				}}]
			}`,
		},
		{
			name:      "E-09: enum default changed",
			wantRule:  "E-09",
			wantLevel: model.LevelPatch,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE", "INACTIVE"], "default": "ACTIVE"
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE", "INACTIVE"], "default": "INACTIVE"
				}}]
			}`,
		},
		{
			name:      "E-10: enum namespace changed",
			wantRule:  "E-10",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"namespace": "com.old",
					"symbols": ["ACTIVE"]
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"namespace": "com.new",
					"symbols": ["ACTIVE"]
				}}]
			}`,
		},
		{
			// Same change (symbol added, no default), but BACKWARD mode only checks
			// that new schema can read old data — old data has no new symbol, so SAFE.
			name:      "E-02 in BACKWARD mode: symbol added is SAFE (old data has no new symbol)",
			wantRule:  "E-02",
			wantLevel: model.LevelMinor,
			mode:      model.ModeBackward,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE"]
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE", "INACTIVE"]
				}}]
			}`,
		},
		{
			name:      "E-11: enum doc changed",
			wantRule:  "E-11",
			wantLevel: model.LevelPatch,
			base: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE"], "doc": "Old doc."
				}}]
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": [{"name": "status", "type": {
					"type": "enum", "name": "Status",
					"symbols": ["ACTIVE"], "doc": "New doc."
				}}]
			}`,
		},
	})
}
