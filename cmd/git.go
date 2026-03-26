package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nikitagold/avrodiff/model"
)

// readSchema reads an Avro schema from either a file path or a git ref.
// Git refs use the "ref:path" format, e.g. "main:avro/user.avsc" or "HEAD~1:avro/user.avsc".
// Any string containing ":" is treated as a git ref; everything else is read from disk.
func readSchema(pathOrRef string) (*model.Schema, error) {
	if isGitRef(pathOrRef) {
		return readSchemaFromGit(pathOrRef)
	}

	return model.ReadSchema(pathOrRef)
}

// isGitRef reports whether s looks like a git ref (contains ":").
// Examples: "main:avro/user.avsc", "HEAD~1:schemas/event.avsc", "abc123:file.avsc".
func isGitRef(s string) bool {
	return strings.Contains(s, ":")
}

// readSchemaFromGit runs "git show <refPath>" and parses the resulting JSON.
// refPath must be in "ref:path" format understood by git show.
func readSchemaFromGit(refPath string) (*model.Schema, error) {
	out, err := exec.Command("git", "show", refPath).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("git show %s: %s", refPath, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git show %s: %w", refPath, err)
	}
	return model.ParseSchema(out)
}
