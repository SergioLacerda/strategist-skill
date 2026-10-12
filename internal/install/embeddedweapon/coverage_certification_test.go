package embeddedweapon

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rangerCandidate() plugincatalog.Provider {
	return plugincatalog.Provider{
		ID: "brainstorming", Version: "1.0.0", Roles: []string{"ranger"}, CanonicalRole: "ranger",
		CompatibilitySource: "native_role",
	}
}

func TestCertifyRankedCandidateFailureModes(t *testing.T) {
	t.Parallel()
	defaults := filepath.Join(RepoRoot(), "internal", "embed", "defaults")

	absent := &plugincatalog.PluginCatalog{Providers: []plugincatalog.Provider{{ID: "other", Version: "1"}}}
	require.NoError(t, CertifyRankedCandidate(absent, defaults, "brainstorming@1.0.0", "ranger"), "an absent pairing is skipped")
	assert.Equal(t, -1, IndexOfCatalogProviderIdentity(absent.Providers, "brainstorming@1.0.0"))

	wrongRole := rangerCandidate()
	wrongRole.Roles, wrongRole.CanonicalRole = []string{"archivist"}, "archivist"
	err := CertifyRankedCandidate(&plugincatalog.PluginCatalog{Providers: []plugincatalog.Provider{wrongRole}}, defaults, "brainstorming@1.0.0", "ranger")
	require.ErrorContains(t, err, "certify ranked candidate")
	require.ErrorContains(t, err, "role affinity missing")

	external := rangerCandidate()
	external.CompatibilitySource = "embedded"
	err = CertifyRankedCandidate(&plugincatalog.PluginCatalog{Providers: []plugincatalog.Provider{external}}, defaults, "brainstorming@1.0.0", "ranger")
	require.Error(t, err)

	err = CertifyRankedCandidate(&plugincatalog.PluginCatalog{Providers: []plugincatalog.Provider{rangerCandidate()}}, t.TempDir(), "brainstorming@1.0.0", "ranger")
	require.Error(t, err, "missing role contract files cannot be digested")
	_, err = rankedConformanceEvidence(t.TempDir(), "ranger", "brainstorming")
	require.Error(t, err)
	_, err = rankedConformanceEvidence(defaults, "scout", "x")
	require.ErrorContains(t, err, "no conformance test file pinned")
}

func TestCertifyRankedCandidateStampsEvidence(t *testing.T) {
	t.Parallel()
	defaults := filepath.Join(RepoRoot(), "internal", "embed", "defaults")
	catalog := &plugincatalog.PluginCatalog{Providers: []plugincatalog.Provider{rangerCandidate()}}
	require.NoError(t, CertifyRankedCandidate(catalog, defaults, "brainstorming@1.0.0", "ranger"))
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
	require.ErrorContains(t, ValidateRankedCandidate(scout, "scout"), "declares no handoff schema")

	badRuntime := rangerCandidate()
	badRuntime.Runtime.Kind = "bogus"
	require.ErrorContains(t, ValidateRankedCandidate(badRuntime, "ranger"), "runtime contract invalid")
	require.NoError(t, ValidateRankedCandidate(rangerCandidate(), "ranger"))
}
