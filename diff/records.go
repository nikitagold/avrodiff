package diff

import (
	"encoding/json"
	"fmt"

	"github.com/nikitagold/avrodiff/model"
)

// recordPairRules contains property-level rules for two records with the same name.
var recordPairRules = []PairRule[model.Schema]{
	{
		// R-04: namespace changed → MAJOR (changes the fully-qualified name)
		ID:            "R-04",
		Reason:        "namespace is part of the fully-qualified name used as the schema identifier",
		AffectedModes: modesAll,
		Applies: func(b, h model.Schema) bool {
			return b.Namespace != h.Namespace
		},
		Describe: func(b, h model.Schema) string {
			return fmt.Sprintf("record %q namespace changed from %q to %q", b.Name, b.Namespace, h.Namespace)
		},
	},
	{
		// R-05: alias removed → MAJOR (consumers using that alias break)
		ID:            "R-05",
		Reason:        "consumers referencing this record by alias will fail to deserialize",
		AffectedModes: modesAll,
		Applies: func(b, h model.Schema) bool {
			return hasRemovedAlias(b.Aliases, h.Aliases)
		},
		Describe: func(b, h model.Schema) string {
			removed := removedAliases(b.Aliases, h.Aliases)
			return fmt.Sprintf("record %q alias %v removed", b.Name, removed)
		},
	},
	{
		// R-06: alias added → MINOR (adds alternative name, nothing breaks)
		ID:            "R-06",
		Reason:        "adding an alias provides an alternative name without breaking existing consumers",
		AffectedModes: modesNone,
		Applies: func(b, h model.Schema) bool {
			return hasRemovedAlias(h.Aliases, b.Aliases)
		},
		Describe: func(b, h model.Schema) string {
			added := removedAliases(h.Aliases, b.Aliases)
			return fmt.Sprintf("record %q alias %v added", b.Name, added)
		},
	},
	{
		// R-07: doc changed → PATCH
		ID:       "R-07",
		Reason:   "documentation only, no compatibility impact",
		Cosmetic: true,
		Applies: func(b, h model.Schema) bool {
			return b.Doc != h.Doc
		},
		Describe: func(b, h model.Schema) string {
			return fmt.Sprintf("record %q doc changed", b.Name)
		},
	},
}

// diffNestedRecord recursively diffs two inline record schemas.
func diffNestedRecord(base, head model.Schema, path string, ctx *DiffContext) []model.Change {
	return diffRecord(base, head, path, ctx)
}

// diffRecord compares two Avro record schemas at the same position.
// It checks record-level rules (name, namespace, aliases, doc) then delegates to diffFields.
func diffRecord(base, head model.Schema, path string, ctx *DiffContext) []model.Change {
	var changes []model.Change

	// R-02/R-03: record renamed
	if base.Name != head.Name {
		if sliceContains(head.Aliases, base.Name) {
			// R-03: alias covers the old name → MINOR
			changes = append(changes, makeChange(
				"R-03",
				path,
				fmt.Sprintf("record renamed from %q to %q (alias preserved)", base.Name, head.Name),
				"old name is available as an alias; consumers using the old name remain compatible",
				modesNone,
				ctx,
			))
		} else {
			// R-02: no alias → MAJOR
			changes = append(changes, makeChange(
				"R-02",
				path,
				fmt.Sprintf("record renamed from %q to %q without alias", base.Name, head.Name),
				"the fully-qualified name is the schema identifier; renaming without an alias breaks all consumers",
				modesAll,
				ctx,
			))
		}
	}

	changes = append(changes, applyPairRules(recordPairRules, base, head, path, ctx)...)
	changes = append(changes, diffFields(&base, &head, path, ctx)...)

	return changes
}

func sliceContains(s []string, v string) bool {
	for _, elem := range s {
		if elem == v {
			return true
		}
	}
	return false
}

// toRecordSchema tries to parse an interface{} value as a record Schema.
// Returns (schema, true) if the value is an inline Avro record type.
func toRecordSchema(v interface{}) (model.Schema, bool) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return model.Schema{}, false
	}
	t, _ := m["type"].(string)
	if t != "record" {
		return model.Schema{}, false
	}
	data, err := json.Marshal(m)
	if err != nil {
		return model.Schema{}, false
	}
	var s model.Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return model.Schema{}, false
	}
	return s, true
}