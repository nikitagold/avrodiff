package diff

import (
	"path/filepath"
	"testing"

	"github.com/nikitagold/avrodiff/model"
)

func TestExamples(t *testing.T) {
	examples := []struct {
		dir         string
		mode        model.CompatMode // defaults to ModeFull if zero
		wantLevel   model.SemverLevel
		wantChanges int // 0 = don't check count
		desc        string
	}{
		// MAJOR — breaking changes (FULL mode)
		{
			dir:       "major-field-removed",
			wantLevel: model.LevelMajor,
			desc:      "removing a required field breaks consumers",
		},
		{
			dir:       "major-type-changed",
			wantLevel: model.LevelMajor,
			desc:      "changing field type breaks binary compatibility",
		},
		{
			dir:       "major-enum-reordered",
			wantLevel: model.LevelMajor,
			desc:      "reordering enum symbols changes their index",
		},

		// MINOR — safe additions (FULL mode)
		{
			dir:       "minor-field-added",
			wantLevel: model.LevelMinor,
			desc:      "adding optional field with null default is backward compatible",
		},
		{
			// In FULL mode, enum symbol added is MAJOR (old readers can't handle new index)
			dir:       "minor-enum-symbol-added",
			wantLevel: model.LevelMajor,
			desc:      "adding enum symbol is BREAKING in FULL mode (old readers can't deserialize new index)",
		},
		{
			// In BACKWARD mode, enum symbol added is MINOR (old data doesn't contain new symbol)
			dir:       "minor-enum-symbol-added",
			mode:      model.ModeBackward,
			wantLevel: model.LevelMinor,
			desc:      "adding enum symbol is SAFE in BACKWARD mode",
		},

		// PATCH — cosmetic changes
		{
			dir:       "patch-doc-changed",
			wantLevel: model.LevelPatch,
			desc:      "doc change is cosmetic, no compatibility impact",
		},
		{
			dir:       "patch-default-changed",
			wantLevel: model.LevelPatch,
			desc:      "changing an existing default value is cosmetic",
		},

		// NONE — no structural changes
		{
			dir:       "none-no-changes",
			wantLevel: model.LevelNone,
			desc:      "identical schemas produce no changes",
		},
		{
			dir:       "none-doc-changed",
			wantLevel: model.LevelNone,
			desc:      "schema without doc fields — still no changes",
		},

		// EDGE CASES
		{
			dir:       "edge-null-default",
			wantLevel: model.LevelMinor,
			desc:      `"default": null is a valid default (HasDefault=true), field addition is SAFE`,
		},
		{
			dir:         "edge-alias-rename",
			wantLevel:   model.LevelMinor,
			wantChanges: 1,
			desc:        "rename with alias preserved is SAFE",
		},
		{
			dir:       "edge-union-widening",
			wantLevel: model.LevelMinor,
			desc:      `"string" → ["null","string"] is SAFE widening`,
		},
		{
			dir:       "edge-nested-field-removed",
			wantLevel: model.LevelMajor,
			desc:      "removing field inside nested record is BREAKING",
		},
	}

	for _, ex := range examples {
		ex := ex
		mode := ex.mode
		if mode == "" {
			mode = model.ModeFull
		}
		name := ex.dir
		if ex.mode != "" {
			name += "/" + string(ex.mode)
		}
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("../testdata/examples", ex.dir)
			base, err := model.ReadSchema(filepath.Join(dir, "base.avsc"))
			if err != nil {
				t.Fatalf("parse base: %v", err)
			}
			head, err := model.ReadSchema(filepath.Join(dir, "head.avsc"))
			if err != nil {
				t.Fatalf("parse head: %v", err)
			}

			result := DiffSchemas(base, head, mode)

			if result.Level != ex.wantLevel {
				t.Errorf("level: got %s, want %s\n  desc: %s\n  changes: %v",
					result.Level, ex.wantLevel, ex.desc, result.Changes)
			}
			if ex.wantChanges > 0 && len(result.Changes) != ex.wantChanges {
				t.Errorf("changes count: got %d, want %d — %v",
					len(result.Changes), ex.wantChanges, result.Changes)
			}
		})
	}
}
