package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateActiveYAML_InvalidMode(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: experimental
base_path: .analysis
slots:
  discovery: brainstorming
`), 0o644))
	err := ActiveYAML(filepath.Join(dir, "active.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mode")
}

func TestValidatePersonasDir_MissingDir(t *testing.T) {
	dir := t.TempDir()
	errs, checks := PersonasDir(filepath.Join(dir, "missing-personas"))
	assert.Zero(t, checks)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "personas/")
}

func TestValidateRolesDir_MissingDir(t *testing.T) {
	dir := t.TempDir()
	errs, checks := RolesDir(filepath.Join(dir, "missing-roles"))
	assert.Zero(t, checks)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "roles/")
}

func TestValidateYAMLFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte("not: [yaml"), 0o644))
	err := YAMLFile(filepath.Join(dir, "bad.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid YAML")
}

func TestValidateRoleFile_RoleDefinitionAndSlotMapErrors(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "role.yaml"), []byte(`
role: scout
specialization:
  slot: invalid
`), 0o644))
	errs := validateRoleFile(dir, "role.yaml")
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "roles/role.yaml")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "slots.yaml"), []byte(`
discovery: scout
`), 0o644))
	errs = validateRoleFile(dir, "slots.yaml")
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "roles/slots.yaml")
}
