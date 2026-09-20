package rolevalidation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestValidateRuntimeBindingsAcceptsValidExternalBindings(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
  - slot: refinement
    installed_instance_id: openspec-propose
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	require.Empty(t, ValidateRuntimeBindings(root, active))
}

func TestValidateRuntimeBindingsRejectsMissingBinding(t *testing.T) {
	root := writeValidationRoot(t, "")
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	failures := ValidateRuntimeBindings(root, active)
	require.Len(t, failures, 2)
	require.Contains(t, failures[0].Error(), "slot=discovery")
	require.Contains(t, failures[0].Error(), "no persisted weapon binding")
}

func TestValidateRuntimeBindingsRejectsRoleMismatch(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: openspec-propose
  - slot: refinement
    installed_instance_id: openspec-propose
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "openspec-propose", "refinement": "openspec-propose",
	}}
	failures := ValidateRuntimeBindings(root, active)
	require.NotEmpty(t, failures)
	require.Contains(t, failures[0].Error(), "role affinity")
}

func TestValidateSkillManifestRejectsAuxiliaryToolAsMissionProvider(t *testing.T) {
	failures := validateSkillManifest("refinement", "archivist", "writing-plans", []byte("risk_score: write_analysis\ncanonical_role: auxiliary\nroles: [auxiliary]\n"))
	require.Len(t, failures, 1)
	require.Contains(t, failures[0].Error(), "role affinity")
}

func TestValidateRuntimeBindingsRejectsUncertifiedRankedMode(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: ranked
  - slot: refinement
    installed_instance_id: openspec-propose
`)
	writeRankedCatalogFile(t, root, `
schema_version: strategist-plugin-catalog/v1
providers:
  - id: brainstorming
    canonical_role: ranger
    roles: [ranger]
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	failures := ValidateRuntimeBindings(root, active)
	require.NotEmpty(t, failures)
	require.Contains(t, failures[0].Error(), "not a certified ranked candidate")
}

func TestValidateRuntimeBindingsAcceptsCertifiedRankedMode(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: ranked
  - slot: refinement
    installed_instance_id: openspec-propose
`)
	writeRankedCatalogFile(t, root, `
schema_version: strategist-plugin-catalog/v1
providers:
  - id: brainstorming
    canonical_role: ranger
    roles: [ranger]
    ranked: true
    certification_digest: sha256:1111111111111111111111111111111111111111111111111111111111111111
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	require.Empty(t, ValidateRuntimeBindings(root, active))
}

// TestValidateRuntimeBindingsAcceptsCertifiedRankedModeOnRefinementSlot is
// the 20260916-ranked-skills-end-to-end-evaluation Layer 3 coverage-gap
// fix: the refinement-slot counterpart to
// TestValidateRuntimeBindingsAcceptsCertifiedRankedMode, which only ever
// exercised mode: ranked on the discovery binding.
func TestValidateRuntimeBindingsAcceptsCertifiedRankedModeOnRefinementSlot(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
  - slot: refinement
    installed_instance_id: openspec-propose
    mode: ranked
`)
	writeRankedCatalogFile(t, root, `
schema_version: strategist-plugin-catalog/v1
providers:
  - id: openspec-propose
    canonical_role: archivist
    roles: [archivist]
    ranked: true
    certification_digest: sha256:1111111111111111111111111111111111111111111111111111111111111111
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	require.Empty(t, ValidateRuntimeBindings(root, active))
}

func TestValidateRuntimeBindingsRejectsRankedModeMissingFromCatalog(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: ranked
  - slot: refinement
    installed_instance_id: openspec-propose
`)
	writeRankedCatalogFile(t, root, `
schema_version: strategist-plugin-catalog/v1
providers:
  - id: openspec-propose
    canonical_role: archivist
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	failures := ValidateRuntimeBindings(root, active)
	require.NotEmpty(t, failures)
	require.Contains(t, failures[0].Error(), "not found in catalog")
}

func writeRankedCatalogFile(t *testing.T, root, catalogYAML string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte(catalogYAML), 0o644))
}

func TestValidateRuntimeBindingsRejectsInvalidMode(t *testing.T) {
	root := writeValidationRoot(t, `
  - slot: discovery
    installed_instance_id: brainstorming
    mode: typo
  - slot: refinement
    installed_instance_id: openspec-propose
`)
	active := domain.ActiveConfig{Slots: map[string]string{
		"discovery": "brainstorming", "refinement": "openspec-propose",
	}}
	failures := ValidateRuntimeBindings(root, active)
	require.NotEmpty(t, failures)
	require.Contains(t, failures[0].Error(), "invalid mode")
}

func writeValidationRoot(t *testing.T, bindings string) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "roles"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "skills", "brainstorming"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "skills", "openspec-propose"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "default.yaml"), []byte("discovery: ranger\nrefinement: archivist\nexecution: sniper\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "skills", "brainstorming", "skill.yaml"), []byte("risk_score: write_analysis\nroles:\n  - ranger\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "skills", "openspec-propose", "skill.yaml"), []byte("risk_score: write_analysis\nroles:\n  - archivist\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("schema_version: strategist-plugin-lock-file/v1\nbindings:\n"+bindings), 0o644))
	return root
}
