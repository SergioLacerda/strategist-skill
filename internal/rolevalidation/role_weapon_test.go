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
