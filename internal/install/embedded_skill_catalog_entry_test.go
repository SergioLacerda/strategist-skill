package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillDescription_MissingFileReturnsEmpty(t *testing.T) {
	t.Parallel()
	if got := skillDescription(t.TempDir()); got != "" {
		t.Fatalf("expected empty description for missing SKILL.md, got %q", got)
	}
}

func TestSkillDescription_NoFrontmatterDelimiterReturnsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# no frontmatter here"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if got := skillDescription(dir); got != "" {
		t.Fatalf("expected empty description, got %q", got)
	}
}

func TestSkillDescription_UnterminatedFrontmatterReturnsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "---\ndescription: unterminated\nno closing delimiter"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if got := skillDescription(dir); got != "" {
		t.Fatalf("expected empty description, got %q", got)
	}
}

func TestSkillDescription_MalformedYAMLReturnsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "---\ndescription: [unclosed\n---\nbody"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if got := skillDescription(dir); got != "" {
		t.Fatalf("expected empty description for malformed yaml, got %q", got)
	}
}

func TestSkillDescription_ValidFrontmatterReturnsDescription(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "---\ndescription: does the thing\n---\nbody"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if got := skillDescription(dir); got != "does the thing" {
		t.Fatalf("skillDescription = %q, want %q", got, "does the thing")
	}
}
