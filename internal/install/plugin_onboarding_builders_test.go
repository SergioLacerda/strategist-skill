package install

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInventoryFromLock_PackageKindCarriesPackageDigest(t *testing.T) {
	t.Parallel()

	lock := domain.PluginLock{
		GraphDigest: "sha256:test",
		Nodes: []domain.PluginLockNode{
			{ID: "sample", Kind: "package", Digest: "sha256:package-digest"},
		},
	}
	instances := inventoryFromLock(lock)
	require.Len(t, instances, 1)
	assert.Equal(t, "sha256:package-digest", instances[0].PackageDigest)
	assert.Empty(t, instances[0].AdapterDigest)
}

func TestBindingsFromSlots_SetsCustomMode(t *testing.T) {
	t.Parallel()

	lock := domain.PluginLock{
		Nodes: []domain.PluginLockNode{
			{ID: "brainstorming", Kind: "adapter_contract", Digest: "sha256:test"},
		},
	}
	bindings, err := bindingsFromSlots(map[string]string{"discovery": "brainstorming"}, lock)
	require.NoError(t, err)
	require.Len(t, bindings, 1)
	assert.Equal(t, domain.SlotBindingModeCustom, bindings[0].Mode)
}

func TestBindingsFromSlots_UnresolvedBindingErrorPropagates(t *testing.T) {
	t.Parallel()

	_, err := bindingsFromSlots(map[string]string{"discovery": "brainstorming"}, domain.PluginLock{})
	require.Error(t, err)
	require.ErrorContains(t, err, "unresolved_binding: discovery provider brainstorming")
}
