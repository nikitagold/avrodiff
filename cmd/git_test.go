package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// mustGit runs a git command in dir, failing the test on error.
func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// initRepo creates a minimal git repo in dir with a given file committed.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, dir, "init")
	mustGit(t, dir, "config", "user.email", "test@avrodiff.test")
	mustGit(t, dir, "config", "user.name", "avrodiff test")
	return dir
}

func TestIsGitRef(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"main:avro/user.avsc", true},
		{"HEAD~1:schemas/event.avsc", true},
		{"abc123def:file.avsc", true},
		{"origin/main:avro/user.avsc", true},
		{"/path/to/file.avsc", false},
		{"./avro/user.avsc", false},
		{"user.avsc", false},
	}
	for _, c := range cases {
		if got := isGitRef(c.input); got != c.want {
			t.Errorf("isGitRef(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func TestReadSchemaFromGit(t *testing.T) {
	dir := initRepo(t)

	schema := `{"type":"record","name":"User","fields":[{"name":"id","type":"string"}]}`
	if err := os.WriteFile(filepath.Join(dir, "user.avsc"), []byte(schema), 0644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", "user.avsc")
	mustGit(t, dir, "commit", "-m", "add schema")

	// Run from inside the repo
	oldDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	s, err := readSchemaFromGit("HEAD:user.avsc")
	if err != nil {
		t.Fatalf("readSchemaFromGit: %v", err)
	}
	if s.Name != "User" {
		t.Errorf("schema name: got %q, want %q", s.Name, "User")
	}
	if len(s.Fields) != 1 || s.Fields[0].Name != "id" {
		t.Errorf("unexpected fields: %v", s.Fields)
	}
}

func TestReadSchemaFromGit_CommitHash(t *testing.T) {
	dir := initRepo(t)

	v1 := `{"type":"record","name":"User","fields":[{"name":"id","type":"string"}]}`
	v2 := `{"type":"record","name":"User","fields":[{"name":"id","type":"string"},{"name":"email","type":["null","string"],"default":null}]}`

	writeAndCommit := func(content, msg string) string {
		os.WriteFile(filepath.Join(dir, "user.avsc"), []byte(content), 0644)
		mustGit(t, dir, "add", "user.avsc")
		mustGit(t, dir, "commit", "-m", msg)
		// Return the commit hash
		out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
		if err != nil {
			t.Fatal(err)
		}
		return string(out[:40]) // first 40 chars = full hash
	}

	firstCommit := writeAndCommit(v1, "v1")
	writeAndCommit(v2, "v2")

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	// Read v1 via its commit hash
	s, err := readSchemaFromGit(firstCommit + ":user.avsc")
	if err != nil {
		t.Fatalf("readSchemaFromGit: %v", err)
	}
	if len(s.Fields) != 1 {
		t.Errorf("expected 1 field in v1, got %d: %v", len(s.Fields), s.Fields)
	}

	// Read v2 via HEAD
	s2, err := readSchemaFromGit("HEAD:user.avsc")
	if err != nil {
		t.Fatalf("readSchemaFromGit HEAD: %v", err)
	}
	if len(s2.Fields) != 2 {
		t.Errorf("expected 2 fields in v2 (HEAD), got %d: %v", len(s2.Fields), s2.Fields)
	}
}

func TestReadSchemaFromGit_NotFound(t *testing.T) {
	dir := initRepo(t)
	mustGit(t, dir, "commit", "--allow-empty", "-m", "empty")

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	_, err := readSchemaFromGit("HEAD:nonexistent.avsc")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestReadSchema_Dispatch(t *testing.T) {
	dir := initRepo(t)

	schema := `{"type":"record","name":"Event","fields":[]}`
	schemaPath := filepath.Join(dir, "event.avsc")
	os.WriteFile(schemaPath, []byte(schema), 0644)
	mustGit(t, dir, "add", "event.avsc")
	mustGit(t, dir, "commit", "-m", "add event")

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	// File path → reads from disk
	s1, err := readSchema("event.avsc")
	if err != nil {
		t.Fatalf("readSchema file: %v", err)
	}
	if s1.Name != "Event" {
		t.Errorf("file: got %q", s1.Name)
	}

	// Git ref → reads from git
	s2, err := readSchema("HEAD:event.avsc")
	if err != nil {
		t.Fatalf("readSchema git ref: %v", err)
	}
	if s2.Name != "Event" {
		t.Errorf("git ref: got %q", s2.Name)
	}
}
