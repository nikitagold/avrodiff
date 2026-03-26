package diff

import (
	"fmt"
	"reflect"

	"github.com/nikitagold/avrodiff/model"
)

// toArraySchema returns the raw map and true if v is an Avro array type.
func toArraySchema(v interface{}) (map[string]interface{}, bool) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, false
	}
	if t, _ := m["type"].(string); t != "array" {
		return nil, false
	}
	return m, true
}

// toMapSchema returns the raw map and true if v is an Avro map type.
func toMapSchema(v interface{}) (map[string]interface{}, bool) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, false
	}
	if t, _ := m["type"].(string); t != "map" {
		return nil, false
	}
	return m, true
}

// diffArrays compares two Avro array schemas.
// path is the already-joined path to the field (e.g. "fields.tags").
func diffArrays(base, head map[string]interface{}, path string, ctx *DiffContext) []model.Change {
	baseItems := base["items"]
	headItems := head["items"]

	baseResolved := resolveType(baseItems, ctx.BaseTypes)
	headResolved := resolveType(headItems, ctx.HeadTypes)

	if reflect.DeepEqual(baseResolved, headResolved) {
		return nil
	}

	itemsPath := path + ".items"

	// Both items are records with the same name → recurse
	if baseRec, ok := toRecordSchema(baseResolved); ok {
		if headRec, ok := toRecordSchema(headResolved); ok && baseRec.Name == headRec.Name {
			return diffNestedRecord(baseRec, headRec, itemsPath, ctx)
		}
	}

	// Both items are enums with the same name → diff symbols
	if baseEnum, ok := toEnumSchema(baseResolved); ok {
		if headEnum, ok := toEnumSchema(headResolved); ok && baseEnum.Name == headEnum.Name {
			return diffEnums(baseEnum, headEnum, itemsPath, ctx)
		}
	}

	// Safe widening: "T" → ["null", "T"]
	if isSafeNullableWidening(baseResolved, headResolved) {
		return []model.Change{makeChange(
			path,
			fmt.Sprintf("array items type widened to nullable %v", headItems),
			"adding null to items union is backward and forward compatible",
			modesNone,
			ctx,
		)}
	}

	// Narrowing: ["null", "T"] → "T"
	if isSafeNullableWidening(headResolved, baseResolved) {
		return []model.Change{makeChange(
			path,
			fmt.Sprintf("array items type narrowed from nullable %v to %v", baseItems, headItems),
			"removing null from items union breaks consumers that stored null values",
			modesAll,
			ctx,
		)}
	}

	return []model.Change{makeChange(
		path,
		fmt.Sprintf("array items type changed from %v to %v", baseItems, headItems),
		"consumers cannot deserialize existing array elements",
		modesAll,
		ctx,
	)}
}

// diffMaps compares two Avro map schemas.
// path is the already-joined path to the field (e.g. "fields.metadata").
func diffMaps(base, head map[string]interface{}, path string, ctx *DiffContext) []model.Change {
	baseValues := base["values"]
	headValues := head["values"]

	baseResolved := resolveType(baseValues, ctx.BaseTypes)
	headResolved := resolveType(headValues, ctx.HeadTypes)

	if reflect.DeepEqual(baseResolved, headResolved) {
		return nil
	}

	valuesPath := path + ".values"

	// Both values are records with the same name → recurse
	if baseRec, ok := toRecordSchema(baseResolved); ok {
		if headRec, ok := toRecordSchema(headResolved); ok && baseRec.Name == headRec.Name {
			return diffNestedRecord(baseRec, headRec, valuesPath, ctx)
		}
	}

	// Both values are enums with the same name → diff symbols
	if baseEnum, ok := toEnumSchema(baseResolved); ok {
		if headEnum, ok := toEnumSchema(headResolved); ok && baseEnum.Name == headEnum.Name {
			return diffEnums(baseEnum, headEnum, valuesPath, ctx)
		}
	}

	// Safe widening: "T" → ["null", "T"]
	if isSafeNullableWidening(baseResolved, headResolved) {
		return []model.Change{makeChange(
			path,
			fmt.Sprintf("map values type widened to nullable %v", headValues),
			"adding null to values union is backward and forward compatible",
			modesNone,
			ctx,
		)}
	}

	// Narrowing: ["null", "T"] → "T"
	if isSafeNullableWidening(headResolved, baseResolved) {
		return []model.Change{makeChange(
			path,
			fmt.Sprintf("map values type narrowed from nullable %v to %v", baseValues, headValues),
			"removing null from values union breaks consumers that stored null values",
			modesAll,
			ctx,
		)}
	}

	return []model.Change{makeChange(
		path,
		fmt.Sprintf("map values type changed from %v to %v", baseValues, headValues),
		"consumers cannot deserialize existing map values",
		modesAll,
		ctx,
	)}
}
