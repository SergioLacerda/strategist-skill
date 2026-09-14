package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPluginLockFile_MissingFileReturnsZeroValue(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	f, err := readPluginLockFile(dir)
	require.NoError(t, err)
	assert.Equal(t, domain.PluginLockFileSchemaVersion, f.SchemaVersion)
	assert.Empty(t, f.Inventory.Instances)
	assert.Empty(t, f.Bindings)
}

func TestReadPluginLockFile_CorruptYAMLErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, pluginLockFileName), []byte("not: [valid yaml"), 0o644))

	_, err := readPluginLockFile(dir)
	require.Error(t, err)
	assert.ErrorContains(t, err, pluginLockFileName)
}

func TestWritePluginLockFile_RoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	want := domain.PluginLockFile{
		Inventory: domain.PluginInventory{
			Instances: []domain.InstalledInstance{{ID: "brainstorming", State: "active", LastKnownGood: true}},
		},
		Bindings: []domain.SlotBinding{
			{SchemaVersion: "strategist-plugin-binding/v1", Slot: "discovery", InstalledInstanceID: "brainstorming", Generation: 1, Status: "enabled"},
		},
	}

	require.NoError(t, writePluginLockFile(dir, want))

	got, err := readPluginLockFile(dir)
	require.NoError(t, err)
	assert.Equal(t, domain.PluginLockFileSchemaVersion, got.SchemaVersion)
	assert.Equal(t, want.Inventory, got.Inventory)
	assert.Equal(t, want.Bindings, got.Bindings)
}

func TestWritePluginLockFile_StampsSchemaVersionRegardlessOfInput(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	require.NoError(t, writePluginLockFile(dir, domain.PluginLockFile{SchemaVersion: "bogus"}))

	got, err := readPluginLockFile(dir)
	require.NoError(t, err)
	assert.Equal(t, domain.PluginLockFileSchemaVersion, got.SchemaVersion)
}

func TestWritePluginLockFile_WriteErrorPropagates(t *testing.T) {
	t.Parallel()
	skipIfPermissionTestUnsupported(t)
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := writePluginLockFile(dir, domain.PluginLockFile{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "write "+pluginLockFileName)
}
