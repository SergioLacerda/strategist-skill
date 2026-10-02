package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeRoleFixture(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, "roles", name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestBuildCompiledRolesFailureModes(t *testing.T) {
	t.Parallel()
	defaults := defaultsExtractor{}
	mapRaw, err := defaults.ReadFile(roleSlotMapPath)
	require.NoError(t, err)

	_, err = buildCompiledRoles(t.TempDir())
	require.ErrorContains(t, err, "read compiled Role map")

	bad := t.TempDir()
	writeRoleFixture(t, bad, "default.yaml", "slots: [unclosed")
	_, err = buildCompiledRoles(bad)
	require.ErrorContains(t, err, "parse compiled Role map")

	incomplete := t.TempDir()
	writeRoleFixture(t, incomplete, "default.yaml", "discovery: ranger\n")
	_, err = buildCompiledRoles(incomplete)
	require.ErrorContains(t, err, "validate compiled Role map")

	noRole := t.TempDir()
	writeRoleFixture(t, noRole, "default.yaml", string(mapRaw))
	_, err = buildCompiledRoles(noRole)
	require.ErrorContains(t, err, "read compiled Role")

	_, err = buildCompiledRegistry(pluginCatalog{}, noRole)
	require.ErrorContains(t, err, "read compiled Role")
}

func TestCompileRoleFailureModes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	_, err := compileRole(root, "discovery", "ranger")
	require.ErrorContains(t, err, `read compiled Role "ranger"`)

	writeRoleFixture(t, root, "ranger.yaml", "role: [unclosed")
	_, err = compileRole(root, "discovery", "ranger")
	require.ErrorContains(t, err, `parse compiled Role "ranger"`)

	writeRoleFixture(t, root, "ranger.yaml", "{}\n")
	_, err = compileRole(root, "discovery", "ranger")
	require.ErrorContains(t, err, `validate compiled Role "ranger"`)

	raw, readErr := defaultsExtractor{}.ReadFile("roles/ranger.yaml")
	require.NoError(t, readErr)
	writeRoleFixture(t, root, "ranger.yaml", string(raw))
	role, err := compileRole(root, "discovery", "ranger")
	require.NoError(t, err)
	assert.Equal(t, "ranger", role.ID)
	assert.Contains(t, role.ContractDigest, "sha256:")
}
