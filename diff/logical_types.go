package diff

import (
	"fmt"

	"github.com/nikitagold/avrodiff/model"
)

// ltInfo holds extracted logicalType metadata from a resolved Avro type.
type ltInfo struct {
	underlying string  // Avro primitive type: "bytes", "int", "long", etc.
	lt         string  // logicalType value: "decimal", "date", "timestamp-millis", etc.
	precision  float64 // decimal precision (only meaningful when lt == "decimal")
	scale      float64 // decimal scale    (only meaningful when lt == "decimal")
}

// extractLTInfo extracts logicalType metadata from a resolved type value.
// v can be a primitive string ("bytes") or a map[string]interface{} with logicalType.
func extractLTInfo(v interface{}) ltInfo {
	if s, ok := v.(string); ok {
		return ltInfo{underlying: s}
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return ltInfo{}
	}
	info := ltInfo{}
	info.underlying, _ = m["type"].(string)
	info.lt, _ = m["logicalType"].(string)
	info.precision, _ = m["precision"].(float64)
	info.scale, _ = m["scale"].(float64)
	return info
}

// diffLogicalTypes checks L-01..L-07 rules between two resolved type values.
// Returns (changes, true) when logical type rules apply; (nil, false) to fall through.
func diffLogicalTypes(base, head interface{}, path string, ctx *DiffContext) ([]model.Change, bool) {
	b := extractLTInfo(base)
	h := extractLTInfo(head)

	// Neither has a logicalType → not our concern, fall through.
	if b.lt == "" && h.lt == "" {
		return nil, false
	}

	// L-01: logicalType added
	if b.lt == "" && h.lt != "" {
		return []model.Change{makeChange(
			"L-01",
			path,
			fmt.Sprintf("logicalType %q added to %q field", h.lt, h.underlying),
			"adding a logicalType changes deserialization semantics for readers that understand it",
			modesAll,
			ctx,
		)}, true
	}

	// L-02: logicalType removed
	if b.lt != "" && h.lt == "" {
		return []model.Change{makeChange(
			"L-02",
			path,
			fmt.Sprintf("logicalType %q removed from %q field", b.lt, b.underlying),
			"removing a logicalType causes readers to lose semantic meaning of the data",
			modesAll,
			ctx,
		)}, true
	}

	// L-03: logicalType changed to a different one
	if b.lt != h.lt {
		return []model.Change{makeChange(
			"L-03",
			path,
			fmt.Sprintf("logicalType changed from %q to %q", b.lt, h.lt),
			"different logicalTypes are semantically incompatible",
			modesAll,
			ctx,
		)}, true
	}

	// Same logicalType from here on.
	var changes []model.Change

	// L-04..L-06: decimal-specific parameter changes
	if b.lt == "decimal" {
		if b.precision != h.precision {
			changes = append(changes, makeChange(
				"L-04",
				path,
				fmt.Sprintf("decimal precision changed from %g to %g", b.precision, h.precision),
				"changing precision alters the valid value range for decimal data",
				modesAll,
				ctx,
			))
		}
		if b.scale > h.scale {
			changes = append(changes, makeChange(
				"L-05",
				path,
				fmt.Sprintf("decimal scale decreased from %g to %g", b.scale, h.scale),
				"decreasing scale causes loss of fractional precision in existing data",
				modesAll,
				ctx,
			))
		} else if b.scale < h.scale {
			changes = append(changes, makeChange(
				"L-06",
				path,
				fmt.Sprintf("decimal scale increased from %g to %g", b.scale, h.scale),
				"increasing scale means old readers interpret the exponent incorrectly",
				modesAll,
				ctx,
			))
		}
	}

	// L-07: underlying type changed while keeping the same logicalType
	if b.underlying != h.underlying {
		changes = append(changes, makeChange(
			"L-07",
			path,
			fmt.Sprintf("underlying type changed from %q to %q (logicalType %q preserved)", b.underlying, h.underlying, b.lt),
			"the underlying type determines binary encoding; changing it breaks deserialization",
			modesAll,
			ctx,
		))
	}

	return changes, true
}
