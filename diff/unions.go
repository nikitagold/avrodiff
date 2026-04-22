package diff

import (
	"fmt"

	"github.com/nikitagold/avrodiff/model"
)

func diffUnions(base, head []interface{}, path string, ctx *DiffContext) []model.Change {
	var changes []model.Change

	baseKeys := make(map[string]int, len(base)) // key → index
	for i, t := range base {
		baseKeys[unionTypeKey(t)] = i
	}

	headKeys := make(map[string]int, len(head))
	for i, t := range head {
		headKeys[unionTypeKey(t)] = i
	}

	// U-01: Removed types: BREAKING for BACKWARD and FULL (old data may contain this type)
	for _, t := range base {
		key := unionTypeKey(t)
		if _, ok := headKeys[key]; !ok {
			changes = append(changes, makeChange(
				"U-01",
				path,
				fmt.Sprintf("union type %q removed", key),
				"old data may contain this type; new schema cannot deserialize it",
				modesBackwardFull,
				ctx,
			))
		}
	}

	// U-02: Added types: BREAKING for FORWARD and FULL (old readers don't know this type)
	for _, t := range head {
		key := unionTypeKey(t)
		if _, ok := baseKeys[key]; !ok {
			changes = append(changes, makeChange(
				"U-02",
				path,
				fmt.Sprintf("union type %q added", key),
				"old schema readers don't know this type and cannot deserialize it",
				modesForwardFull,
				ctx,
			))
		}
	}

	// U-04: null moved to a non-first position — checked before U-03 so that the more
	// specific rule takes precedence and U-03 does not fire redundantly for the same reorder.
	nullBaseIdx := nullIndex(base)
	nullHeadIdx := nullIndex(head)
	if nullHeadIdx > 0 && nullHeadIdx != nullBaseIdx {
		changes = append(changes, makeChange(
			"U-04",
			path,
			fmt.Sprintf("null is at position %d in union, must be first", nullHeadIdx),
			`["null","T"] and ["T","null"] have different semantics in Avro JSON encoding; null must be first`,
			modesAll,
			ctx,
		))
	}

	// U-03: any other order change (same set of types, different order): BREAKING in all modes
	if len(changes) == 0 && !unionOrderEqual(base, head) {
		changes = append(changes, makeChange(
			"U-03",
			path,
			"union type order changed",
			"Avro binary encodes union as index; reordering changes the meaning of existing data",
			modesAll,
			ctx,
		))
	}

	// U-05: internal changes within matching union members (recursive diff).
	// A U-05 change is emitted when any breaking inner change is found; the inner
	// changes (F-01, E-01, etc.) are also included to preserve detail.
	baseByKey := make(map[string]interface{}, len(base))
	for _, t := range base {
		baseByKey[unionTypeKey(t)] = t
	}
	for _, ht := range head {
		key := unionTypeKey(ht)
		bt, ok := baseByKey[key]
		if !ok {
			continue // already reported by U-02
		}
		memberChanges := diffUnionMember(bt, ht, path, ctx)
		if len(memberChanges) > 0 {
			hasBreaking := false
			for _, c := range memberChanges {
				if c.Severity == model.Breaking {
					hasBreaking = true
					break
				}
			}
			if hasBreaking {
				changes = append(changes, makeChange(
					"U-05",
					path,
					fmt.Sprintf("incompatible change inside union member %q", key),
					"breaking changes inside a union member affect all consumers reading existing data",
					modesAll,
					ctx,
				))
			}
			changes = append(changes, memberChanges...)
		}
	}

	return changes
}

// diffUnionMember recursively diffs two union member types with the same key.
func diffUnionMember(base, head interface{}, path string, ctx *DiffContext) []model.Change {
	if baseRec, ok := toRecordSchema(base); ok {
		if headRec, ok := toRecordSchema(head); ok {
			return diffRecord(baseRec, headRec, path, ctx)
		}
	}
	if baseEnum, ok := toEnumSchema(base); ok {
		if headEnum, ok := toEnumSchema(head); ok {
			return diffEnums(baseEnum, headEnum, path, ctx)
		}
	}
	if baseArr, ok := toArraySchema(base); ok {
		if headArr, ok := toArraySchema(head); ok {
			return diffArrays(baseArr, headArr, path, ctx)
		}
	}
	if baseMap, ok := toMapSchema(base); ok {
		if headMap, ok := toMapSchema(head); ok {
			return diffMaps(baseMap, headMap, path, ctx)
		}
	}
	return nil
}

// nullIndex returns the index of "null" in the union slice, or -1 if absent.
func nullIndex(union []interface{}) int {
	for i, t := range union {
		if s, ok := t.(string); ok && s == "null" {
			return i
		}
	}
	return -1
}

// unionTypeKey returns a stable string identifier for a union member type.
func unionTypeKey(t interface{}) string {
	switch v := t.(type) {
	case string:
		return v
	case map[string]interface{}:
		typeName, _ := v["type"].(string)
		switch typeName {
		case "record", "enum", "fixed":
			name, _ := v["name"].(string)
			if name == "" {
				return fmt.Sprintf("%v", v)
			}
			return typeName + "." + name
		case "array":
			if items, ok := v["items"].(string); ok && items != "" {
				return "array." + items
			}
			return fmt.Sprintf("%v", v)
		case "map":
			if values, ok := v["values"].(string); ok && values != "" {
				return "map." + values
			}
			return fmt.Sprintf("%v", v)
		default:
			return typeName
		}
	}
	return fmt.Sprintf("%v", t)
}

func unionOrderEqual(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if unionTypeKey(a[i]) != unionTypeKey(b[i]) {
			return false
		}
	}
	return true
}
