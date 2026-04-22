package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/nikitagold/avrodiff/model"
)

// labelWidth is the column width for the severity label.
// "BREAKING" and "COSMETIC" are 8 characters; +1 gives one space of breathing room.
const labelWidth = 9

// PrintText writes a human-readable diff report.
//
// Example output with changes:
//
//	user.avsc  [BACKWARD]
//
//	  BREAKING  field "email" removed
//	            old schema readers expect this field; without a default they cannot read new data
//
//	  SAFE      field "phone" added (default: nil)
//	            backward and forward compatible
//	            (breaking in: FORWARD, FULL)
//
//	Result: MAJOR — breaking changes present
//
// Example output with no changes:
//
//	user.avsc
//
//	no changes
//	Result: NONE
func PrintText(w io.Writer, schemaName string, result model.DiffResult) {
	if result.Mode != "" {
		_, _ = fmt.Fprintf(w, "%s  [%s]\n\n", schemaName, result.Mode)
	} else {
		_, _ = fmt.Fprintf(w, "%s\n\n", schemaName)
	}

	if len(result.Changes) == 0 {
		_, _ = fmt.Fprintln(w, "no changes")
	} else {
		for _, c := range result.Changes {
			writeChange(w, c)
		}
	}

	suffix := levelSuffix(result.Level)
	if suffix != "" {
		_, _ = fmt.Fprintf(w, "Result: %s — %s\n", result.Level, suffix)
	} else {
		_, _ = fmt.Fprintf(w, "Result: %s\n", result.Level)
	}
}

func writeChange(w io.Writer, c model.Change) {
	pad := strings.Repeat(" ", labelWidth)
	_, _ = fmt.Fprintf(w, "  %-*s %s\n", labelWidth, c.Severity, c.Description)
	_, _ = fmt.Fprintf(w, "  %s %s\n", pad, c.Reason)
	if c.Severity == model.Safe && len(c.AffectedModes) > 0 {
		modes := make([]string, len(c.AffectedModes))
		for i, m := range c.AffectedModes {
			modes[i] = string(m)
		}
		_, _ = fmt.Fprintf(w, "  %s (breaking in: %s)\n", pad, strings.Join(modes, ", "))
	}
	_, _ = fmt.Fprintln(w)
}

func PrintJSON(w io.Writer, schemaName string, result model.DiffResult) error {
	out := struct {
		Schema  string         `json:"schema"`
		Mode    string         `json:"mode,omitempty"`
		Level   string         `json:"level"`
		Changes []model.Change `json:"changes"`
	}{
		Schema:  schemaName,
		Mode:    string(result.Mode),
		Level:   string(result.Level),
		Changes: result.Changes,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func levelSuffix(level model.SemverLevel) string {
	switch level {
	case model.LevelMajor:
		return "breaking changes present"
	case model.LevelMinor:
		return "backward compatible additions"
	case model.LevelPatch:
		return "cosmetic changes only: doc, defaults"
	default:
		return ""
	}
}
