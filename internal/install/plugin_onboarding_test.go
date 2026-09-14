package install

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanPluginOnboardingFromActiveSlotsProducesPreviewableBindings(t *testing.T) {
	t.Parallel()

	catalog, err := loadPluginCatalog(defaultsExtractor{})
	require.NoError(t, err)

	plan, err := planPluginOnboarding(defaultsExtractor{}, catalog, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "openspec-propose",
		"execution":  "sniper",
	})
	require.NoError(t, err)

	assert.True(t, plan.RequiresConfirmation)
	assert.Equal(t, "strategist-plugin-onboarding-plan/v1", plan.SchemaVersion)
	assert.Len(t, plan.Inventory.Instances, 3)
	assert.Len(t, plan.Bindings, 3)
	// 3 legacy adapter_contract nodes + 3 role_provider_binding nodes — this
	// slot combination (brainstorming/ranger, openspec-propose/archivist,
	// sniper/sniper) resolves fully for every slot (see
	// TestPlanRoleProviderMigrationValidatesOpenspecProposeAsArchivistMigrationCase).
	assert.Len(t, plan.Lock.Nodes, 6)
	assert.Contains(t, plan.Preview(), "slot discovery -> brainstorming@")
	assert.Contains(t, plan.Preview(), "lock ")
	for _, binding := range plan.Bindings {
		assert.NotEmpty(t, binding.InstalledInstanceID)
		assert.Equal(t, "enabled", binding.Status)
	}
}

// TestPlanPluginOnboardingIncludesRoleProviderBindingLockNodes proves
// tasks.md Task 3.3 ("Record effective Role -> Provider bindings in the
// existing lock") for real: the role/provider bindings resolved for these
// slots participate in the same plan.Lock the legacy adapter_contract nodes
// already do, and the lock's GraphDigest covers them (a tampered/missing
// role_provider_binding node changes the digest).
func TestPlanPluginOnboardingIncludesRoleProviderBindingLockNodes(t *testing.T) {
	t.Parallel()

	catalog, err := loadPluginCatalog(defaultsExtractor{})
	require.NoError(t, err)
	slots := map[string]string{
		"discovery":  "brainstorming",
		"refinement": "openspec-propose",
		"execution":  "sniper",
	}

	plan, err := planPluginOnboarding(defaultsExtractor{}, catalog, slots)
	require.NoError(t, err)

	var roleNodes int
	for _, node := range plan.Lock.Nodes {
		if node.Kind != plugins.RoleBindingLockKind {
			continue
		}
		roleNodes++
		assert.NotEmpty(t, node.Digest)
	}
	assert.Equal(t, 3, roleNodes, "expected one role_provider_binding node per resolved slot")
	assert.Equal(t, plugins.DigestLockNodes(plan.Lock.Nodes), plan.Lock.GraphDigest)
}

// TestPlanPluginOnboardingRoleBindingLockNodesReplayDeterministically proves
// tasks.md Task 3.3's "offline replay" requirement: re-running the same plan
// from local embedded defaults — no network, no reselection — reproduces the
// identical lock graph digest byte-for-byte.
func TestPlanPluginOnboardingRoleBindingLockNodesReplayDeterministically(t *testing.T) {
	t.Parallel()

	catalog, err := loadPluginCatalog(defaultsExtractor{})
	require.NoError(t, err)
	slots := map[string]string{
		"discovery":  "brainstorming",
		"refinement": "openspec-propose",
		"execution":  "sniper",
	}

	first, err := planPluginOnboarding(defaultsExtractor{}, catalog, slots)
	require.NoError(t, err)
	second, err := planPluginOnboarding(defaultsExtractor{}, catalog, slots)
	require.NoError(t, err)

	assert.Equal(t, first.Lock.GraphDigest, second.Lock.GraphDigest)
	assert.ElementsMatch(t, first.Lock.Nodes, second.Lock.Nodes)
}

// TestPlanPluginOnboardingRoleBindingEvidenceCoversEveryEntry proves the
// evidence half of tasks.md Task 6.1: every slot's role/provider resolution
// outcome (resolved or not) is available as evidence, not just the
// successfully resolved ones folded into the lock.
func TestPlanPluginOnboardingRoleBindingEvidenceCoversEveryEntry(t *testing.T) {
	t.Parallel()

	catalog, err := loadPluginCatalog(defaultsExtractor{})
	require.NoError(t, err)
	slots := map[string]string{
		"discovery":  "brainstorming",
		"refinement": "openspec-propose",
		"execution":  "sniper",
	}

	plan, err := planPluginOnboarding(defaultsExtractor{}, catalog, slots)
	require.NoError(t, err)

	events := plan.RoleMigration.Evidence()
	assert.Len(t, events, len(plan.RoleMigration.Entries))
	assert.True(t, plan.RoleMigration.FullyResolved())
}

func TestPlanPluginOnboardingRejectsUnresolvedActiveSlot(t *testing.T) {
	t.Parallel()

	catalog, err := loadPluginCatalog(defaultsExtractor{})
	require.NoError(t, err)

	_, err = planPluginOnboarding(defaultsExtractor{}, catalog, map[string]string{
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
	plan, err := planPluginOnboarding(defaultsExtractor{}, catalog, map[string]string{
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
	plan, err := planPluginOnboarding(defaultsExtractor{}, catalog, map[string]string{
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

	// checkCustomSkillAvailability (tasks.md Task 6,
	// .analysis/refined/20260913-embedded-skill-directory-catalog) now runs
	// before planPluginOnboarding and pauses on any registry-unknown value
	// that also fails already-installed resolution — "missing-refinement" is
	// exactly that case, so it is now caught here with a clearer
	// "configured_unverified" message instead of surfacing as a raw
	// "plugin onboarding plan"/"unresolved_active_slot" error. The wizard
	// still blocks (require.Error) either way; only the message improved.
	require.Error(t, err)
	require.ErrorContains(t, err, "configured_unverified")
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
