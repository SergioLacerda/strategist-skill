package treasurecli

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/roles/jeweler"
	"github.com/stretchr/testify/require"
)

func TestTreasureChestProviderQueryFailsClosedWithoutExternalBinding(t *testing.T) {
	root := t.TempDir()
	previousRoot := treasureChestRootFromCmd(treasureChestProviderQueryCmd)
	treasureChestCmd.PersistentFlags().Lookup(flagRoot).Value.Set(root)
	t.Cleanup(func() {
		_ = treasureChestCmd.PersistentFlags().Lookup(flagRoot).Value.Set(previousRoot)
	})

	err := runTreasureChestProviderQuery(treasureChestProviderQueryCmd, treasureChestProviderQueryOptions{
		treasureChestProviderOptions: treasureChestProviderOptions{
			Package:    "treasure-chest-tmp",
			Version:    "0.1.0",
			API:        "knowledge-api/v1",
			Digest:     "sha256:test",
			Provenance: "test-only",
		},
		Requester:   "ranger",
		MissionID:   "mission-test",
		Correlation: "corr-test",
		Limit:       1,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "external provider executable is not configured")
	require.ErrorContains(t, err, "verify the explicit provider binding")
}

func TestTreasureChestProviderAdmissionRequiresPinnedIdentity(t *testing.T) {
	identity := jeweler.AdmittedTreasureChestTMPIdentity()
	evidence := jeweler.TreasureChestTMPAdmissionEvidence(identity, "/bin/treasure-chest")
	require.Equal(t, jeweler.EvidenceCertified, evidence.Package)
	require.Equal(t, jeweler.EvidenceCertified, evidence.Identity)
	require.Equal(t, jeweler.EvidenceCertified, evidence.Provenance)
	require.Equal(t, jeweler.EvidenceCertified, evidence.API)
	require.Equal(t, jeweler.EvidenceCertified, evidence.Delegate)

	identity.Digest = "sha256:stale"
	evidence = jeweler.TreasureChestTMPAdmissionEvidence(identity, "/bin/treasure-chest")
	require.Equal(t, jeweler.EvidenceBlocked, evidence.Identity)

	identity = jeweler.AdmittedTreasureChestTMPIdentity()
	identity.Provenance = "mutable/main"
	evidence = jeweler.TreasureChestTMPAdmissionEvidence(identity, "/bin/treasure-chest")
	require.Equal(t, jeweler.EvidenceBlocked, evidence.Provenance)
}

func TestTreasureChestProviderAdmissionDoesNotCertifyMissingBinding(t *testing.T) {
	evidence := jeweler.TreasureChestTMPAdmissionEvidence(jeweler.AdmittedTreasureChestTMPIdentity(), "")
	require.Equal(t, jeweler.EvidenceUnknown, evidence.Delegate)
}
