package conformance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectEvidenceRequiresAuthorityAndLiveCertification(t *testing.T) {
	projection, err := ProjectEvidence(AuthorityRankedCatalog, "provider", "catalog:v1", StateCertified, false)
	require.NoError(t, err)
	require.Equal(t, StateUnknown, projection.State)

	_, err = ProjectEvidence("", "provider", "catalog:v1", StateCertified, true)
	require.ErrorContains(t, err, "unknown authority")
}

func TestRejectAuthorityConflictFailsClosed(t *testing.T) {
	custom, err := ProjectEvidence(AuthorityCustomLock, "custom", "lock:v1", StateUnknown, false)
	require.NoError(t, err)
	ranked, err := ProjectEvidence(AuthorityRankedCatalog, "ranked", "catalog:v1", StateCertified, true)
	require.NoError(t, err)
	require.ErrorContains(t, RejectAuthorityConflict(custom, ranked), "authority conflict")
}

func TestProjectEvidenceRequiresProvenanceProviderAndKnownState(t *testing.T) {
	_, err := ProjectEvidence(AuthorityRankedCatalog, "", "catalog:v1", StateUnknown, false)
	require.ErrorContains(t, err, "provider, provenance, and state are required")
	_, err = ProjectEvidence(AuthorityRankedCatalog, "provider", "", StateUnknown, false)
	require.ErrorContains(t, err, "provider, provenance, and state are required")
	_, err = ProjectEvidence(AuthorityCustomLock, "provider", "lock:v1", EvidenceState("bogus"), false)
	require.ErrorContains(t, err, "provider, provenance, and state are required")
}

func TestProjectEvidencePreservesNonCertifiedStatesAndAuthority(t *testing.T) {
	for _, state := range []EvidenceState{StateUnknown, StateUnavailable, StateFailed, StateStale, StateUnauthorized} {
		projection, err := ProjectEvidence(AuthorityCustomLock, "custom", "lock:v1", state, false)
		require.NoError(t, err)
		require.Equal(t, state, projection.State, "static/unavailable/failed states must not be rewritten")
		require.Equal(t, AuthorityCustomLock, projection.Authority)
		require.False(t, projection.Live)
	}
	live, err := ProjectEvidence(AuthorityRankedCatalog, "ranked", "catalog:v1", StateCertified, true)
	require.NoError(t, err)
	require.Equal(t, StateCertified, live.State, "only live evidence may stay certified")
}

func TestRejectAuthorityConflictAcceptsSameProviderAndRejectsSwappedAuthorities(t *testing.T) {
	custom, err := ProjectEvidence(AuthorityCustomLock, "same", "lock:v1", StateUnknown, false)
	require.NoError(t, err)
	ranked, err := ProjectEvidence(AuthorityRankedCatalog, "same", "catalog:v1", StateCertified, true)
	require.NoError(t, err)
	require.NoError(t, RejectAuthorityConflict(custom, ranked))
	require.ErrorContains(t, RejectAuthorityConflict(ranked, custom), "expected Custom and Ranked authorities")
}
