package avrodiff_test

import (
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

// TestRulesR covers record-level rules R-02..R-07.
// R-01 (record deleted) is not a distinct rule: caught as S-01 when the root
// type changes, or as F-09 when a field's type changes from record to something else.
func TestRulesR(t *testing.T) {
	runRuleCases(t, []ruleCase{
		{
			name:      "R-02: record renamed without alias",
			wantRule:  "R-02",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": []
			}`,
			head: `{
				"type": "record",
				"name": "Customer",
				"fields": []
			}`,
		},
		{
			name:      "R-03: record renamed, old name kept as alias",
			wantRule:  "R-03",
			wantLevel: model.LevelMinor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": []
			}`,
			head: `{
				"type": "record",
				"name": "Customer",
				"aliases": ["User"],
				"fields": []
			}`,
		},
		{
			name:      "R-04: namespace changed",
			wantRule:  "R-04",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"namespace": "com.old",
				"fields": []
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"namespace": "com.new",
				"fields": []
			}`,
		},
		{
			name:      "R-05: record alias removed",
			wantRule:  "R-05",
			wantLevel: model.LevelMajor,
			base: `{
				"type": "record",
				"name": "User",
				"aliases": ["LegacyUser"],
				"fields": []
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"fields": []
			}`,
		},
		{
			name:      "R-06: record alias added",
			wantRule:  "R-06",
			wantLevel: model.LevelMinor,
			base: `{
				"type": "record",
				"name": "User",
				"fields": []
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"aliases": ["LegacyUser"],
				"fields": []
			}`,
		},
		{
			name:      "R-07: record doc changed",
			wantRule:  "R-07",
			wantLevel: model.LevelPatch,
			base: `{
				"type": "record",
				"name": "User",
				"doc": "Old documentation.",
				"fields": []
			}`,
			head: `{
				"type": "record",
				"name": "User",
				"doc": "New documentation.",
				"fields": []
			}`,
		},
	})
}
