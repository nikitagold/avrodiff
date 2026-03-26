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
		// Renamed with alias preserved → SAFE in all modes
		if newName, ok := aliasToHead[bf.Name]; ok {
			renamedHeadFields[newName] = true
			changes = append(changes, makeChange(
				joinPath(path, bf.Name),
				fmt.Sprintf("field %q renamed to %q (alias preserved)", bf.Name, newName),
				"old name available as alias, backward compatible",
				modesNone,
				ctx,
			))
			continue
		}
		if bf.HasDefault {
			// SAFE in all modes: old readers fall back to their default when the field is absent
			changes = append(changes, makeChange(
				joinPath(path, bf.Name),
				fmt.Sprintf("field %q removed (had default: %v)", bf.Name, bf.Default),
				"old schema readers can fall back to the default value",
				modesNone,
				ctx,
			))
		} else {
			// BREAKING for FORWARD and FULL: old readers expect the field and have no fallback
			changes = append(changes, makeChange(
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
			// SAFE in all modes: new readers use the default when reading old data
			changes = append(changes, makeChange(
				joinPath(path, hf.Name),
				fmt.Sprintf("field %q added (default: %v)", hf.Name, hf.Default),
				"backward and forward compatible",
				modesNone,
				ctx,
			))
		} else {
			// BREAKING for BACKWARD and FULL: new readers cannot find this field in old data
			changes = append(changes, makeChange(
				joinPath(path, hf.Name),
				fmt.Sprintf("field %q added without default", hf.Name),
				"new schema readers cannot find this field in old data",
				modesBackwardFull,
				ctx,
			))
		}
	}

	// Changed fields (in both)
	for _, bf := range base.Fields {
		hf, ok := headByName[bf.Name]
		if !ok {
			continue
		}
		changes = append(changes, diffFieldType(bf, hf, path, ctx)...)
		changes = append(changes, diffFieldCosmetic(bf, hf, path)...)
	}

	return changes
}

func diffFieldCosmetic(base, head model.Field, path string) []model.Change {
	var changes []model.Change

	if base.Doc != head.Doc {
		changes = append(changes, model.Change{
			Path:        joinPath(path, base.Name),
			Description: fmt.Sprintf("field %q doc changed", base.Name),
			Reason:      "documentation only, no compatibility impact",
			Severity:    model.Cosmetic,
		})
	}

	// Default value changed (both have a default, but the value differs)
	if base.HasDefault && head.HasDefault {
		baseVal := fmt.Sprintf("%v", base.Default)
		headVal := fmt.Sprintf("%v", head.Default)
		if baseVal != headVal {
			changes = append(changes, model.Change{
				Path:        joinPath(path, base.Name),
				Description: fmt.Sprintf("field %q default changed from %v to %v", base.Name, base.Default, head.Default),
				Reason:      "does not affect already written data",
				Severity:    model.Cosmetic,
			})
		}
	}

	return changes
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

	// Both are enums with the same name → diff their symbols
	if baseEnum, ok := toEnumSchema(baseType); ok {
		if headEnum, ok := toEnumSchema(headType); ok && baseEnum.Name == headEnum.Name {
			return diffEnums(baseEnum, headEnum, joinPath(path, base.Name), ctx)
		}
	}

	// Both are unions → diff their members
	if baseUnion, ok := baseType.([]interface{}); ok {
		if headUnion, ok := headType.([]interface{}); ok {
			return diffUnions(baseUnion, headUnion, joinPath(path, base.Name), ctx)
		}
	}

	// Both are arrays → diff their items
	if baseArr, ok := toArraySchema(baseType); ok {
		if headArr, ok := toArraySchema(headType); ok {
			return diffArrays(baseArr, headArr, joinPath(path, base.Name), ctx)
		}
	}

	// Both are maps → diff their values
	if baseMap, ok := toMapSchema(baseType); ok {
		if headMap, ok := toMapSchema(headType); ok {
			return diffMaps(baseMap, headMap, joinPath(path, base.Name), ctx)
		}
	}

	// Safe union widening: "T" → ["null", "T"]
	if isSafeNullableWidening(baseType, headType) {
		return []model.Change{makeChange(
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
			joinPath(path, base.Name),
			fmt.Sprintf("field %q type narrowed from nullable %v to %v", base.Name, base.Type, head.Type),
			"removing null from a union breaks consumers that wrote null values",
			modesAll,
			ctx,
		)}
	}

	return []model.Change{makeChange(
		joinPath(path, base.Name),
		fmt.Sprintf("field %q type changed from %v to %v", base.Name, base.Type, head.Type),
		"type mismatch causes binary incompatibility",
		modesAll,
		ctx,
	)}
}

// isSafeNullableWidening checks if from="T" and to=["null","T"] (or ["T","null"]).
func isSafeNullableWidening(from, to interface{}) bool {
	fromStr, ok := from.(string)
	if !ok {
		return false
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
