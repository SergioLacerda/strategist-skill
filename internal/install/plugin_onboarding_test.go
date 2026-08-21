package install

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanPluginOnboardingFromActiveSlotsProducesPreviewableBindings(t *testing.T) {
	t.Parallel()

	catalog, err := loadPluginCatalog(defaultsExtractor{})
	require.NoError(t, err)

	plan, err := planPluginOnboarding(catalog, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "openspec-explore",
		"execution":  "sniper",
	})
	require.NoError(t, err)

	assert.True(t, plan.RequiresConfirmation)
	assert.Equal(t, "strategist-plugin-onboarding-plan/v1", plan.SchemaVersion)
	assert.Len(t, plan.Inventory.Instances, 3)
	assert.Len(t, plan.Bindings, 3)
	assert.Len(t, plan.Lock.Nodes, 3)
	assert.Contains(t, plan.Preview(), "slot discovery -> brainstorming@")
	assert.Contains(t, plan.Preview(), "lock ")
	for _, binding := range plan.Bindings {
		assert.NotEmpty(t, binding.InstalledInstanceID)
		assert.Equal(t, "enabled", binding.Status)
	}
}

func TestPlanPluginOnboardingRejectsUnresolvedActiveSlot(t *testing.T) {
	t.Parallel()

	catalog, err := loadPluginCatalog(defaultsExtractor{})
	require.NoError(t, err)

	_, err = planPluginOnboarding(catalog, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "missing-provider",
		"execution":  "sniper",
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "unresolved_active_slot")
	require.ErrorContains(t, err, "missing-provider")
}

func TestPlanPluginOnboardingApplyActivatesAfterProbe(t *testing.T) {
	t.Parallel()

	catalog, err := loadPluginCatalog(defaultsExtractor{})
	require.NoError(t, err)
	plan, err := planPluginOnboarding(catalog, map[string]string{
		"discovery": "brainstorming",
	})
	require.NoError(t, err)

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: "native-ranger", State: lifecycle.StateActive, LastKnownGood: true}}
	store.Bindings = []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "native-ranger", Generation: 2, Status: "enabled"}}

	require.NoError(t, applyPluginOnboardingPlan(store, plan, func(domain.SlotBinding, domain.InstalledInstance) bool {
		return true
	}))

	binding, ok := store.Binding("discovery")
	require.True(t, ok)
	assert.NotEqual(t, "native-ranger", binding.InstalledInstanceID)
	assert.Equal(t, int64(3), binding.Generation)
}

func TestPlanPluginOnboardingApplyRollsBackOnProbeFailure(t *testing.T) {
	t.Parallel()

	catalog, err := loadPluginCatalog(defaultsExtractor{})
	require.NoError(t, err)
	plan, err := planPluginOnboarding(catalog, map[string]string{
		"refinement": "openspec-explore",
	})
	require.NoError(t, err)

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: "native-archivist", State: lifecycle.StateActive, LastKnownGood: true}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: "native-archivist", Generation: 5, Status: "enabled"}}

	err = applyPluginOnboardingPlan(store, plan, func(domain.SlotBinding, domain.InstalledInstance) bool {
		return false
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "activation_requires_successful_probe")
	binding, ok := store.Binding("refinement")
	require.True(t, ok)
	assert.Equal(t, "native-archivist", binding.InstalledInstanceID)
}

func TestPlanPluginOnboardingDualReadMatchesWizardSlots(t *testing.T) {
	t.Parallel()

	wc := domain.WizardConfig{
		DiscoveryProvider:  "brainstorming",
		RefinementProvider: "openspec-explore",
		ExecutionProvider:  "sniper",
	}
	active := domain.ActiveConfig{Slots: wizardSlots(wc)}

	assert.Equal(t, active.Slots, wizardSlots(wc))
}

func TestRunWizardBlocksUnresolvedPluginPlanWhenCatalogExists(t *testing.T) {
	t.Parallel()

	ext := wizardCatalogExtractor{catalog: []byte(`
schema_version: strategist-plugin-catalog/v1
providers:
  - id: brainstorming
    risk_score: write_analysis
  - id: sniper
    risk_score: controlled
`)}

	_, err := runWizard(context.Background(), NewTextPrompter(strings.NewReader(
		"en\nen\nen\nen\nepic\n.analysis\nbrainstorming\nmissing-refinement\nsniper\n\n",
	)), ext)

	require.Error(t, err)
	require.ErrorContains(t, err, "plugin onboarding plan")
	require.ErrorContains(t, err, "missing-refinement")
}

type wizardCatalogExtractor struct {
	catalog []byte
}

func (w wizardCatalogExtractor) Extract(_ string, _ bool) error { return nil }
func (w wizardCatalogExtractor) ReadFile(relPath string) ([]byte, error) {
	switch relPath {
	case pluginCatalogPath:
		return w.catalog, nil
	case skillYAMLName:
		return []byte("active_config:\n  language:\n    values: [en, pt-BR]\n  mode:\n    values: [pragmatic, epic]\n"), nil
	default:
		return nil, fmt.Errorf("wizardCatalogExtractor: not found: %s", relPath)
	}
}
