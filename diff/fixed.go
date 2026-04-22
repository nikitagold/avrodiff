package diff

import (
	"fmt"

	"github.com/nikitagold/avrodiff/model"
)

func toFixedSchema(v interface{}) (map[string]interface{}, bool) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, false
	}
	if t, _ := m["type"].(string); t != "fixed" {
		return nil, false
	}
	return m, true
}

func diffFixed(base, head map[string]interface{}, path string, ctx *DiffContext) []model.Change {
	name, _ := base["name"].(string)
	baseSize, _ := base["size"].(float64) // JSON numbers unmarshal to float64
	headSize, _ := head["size"].(float64)
	if baseSize == headSize {
		return nil
	}
	return []model.Change{makeChange(
		"FIXED-01",
		path,
		fmt.Sprintf("fixed type %q size changed from %d to %d", name, int(baseSize), int(headSize)),
		"fixed type is binary-encoded as exactly size bytes; size change breaks all consumers",
		modesAll,
		ctx,
	)}
}