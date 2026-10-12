package rankedbinding

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/stretchr/testify/require"
)

func TestPrepareRankedBindingRejectsRuntimeSymlinkEscape(t *testing.T) {
	installtest.RequireSymlinkSupport(t)
	dir := t.TempDir()
	strategist := filepath.Join(dir, ".strategist")
	installtest.WriteRankedRuntimeFixture(t, dir, "openspec-propose", "refinement", domain.WeaponRuntime{
		Kind: domain.RankedRuntimeOpenSpecRoot, Root: ".strategist/openspec",
		Bootstrap: "openspec init", Healthcheck: "openspec context --json",
	})
	external := t.TempDir()
	require.NoError(t, os.Symlink(external, filepath.Join(strategist, "openspec")))

	roles, catalog, err := LoadRankedRuntimeInputs(strategist)
	require.NoError(t, err)
	bindings, err := LoadRankedBindings(strategist)
	require.NoError(t, err)
	_, _, err = PrepareRankedBinding(context.Background(), strategist, roles, catalog, bindings[0])
	require.Error(t, err)
	require.Contains(t, err.Error(), "symlink outside root")
	_, statErr := os.Stat(filepath.Join(external, "config.yaml"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
}
