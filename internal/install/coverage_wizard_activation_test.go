package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func defaultOnboardingPlan(t *testing.T) (pluginCatalog, pluginOnboardingPlan, domain.WizardConfig) {
	t.Helper()
	extractor := defaultsExtractor{}
	catalog, err := loadPluginCatalog(extractor)
	require.NoError(t, err)
	slots := map[string]string{"discovery": "brainstorming", "refinement": "openspec-propose", "execution": "sniper"}
	plan, err := planPluginOnboarding(extractor, catalog, slots)
	require.NoError(t, err)
	wc := domain.WizardConfig{DiscoveryProvider: "brainstorming", RefinementProvider: "openspec-propose", ExecutionProvider: "sniper"}
	return catalog, plan, wc
}

func TestActivateWizardPlanFailureModes(t *testing.T) {
	catalog, plan, wc := defaultOnboardingPlan(t)

	unresolved := plan
	unresolved.RoleMigration = RoleProviderMigrationPreview{}
	_, err := activateWizardPlan(defaultsExtractor{}, catalog, wc, t.TempDir(), unresolved, false)
	require.ErrorContains(t, err, "wizard:")

	badPolicy := t.TempDir()
	policy := filepath.Join(badPolicy, levelingPolicyPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(policy), 0o755))
	require.NoError(t, os.WriteFile(policy, []byte("policy: [unclosed"), 0o644))
	_, err = activateWizardPlan(defaultsExtractor{}, catalog, wc, badPolicy, plan, false)
	require.ErrorContains(t, err, "wizard:")

	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	_, err = activateWizardPlan(defaultsExtractor{}, catalog, wc, blocker, plan, false)
	require.Error(t, err)
}
