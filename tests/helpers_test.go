package avrodiff_test

import (
	"strings"
	"testing"

	"github.com/nikitagold/avrodiff/diff"
	"github.com/nikitagold/avrodiff/model"
)

// ruleCase is one row in a rule table test.
type ruleCase struct {
	name      string            // human-readable description
	base      string            // base schema JSON
	head      string            // head schema JSON
	wantRule  string            // expected rule ID in the result changes
	wantLevel model.SemverLevel // expected overall result level
	mode      model.CompatMode  // defaults to ModeFull when empty
}

// runRuleCases runs a slice of ruleCase entries as subtests.
func runRuleCases(t *testing.T, cases []ruleCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mode := tc.mode
			if mode == "" {
				mode = model.ModeFull
			}

			base, err := model.ParseSchema([]byte(tc.base))
			if err != nil {
				t.Fatalf("parse base: %v", err)
			}
			head, err := model.ParseSchema([]byte(tc.head))
			if err != nil {
				t.Fatalf("parse head: %v", err)
			}

			result := diff.DiffSchemas(base, head, mode)

			wantRule := strings.ToUpper(tc.wantRule)
			var found bool
			for _, c := range result.Changes {
				if c.Rule == wantRule {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("rule %s not found\n  changes: %v", wantRule, result.Changes)
			}
			if result.Level != tc.wantLevel {
				t.Errorf("level: got %s, want %s", result.Level, tc.wantLevel)
			}
		})
	}
}
