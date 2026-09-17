package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

// TestCatalogProviderFromIngestedSkill_CarriesUpstreamProvenance is the
// ADR-0029 DEC-002 regression test: a package whose strategist.yaml
// declares upstream provenance must have every field flow through to the
// catalog entry unchanged.
func TestCatalogProviderFromIngestedSkill_CarriesUpstreamProvenance(t *testing.T) {
	t.Parallel()
	skill := IngestedSkill{
		ID: "brainstorming",
		Adapter: externalSkillAdapter{
			CanonicalRole:         "ranger",
			RiskScore:             "write_analysis",
			UpstreamRepo:          "obra/superpowers",
			UpstreamSkillPath:     "skills/brainstorming/SKILL.md",
			UpstreamVersion:       "6.3.0",
			UpstreamCommit:        "b36e0829c6d0140e93cfef2ca599b1b07d4a7797",
			UpstreamContentDigest: "sha256:74edf03ea6d24ef53db48677b93558d14a979bdf052ca3f57ecdca0c66791608",
		},
		Package: domain.PluginPackage{Version: "1.0.0"},
	}

	entry := catalogProviderFromIngestedSkill(skill)
	assert.Equal(t, "obra/superpowers", entry.UpstreamRepo)
	assert.Equal(t, "skills/brainstorming/SKILL.md", entry.UpstreamSkillPath)
	assert.Equal(t, "6.3.0", entry.UpstreamVersion)
	assert.Equal(t, "b36e0829c6d0140e93cfef2ca599b1b07d4a7797", entry.UpstreamCommit)
	assert.Equal(t, "sha256:74edf03ea6d24ef53db48677b93558d14a979bdf052ca3f57ecdca0c66791608", entry.UpstreamContentDigest)
}

// TestCatalogProviderFromIngestedSkill_UpstreamProvenanceEmptyWhenUndeclared
// confirms an unresearched package (e.g. openspec-propose, as of this
// mission) never gets fabricated or borrowed provenance.
func TestCatalogProviderFromIngestedSkill_UpstreamProvenanceEmptyWhenUndeclared(t *testing.T) {
	t.Parallel()
	skill := IngestedSkill{
		ID:      "openspec-propose",
		Adapter: externalSkillAdapter{CanonicalRole: "archivist", RiskScore: "write_analysis"},
		Package: domain.PluginPackage{Version: "1.0"},
	}

	entry := catalogProviderFromIngestedSkill(skill)
	assert.Empty(t, entry.UpstreamRepo)
	assert.Empty(t, entry.UpstreamCommit)
	assert.Empty(t, entry.UpstreamContentDigest)
}

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
