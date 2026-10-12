package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JEWELER owns the non-pipeline `treasure` support slot, which is
// intentionally absent from roles/default.yaml (that file maps only the three
// mission checkpoints). Its binding must verify without a default.yaml entry.
func TestVerifyEmbeddedWeaponBindings_SupportSlotNeedsNoDefaultMapEntry(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeDefaultRoleSlotMap(t, root) // discovery, refinement, execution only
	writeFullRoster(t, root)
	writeWeaponFixture(t, root, "treasure-chest", "jeweler", "role: jeweler\nslot: treasure\norigin: external\n")

	bindings, err := verifyEmbeddedWeaponBindings(root)
	require.NoError(t, err)
	b, found := findBinding(bindings, "treasure-chest")
	require.True(t, found)
	assert.Equal(t, "treasure", b.Slot)
	assert.True(t, b.OK, "support slot binding failed: %s", b.Reason)
	assert.Empty(t, b.Reason)
}

func TestVerifyEmbeddedWeaponBindings_SupportSlotDoesNotDependOnTheDefaultMapFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFullRoster(t, root)
	require.NoError(t, os.RemoveAll(filepath.Join(root, "roles", "default.yaml")))
	writeWeaponFixture(t, root, "treasure-chest", "jeweler", "role: jeweler\nslot: treasure\norigin: external\n")

	bindings, err := verifyEmbeddedWeaponBindings(root)
	require.NoError(t, err)
	b, found := findBinding(bindings, "treasure-chest")
	require.True(t, found)
	assert.True(t, b.OK, "support slot binding failed: %s", b.Reason)
}

// A pipeline slot is still held to the default.yaml mapping.
func TestVerifyEmbeddedWeaponBindings_PipelineSlotStillRequiresTheDefaultMap(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeDefaultRoleSlotMap(t, root)
	writeFullRoster(t, root)
	writeWeaponFixture(t, root, "other-ranger", "ranger", "role: ranger\nslot: refinement\n")

	bindings, err := verifyEmbeddedWeaponBindings(root)
	require.NoError(t, err)
	b, found := findBinding(bindings, "other-ranger")
	require.True(t, found)
	assert.False(t, b.OK)
	assert.Contains(t, b.Reason, "roles/default.yaml maps slot")
}
