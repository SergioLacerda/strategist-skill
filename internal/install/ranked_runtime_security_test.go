package install

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func requireSymlinkSupport(t *testing.T) {
	t.Helper()
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink tests require Windows Developer Mode or symlink privilege: %v", err)
	}
}

func TestPrepareRankedBindingRejectsRuntimeSymlinkEscape(t *testing.T) {
	requireSymlinkSupport(t)
	dir := t.TempDir()
	strategist := filepath.Join(dir, ".strategist")
	writeRankedRuntimeFixture(t, dir, "openspec-propose", "refinement", domain.WeaponRuntime{
		Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec",
		Bootstrap: "openspec init", Healthcheck: "openspec context --json",
	})
	external := t.TempDir()
	require.NoError(t, os.Symlink(external, filepath.Join(strategist, "openspec")))

	roles, catalog, err := loadRankedRuntimeInputs(strategist)
	require.NoError(t, err)
	bindings, err := loadRankedBindings(strategist)
	require.NoError(t, err)
	_, _, err = prepareRankedBinding(context.Background(), strategist, roles, catalog, bindings[0])
	require.Error(t, err)
	require.Contains(t, err.Error(), "symlink outside root")
	_, statErr := os.Stat(filepath.Join(external, "config.yaml"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
}
