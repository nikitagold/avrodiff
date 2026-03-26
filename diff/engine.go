package diff

import "github.com/nikitagold/avrodiff/model"

func DiffSchemas(base, head *model.Schema, mode model.CompatMode) model.DiffResult {
	ctx := newCtx(base, head, mode)
	changes := diffFields(base, head, "", ctx)
	return model.DiffResult{
		Changes: changes,
		Level:   classifyLevel(changes),
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
func makeChange(path, description, reason string, affectedModes []model.CompatMode, ctx *DiffContext) model.Change {
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
		Path:          path,
		Description:   description,
		Reason:        reason,
		Severity:      sev,
		AffectedModes: affectedModes,
	}
}
