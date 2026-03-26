package diff

import (
	"encoding/json"

	"github.com/nikitagold/avrodiff/model"
)

// diffNestedRecord recursively diffs two inline record schemas.
func diffNestedRecord(base, head model.Schema, path string, ctx *DiffContext) []model.Change {
	return diffFields(&base, &head, path, ctx)
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
