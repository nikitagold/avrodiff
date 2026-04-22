package diff

import (
	"encoding/json"
	"fmt"

	"github.com/nikitagold/avrodiff/model"
)

// enumPairRules contains property-level rules for two enums with the same name.
var enumPairRules = []PairRule[model.EnumSchema]{
	{
		// E-07: alias removed → MAJOR
		ID:            "E-07",
		Reason:        "consumers referencing this enum by alias will fail to deserialize",
		AffectedModes: modesAll,
		Applies: func(b, h model.EnumSchema) bool {
			return hasRemovedAlias(b.Aliases, h.Aliases)
		},
		Describe: func(b, h model.EnumSchema) string {
			removed := removedAliases(b.Aliases, h.Aliases)
			return fmt.Sprintf("enum %q alias %v removed", b.Name, removed)
		},
	},
	{
		// E-08: alias added → MINOR
		ID:            "E-08",
		Reason:        "adding an alias provides an alternative name without breaking existing consumers",
		AffectedModes: modesNone,
		Applies: func(b, h model.EnumSchema) bool {
			return hasRemovedAlias(h.Aliases, b.Aliases)
		},
		Describe: func(b, h model.EnumSchema) string {
			added := removedAliases(h.Aliases, b.Aliases)
			return fmt.Sprintf("enum %q alias %v added", b.Name, added)
		},
	},
	{
		// E-09: default changed → PATCH
		ID:       "E-09",
		Reason:   "affects only how new readers handle unknown symbols; no impact on existing data",
		Cosmetic: true,
		Applies: func(b, h model.EnumSchema) bool {
			return b.Default != h.Default && b.Default != "" && h.Default != ""
		},
		Describe: func(b, h model.EnumSchema) string {
			return fmt.Sprintf("enum %q default changed from %q to %q", b.Name, b.Default, h.Default)
		},
	},
	{
		// E-10: namespace changed → MAJOR (changes the fully-qualified name)
		ID:            "E-10",
		Reason:        "namespace is part of the fully-qualified name used as the schema identifier",
		AffectedModes: modesAll,
		Applies: func(b, h model.EnumSchema) bool {
			return b.Namespace != h.Namespace
		},
		Describe: func(b, h model.EnumSchema) string {
			return fmt.Sprintf("enum %q namespace changed from %q to %q", b.Name, b.Namespace, h.Namespace)
		},
	},
	{
		// E-11: doc changed → PATCH
		ID:       "E-11",
		Reason:   "documentation only, no compatibility impact",
		Cosmetic: true,
		Applies: func(b, h model.EnumSchema) bool {
			return b.Doc != h.Doc
		},
		Describe: func(b, h model.EnumSchema) string {
			return fmt.Sprintf("enum %q doc changed", b.Name)
		},
	},
}

func diffEnums(base, head model.EnumSchema, path string, ctx *DiffContext) []model.Change {
	var changes []model.Change

	baseSymbols := make(map[string]int, len(base.Symbols))
	for i, s := range base.Symbols {
		baseSymbols[s] = i
	}

	headSymbols := make(map[string]int, len(head.Symbols))
	for i, s := range head.Symbols {
		headSymbols[s] = i
	}

	// E-01: Removed symbols: BREAKING for BACKWARD and FULL (old data contains this value)
	for _, s := range base.Symbols {
		if _, ok := headSymbols[s]; !ok {
			changes = append(changes, makeChange(
				"E-01",
				path,
				fmt.Sprintf("enum symbol %q removed", s),
				"old data contains this value; new schema cannot deserialize it",
				modesBackwardFull,
				ctx,
			))
		}
	}

	// E-02/E-03: Added symbols
	for _, s := range head.Symbols {
		if _, ok := baseSymbols[s]; !ok {
			if head.Default != "" {
				// E-03: enum has a default → SAFE (old readers fall back to default for unknown symbols)
				changes = append(changes, makeChange(
					"E-03",
					path,
					fmt.Sprintf("enum symbol %q added (enum default: %q)", s, head.Default),
					"old schema readers use the enum default for unknown symbols",
					modesNone,
					ctx,
				))
			} else {
				// E-02: no default → BREAKING for FORWARD and FULL
				changes = append(changes, makeChange(
					"E-02",
					path,
					fmt.Sprintf("enum symbol %q added without enum default", s),
					"old schema readers don't know this symbol and cannot deserialize it",
					modesForwardFull,
					ctx,
				))
			}
		}
	}

	// E-04: Order changed (only when set is identical): BREAKING in all modes
	if len(changes) == 0 && !symbolOrderEqual(base.Symbols, head.Symbols) {
		changes = append(changes, makeChange(
			"E-04",
			path,
			"enum symbol order changed",
			"enum is encoded as index; reordering changes the meaning of existing data",
			modesAll,
			ctx,
		))
	}

	// E-07...E-11: property-level rules (aliases, default, namespace, doc)
	changes = append(changes, applyPairRules(enumPairRules, base, head, path, ctx)...)

	return changes
}

func symbolOrderEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// toEnumSchema tries to parse an interface{} value as an EnumSchema.
// Returns (schema, true) if the value is an Avro enum type.
func toEnumSchema(v interface{}) (model.EnumSchema, bool) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return model.EnumSchema{}, false
	}
	t, _ := m["type"].(string)
	if t != "enum" {
		return model.EnumSchema{}, false
	}
	// Re-marshal and unmarshal into EnumSchema for clean extraction
	data, err := json.Marshal(m)
	if err != nil {
		return model.EnumSchema{}, false
	}
	var e model.EnumSchema
	if err := json.Unmarshal(data, &e); err != nil {
		return model.EnumSchema{}, false
	}
	return e, true
}
