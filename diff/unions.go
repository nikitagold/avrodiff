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

	// Removed types: BREAKING for BACKWARD and FULL (old data may contain this type)
	for _, t := range base {
		key := unionTypeKey(t)
		if _, ok := headKeys[key]; !ok {
			changes = append(changes, makeChange(
				path,
				fmt.Sprintf("union type %q removed", key),
				"old data may contain this type; new schema cannot deserialize it",
				modesBackwardFull,
				ctx,
			))
		}
	}

	// Added types: BREAKING for FORWARD and FULL (old readers don't know this type)
	for _, t := range head {
		key := unionTypeKey(t)
		if _, ok := baseKeys[key]; !ok {
			changes = append(changes, makeChange(
				path,
				fmt.Sprintf("union type %q added", key),
				"old schema readers don't know this type and cannot deserialize it",
				modesForwardFull,
				ctx,
			))
		}
	}

	// Order changed (same set of types, different order): BREAKING in all modes
	if len(changes) == 0 && !unionOrderEqual(base, head) {
		changes = append(changes, makeChange(
			path,
			"union type order changed",
			"Avro binary encodes union as index; reordering changes the meaning of existing data",
			modesAll,
			ctx,
		))
	}

	return changes
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
			return typeName + "." + name
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
