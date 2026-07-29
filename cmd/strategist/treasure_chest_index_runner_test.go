package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanRegisteredChestsForPotions_RunbookDirUnreadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not reliable on windows")
	}
	root := t.TempDir()
	strategistDir := filepath.Join(root, ".strategist")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	runbooksDir := filepath.Join(root, "docs", "runbooks")
	require.NoError(t, os.MkdirAll(runbooksDir, 0o755))
	require.NoError(t, os.Chmod(runbooksDir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(runbooksDir, 0o755) })
	if os.Getuid() == 0 {
		t.Skip("running as root — directory permission checks do not apply")
	}

	governed := map[string]treasure.GovernedChest{
		"runbooks": {ID: "runbooks", Path: "docs/runbooks", Trust: treasure.GovernedTrust{Tier: "T2"}},
	}
	_, err := scanRegisteredChestsForPotions(strategistDir, governed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest index")
}

func TestScanRegisteredChestsForPotions_SkipsNonRunbookChests(t *testing.T) {
	root := t.TempDir()
	governed := map[string]treasure.GovernedChest{
		"source": {ID: "source", Path: ".sdd/source", Trust: treasure.GovernedTrust{Tier: "T1"}},
	}
	candidates, err := scanRegisteredChestsForPotions(root, governed)
	require.NoError(t, err)
	assert.Empty(t, candidates)
}
