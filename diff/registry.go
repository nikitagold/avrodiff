package diff

import "github.com/nikitagold/avrodiff/model"

// DiffContext holds named type registries and the active compatibility mode.
type DiffContext struct {
	BaseTypes map[string]interface{}
	HeadTypes map[string]interface{}
	Mode      model.CompatMode
}

func newCtx(base, head *model.Schema, mode model.CompatMode) *DiffContext {
	return &DiffContext{
		BaseTypes: buildTypeRegistry(base),
		HeadTypes: buildTypeRegistry(head),
		Mode:      mode,
	}
}

// Predefined affected-mode sets for common change categories.
var (
	modesAll          = []model.CompatMode{model.ModeBackward, model.ModeForward, model.ModeFull}
	modesForwardFull  = []model.CompatMode{model.ModeForward, model.ModeFull}
	modesBackwardFull = []model.CompatMode{model.ModeBackward, model.ModeFull}
	modesNone         = []model.CompatMode{}
)

// buildTypeRegistry walks an Avro schema and collects all named type definitions
// (record, enum, fixed) indexed by their short name and fully-qualified name.
func buildTypeRegistry(schema *model.Schema) map[string]interface{} {
	reg := make(map[string]interface{})
	for _, f := range schema.Fields {
		collectFromInterface(f.Type, reg)
	}
	return reg
}

// collectFromInterface recursively finds named type definitions inside an interface{}
// value (as produced by JSON unmarshaling) and adds them to reg.
func collectFromInterface(v interface{}, reg map[string]interface{}) {
	switch t := v.(type) {
	case map[string]interface{}:
		typeName, _ := t["type"].(string)
		switch typeName {
		case "record":
			name, _ := t["name"].(string)
			if name != "" {
				reg[name] = t
				if ns, _ := t["namespace"].(string); ns != "" {
					reg[ns+"."+name] = t
				}
			}
			// Recurse into nested fields
			if fields, ok := t["fields"].([]interface{}); ok {
				for _, f := range fields {
					if fm, ok := f.(map[string]interface{}); ok {
						if ft, ok := fm["type"]; ok {
							collectFromInterface(ft, reg)
						}
					}
				}
			}
		case "enum", "fixed":
			name, _ := t["name"].(string)
			if name != "" {
				reg[name] = t
				if ns, _ := t["namespace"].(string); ns != "" {
					reg[ns+"."+name] = t
				}
			}
		case "array":
			if items, ok := t["items"]; ok {
				collectFromInterface(items, reg)
			}
		case "map":
			if values, ok := t["values"]; ok {
				collectFromInterface(values, reg)
			}
		}
	case []interface{}:
		for _, elem := range t {
			collectFromInterface(elem, reg)
		}
	}
}

// resolveType resolves a named type reference (a string) to its definition in reg.
// Primitive types ("string", "int", etc.) are not in reg and are returned as-is.
func resolveType(t interface{}, reg map[string]interface{}) interface{} {
	name, ok := t.(string)
	if !ok {
		return t
	}
	if resolved, found := reg[name]; found {
		return resolved
	}
	return t
}
