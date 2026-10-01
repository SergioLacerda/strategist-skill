package install

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rangerCandidate() pluginCatalogProvider {
	return pluginCatalogProvider{
		ID: "brainstorming", Version: "1.0.0", Roles: []string{"ranger"}, CanonicalRole: "ranger",
		CompatibilitySource: "native_role",
	}
}

func TestCertifyRankedCandidateFailureModes(t *testing.T) {
	t.Parallel()
	defaults := filepath.Join(repoRoot(), "internal", "embed", "defaults")

	absent := &pluginCatalog{Providers: []pluginCatalogProvider{{ID: "other", Version: "1"}}}
	require.NoError(t, certifyRankedCandidate(absent, defaults, "brainstorming@1.0.0", "ranger"), "an absent pairing is skipped")
	assert.Equal(t, -1, indexOfCatalogProviderIdentity(absent.Providers, "brainstorming@1.0.0"))

	wrongRole := rangerCandidate()
	wrongRole.Roles, wrongRole.CanonicalRole = []string{"archivist"}, "archivist"
	err := certifyRankedCandidate(&pluginCatalog{Providers: []pluginCatalogProvider{wrongRole}}, defaults, "brainstorming@1.0.0", "ranger")
	require.ErrorContains(t, err, "certify ranked candidate")
	require.ErrorContains(t, err, "role affinity missing")

	external := rangerCandidate()
	external.CompatibilitySource = "embedded"
	err = certifyRankedCandidate(&pluginCatalog{Providers: []pluginCatalogProvider{external}}, defaults, "brainstorming@1.0.0", "ranger")
	require.Error(t, err)

	err = certifyRankedCandidate(&pluginCatalog{Providers: []pluginCatalogProvider{rangerCandidate()}}, t.TempDir(), "brainstorming@1.0.0", "ranger")
	require.Error(t, err, "missing role contract files cannot be digested")
	_, err = rankedConformanceEvidence(t.TempDir(), "ranger", "brainstorming")
	require.Error(t, err)
	_, err = rankedConformanceEvidence(defaults, "scout", "x")
	require.ErrorContains(t, err, "no conformance test file pinned")
}

func TestCertifyRankedCandidateStampsEvidence(t *testing.T) {
	t.Parallel()
	defaults := filepath.Join(repoRoot(), "internal", "embed", "defaults")
	catalog := &pluginCatalog{Providers: []pluginCatalogProvider{rangerCandidate()}}
	require.NoError(t, certifyRankedCandidate(catalog, defaults, "brainstorming@1.0.0", "ranger"))
	stamped := catalog.Providers[0]
	assert.True(t, stamped.Ranked)
	assert.EqualValues(t, 1, stamped.RankedBindingGeneration)
	assert.NotEmpty(t, stamped.CertificationDigest)
	assert.NotEmpty(t, stamped.HostAPIDigest)
	assert.NotEmpty(t, stamped.ConnectorDigest)
	assert.NotEmpty(t, stamped.TestSuiteDigest)
	assert.NotEmpty(t, stamped.PolicyDigest)
}

func TestValidateRankedCandidateRoleAndRuntime(t *testing.T) {
	t.Parallel()
	scout := rangerCandidate()
	scout.Roles, scout.CanonicalRole = []string{"scout"}, "scout"
	require.ErrorContains(t, validateRankedCandidate(scout, "scout"), "declares no handoff schema")

	badRuntime := rangerCandidate()
	badRuntime.Runtime.Kind = "bogus"
	require.ErrorContains(t, validateRankedCandidate(badRuntime, "ranger"), "runtime contract invalid")
	require.NoError(t, validateRankedCandidate(rangerCandidate(), "ranger"))
}
