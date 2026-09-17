package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func verifiedBrainstormingProvider(provider pluginCatalogProvider) pluginCatalogProvider {
	provider.UpstreamRepo = "obra/superpowers"
	provider.UpstreamSkillPath = "skills/brainstorming/SKILL.md"
	provider.UpstreamVersion = "6.3.0"
	provider.UpstreamCommit = "b36e0829c6d0140e93cfef2ca599b1b07d4a7797"
	provider.UpstreamContentDigest = "sha256:74edf03ea6d24ef53db48677b93558d14a979bdf052ca3f57ecdca0c66791608"
	provider.License = "MIT"
	return provider
}

// writeRoleContractFixture writes a minimal roles/<role>.yaml +
// internal_skills/<role>/SKILL.md pair under defaultsRoot — the two files
// hostAPIContractDigest (ADR-0043 DEC-006) reads.
func writeRoleContractFixture(t *testing.T, defaultsRoot, role string) {
	t.Helper()
	rolesDir := filepath.Join(defaultsRoot, "roles")
	skillDir := filepath.Join(defaultsRoot, "internal_skills", role)
	require.NoError(t, os.MkdirAll(rolesDir, 0o755))
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rolesDir, role+".yaml"), []byte("role: "+role+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# "+role+"\n"), 0o644))
}

func TestCertifyRankedCandidates_StampsPinnedPairing(t *testing.T) {
	defaultsRoot := t.TempDir()
	writeRoleContractFixture(t, defaultsRoot, "ranger")
	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		verifiedBrainstormingProvider(pluginCatalogProvider{ID: "brainstorming", RiskScore: "write_analysis", CanonicalRole: "ranger", Roles: []string{"ranger"}, Default: true}),
		{ID: "openspec-explore", RiskScore: "write_analysis", CanonicalRole: "ranger"},
	}}

	require.NoError(t, certifyRankedCandidates(&catalog, defaultsRoot))

	brainstorming, ok := findCatalogProvider(catalog, "brainstorming")
	require.True(t, ok)
	assert.True(t, brainstorming.Ranked)
	assert.NotEmpty(t, brainstorming.CertificationDigest)
	assert.Equal(t, int64(1), brainstorming.RankedBindingGeneration)
	assert.Equal(t, "active", brainstorming.RankedBindingStatus)
	assert.NotEmpty(t, brainstorming.HostAPIDigest)
	assert.NotEmpty(t, brainstorming.ConnectorDigest)
	assert.NotEmpty(t, brainstorming.TestSuiteDigest)
	assert.Equal(t, "C1", brainstorming.ConformanceLevel)
	// Default is untouched — Ranked and Default stay independent (DEC-001).
	assert.True(t, brainstorming.Default)

	// openspec-explore is not in rankedCertificationPairs (only
	// openspec-propose is, as Archivist's pairing) — confirms only pinned
	// pairings are certified, never every catalog entry.
	openspecExplore, ok := findCatalogProvider(catalog, "openspec-explore")
	require.True(t, ok)
	assert.False(t, openspecExplore.Ranked, "only the pinned pairing is certified, never every catalog entry")
}

