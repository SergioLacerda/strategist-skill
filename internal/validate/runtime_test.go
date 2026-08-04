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

func TestValidateActiveYAML_Success(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sniper
`), 0o644))
	require.NoError(t, ActiveYAML(filepath.Join(dir, "active.yaml")))
}

func TestValidateActiveYAML_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	err := ActiveYAML(filepath.Join(dir, "missing.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestValidateActiveYAML_MissingMode(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
base_path: .analysis
slots:
  discovery: brainstorming
`), 0o644))
	err := ActiveYAML(filepath.Join(dir, "active.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field: mode")
}

func TestValidateActiveYAML_MissingBasePath(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
slots:
  discovery: brainstorming
`), 0o644))
	err := ActiveYAML(filepath.Join(dir, "active.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field: base_path")
}

func TestValidateActiveYAML_MissingSlots(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
`), 0o644))
	err := ActiveYAML(filepath.Join(dir, "active.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field: slots")
}

func TestValidatePersonasDir_MissingDir(t *testing.T) {
	dir := t.TempDir()
	errs, checks := PersonasDir(filepath.Join(dir, "missing-personas"))
	assert.Zero(t, checks)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "personas/")
}

func TestValidatePersonasDir_Success(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "epic.yaml"), []byte(`
id: epic
tone_directive: heroic
phase_labels:
  discovery: Ranger
  refinement: Archivist
  execution: Sniper
diagnostics:
  format: raw
  pipeline_header: "STRATEGIST"
  bootstrap_origin: standard_path
`), 0o644))
	errs, checks := PersonasDir(dir)
	assert.Equal(t, 1, checks)
	assert.Empty(t, errs)
}

func TestValidatePersonasDir_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte("not: [yaml"), 0o644))
	errs, checks := PersonasDir(dir)
	assert.Equal(t, 1, checks)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "invalid YAML")
}

func TestValidatePersonasDir_ValidateForRuntimeFailure(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "incomplete.yaml"), []byte(`
id: incomplete
`), 0o644))
	errs, checks := PersonasDir(dir)
	assert.Equal(t, 1, checks)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "personas/incomplete.yaml")
}

func TestValidateRolesDir_MissingDir(t *testing.T) {
	dir := t.TempDir()
	errs, checks := RolesDir(filepath.Join(dir, "missing-roles"))
	assert.Zero(t, checks)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "roles/")
}

func TestValidateRolesDir_Success(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ranger.yaml"), []byte(`
role: ranger
slot: discovery
`), 0o644))
	errs, checks := RolesDir(dir)
	assert.Equal(t, 1, checks)
	assert.Empty(t, errs)
}

func TestValidateRoleFile_ValidRoleDefinition(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sniper.yaml"), []byte(`
role: sniper
slot: execution
`), 0o644))
	errs := validateRoleFile(dir, "sniper.yaml")
	assert.Empty(t, errs)
}

func TestValidateRoleFile_ValidSlotMap(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "slots.yaml"), []byte(`
discovery: brainstorming
refinement: openspec-explore
execution: sniper
`), 0o644))
	errs := validateRoleFile(dir, "slots.yaml")
	assert.Empty(t, errs)
}

func TestValidateRoleFile_ReadError(t *testing.T) {
	dir := t.TempDir()
	errs := validateRoleFile(dir, "missing.yaml")
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "roles/missing.yaml")
}

func TestValidateYAMLFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte("not: [yaml"), 0o644))
	err := YAMLFile(filepath.Join(dir, "bad.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid YAML")
}

func TestValidateYAMLFile_Valid(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "good.yaml"), []byte("key: value\n"), 0o644))
	require.NoError(t, YAMLFile(filepath.Join(dir, "good.yaml")))
}

func TestValidateYAMLFile_ReadError(t *testing.T) {
	dir := t.TempDir()
	err := YAMLFile(filepath.Join(dir, "missing.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read:")
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
