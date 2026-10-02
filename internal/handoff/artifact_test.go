package handoff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateRangerArtifactRequiresNormalizedSections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	content := `---
schema_version: strategist-ranger-discovery/v1
mission_id: m-1
mission_status: ranger_done
sources_consulted: []
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

func TestValidateRangerArtifactForRefinementAcceptsArchivistPending(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	content := `---
schema_version: strategist-ranger-discovery/v1
mission_id: m-1
mission_status: archivist_pending
sources_consulted: []
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
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	require.NoError(t, ValidateRangerArtifactForRefinement(path, "m-1"))
}

func TestValidateRangerArtifactForRefinementRejectsPostRefinementStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	content := "---\nschema_version: strategist-ranger-discovery/v1\nmission_id: m-1\nmission_status: gate_pending\nsources_consulted: []\n---\n\n## mission_objective\n## known_facts\n## confidence_summary\n## handoff\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	require.ErrorContains(t, ValidateRangerArtifactForRefinement(path, "m-1"), "mission_status")
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

func TestValidateRangerArtifactRejectsMalformedFrontmatter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	require.NoError(t, os.WriteFile(path, []byte("not frontmatter\n"), 0o600))
	require.ErrorContains(t, ValidateRangerArtifact(path, "m-1"), "frontmatter is missing")
}

func TestValidateRangerArtifactRejectsMissingSourcesList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	content := "---\nmission_id: m-1\nmission_status: ranger_done\n---\n\n## mission_objective\nobjective\n## known_facts\nfacts\n## confidence_summary\nsummary\n## handoff\nhandoff\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	require.ErrorContains(t, ValidateRangerArtifact(path, "m-1"), "sources_consulted")
}

func TestValidateRangerArtifactRejectsMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.md")
	err := ValidateRangerArtifact(path, "m-1")
	require.ErrorContains(t, err, "read Ranger artifact")
}

func TestValidateRangerArtifactRejectsMismatchedMission(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	content := "---\nmission_id: m-2\nmission_status: ranger_done\n---\n\n## mission_objective\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	err := ValidateRangerArtifact(path, "m-1")
	require.ErrorContains(t, err, "mission_id")
}

func TestHasHandoffMetadata(t *testing.T) {
	t.Run("frontmatter present", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "artifact.md")
		require.NoError(t, os.WriteFile(path, []byte("---\nmission_id: m-1\n---\nbody\n"), 0o600))
		require.True(t, HasHandoffMetadata(path))
	})
	t.Run("no frontmatter", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "artifact.md")
		require.NoError(t, os.WriteFile(path, []byte("# plain markdown\n"), 0o600))
		require.False(t, HasHandoffMetadata(path))
	})
	t.Run("missing file", func(t *testing.T) {
		require.False(t, HasHandoffMetadata(filepath.Join(t.TempDir(), "missing.md")))
	})
}

func TestHasNormalizedRangerMetadata(t *testing.T) {
	t.Run("normalized envelope", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "artifact.md")
		require.NoError(t, os.WriteFile(path, []byte("---\nschema_version: strategist-ranger-discovery/v1\n---\nbody\n"), 0o600))
		require.True(t, HasNormalizedRangerMetadata(path))
	})
	t.Run("generic frontmatter without schema marker", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "artifact.md")
		require.NoError(t, os.WriteFile(path, []byte("---\nmission_id: m-1\n---\nbody\n"), 0o600))
		require.False(t, HasNormalizedRangerMetadata(path))
	})
	t.Run("no frontmatter", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "artifact.md")
		require.NoError(t, os.WriteFile(path, []byte("# plain markdown\n"), 0o600))
		require.False(t, HasNormalizedRangerMetadata(path))
	})
	t.Run("missing file", func(t *testing.T) {
		require.False(t, HasNormalizedRangerMetadata(filepath.Join(t.TempDir(), "missing.md")))
	})
}

func TestValidateArchivistPackageRejectsMissingAnalysis(t *testing.T) {
	dir := t.TempDir()
	err := ValidateArchivistPackage(dir, "m-1")
	require.ErrorContains(t, err, "read Archivist analysis")
}

func TestValidateArchivistPackageRejectsMalformedAnalysis(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte("not frontmatter\n"), 0o600))
	require.ErrorContains(t, ValidateArchivistPackage(dir, "m-1"), "frontmatter is missing")
}

func TestValidateArchivistPackageRejectsInvalidAnalysisStatus(t *testing.T) {
	dir := t.TempDir()
	analysis := "---\nmission_id: m-1\nmission_status: ranger_done\n---\n\n# Analysis\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(analysis), 0o600))
	require.ErrorContains(t, ValidateArchivistPackage(dir, "m-1"), "mission_status")
}

func TestValidateArchivistPackageRejectsMissingSupportingFile(t *testing.T) {
	dir := t.TempDir()
	analysis := "---\nmission_id: m-1\nmission_status: archivist_done\n---\n\n# Analysis\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(analysis), 0o600))
	// proposal.md is intentionally absent.
	err := ValidateArchivistPackage(dir, "m-1")
	require.ErrorContains(t, err, "read Archivist proposal.md")
}

func TestValidateArchivistPackageRejectsEmptySupportingFile(t *testing.T) {
	dir := t.TempDir()
	analysis := "---\nmission_id: m-1\nmission_status: archivist_done\n---\n\n# Analysis\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(analysis), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("   \n"), 0o600))
	err := ValidateArchivistPackage(dir, "m-1")
	require.ErrorContains(t, err, "proposal.md")
}

func TestValidateArchivistPackageRejectsEmptyLaterSupportingFile(t *testing.T) {
	dir := t.TempDir()
	analysis := "---\nmission_id: m-1\nmission_status: archivist_done\n---\n\n# Analysis\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(analysis), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("proposal\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "design.md"), []byte("\n"), 0o600))
	require.ErrorContains(t, ValidateArchivistPackage(dir, "m-1"), "artifact is empty")
}

func TestValidateArchivistPackageRejectsMissingTasks(t *testing.T) {
	dir := t.TempDir()
	analysis := "---\nmission_id: m-1\nmission_status: archivist_done\n---\n\n# Analysis\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "analysis.md"), []byte(analysis), 0o600))
	for _, name := range []string{"proposal.md", "design.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("# "+name), 0o600))
	}
	// tasks.md is intentionally absent.
	err := ValidateArchivistPackage(dir, "m-1")
	require.ErrorContains(t, err, "read Archivist tasks")
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

func TestValidateRangerArtifactRequiresSourcesConsulted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	content := "---\nmission_id: m-1\nmission_status: ranger_done\n---\n\n## mission_objective\n## known_facts\n## confidence_summary\n## handoff\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	require.ErrorContains(t, ValidateRangerArtifact(path, "m-1"), "sources_consulted")
}