// TestCertifyRankedCandidates_StampsBothPinnedPairings is the
// docs/adr/0045 regression test: two distinct Role→provider pairings
// (Ranger↔brainstorming, Archivist↔openspec-propose) are certified
// simultaneously, each with its own role-specific HostAPIDigest — proving
// hostAPIContractDigest's role-parameterization is exercised end-to-end
// with two roles, not just per-role in isolation.
func TestCertifyRankedCandidates_StampsBothPinnedPairings(t *testing.T) {
	defaultsRoot := t.TempDir()
	writeRoleContractFixture(t, defaultsRoot, "ranger")
	writeRoleContractFixture(t, defaultsRoot, "archivist")
	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		verifiedBrainstormingProvider(pluginCatalogProvider{ID: "brainstorming", RiskScore: "write_analysis", CanonicalRole: "ranger", Roles: []string{"ranger"}, Default: true}),
		{ID: "openspec-propose", RiskScore: "write_analysis", CanonicalRole: "archivist", Roles: []string{"archivist"}, Default: true,
			UpstreamRepo: "Fission-AI/OpenSpec", UpstreamSkillPath: "skills/openspec-propose/SKILL.md", UpstreamVersion: "1.10.0",
			UpstreamCommit: "1ebddd17f40dde15dfd28289e4493c3cf05ee9df", UpstreamContentDigest: "sha256:c0537ce311115878e7e0a04e6ff4fc6456056f21024079c228b6b24325e38613", License: "MIT"},
	}}

	require.NoError(t, certifyRankedCandidates(&catalog, defaultsRoot))

	brainstorming, ok := findCatalogProvider(catalog, "brainstorming")
	require.True(t, ok)
	assert.True(t, brainstorming.Ranked)
	assert.NotEmpty(t, brainstorming.HostAPIDigest)
	assert.NotEmpty(t, brainstorming.ConnectorDigest)
	assert.NotEmpty(t, brainstorming.TestSuiteDigest)
	assert.Equal(t, "C1", brainstorming.ConformanceLevel)

	openspecPropose, ok := findCatalogProvider(catalog, "openspec-propose")
	require.True(t, ok)
	assert.True(t, openspecPropose.Ranked)
	assert.NotEmpty(t, openspecPropose.HostAPIDigest)
	assert.NotEmpty(t, openspecPropose.ConnectorDigest)
	assert.NotEmpty(t, openspecPropose.TestSuiteDigest)
	assert.Equal(t, "C1", openspecPropose.ConformanceLevel)

	assert.NotEqual(t, brainstorming.HostAPIDigest, openspecPropose.HostAPIDigest,
		"each role's HostAPIDigest must be computed from its own roles/<role>.yaml + internal_skills/<role>/SKILL.md, not shared")
	// ConnectorDigest remains role-agnostic, while each role now has its own
	// conformance test file and therefore a distinct TestSuiteDigest.
	assert.Equal(t, brainstorming.ConnectorDigest, openspecPropose.ConnectorDigest)
	assert.NotEqual(t, brainstorming.TestSuiteDigest, openspecPropose.TestSuiteDigest)
}

func TestCertifyRankedCandidates_SkipsAbsentPinnedPairing(t *testing.T) {
	// openspec-explore is not in rankedCertificationPairs (neither
	// brainstorming nor openspec-propose) — this fixture catalog contains
	// no pinned pairing at all.
	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "openspec-explore", RiskScore: "write_analysis", CanonicalRole: "ranger"},
	}}

	require.NoError(t, certifyRankedCandidates(&catalog, t.TempDir()), "a pinned pairing absent from an unrelated fixture catalog is not an error")
}

func TestCertifyRankedCandidates_RejectsMissingRoleAffinity(t *testing.T) {
	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "brainstorming", RiskScore: "write_analysis", CanonicalRole: "archivist", Roles: []string{"archivist"}},
	}}

	err := certifyRankedCandidates(&catalog, t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role affinity")
}

func TestCertifyRankedCandidates_RejectsIncompleteProvenance(t *testing.T) {
	defaultsRoot := t.TempDir()
	writeRoleContractFixture(t, defaultsRoot, "ranger")
	catalog := pluginCatalog{Providers: []pluginCatalogProvider{{
		ID: "brainstorming", RiskScore: "write_analysis", CanonicalRole: "ranger", Roles: []string{"ranger"},
	}}}

	err := certifyRankedCandidates(&catalog, defaultsRoot)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "verified upstream provenance incomplete")
}

func TestCertifyRankedCandidatesLeavesIncompleteNonRankedProviderUncertified(t *testing.T) {
	catalog := pluginCatalog{Providers: []pluginCatalogProvider{{
		ID: "openspec-explore", RiskScore: "write_analysis", CanonicalRole: "ranger", Roles: []string{"ranger"},
	}}}

	require.NoError(t, certifyRankedCandidates(&catalog, t.TempDir()))
	provider, ok := findCatalogProvider(catalog, "openspec-explore")
	require.True(t, ok)
	assert.False(t, provider.Ranked)
	assert.Empty(t, provider.CertificationDigest)
	assert.Empty(t, provider.License)
}

func TestCertifyRankedCandidates_IsDeterministic(t *testing.T) {
	defaultsRoot := t.TempDir()
	writeRoleContractFixture(t, defaultsRoot, "ranger")
	build := func() pluginCatalog {
		return pluginCatalog{Providers: []pluginCatalogProvider{
			verifiedBrainstormingProvider(pluginCatalogProvider{ID: "brainstorming", Version: "1.0.0", RiskScore: "write_analysis", CanonicalRole: "ranger", Roles: []string{"ranger"}}),
		}}
	}
	a, b := build(), build()
	require.NoError(t, certifyRankedCandidates(&a, defaultsRoot))
	require.NoError(t, certifyRankedCandidates(&b, defaultsRoot))
	assert.Equal(t, a.Providers[0].CertificationDigest, b.Providers[0].CertificationDigest)
	assert.Equal(t, a.Providers[0].HostAPIDigest, b.Providers[0].HostAPIDigest)
}
