package diff

import (
	"encoding/json"
	"fmt"

	"github.com/nikitagold/avrodiff/model"
)

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

	// Removed symbols: BREAKING for BACKWARD and FULL (old data contains this value)
	for _, s := range base.Symbols {
		if _, ok := headSymbols[s]; !ok {
			changes = append(changes, makeChange(
				path,
				fmt.Sprintf("enum symbol %q removed", s),
				"old data contains this value; new schema cannot deserialize it",
				modesBackwardFull,
				ctx,
			))
		}
	}

	// Added symbols: BREAKING for FORWARD and FULL (old readers don't know this index)
	for _, s := range head.Symbols {
		if _, ok := baseSymbols[s]; !ok {
			changes = append(changes, makeChange(
				path,
				fmt.Sprintf("enum symbol %q added", s),
				"old schema readers don't know this symbol and cannot deserialize it",
				modesForwardFull,
				ctx,
			))
		}
	}

	// Order changed (only when set is identical): BREAKING in all modes
	if len(changes) == 0 && !symbolOrderEqual(base.Symbols, head.Symbols) {
		changes = append(changes, makeChange(
			path,
			"enum symbol order changed",
			"enum is encoded as index; reordering changes the meaning of existing data",
			modesAll,
			ctx,
		))
	}

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
