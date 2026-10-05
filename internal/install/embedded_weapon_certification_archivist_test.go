package install

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writingPlansCandidate is the catalog entry shape of the embedded
// writing-plans Weapon once it declares the prompt-bridge runtime
// (ADR-0067). The runtime is a parameter so the same fixture proves what is
// and is not certifiable.
func writingPlansCandidate(version string, runtime domain.WeaponRuntime) pluginCatalogProvider {
	return pluginCatalogProvider{
		ID: "writing-plans", Version: version, RiskScore: "write_analysis", CanonicalRole: "archivist", Roles: []string{"archivist"},
		SupportedSlots: []string{"refinement"}, Runtime: runtime,
		UpstreamRepo: "obra/superpowers", UpstreamSkillPath: "skills/writing-plans/SKILL.md", UpstreamVersion: "6.3.0",
		UpstreamCommit:        "b36e0829c6d0140e93cfef2ca599b1b07d4a7797",
		UpstreamContentDigest: "sha256:2f24891aab2623894a34e3322fbd1b4fcb2254d089293ef78ab1740a79f79c49", License: "MIT",
	}
}

func embeddedPromptBridge() domain.WeaponRuntime {
	return domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModePromptBridge}
}

func TestCertifyRankedCandidates_CertifiesTheEmbeddedWritingPlansArchivist(t *testing.T) {
	defaultsRoot := t.TempDir()
	writeRoleContractFixture(t, defaultsRoot, "archivist")
	catalog := pluginCatalog{Providers: []pluginCatalogProvider{writingPlansCandidate("1.0.0", embeddedPromptBridge())}}

	require.NoError(t, certifyRankedCandidates(&catalog, defaultsRoot))

	provider, ok := findCatalogProviderVersion(catalog, "writing-plans", "1.0.0")
	require.True(t, ok)
	assert.True(t, provider.Ranked)
	assert.NotEmpty(t, provider.CertificationDigest)
	assert.Equal(t, int64(1), provider.RankedBindingGeneration)
	assert.Equal(t, "active", provider.RankedBindingStatus)
	assert.False(t, provider.Default, "certification never makes a Weapon the default")
}

func TestCertifyRankedCandidates_RejectsAHostRuntimeArchivist(t *testing.T) {
	defaultsRoot := t.TempDir()
	writeRoleContractFixture(t, defaultsRoot, "archivist")
	hostRuntime := domain.WeaponRuntime{Kind: domain.RankedRuntimeHost, HostAPI: "strategist-host-skill/v1"}
	catalog := pluginCatalog{Providers: []pluginCatalogProvider{writingPlansCandidate("1.0.0", hostRuntime)}}

	err := certifyRankedCandidates(&catalog, defaultsRoot)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "runtime kind")
	assert.False(t, catalog.Providers[0].Ranked, "a rejected candidate is never stamped")
}

func TestCertifyRankedCandidates_DoesNotCertifyAnUnpinnedWritingPlansVersion(t *testing.T) {
	defaultsRoot := t.TempDir()
	writeRoleContractFixture(t, defaultsRoot, "archivist")
	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		writingPlansCandidate("1.0.0", embeddedPromptBridge()),
		writingPlansCandidate("2.0.0", embeddedPromptBridge()),
	}}

	require.NoError(t, certifyRankedCandidates(&catalog, defaultsRoot))

	pinned, _ := findCatalogProviderVersion(catalog, "writing-plans", "1.0.0")
	other, _ := findCatalogProviderVersion(catalog, "writing-plans", "2.0.0")
	assert.True(t, pinned.Ranked)
	assert.False(t, other.Ranked, "an unpinned version is only selectable as Custom")
}

func TestCertifyRankedCandidates_KeepsBothArchivistPairingsIndependent(t *testing.T) {
	defaultsRoot := t.TempDir()
	writeRoleContractFixture(t, defaultsRoot, "archivist")
	openspec := pluginCatalogProvider{
		ID: "openspec-propose", Version: "1.0", RiskScore: "write_analysis", CanonicalRole: "archivist", Roles: []string{"archivist"}, Default: true,
		UpstreamRepo: "Fission-AI/OpenSpec", UpstreamSkillPath: "skills/openspec-propose/SKILL.md", UpstreamVersion: "1.10.0",
		UpstreamCommit: "1ebddd17f40dde15dfd28289e4493c3cf05ee9df", UpstreamContentDigest: "sha256:c0537ce311115878e7e0a04e6ff4fc6456056f21024079c228b6b24325e38613", License: "MIT",
	}
	catalog := pluginCatalog{Providers: []pluginCatalogProvider{openspec, writingPlansCandidate("1.0.0", embeddedPromptBridge())}}

	require.NoError(t, certifyRankedCandidates(&catalog, defaultsRoot))

	openspecOut, _ := findCatalogProvider(catalog, "openspec-propose")
	writingPlans, _ := findCatalogProvider(catalog, "writing-plans")
	assert.True(t, openspecOut.Ranked)
	assert.True(t, writingPlans.Ranked)
	assert.True(t, openspecOut.Default, "openspec-propose stays the default Archivist")
	assert.NotEqual(t, openspecOut.CertificationDigest, writingPlans.CertificationDigest)
}
