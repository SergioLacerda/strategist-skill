package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/onboarding"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/install/wizard"
	"github.com/stretchr/testify/require"
)

func defaultOnboardingPlan(t *testing.T) (plugincatalog.PluginCatalog, onboarding.PluginOnboardingPlan, domain.WizardConfig) {
	t.Helper()
	extractor := installtest.DefaultsExtractor{}
	catalog, err := plugincatalog.LoadPluginCatalog(extractor)
	require.NoError(t, err)
	slots := map[string]string{"discovery": "brainstorming", "refinement": "openspec-propose", "execution": "sniper"}
	plan, err := onboarding.PlanPluginOnboarding(extractor, catalog, slots)
	require.NoError(t, err)
	wc := domain.WizardConfig{DiscoveryProvider: "brainstorming", RefinementProvider: "openspec-propose", ExecutionProvider: "sniper"}
	return catalog, plan, wc
}

func TestActivateWizardPlanFailureModes(t *testing.T) {
	catalog, plan, wc := defaultOnboardingPlan(t)

	unresolved := plan
	unresolved.RoleMigration = onboarding.RoleProviderMigrationPreview{}
	_, err := wizard.ActivateWizardPlan(installtest.DefaultsExtractor{}, catalog, wc, t.TempDir(), unresolved, false)
	require.ErrorContains(t, err, "wizard:")

	badPolicy := t.TempDir()
	policy := filepath.Join(badPolicy, names.LevelingPolicyPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(policy), 0o755))
	require.NoError(t, os.WriteFile(policy, []byte("policy: [unclosed"), 0o644))
	_, err = wizard.ActivateWizardPlan(installtest.DefaultsExtractor{}, catalog, wc, badPolicy, plan, false)
	require.ErrorContains(t, err, "wizard:")

	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	_, err = wizard.ActivateWizardPlan(installtest.DefaultsExtractor{}, catalog, wc, blocker, plan, false)
	require.Error(t, err)
}

func TestActivateWizardPlanProducesBindingArtifacts(t *testing.T) {
	catalog, plan, wc := defaultOnboardingPlan(t)

	lock, err := wizard.ActivateWizardPlan(installtest.DefaultsExtractor{}, catalog, wc, "", plan, false)
	require.NoError(t, err)
	require.NotEmpty(t, lock.BindingArtifacts)
	require.LessOrEqual(t, len(lock.BindingArtifacts), len(lock.Bindings))
	require.NoError(t, domain.ValidateWeaponBindingArtifacts(lock.Bindings, lock.BindingArtifacts))
}
