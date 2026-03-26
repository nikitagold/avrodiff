package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/nikitagold/avrodiff/model"
)

func PrintText(w io.Writer, schemaName string, result model.DiffResult) {
	if result.Mode != "" {
		_, _ = fmt.Fprintf(w, "%s  [%s]\n\n", schemaName, result.Mode)
	} else {
		_, _ = fmt.Fprintf(w, "%s\n\n", schemaName)
	}

	if len(result.Changes) == 0 {
		_, _ = fmt.Fprintln(w, "  no changes")
	} else {
		for _, c := range result.Changes {
			_, _ = fmt.Fprintf(w, "  %-9s %s\n", c.Severity, c.Description)
			_, _ = fmt.Fprintf(w, "  %9s %s\n", "", c.Reason)
			// Show a hint when the change is SAFE for the current mode but breaking in others
			if c.Severity == model.Safe && len(c.AffectedModes) > 0 {
				modes := make([]string, len(c.AffectedModes))
				for i, m := range c.AffectedModes {
					modes[i] = string(m)
				}
				_, _ = fmt.Fprintf(w, "  %9s (breaking in: %s)\n", "", strings.Join(modes, ", "))
			}
			_, _ = fmt.Fprintln(w)
		}
	}

	_, _ = fmt.Fprintf(w, "Result: %s%s\n", result.Level, levelSuffix(result.Level))
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
		return " (breaking changes present)"
	case model.LevelMinor:
		return " (backward compatible additions)"
	case model.LevelPatch:
		return " (cosmetic changes only: doc, defaults)"
	default:
		return ""
	}
}
