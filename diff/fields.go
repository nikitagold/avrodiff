package diff

import (
	"fmt"
	"reflect"

	"github.com/nikitagold/avrodiff/model"
)

func diffFields(base, head *model.Schema, path string, ctx *DiffContext) []model.Change {
	var changes []model.Change

	baseByName := make(map[string]model.Field, len(base.Fields))
	for _, f := range base.Fields {
		baseByName[f.Name] = f
	}

	headByName := make(map[string]model.Field, len(head.Fields))
	for _, f := range head.Fields {
		headByName[f.Name] = f
	}

	// alias → head field name: for detecting safe renames
	aliasToHead := make(map[string]string)
	for _, hf := range head.Fields {
		for _, alias := range hf.Aliases {
			aliasToHead[alias] = hf.Name
		}
	}
	// head fields that are safe renames of base fields (skip in "added" loop)
	renamedHeadFields := make(map[string]bool)

	// Removed fields (in base, not in head)
	for _, bf := range base.Fields {
		if _, ok := headByName[bf.Name]; ok {
			continue
		}
		// Renamed with alias preserved → SAFE in all modes (F-06)
		if newName, ok := aliasToHead[bf.Name]; ok {
			renamedHeadFields[newName] = true
			changes = append(changes, makeChange(
				"F-06",
				joinPath(path, bf.Name),
				fmt.Sprintf("field %q renamed to %q (alias preserved)", bf.Name, newName),
				"old name available as alias, backward compatible",
				modesNone,
				ctx,
			))
			continue
		}
		if bf.HasDefault {
			// F-02: SAFE in all modes
			changes = append(changes, makeChange(
				"F-02",
				joinPath(path, bf.Name),
				fmt.Sprintf("field %q removed (had default: %v)", bf.Name, bf.Default),
				"old schema readers can fall back to the default value",
				modesNone,
				ctx,
			))
		} else {
			// F-01: BREAKING for FORWARD and FULL
			changes = append(changes, makeChange(
				"F-01",
				joinPath(path, bf.Name),
				fmt.Sprintf("field %q removed", bf.Name),
				"old schema readers expect this field; without a default they cannot read new data",
				modesForwardFull,
				ctx,
			))
		}
	}

	// Added fields (in head, not in base)
	for _, hf := range head.Fields {
		if _, ok := baseByName[hf.Name]; ok {
			continue
		}
		if renamedHeadFields[hf.Name] {
			continue
		}
		if hf.HasDefault {
			// F-04: SAFE in all modes
			changes = append(changes, makeChange(
				"F-04",
				joinPath(path, hf.Name),
				fmt.Sprintf("field %q added (default: %v)", hf.Name, hf.Default),
				"backward and forward compatible",
				modesNone,
				ctx,
			))
		} else {
			// F-03: BREAKING for BACKWARD and FULL
			changes = append(changes, makeChange(
				"F-03",
				joinPath(path, hf.Name),
				fmt.Sprintf("field %q added without default", hf.Name),
				"new schema readers cannot find this field in old data",
				modesBackwardFull,
				ctx,
			))
		}
	}

	// F-11: field order changed among common fields → MAJOR
	// Avro binary format encodes fields by position; reordering breaks consumers
	// that wrote data without an embedded schema.
	baseOrder := make([]string, 0, len(base.Fields))
	for _, bf := range base.Fields {
		if _, ok := headByName[bf.Name]; ok {
			baseOrder = append(baseOrder, bf.Name)
		}
	}
	headOrder := make([]string, 0, len(head.Fields))
	for _, hf := range head.Fields {
		if _, ok := baseByName[hf.Name]; ok {
			headOrder = append(headOrder, hf.Name)
		}
	}
	if !fieldOrderEqual(baseOrder, headOrder) {
		changes = append(changes, makeChange(
			"F-11",
			path,
			"field order changed",
			"Avro binary encodes fields by position; reordering breaks consumers reading data without an embedded schema",
			modesAll,
			ctx,
		))
	}

	// Changed fields (in both)
	for _, bf := range base.Fields {
		hf, ok := headByName[bf.Name]
		if !ok {
			continue
		}
		changes = append(changes, diffFieldType(bf, hf, path, ctx)...)
		changes = append(changes, applyPairRules(fieldPairRules, bf, hf, joinPath(path, bf.Name), ctx)...)
	}

	return changes
}

