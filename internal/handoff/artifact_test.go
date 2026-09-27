package handoff

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRangerArtifactRequiresNormalizedSections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	content := `---
schema_version: strategist-ranger-discovery/v1
mission_id: m-1
mission_status: ranger_done
---

## mission_objective
objective
## known_facts
facts
## confidence_summary
summary
## handoff
handoff
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRangerArtifact(path, "m-1"); err != nil {
		t.Fatalf("ValidateRangerArtifact: %v", err)
	}
}

func TestValidateRangerArtifactRejectsMissingSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	content := "---\nmission_id: m-1\nmission_status: ranger_done\n---\n\n## mission_objective\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRangerArtifact(path, "m-1"); err == nil {
		t.Fatal("expected missing-section error")
	}
}

func TestValidateArchivistPackageRequiresClassifiedTasks(t *testing.T) {
	dir := t.TempDir()
	analysis := "---\nmission_id: m-1\nmission_status: archivist_done\n---\n\n# Analysis\n"
	if err := os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(analysis), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"proposal.md", "design.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("# "+name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("- [ ] 1.1 [implementation_handoff] change source\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateArchivistPackage(dir, "m-1"); err != nil {
		t.Fatalf("ValidateArchivistPackage: %v", err)
	}
}
