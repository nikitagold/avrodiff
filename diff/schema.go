package diff

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nikitagold/avrodiff/model"
)

// diffSchemaVersion runs S-02..S-04 lint checks on the version field.
// contentLevel is the change level computed from the schema content (before lint).
// Lint changes are always Breaking regardless of compat mode.
// Checks only run when base.Version is set — schemas without versioning are skipped.
func diffSchemaVersion(base, head *model.Schema, contentLevel model.SemverLevel) []model.Change {
	// If the base schema has no version, this project hasn't opted into version tracking.
	if base.Version == "" {
		return nil
	}

	var changes []model.Change

	// S-02: version field absent in head schema (base had one, head dropped it)
	if head.Version == "" {
		changes = append(changes, model.Change{
			Rule:        "S-02",
			Path:        "version",
			Description: "schema is missing a \"version\" field",
			Reason:      "version field is required for semver tracking",
			Severity:    model.Breaking,
		})
		return changes // S-03 and S-04 require a parseable version, stop here
	}

	headVer, headErr := parseSemver(head.Version)
	baseVer, baseErr := parseSemver(base.Version)

	if headErr != nil {
		changes = append(changes, model.Change{
			Rule:        "S-02",
			Path:        "version",
			Description: fmt.Sprintf("version %q is not valid semver (expected MAJOR.MINOR.PATCH)", head.Version),
			Reason:      "version must follow semver format for automated checking",
			Severity:    model.Breaking,
		})
		return changes
	}

	// S-04: version decreased (only when base also has a parseable version)
	if baseErr == nil && semverLess(headVer, baseVer) {
		changes = append(changes, model.Change{
			Rule:        "S-04",
			Path:        "version",
			Description: fmt.Sprintf("version decreased from %s to %s", base.Version, head.Version),
			Reason:      "semver versions must never decrease",
			Severity:    model.Breaking,
		})
		return changes // S-03 is redundant when version went backwards
	}

	// S-03: version bump doesn't match the content change level
	// Over-bumping is allowed (e.g. MAJOR bump for MINOR changes); under-bumping is not.
	if baseErr == nil {
		bumpLevel := semverBumpLevel(baseVer, headVer)
		if semverLevelLess(bumpLevel, contentLevel) {
			changes = append(changes, model.Change{
				Rule:  "S-03",
				Path:  "version",
				Description: fmt.Sprintf(
					"version bumped from %s to %s (%s), but changes require a %s bump",
					base.Version, head.Version, bumpLevel, contentLevel,
				),
				Reason:   "version bump must reflect the highest impact change in the schema",
				Severity: model.Breaking,
			})
		}
	}

	return changes
}

// semver holds a parsed MAJOR.MINOR.PATCH version.
type semver struct{ major, minor, patch int }

func parseSemver(s string) (semver, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return semver{}, fmt.Errorf("invalid semver %q", s)
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	patch, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return semver{}, fmt.Errorf("invalid semver %q", s)
	}
	return semver{major, minor, patch}, nil
}

// semverBumpLevel returns the SemverLevel that describes the bump from base to head.
func semverBumpLevel(base, head semver) model.SemverLevel {
	if head.major > base.major {
		return model.LevelMajor
	}
	if head.minor > base.minor {
		return model.LevelMinor
	}
	if head.patch > base.patch {
		return model.LevelPatch
	}
	return model.LevelNone
}

// semverLess reports whether a < b (any component decreased).
func semverLess(a, b semver) bool {
	if a.major != b.major {
		return a.major < b.major
	}
	if a.minor != b.minor {
		return a.minor < b.minor
	}
	return a.patch < b.patch
}

// semverLevelLess reports whether level a is strictly less impactful than b.
// Order: NONE < PATCH < MINOR < MAJOR
func semverLevelLess(a, b model.SemverLevel) bool {
	return levelWeight(a) < levelWeight(b)
}

func levelWeight(l model.SemverLevel) int {
	switch l {
	case model.LevelMajor:
		return 3
	case model.LevelMinor:
		return 2
	case model.LevelPatch:
		return 1
	default:
		return 0
	}
}
