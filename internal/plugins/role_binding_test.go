package plugins_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rangerRole() domain.RoleContract {
	return domain.RoleContract{
		SchemaVersion: domain.RoleContractSchemaVersion,
		Role:          "ranger",
		Slot:          "discovery",
	}
}

func rangerProvider(source domain.ProviderSource, version string) domain.ProviderContract {
	return domain.ProviderContract{
		SchemaVersion:                 "strategist-provider-contract/v1",
		ID:                            "brainstorming",
		Version:                       version,
		ProviderSchemaVersion:         "1",
		CanonicalRole:                 "ranger",
		RiskScore:                     "write_analysis",
		Source:                        source,
		SupportedRoleContractVersions: []string{domain.RoleContractSchemaVersion},
	}
}

func TestResolveRoleBindingFiltersByCanonicalRole(t *testing.T) {
	t.Parallel()

	role := rangerRole()
	candidates := []domain.ProviderContract{
		rangerProvider(domain.ProviderSourceEmbedded, "1.0.0"),
		{
			SchemaVersion:                 "strategist-provider-contract/v1",
			ID:                            "sniper",
			Version:                       "1.0.0",
			ProviderSchemaVersion:         "1",
			CanonicalRole:                 "sniper",
			RiskScore:                     "controlled",
			Source:                        domain.ProviderSourceNativeRole,
			SupportedRoleContractVersions: []string{domain.RoleContractSchemaVersion},
		},
	}

	binding, err := plugins.ResolveRoleBinding(role, candidates, "", "")
	require.NoError(t, err)
	assert.Equal(t, "brainstorming", binding.Provider.ID)
	assert.True(t, binding.Compatibility.Compatible)
}

func TestResolveRoleBindingRejectsImplicitIDShadowing(t *testing.T) {
	t.Parallel()

	role := rangerRole()
	candidates := []domain.ProviderContract{
		rangerProvider(domain.ProviderSourceEmbedded, "1.0.0"),
		rangerProvider(domain.ProviderSourceExternal, "1.0.0"),
	}

	_, err := plugins.ResolveRoleBinding(role, candidates, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id_shadowing")
	assert.Contains(t, err.Error(), "brainstorming")
}

func TestResolveRoleBindingAcceptsExplicitShadowOverride(t *testing.T) {
	t.Parallel()

	role := rangerRole()
	candidates := []domain.ProviderContract{
		rangerProvider(domain.ProviderSourceEmbedded, "1.0.0"),
		rangerProvider(domain.ProviderSourceExternal, "2.0.0"),
	}

	binding, err := plugins.ResolveRoleBinding(role, candidates, domain.ProviderSourceExternal, "")
	require.NoError(t, err)
	assert.Equal(t, domain.ProviderSourceExternal, binding.Provider.Source)
	assert.Equal(t, "2.0.0", binding.Provider.Version)
}

func TestResolveRoleBindingReportsShadowOverrideNotPresent(t *testing.T) {
	t.Parallel()

	role := rangerRole()
	candidates := []domain.ProviderContract{
		rangerProvider(domain.ProviderSourceEmbedded, "1.0.0"),
		rangerProvider(domain.ProviderSourceExternal, "1.0.0"),
	}

	_, err := plugins.ResolveRoleBinding(role, candidates, domain.ProviderSourceNativeRole, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id_shadowing")
	assert.Contains(t, err.Error(), "does not include override source native_role")
}

func TestResolveRoleBindingReportsMissingCompatibleProvider(t *testing.T) {
	t.Parallel()

	role := rangerRole()
	incompatible := rangerProvider(domain.ProviderSourceEmbedded, "1.0.0")
	incompatible.SupportedRoleContractVersions = []string{"strategist-role-contract/v0"}

	_, err := plugins.ResolveRoleBinding(role, []domain.ProviderContract{incompatible}, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role_binding_missing")
}

func TestResolveRoleBindingReportsAmbiguousCompatibleProviders(t *testing.T) {
	t.Parallel()

	role := rangerRole()
	first := rangerProvider(domain.ProviderSourceEmbedded, "1.0.0")
	first.ID = "brainstorming-a"
	second := rangerProvider(domain.ProviderSourceEmbedded, "1.0.0")
	second.ID = "brainstorming-b"

	_, err := plugins.ResolveRoleBinding(role, []domain.ProviderContract{first, second}, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role_binding_ambiguous")
	assert.Contains(t, err.Error(), "brainstorming-a")
	assert.Contains(t, err.Error(), "brainstorming-b")
}

func TestResolveRoleBindingPreferredProviderIDDisambiguatesLegitimateAmbiguity(t *testing.T) {
	t.Parallel()

	role := rangerRole()
	first := rangerProvider(domain.ProviderSourceEmbedded, "1.0.0")
	first.ID = "brainstorming-a"
	second := rangerProvider(domain.ProviderSourceEmbedded, "1.0.0")
	second.ID = "brainstorming-b"

	binding, err := plugins.ResolveRoleBinding(role, []domain.ProviderContract{first, second}, "", "brainstorming-b")
	require.NoError(t, err)
	assert.Equal(t, "brainstorming-b", binding.Provider.ID)
}

func TestResolveRoleBindingPreferredProviderIDNotAmongCompatibleStaysAmbiguous(t *testing.T) {
	t.Parallel()

	role := rangerRole()
	first := rangerProvider(domain.ProviderSourceEmbedded, "1.0.0")
	first.ID = "brainstorming-a"
	second := rangerProvider(domain.ProviderSourceEmbedded, "1.0.0")
	second.ID = "brainstorming-b"

	_, err := plugins.ResolveRoleBinding(role, []domain.ProviderContract{first, second}, "", "brainstorming-c")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role_binding_ambiguous")
}

func TestRoleBindingLockNodeIsDeterministicAndParticipatesInLockDigest(t *testing.T) {
	t.Parallel()

	role := rangerRole()
	provider := rangerProvider(domain.ProviderSourceEmbedded, "1.0.0")
	binding := domain.ProviderBinding{Role: role, Provider: provider, Compatibility: provider.CheckRoleAffinity(role)}

	nodeA := plugins.RoleBindingLockNode(binding)
	nodeB := plugins.RoleBindingLockNode(binding)
	assert.Equal(t, nodeA, nodeB, "lock node must be deterministic for the same binding")
	assert.Equal(t, "role_provider_binding", nodeA.Kind)
	assert.Equal(t, "ranger:brainstorming", nodeA.ID)
	assert.Regexp(t, `^sha256:[a-f0-9]{64}$`, nodeA.Digest)

	// The binding node hashes into the same digest-pinned lock graph as any
	// other resolved resource — no parallel lock store (Decision 4).
	otherNode := domain.PluginLockNode{ID: "adapter/b", Kind: "adapter_contract", Digest: digestB1}
	combined := plugins.DigestLockNodes([]domain.PluginLockNode{nodeA, otherNode})
	require.NotEmpty(t, combined)
	assert.Regexp(t, `^sha256:[a-f0-9]{64}$`, combined)
}
