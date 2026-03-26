package model

// Severity describes the compatibility impact of a single change.
type Severity string

const (
	Breaking Severity = "BREAKING" // consumers will fail to read existing data
	Safe     Severity = "SAFE"     // backward and forward compatible
	Cosmetic Severity = "COSMETIC" // doc or default only — no compatibility impact
)

// CompatMode controls which direction of compatibility is checked.
type CompatMode string

const (
	ModeBackward CompatMode = "BACKWARD" // new schema can read data written by old schema
	ModeForward  CompatMode = "FORWARD"  // old schema can read data written by new schema
	ModeFull     CompatMode = "FULL"     // both directions
	ModeNone     CompatMode = "NONE"     // no checks, report all changes as-is
)

// Change represents a single detected difference between two schemas.
type Change struct {
	Path          string       `json:"path"`        // dot-separated field path, e.g. "fields.address.fields.city"
	Description   string       `json:"description"` // human-readable description of the change
	Reason        string       `json:"reason"`      // why this change has the given severity
	Severity      Severity     `json:"severity"`
	AffectedModes []CompatMode `json:"affected_modes,omitempty"` // modes in which this change is breaking
}

// SemverLevel is the aggregate compatibility impact of all changes in a DiffResult.
type SemverLevel string

const (
	LevelMajor SemverLevel = "MAJOR" // breaking changes present
	LevelMinor SemverLevel = "MINOR" // backward compatible additions only
	LevelPatch SemverLevel = "PATCH" // cosmetic changes only (doc, defaults)
	LevelNone  SemverLevel = "NONE"  // no changes
)

// DiffResult is the output of comparing two Avro schemas.
type DiffResult struct {
	Changes []Change
	Level   SemverLevel
	Mode    CompatMode
}
