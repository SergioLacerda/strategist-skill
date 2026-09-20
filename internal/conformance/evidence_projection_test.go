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
