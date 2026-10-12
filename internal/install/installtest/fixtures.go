package installtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/install/sidecar"
	"github.com/stretchr/testify/require"
)

// RequireSymlinkSupport skips the test where symlinks are unavailable.
func RequireSymlinkSupport(t *testing.T) {
	t.Helper()
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink tests require Windows Developer Mode or symlink privilege: %v", err)
	}
}

// ScaffoldHostPackage writes the Strategist sidecar a wizard-selected host
// Weapon must declare, through the real generator, never inferred by the wizard.
func ScaffoldHostPackage(t *testing.T, dir, role, slot string) {
	t.Helper()
	_, err := sidecar.ScaffoldSidecar(sidecar.ScaffoldOptions{PackageDir: dir, Roles: []string{role}, Slots: []string{slot}})
	require.NoError(t, err)
}

// FindSlotBinding returns the binding of slot in bindings.
func FindSlotBinding(bindings []domain.SlotBinding, slot string) (domain.SlotBinding, bool) {
	for _, b := range bindings {
		if b.Slot == slot {
			return b, true
		}
	}
	return domain.SlotBinding{}, false
}

// FindBindingBySlot returns the binding of slot in bindings or fails the test.
func FindBindingBySlot(t *testing.T, bindings []domain.SlotBinding, slot string) domain.SlotBinding {
	t.Helper()
	for _, b := range bindings {
		if b.Slot == slot {
			return b
		}
	}
	t.Fatalf("no binding found for slot %q", slot)
	return domain.SlotBinding{}
}

// RankedVersionsCatalog returns a catalog holding one compiled Ranked binding per version.
func RankedVersionsCatalog(versions ...string) plugincatalog.PluginCatalog {
	catalog := plugincatalog.PluginCatalog{SchemaVersion: plugincatalog.PluginCatalogSchemaVersion}
	for _, version := range versions {
		catalog.RankedBindings = append(catalog.RankedBindings, domain.CompiledRankedBinding{
			Role: "ranger", Slot: "discovery", WeaponID: "demo", WeaponVersion: version,
			WeaponDigest: "sha256:w-" + version, BindingDigest: "sha256:b-" + version, CertificationDigest: "sha256:c-" + version,
			ConnectorID: "strategist-embedded", Entrypoint: "discover", Generation: 1, Status: "active",
			Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded},
		})
	}
	return catalog
}

// EmbeddedDefaultsCatalog loads the plugin catalog of the embedded defaults tree.
func EmbeddedDefaultsCatalog(t *testing.T) plugincatalog.PluginCatalog {
	t.Helper()
	catalog, err := plugincatalog.LoadPluginCatalog(DefaultsExtractor{})
	require.NoError(t, err)
	return catalog
}
