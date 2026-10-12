package embeddedweapon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
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
	defaults := installtest.DefaultsExtractor{}
	mapRaw, err := defaults.ReadFile(names.RoleSlotMapPath)
	require.NoError(t, err)

	_, err = BuildCompiledRoles(t.TempDir())
	require.ErrorContains(t, err, "read compiled Role map")

	bad := t.TempDir()
	writeRoleFixture(t, bad, "default.yaml", "slots: [unclosed")
	_, err = BuildCompiledRoles(bad)
	require.ErrorContains(t, err, "parse compiled Role map")

	incomplete := t.TempDir()
	writeRoleFixture(t, incomplete, "default.yaml", "discovery: ranger\n")
	_, err = BuildCompiledRoles(incomplete)
	require.ErrorContains(t, err, "validate compiled Role map")

	noRole := t.TempDir()
	writeRoleFixture(t, noRole, "default.yaml", string(mapRaw))
	_, err = BuildCompiledRoles(noRole)
	require.ErrorContains(t, err, "read compiled Role")

	_, err = BuildCompiledRegistry(plugincatalog.PluginCatalog{}, noRole)
	require.ErrorContains(t, err, "read compiled Role")
}

func TestCompileRoleFailureModes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	_, err := CompileRole(root, "discovery", "ranger")
	require.ErrorContains(t, err, `read compiled Role "ranger"`)

	writeRoleFixture(t, root, "ranger.yaml", "role: [unclosed")
	_, err = CompileRole(root, "discovery", "ranger")
	require.ErrorContains(t, err, `parse compiled Role "ranger"`)

	writeRoleFixture(t, root, "ranger.yaml", "{}\n")
	_, err = CompileRole(root, "discovery", "ranger")
	require.ErrorContains(t, err, `validate compiled Role "ranger"`)

	raw, readErr := installtest.DefaultsExtractor{}.ReadFile("roles/ranger.yaml")
	require.NoError(t, readErr)
	writeRoleFixture(t, root, "ranger.yaml", string(raw))
	role, err := CompileRole(root, "discovery", "ranger")
	require.NoError(t, err)
	assert.Equal(t, "ranger", role.ID)
	assert.Contains(t, role.ContractDigest, "sha256:")
}
