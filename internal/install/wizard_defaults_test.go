package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func archivistChoiceCatalog() pluginCatalog {
	provider := func(id, role string, certified, isDefault bool) pluginCatalogProvider {
		p := pluginCatalogProvider{
			ID: id, Version: "1.0.0", RiskScore: "write_analysis", CanonicalRole: role,
			CompatibilitySource: "embedded", Default: isDefault,
		}
		if certified {
			p.Ranked, p.CertificationDigest = true, "sha256:cert-"+id
		}
		return p
	}
	return pluginCatalog{Providers: []pluginCatalogProvider{
		provider("brainstorming", "ranger", true, true),
		provider("openspec-propose", "archivist", true, true),
		provider("writing-plans", "archivist", true, false),
	}}
}

func enterThrough(t *testing.T, d wizardDefaults) (discovery, refinement, refinementMode string) {
	t.Helper()
	b := i18n.BundleFor("en")
	discovery, refinement, _, _, refinementMode, _, err := promptSlots(
		NewTextPrompter(strings.NewReader("\n\n\n")), b, archivistChoiceCatalog(), knownProviderRisk, false, d)
	require.NoError(t, err)
	return discovery, refinement, refinementMode
}

func TestWizardWithoutAnExistingConfigurationKeepsTheCatalogDefaults(t *testing.T) {
	t.Parallel()
	_, refinement, mode := enterThrough(t, wizardDefaults{})
	assert.Equal(t, "openspec-propose", domainProviderID(refinement))
	assert.Equal(t, domain.SlotBindingModeRanked, mode)
}

// Accepting every default in the wizard used to reset the slots, because the
// defaults came from the catalog and never from the current active.yaml.
func TestWizardPrefillsTheSlotsFromTheExistingConfiguration(t *testing.T) {
	t.Parallel()
	d := wizardDefaults{
		Slots: map[string]string{"discovery": "brainstorming", "refinement": "writing-plans"},
		Modes: map[string]string{"discovery": domain.SlotBindingModeRanked, "refinement": domain.SlotBindingModeRanked},
	}
	_, refinement, mode := enterThrough(t, d)
	assert.Equal(t, "writing-plans", domainProviderID(refinement), "entering through must keep the current Weapon")
	assert.Equal(t, domain.SlotBindingModeRanked, mode)
}

func TestWizardPrefillsACustomBindingAsCustomNotRanked(t *testing.T) {
	t.Parallel()
	d := wizardDefaults{
		Slots: map[string]string{"refinement": "writing-plans"},
		Modes: map[string]string{"refinement": domain.SlotBindingModeCustom},
	}
	_, refinement, mode := enterThrough(t, d)
	assert.Equal(t, "writing-plans", domainProviderID(refinement))
	assert.Equal(t, domain.SlotBindingModeCustom, mode, "a Custom binding must not silently become Ranked")
}

func TestWizardIgnoresAnExistingSlotThatIsNoLongerOffered(t *testing.T) {
	t.Parallel()
	d := wizardDefaults{Slots: map[string]string{"refinement": "removed-weapon"}}
	_, refinement, _ := enterThrough(t, d)
	assert.Equal(t, "openspec-propose", domainProviderID(refinement))
}

func TestLoadWizardDefaultsReadsActiveYAMLAndTheLockModes(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	body := "mode: epic\nbase_path: .custom\nlanguage:\n  ui: pt-BR\n  docs: en\n  chat: pt-BR\n  code: en\nslots:\n  refinement: writing-plans\nadr:\n  canonical_path: .custom/adr\nrunbooks:\n  canonical_path: docs/runbooks\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(body), 0o600))

	d := loadWizardDefaults(dir)
	assert.Equal(t, "pt-BR", d.UI)
	assert.Equal(t, ".custom", d.BasePath)
	assert.Equal(t, ".custom/adr", d.ADRRoot)
	assert.Equal(t, "docs/runbooks", d.RunbookRoot)
	assert.Equal(t, "writing-plans", d.Slots["refinement"])
}

func TestLoadWizardDefaultsOfAMissingFileIsEmpty(t *testing.T) {
	t.Parallel()
	d := loadWizardDefaults(filepath.Join(t.TempDir(), ".strategist"))
	assert.Empty(t, d.Slots)
	assert.Empty(t, d.BasePath)
}

func TestLoadWizardDefaultsIgnoresMalformedActiveYAMLAndReadsBindingModes(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, activeYAMLName), []byte("mode: [broken"), 0o600))
	assert.Empty(t, loadWizardDefaults(dir).Slots)

	valid := "mode: epic\nslots:\n  refinement: writing-plans\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, activeYAMLName), []byte(valid), 0o600))
	lock := "schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: refinement\n    installed_instance_id: writing-plans@1.0.0\n    mode: custom\n    status: active\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, pluginLockFileName), []byte(lock), 0o600))
	d := loadWizardDefaults(dir)
	assert.Equal(t, domain.SlotBindingModeCustom, d.Modes["refinement"])
}

func TestWizardDefaultsKeepValidValuesAndMatchExistingWeaponReferences(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "catalog", orDefault("", "catalog", []string{"custom"}))
	assert.Equal(t, "catalog", orDefault("missing", "catalog", []string{"custom"}))
	assert.Equal(t, "custom", orDefault("custom", "catalog", []string{"custom"}))

	if got, ok := matchingOption([]string{"writing-plans@1.0.0"}, "writing-plans"); !ok || got != "writing-plans@1.0.0" {
		t.Fatalf("provider id match: %q %t", got, ok)
	}
	if got, ok := matchingOption([]string{"writing-plans"}, "writing-plans"); !ok || got != "writing-plans" {
		t.Fatalf("exact match: %q %t", got, ok)
	}
	if got, ok := matchingOption([]string{"brainstorming"}, "writing-plans"); ok || got != "" {
		t.Fatalf("unexpected missing match: %q %t", got, ok)
	}
}

func TestWizardReplacementBacksTheExistingConfigurationUp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	svc := newSvcW(t, "en\nen\npt-BR\nen\nepic\n/workspace\n\n\nbrainstorming\nopenspec-propose\narchivist\nfixture-provider\n\n")
	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Silent: true}))
	strategistDir := filepath.Join(dir, ".strategist")
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "active.yaml"), []byte(customizedConfiguration), 0o644))

	require.NoError(t, svc.Install(context.Background(), domain.InstallConfig{Target: dir, Wizard: true}))

	stamps, err := ListUpgradeBackups(strategistDir)
	require.NoError(t, err)
	require.NotEmpty(t, stamps)
	saved, err := os.ReadFile(filepath.Join(strategistDir, upgradeBackupRelDir, stamps[0], "active.yaml"))
	require.NoError(t, err)
	assert.Equal(t, customizedConfiguration, string(saved), "the wizard must back up what it replaces")
}
