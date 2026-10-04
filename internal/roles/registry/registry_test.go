package roles_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/roles/registry"
	"github.com/stretchr/testify/require"
)

func writeRole(t *testing.T, dir, name, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600))
}

func TestLoadRoleRegistryUsesBuiltInsWhenDirectoryIsAbsent(t *testing.T) {
	registry, err := roles.LoadRoleRegistry(filepath.Join(t.TempDir(), "missing"))
	require.NoError(t, err)
	assertRolePresent(t, registry, "ranger")
}

func TestLoadRoleRegistryReadsAndOverlaysRoleFiles(t *testing.T) {
	dir := t.TempDir()
	writeRole(t, dir, "ranger.yaml", "role: ranger\nslot: discovery\nphase: 1\nextensibility: pluggable\n")
	writeRole(t, dir, "auditor.yaml", "role: auditor\nphase: 5\nextensibility: fixed\n")
	writeRole(t, dir, "default.yaml", "not a role file\n")

	registry, err := roles.LoadRoleRegistry(dir)
	require.NoError(t, err)
	assertRolePresent(t, registry, "auditor")
	assertRolePresent(t, registry, "ranger")
}

func TestLoadRoleRegistryRejectsMalformedAndInvalidFiles(t *testing.T) {
	dir := t.TempDir()
	writeRole(t, dir, "broken.yaml", ":\n - [broken")
	_, err := roles.LoadRoleRegistry(dir)
	require.ErrorContains(t, err, "broken.yaml")

	dir = t.TempDir()
	writeRole(t, dir, "missing-role.yaml", "slot: discovery\n")
	_, err = roles.LoadRoleRegistry(dir)
	require.ErrorContains(t, err, "role is required")
}

func TestLoadRoleRegistryPropagatesDirectoryReadError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-directory")
	writeRole(t, filepath.Dir(file), filepath.Base(file), "x")
	_, err := roles.LoadRoleRegistry(file)
	require.Error(t, err)
}

func assertRolePresent(t *testing.T, registry interface{ Has(string) bool }, role string) {
	t.Helper()
	require.True(t, registry.Has(role), role)
}
