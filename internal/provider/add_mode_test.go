package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

// stockRankedWorkspace is a runtime whose lock carries the ranked discovery and
// refinement bindings a stock install writes.
func stockRankedWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, copySource(filepath.Join("..", "embed", "defaults"), root))
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: epic\nbase_path: .analysis\nknowledge_index_path: knowledge.index.yaml\nslots:\n  discovery: brainstorming\n  refinement: openspec-propose\n  execution: sniper\n"), 0o644))
	stock := domain.PluginLockFile{
		Inventory: domain.PluginInventory{SchemaVersion: "strategist-plugin-inventory/v1", Instances: []domain.InstalledInstance{
			{ID: "brainstorming", State: "active", LastKnownGood: true},
			{ID: "openspec-propose", State: "active", LastKnownGood: true},
		}},
		Bindings: []domain.SlotBinding{
			{SchemaVersion: "strategist-plugin-binding/v1", Slot: "discovery", InstalledInstanceID: "brainstorming", GrantID: "ranked-grant", Generation: 1, Status: "active", Mode: domain.SlotBindingModeRanked},
			{SchemaVersion: "strategist-plugin-binding/v1", Slot: "refinement", InstalledInstanceID: "openspec-propose", GrantID: "ranked-grant", Generation: 1, Status: "active", Mode: domain.SlotBindingModeRanked},
		},
	}
	require.NoError(t, writeLock(root, stock))
	return root
}

func bindingOf(t *testing.T, root, slot string) domain.SlotBinding {
	t.Helper()
	lock, exists, err := readLock(root)
	require.NoError(t, err)
	require.True(t, exists)
	binding, ok := existingBinding(lock.Bindings, slot)
	require.True(t, ok, "binding for slot %s", slot)
	return binding
}

func TestAddOverARankedSlotYieldsACustomBinding(t *testing.T) {
	root := stockRankedWorkspace(t)
	untouched := bindingOf(t, root, "discovery")

	result, err := Add(root, analysisFixturePath(t), "refinement")

	require.NoError(t, err)
	binding := bindingOf(t, root, "refinement")
	require.Equal(t, domain.SlotBindingModeCustom, binding.Mode, "a package added with provider add is a custom binding, never ranked")
	require.Equal(t, "fixture-provider@1.0.0", binding.InstalledInstanceID)
	require.Equal(t, int64(2), binding.Generation)
	require.Equal(t, result.BindingGeneration, binding.Generation)
	require.Equal(t, untouched, bindingOf(t, root, "discovery"), "an unrelated slot binding is unchanged")
}

func TestAddOverARankedSlotCarriesNoRankedOnlyField(t *testing.T) {
	root := stockRankedWorkspace(t)
	_, err := Add(root, analysisFixturePath(t), "refinement")
	require.NoError(t, err)
	over := bindingOf(t, root, "refinement")

	fresh := stockRankedWorkspace(t)
	require.NoError(t, os.WriteFile(filepath.Join(fresh, "plugins.lock"), []byte("schema_version: strategist-plugin-lock-file/v1\n"), 0o644))
	_, err = Add(fresh, analysisFixturePath(t), "refinement")
	require.NoError(t, err)
	created := bindingOf(t, fresh, "refinement")

	created.Generation = over.Generation
	require.Equal(t, created, over, "an over-ranked add matches a fresh custom binding field by field")
}

func TestAddNewerVersionKeepsTheBindingCustom(t *testing.T) {
	root := stockRankedWorkspace(t)
	_, err := Add(root, analysisFixturePath(t), "refinement")
	require.NoError(t, err)
	newer := analysisFixturePath(t)
	replaceFixtureText(t, filepath.Join(newer, packageManifestName), "1.0.0", "1.1.0")

	_, err = Add(root, newer, "refinement")

	require.NoError(t, err)
	binding := bindingOf(t, root, "refinement")
	require.Equal(t, domain.SlotBindingModeCustom, binding.Mode)
	require.Equal(t, "fixture-provider@1.1.0", binding.InstalledInstanceID)
	require.Equal(t, int64(3), binding.Generation)
}

func TestAddHealsALockWrittenByTheOldRankedModeBug(t *testing.T) {
	root := stockRankedWorkspace(t)
	buggy, _, err := readLock(root)
	require.NoError(t, err)
	buggy.Bindings[1] = domain.SlotBinding{SchemaVersion: "strategist-plugin-binding/v1", Slot: "refinement", InstalledInstanceID: "fixture-provider@1.0.0", Generation: 2, Status: "active", Mode: domain.SlotBindingModeRanked}
	buggy.Inventory.Instances = append(buggy.Inventory.Instances, domain.InstalledInstance{ID: "fixture-provider@1.0.0", State: "active"})
	require.NoError(t, writeLock(root, buggy))

	_, err = Add(root, analysisFixturePath(t), "refinement")

	require.NoError(t, err, "re-adding the same package over a buggy lock is the documented remedy")
	require.Equal(t, domain.SlotBindingModeCustom, bindingOf(t, root, "refinement").Mode)
}
