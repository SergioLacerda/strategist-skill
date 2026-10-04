package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	strategistembed "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMissionInvocationStateFailureLadder(t *testing.T) {
	root := t.TempDir()
	_, _, _, err := loadMissionInvocationState(root)
	require.ErrorContains(t, err, "read active.yaml")

	active := filepath.Join(root, "active.yaml")
	require.NoError(t, os.WriteFile(active, []byte("slots: [unclosed"), 0o644))
	_, _, _, err = loadMissionInvocationState(root)
	require.ErrorContains(t, err, "parse active.yaml")

	require.NoError(t, os.WriteFile(active, []byte("mode: epic\nbase_path: .analysis\n"), 0o644))
	_, _, _, err = loadMissionInvocationState(root)
	require.ErrorContains(t, err, "read plugins.lock")

	lock := filepath.Join(root, "plugins.lock")
	require.NoError(t, os.WriteFile(lock, []byte("bindings: [unclosed"), 0o644))
	_, _, _, err = loadMissionInvocationState(root)
	require.ErrorContains(t, err, "parse plugins.lock")

	require.NoError(t, os.WriteFile(lock, []byte("schema_version: x\n"), 0o644))
	_, _, _, err = loadMissionInvocationState(root)
	require.ErrorContains(t, err, "read compiled catalog")

	catalog := filepath.Join(root, "plugins", "catalog.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(catalog), 0o755))
	require.NoError(t, os.WriteFile(catalog, []byte("providers: [unclosed"), 0o644))
	_, _, _, err = loadMissionInvocationState(root)
	require.ErrorContains(t, err, "parse compiled catalog")

	embedded, readErr := (strategistembed.Extractor{}).ReadFile("plugins/catalog.yaml")
	require.NoError(t, readErr)
	require.NoError(t, os.WriteFile(catalog, embedded, 0o644))
	_, _, registry, err := loadMissionInvocationState(root)
	require.NoError(t, err)
	assert.NotEmpty(t, registry.Weapons)

	require.ErrorContains(t, requireRegistryMatchesBinary(domain.CompiledRegistry{}), "compiled_registry_drift")
}

func TestRequireNoExistingMissionAndADRPath(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, requireNoExistingMission(root, "fresh"))
	require.NoError(t, saveMission(root, domain.MissionEngineStatus{MissionID: "taken", Phase: domain.PhaseBootstrap, State: domain.StateInit}))
	require.ErrorContains(t, requireNoExistingMission(root, "taken"), "already exists")

	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, nil, 0o644))
	if err := requireNoExistingMission(file, "m"); err != nil {
		assert.Contains(t, err.Error(), "inspect existing state")
	}

	_, err := adrCanonicalPath(t.TempDir())
	require.ErrorContains(t, err, "read adr.canonical_path")
	cfgRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(cfgRoot, "active.yaml"), []byte("base_path: .analysis\nadr:\n  canonical_path: docs/adr\n"), 0o644))
	path, err := adrCanonicalPath(cfgRoot)
	require.NoError(t, err)
	assert.Equal(t, "docs/adr", path)
}
