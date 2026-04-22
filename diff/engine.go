package diff

import (
	"fmt"

	"github.com/nikitagold/avrodiff/model"
)

func DiffSchemas(base, head *model.Schema, mode model.CompatMode) model.DiffResult {
	ctx := newCtx(base, head, mode)

	// S-01: root schema type changed (record → enum, etc.) — diff is meaningless, stop early.
	if base.Type != head.Type {
		s01 := model.Change{
			Rule:        "S-01",
			Path:        "",
			Description: fmt.Sprintf("root schema type changed from %q to %q", base.Type, head.Type),
			Reason:      "schema type is the fundamental contract; changing it breaks all consumers",
			Severity:    model.Breaking,
		}
		return model.DiffResult{
			Changes: []model.Change{s01},
			Level:   model.LevelMajor,
			Mode:    mode,
		}
	}

	contentChanges := diffRecord(*base, *head, "", ctx)
	contentLevel := classifyLevel(contentChanges)

	lintChanges := diffSchemaVersion(base, head, contentLevel)

	all := append(contentChanges, lintChanges...)
	return model.DiffResult{
		Changes: all,
		Level:   classifyLevel(all),
		Mode:    mode,
	}
}

func classifyLevel(changes []model.Change) model.SemverLevel {
	if len(changes) == 0 {
		return model.LevelNone
	}
	hasBreaking := false
	hasSafe := false
	hasCosmetic := false
	for _, c := range changes {
		switch c.Severity {
		case model.Breaking:
			hasBreaking = true
		case model.Safe:
			hasSafe = true
		case model.Cosmetic:
			hasCosmetic = true
		}
	}
	if hasBreaking {
		return model.LevelMajor
	}
	if hasSafe {
		return model.LevelMinor
	}
	if hasCosmetic {
		return model.LevelPatch
	}
	return model.LevelNone
}

// makeChange creates a Change whose Severity is computed from ctx.Mode and affectedModes.
// A change is BREAKING if the active mode is listed in affectedModes; SAFE otherwise.
func makeChange(ruleID, path, description, reason string, affectedModes []model.CompatMode, ctx *DiffContext) model.Change {
	sev := model.Safe
	if ctx.Mode != model.ModeNone {
		for _, m := range affectedModes {
			if m == ctx.Mode {
				sev = model.Breaking
				break
			}
		}
	}
	return model.Change{
		Rule:          ruleID,
		Path:          path,
		Description:   description,
		Reason:        reason,
		Severity:      sev,
		AffectedModes: affectedModes,
	}
}
