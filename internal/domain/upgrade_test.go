package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestDecideUpgradeFileState_Orphaned(t *testing.T) {
	t.Parallel()

	state := domain.DecideUpgradeFileState(domain.UpgradeFileInput{
		Exists:       true,
		CurrentHash:  "abc",
		EmbeddedHash: "", // not part of current embedded tree
	})
	assert.Equal(t, domain.UpgradeOrphaned, state)
}

func TestDecideUpgradeFileState_OrphanedTakesPrecedenceOverMissing(t *testing.T) {
	t.Parallel()

	// Not on disk AND not embedded — still orphaned, not "missing" (there is
	// nothing to write; the manifest just remembers a path that no longer
	// exists anywhere).
	state := domain.DecideUpgradeFileState(domain.UpgradeFileInput{
		Exists:       false,
		EmbeddedHash: "",
	})
	assert.Equal(t, domain.UpgradeOrphaned, state)
}

func TestDecideUpgradeFileState_Missing(t *testing.T) {
	t.Parallel()

	state := domain.DecideUpgradeFileState(domain.UpgradeFileInput{
		Exists:       false,
		EmbeddedHash: "embedded-hash",
	})
	assert.Equal(t, domain.UpgradeMissing, state)
}

func TestDecideUpgradeFileState_Managed(t *testing.T) {
	t.Parallel()

	state := domain.DecideUpgradeFileState(domain.UpgradeFileInput{
		Exists:       true,
		CurrentHash:  "same",
		EmbeddedHash: "same",
	})
	assert.Equal(t, domain.UpgradeManaged, state)
}

func TestDecideUpgradeFileState_AutoUpgrade(t *testing.T) {
	t.Parallel()

	state := domain.DecideUpgradeFileState(domain.UpgradeFileInput{
		Exists:       true,
		CurrentHash:  "old-embedded",
		EmbeddedHash: "new-embedded",
		ManifestHash: "old-embedded",
		HasManifest:  true,
	})
	assert.Equal(t, domain.UpgradeAutoUpgrade, state)
}

func TestDecideUpgradeFileState_CustomizedWithManifest(t *testing.T) {
	t.Parallel()

	state := domain.DecideUpgradeFileState(domain.UpgradeFileInput{
		Exists:       true,
		CurrentHash:  "user-edited",
		EmbeddedHash: "new-embedded",
		ManifestHash: "old-embedded",
		HasManifest:  true,
	})
	assert.Equal(t, domain.UpgradeCustomized, state)
}

func TestDecideUpgradeFileState_CustomizedWithoutManifest(t *testing.T) {
	t.Parallel()

	state := domain.DecideUpgradeFileState(domain.UpgradeFileInput{
		Exists:       true,
		CurrentHash:  "unknown-content",
		EmbeddedHash: "new-embedded",
		HasManifest:  false,
	})
	assert.Equal(t, domain.UpgradeCustomized, state)
}

func TestUpgradeFileWillWrite(t *testing.T) {
	t.Parallel()

	assert.True(t, domain.UpgradeFileWillWrite(domain.UpgradeMissing, false))
	assert.True(t, domain.UpgradeFileWillWrite(domain.UpgradeAutoUpgrade, false))
	assert.False(t, domain.UpgradeFileWillWrite(domain.UpgradeCustomized, false))
	assert.True(t, domain.UpgradeFileWillWrite(domain.UpgradeCustomized, true))
	assert.False(t, domain.UpgradeFileWillWrite(domain.UpgradeManaged, true))
	assert.False(t, domain.UpgradeFileWillWrite(domain.UpgradeOrphaned, true))
}

func TestNewFullInstallManifest(t *testing.T) {
	t.Parallel()

	normativePath := domain.NormativeRuntimeDefaultFiles()[0].Path
	hashes := map[string]string{
		normativePath:        "hash-a",
		"some/other/file.md": "hash-b",
	}

	m := domain.NewFullInstallManifest("v1.2.3", hashes)

	assert.Equal(t, "strategist.install-manifest.v1", m.Schema)
	assert.Equal(t, "v1.2.3", m.PackageID)
	assert.NotEmpty(t, m.InstalledAt)
	assert.Len(t, m.Files, 2)

	normativeEntry, ok := m.FileByPath(normativePath)
	assert.True(t, ok)
	assert.Equal(t, domain.RuntimeFileNormative, normativeEntry.Owner)
	assert.Equal(t, "hash-a", normativeEntry.SHA256)

	otherEntry, ok := m.FileByPath("some/other/file.md")
	assert.True(t, ok)
	assert.Equal(t, domain.RuntimeFileUserOwned, otherEntry.Owner)
}

func TestNewFullInstallManifest_Empty(t *testing.T) {
	t.Parallel()

	m := domain.NewFullInstallManifest("dev", nil)
	assert.Empty(t, m.Files)
}
