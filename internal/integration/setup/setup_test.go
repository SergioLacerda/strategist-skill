package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/stretchr/testify/require"
)

func existingEnabled() config.File {
	return config.File{SchemaVersion: config.SchemaVersion, Providers: map[string]config.Provider{ProviderName: {
		Enabled: true, Endpoint: "https://proxy.example.test/v1", CredentialRef: "env:MY_KEY", Model: "jev-2.0.0",
		AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate},
		DataPolicy:          config.DataPolicy{AllowedFields: []string{"objective"}, MaxStateBytes: 1000},
		ApprovalThreshold:   0.95,
	}}}
}

func TestSilentWithoutASelectionLeavesANewInstallDisabledAndWritesNothing(t *testing.T) {
	plan, err := Decide(Input{CredentialResolves: true})
	require.NoError(t, err)
	require.Equal(t, ActionNone, plan.Action)
	require.False(t, plan.Write, "AC-16: a new install with no choice records no decision")
	require.Empty(t, plan.File.Providers)
}

func TestAVariableOrDotenvAloneNeverEnables(t *testing.T) {
	for _, resolves := range []bool{true, false} {
		plan, err := Decide(Input{CredentialResolves: resolves})
		require.NoError(t, err)
		require.False(t, plan.Write)
		require.NotEqual(t, ActionEnable, plan.Action, "AC-03: the credential existing is not consent")
	}
}

func TestAnUpgradePreservesAnExistingDecisionWithoutASelection(t *testing.T) {
	existing := existingEnabled()
	for _, choice := range []Choice{"", ChoiceKeep} {
		plan, err := Decide(Input{Existing: existing, Found: true, Choice: choice, CredentialResolves: true})
		require.NoError(t, err)
		require.Equal(t, ActionPreserve, plan.Action)
		require.False(t, plan.Write, "AC-16: the prior decision is untouched")
		require.Equal(t, existing, plan.File)
	}
}

func TestKeepWithoutAPriorDecisionIsNoDecision(t *testing.T) {
	plan, err := Decide(Input{Choice: ChoiceKeep})
	require.NoError(t, err)
	require.Equal(t, ActionNone, plan.Action)
	require.False(t, plan.Write)
}

func TestEnableProposesTheDocumentedDefaultsAndRecordsPendingCredentials(t *testing.T) {
	ready, err := Decide(Input{Choice: ChoiceEnable, CredentialResolves: true})
	require.NoError(t, err)
	require.Equal(t, ActionEnable, ready.Action)
	require.True(t, ready.Write)
	require.False(t, ready.Pending)
	provider := ready.File.Providers[ProviderName]
	require.True(t, provider.Enabled)
	require.Equal(t, DefaultEndpoint, provider.Endpoint)
	require.Equal(t, DefaultCredentialRef, provider.CredentialRef)
	require.Equal(t, DefaultModel, provider.Model)
	require.NotEmpty(t, provider.DataPolicy.AllowedFields)
	require.NoError(t, ready.File.Validate())

	pending, err := Decide(Input{Choice: ChoiceEnable, CredentialResolves: false})
	require.NoError(t, err)
	require.True(t, pending.Pending, "WIZ-07: a missing file or variable is saved as a pending intent")
	require.True(t, pending.File.Providers[ProviderName].Enabled)
	require.NoError(t, pending.File.Validate())
}

func TestEnableKeepsOperatorValuesAndAppliesOverrides(t *testing.T) {
	plan, err := Decide(Input{Existing: existingEnabled(), Found: true, Choice: ChoiceEnable, Overrides: Overrides{Model: "jev-3.0.0"}, CredentialResolves: true})
	require.NoError(t, err)
	provider := plan.File.Providers[ProviderName]
	require.Equal(t, "https://proxy.example.test/v1", provider.Endpoint, "an existing value is kept")
	require.Equal(t, "jev-3.0.0", provider.Model, "an explicit override wins")
	require.InDelta(t, 0.95, provider.ApprovalThreshold, 1e-9)
}

func TestDisablePreservesTheProviderSettingsAndBlocksNewCalls(t *testing.T) {
	plan, err := Decide(Input{Existing: existingEnabled(), Found: true, Choice: ChoiceDisable})
	require.NoError(t, err)
	require.Equal(t, ActionDisable, plan.Action)
	require.True(t, plan.Write, "the refusal is recorded so it is not asked again as if it were new")
	provider := plan.File.Providers[ProviderName]
	require.False(t, provider.Enabled)
	require.Equal(t, "jev-2.0.0", provider.Model, "WIZ-08: historical settings are preserved")

	fresh, err := Decide(Input{Choice: ChoiceDisable})
	require.NoError(t, err)
	require.False(t, fresh.File.Providers[ProviderName].Enabled)
	require.NoError(t, fresh.File.Validate())
}

func TestUnknownChoiceIsRejected(t *testing.T) {
	_, err := Decide(Input{Choice: "maybe"})
	require.Error(t, err)
}

func TestTheSameSelectionYieldsTheSamePlanFromEveryAdapter(t *testing.T) {
	input := Input{Choice: ChoiceEnable, CredentialResolves: true}
	wizard, err := Decide(input)
	require.NoError(t, err)
	silent, err := Decide(input)
	require.NoError(t, err)
	require.Equal(t, wizard, silent, "one selection, one plan: the planner has no adapter-specific input")
}

func TestDefaultChoiceIsYesOnlyWhenTheCredentialResolves(t *testing.T) {
	require.Equal(t, ChoiceEnable, DefaultChoice(true, false))
	require.Equal(t, ChoiceDisable, DefaultChoice(false, false))
	require.Equal(t, ChoiceKeep, DefaultChoice(true, true), "an existing decision defaults to keeping it")
	require.Equal(t, []Choice{ChoiceEnable, ChoiceDisable}, Offered(false))
	require.Equal(t, []Choice{ChoiceEnable, ChoiceDisable, ChoiceKeep}, Offered(true))
}

func TestPlanDisclosesTheDataCategoriesAndTheEndpoint(t *testing.T) {
	plan, err := Decide(Input{Choice: ChoiceEnable, CredentialResolves: true})
	require.NoError(t, err)
	require.Equal(t, DefaultEndpoint, plan.Endpoint)
	require.Contains(t, plan.DataCategories, "objective")
	require.Contains(t, plan.DataCategories, "implementation_plan")
}

func TestApplyWritesOnlyWhenThePlanSaysSo(t *testing.T) {
	path := filepath.Join(t.TempDir(), config.FileName)
	none, err := Decide(Input{})
	require.NoError(t, err)
	require.NoError(t, Apply(path, none))
	require.NoFileExists(t, path)

	enable, err := Decide(Input{Choice: ChoiceEnable, CredentialResolves: true})
	require.NoError(t, err)
	require.NoError(t, Apply(path, enable))
	loaded, found, err := config.Load(path)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, enable.File, loaded)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Zero(t, info.Mode().Perm()&0o077)
}

func TestResolvesReportsWhetherTheConfiguredCredentialIsAvailable(t *testing.T) {
	workspace := t.TempDir()
	require.False(t, Resolves(DefaultCredentialRef, workspace))
	require.NoError(t, os.WriteFile(filepath.Join(workspace, ".env"), []byte("TYPESAFE_API_KEY=sk-x\n"), 0o600))
	require.True(t, Resolves(DefaultCredentialRef, workspace))
}
