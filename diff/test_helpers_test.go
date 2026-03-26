package diff

import "github.com/nikitagold/avrodiff/model"

// minCtx returns a DiffContext with empty registries for the given mode.
// Used in tests that call internal diff functions directly.
func minCtx(mode model.CompatMode) *DiffContext {
	return &DiffContext{
		Mode:      mode,
		BaseTypes: map[string]interface{}{},
		HeadTypes: map[string]interface{}{},
	}
}
