package diff

import (
	"fmt"

	"github.com/nikitagold/avrodiff/model"
)

// PairRule describes a single declarative check between two values of the same type.
// Used for property-level rules (doc changed, default changed, aliases, etc.)
// where both base and head exist and we simply compare their attributes.
type PairRule[T any] struct {
	ID            string
	Reason        string
	AffectedModes []model.CompatMode
	Cosmetic      bool // if true, severity is always Cosmetic (PATCH), ignores AffectedModes
	Applies       func(base, head T) bool
	Describe      func(base, head T) string
}

// applyPairRules runs all rules against (base, head) and returns the resulting changes.
func applyPairRules[T any](rules []PairRule[T], base, head T, path string, ctx *DiffContext) []model.Change {
	var out []model.Change
	for _, r := range rules {
		if r.Applies(base, head) {
			if r.Cosmetic {
				out = append(out, model.Change{
					Rule:        r.ID,
					Path:        path,
					Description: r.Describe(base, head),
					Reason:      r.Reason,
					Severity:    model.Cosmetic,
				})
			} else {
				out = append(out, makeChange(r.ID, path, r.Describe(base, head), r.Reason, r.AffectedModes, ctx))
			}
		}
	}
	return out
}

// fieldPairRules contains property-level rules for two fields with the same name.
var fieldPairRules = []PairRule[model.Field]{
	{
		// F-07: alias removed from field → MAJOR (consumers using that alias break)
		ID:            "F-07",
		Reason:        "consumers referencing this field by alias will fail to deserialize",
		AffectedModes: modesAll,
		Applies: func(b, h model.Field) bool {
			return hasRemovedAlias(b.Aliases, h.Aliases)
		},
		Describe: func(b, h model.Field) string {
			removed := removedAliases(b.Aliases, h.Aliases)
			return fmt.Sprintf("field %q alias %v removed", b.Name, removed)
		},
	},
	{
		// F-08: alias added to field → MINOR (adds alternative name, nothing breaks)
		ID:            "F-08",
		Reason:        "adding an alias provides an alternative name without breaking existing consumers",
		AffectedModes: modesNone,
		Applies: func(b, h model.Field) bool {
			return hasRemovedAlias(h.Aliases, b.Aliases) // added = removed in the other direction
		},
		Describe: func(b, h model.Field) string {
			added := removedAliases(h.Aliases, b.Aliases)
			return fmt.Sprintf("field %q alias %v added", b.Name, added)
		},
	},
	{
		// F-12: default value changed (both have a default, but value differs)
		ID:       "F-12",
		Reason:   "does not affect already written data",
		Cosmetic: true,
		Applies: func(b, h model.Field) bool {
			return b.HasDefault && h.HasDefault &&
				fmt.Sprintf("%v", b.Default) != fmt.Sprintf("%v", h.Default)
		},
		Describe: func(b, h model.Field) string {
			return fmt.Sprintf("field %q default changed from %v to %v", b.Name, b.Default, h.Default)
		},
	},
	{
		// F-13: default added to a field that had none → MINOR (improves compatibility)
		ID:            "F-13",
		Reason:        "adding a default improves forward compatibility; existing consumers are unaffected",
		AffectedModes: modesNone,
		Applies: func(b, h model.Field) bool {
			return !b.HasDefault && h.HasDefault
		},
		Describe: func(b, h model.Field) string {
			return fmt.Sprintf("field %q default added (value: %v)", b.Name, h.Default)
		},
	},
	{
		// F-14: default removed from a field → MAJOR (worsens compatibility)
		ID:            "F-14",
		Reason:        "removing a default means readers can no longer fall back to a value when the field is absent",
		AffectedModes: modesAll,
		Applies: func(b, h model.Field) bool {
			return b.HasDefault && !h.HasDefault
		},
		Describe: func(b, h model.Field) string {
			return fmt.Sprintf("field %q default removed (was: %v)", b.Name, b.Default)
		},
	},
	{
		// F-15: doc changed
		ID:       "F-15",
		Reason:   "documentation only, no compatibility impact",
		Cosmetic: true,
		Applies: func(b, h model.Field) bool {
			return b.Doc != h.Doc
		},
		Describe: func(b, h model.Field) string {
			return fmt.Sprintf("field %q doc changed", b.Name)
		},
	},
}

// hasRemovedAlias reports whether any alias from 'from' is absent in 'to'.
func hasRemovedAlias(from, to []string) bool {
	toSet := make(map[string]struct{}, len(to))
	for _, a := range to {
		toSet[a] = struct{}{}
	}
	for _, a := range from {
		if _, ok := toSet[a]; !ok {
			return true
		}
	}
	return false
}

// removedAliases returns aliases present in 'from' but absent in 'to'.
func removedAliases(from, to []string) []string {
	toSet := make(map[string]struct{}, len(to))
	for _, a := range to {
		toSet[a] = struct{}{}
	}
	var removed []string
	for _, a := range from {
		if _, ok := toSet[a]; !ok {
			removed = append(removed, a)
		}
	}
	return removed
}