func fieldOrderEqual(a, b []string) bool {
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

func diffFieldType(base, head model.Field, path string, ctx *DiffContext) []model.Change {
	// Resolve named type references before comparison.
	// Primitives ("string", "int", etc.) are not in the registry and pass through unchanged.
	baseType := resolveType(base.Type, ctx.BaseTypes)
	headType := resolveType(head.Type, ctx.HeadTypes)

	if reflect.DeepEqual(baseType, headType) {
		return nil
	}

	// Both are inline records with the same name → recurse
	if baseRec, ok := toRecordSchema(baseType); ok {
		if headRec, ok := toRecordSchema(headType); ok && baseRec.Name == headRec.Name {
			return diffNestedRecord(baseRec, headRec, joinPath(path, base.Name), ctx)
		}
	}

	// Both are enums → diff their symbols, or detect a rename (E-05/E-06)
	if baseEnum, ok := toEnumSchema(baseType); ok {
		if headEnum, ok := toEnumSchema(headType); ok {
			enumPath := joinPath(path, base.Name)
			if baseEnum.Name != headEnum.Name {
				// E-05/E-06: enum renamed
				if sliceContains(headEnum.Aliases, baseEnum.Name) {
					// E-06: old name preserved as alias → MINOR
					return []model.Change{makeChange(
						"E-06",
						enumPath,
						fmt.Sprintf("enum %q renamed to %q (alias preserved)", baseEnum.Name, headEnum.Name),
						"old name is available as an alias; consumers using the old name remain compatible",
						modesNone,
						ctx,
					)}
				}
				// E-05: renamed without alias → MAJOR
				return []model.Change{makeChange(
					"E-05",
					enumPath,
					fmt.Sprintf("enum %q renamed to %q without alias", baseEnum.Name, headEnum.Name),
					"the fully-qualified name is the schema identifier; renaming without an alias breaks all consumers",
					modesAll,
					ctx,
				)}
			}
			return diffEnums(baseEnum, headEnum, enumPath, ctx)
		}
	}

	// Both are fixed types with the same name → diff their size
	if baseFixed, ok := toFixedSchema(baseType); ok {
		if headFixed, ok := toFixedSchema(headType); ok && baseFixed["name"] == headFixed["name"] {
			return diffFixed(baseFixed, headFixed, joinPath(path, base.Name), ctx)
		}
	}

	// Both are unions → diff their members
	if baseUnion, ok := baseType.([]interface{}); ok {
		if headUnion, ok := headType.([]interface{}); ok {
			return diffUnions(baseUnion, headUnion, joinPath(path, base.Name), ctx)
		}
	}

	// Both are arrays → diff their items; one side is array → A-03
	if baseArr, ok := toArraySchema(baseType); ok {
		if headArr, ok := toArraySchema(headType); ok {
			return diffArrays(baseArr, headArr, joinPath(path, base.Name), ctx)
		}
		return []model.Change{makeChange(
			"A-03",
			joinPath(path, base.Name),
			fmt.Sprintf("field %q type changed from array to %v", base.Name, head.Type),
			"replacing an array with a different type is binary incompatible",
			modesAll,
			ctx,
		)}
	}

	// Both are maps → diff their values; one side is map → M-03
	if baseMap, ok := toMapSchema(baseType); ok {
		if headMap, ok := toMapSchema(headType); ok {
			return diffMaps(baseMap, headMap, joinPath(path, base.Name), ctx)
		}
		return []model.Change{makeChange(
			"M-03",
			joinPath(path, base.Name),
			fmt.Sprintf("field %q type changed from map to %v", base.Name, head.Type),
			"replacing a map with a different type is binary incompatible",
			modesAll,
			ctx,
		)}
	}

	// Safe union widening: "T" → ["null", "T"]
	if isSafeNullableWidening(baseType, headType) {
		return []model.Change{makeChange(
			"",
			joinPath(path, base.Name),
			fmt.Sprintf("field %q type widened to nullable %v", base.Name, head.Type),
			"adding null to a union is backward and forward compatible",
			modesNone,
			ctx,
		)}
	}

	// Narrowing: ["null", "T"] → "T"
	if isSafeNullableWidening(headType, baseType) {
		return []model.Change{makeChange(
			"",
			joinPath(path, base.Name),
			fmt.Sprintf("field %q type narrowed from nullable %v to %v", base.Name, base.Type, head.Type),
			"removing null from a union breaks consumers that wrote null values",
			modesAll,
			ctx,
		)}
	}

	// FIX-2: one side is a union, the other is a single type.
	// The 2-element null cases are already caught above by isSafeNullableWidening.
	// Normalize the single type to a slice and delegate to diffUnions so that
	// removed/added members get the correct affected modes.
	if baseUnion, ok := baseType.([]interface{}); ok {
		return diffUnions(baseUnion, []interface{}{headType}, joinPath(path, base.Name), ctx)
	}
	if headUnion, ok := headType.([]interface{}); ok {
		return diffUnions([]interface{}{baseType}, headUnion, joinPath(path, base.Name), ctx)
	}

	// F-10: type promotion (int→long, float→double, etc.) → MINOR
	if isTypePromotion(baseType, headType) {
		return []model.Change{makeChange(
			"F-10",
			joinPath(path, base.Name),
			fmt.Sprintf("field %q type promoted from %v to %v", base.Name, base.Type, head.Type),
			"Avro supports this type promotion; existing data remains readable",
			modesNone,
			ctx,
		)}
	}

	// L-01..L-07: logicalType rules (at least one side has a logicalType annotation)
	if ltChanges, handled := diffLogicalTypes(baseType, headType, joinPath(path, base.Name), ctx); handled {
		return ltChanges
	}

	return []model.Change{makeChange(
		"F-09",
		joinPath(path, base.Name),
		fmt.Sprintf("field %q type changed from %v to %v", base.Name, base.Type, head.Type),
		"type mismatch causes binary incompatibility",
		modesAll,
		ctx,
	)}
}

// isSafeNullableWidening checks if from="T" and to=["null","T"] (or ["T","null"]).
// "T" can be a primitive string ("string", "int", ...) or an inline named type definition
// (map[string]interface{} with a "name" field), in which case the union member is matched
// by name string reference.
func isSafeNullableWidening(from, to interface{}) bool {
	fromStr, ok := from.(string)
	if !ok {
		if m, ok := from.(map[string]interface{}); ok {
			fromStr, _ = m["name"].(string)
		}
		if fromStr == "" {
			return false
		}
	}
	toSlice, ok := to.([]interface{})
	if !ok || len(toSlice) != 2 {
		return false
	}
	hasNull := false
	hasOriginal := false
	for _, t := range toSlice {
		if s, ok := t.(string); ok {
			if s == "null" {
				hasNull = true
			}
			if s == fromStr {
				hasOriginal = true
			}
		}
	}
	return hasNull && hasOriginal
}

func joinPath(prefix, name string) string {
	if prefix == "" {
		return "fields." + name
	}
	return prefix + ".fields." + name
}
