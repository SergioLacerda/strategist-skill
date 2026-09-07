package treasure

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPromotionPackets_MissingFileReturnsNilNotError(t *testing.T) {
	t.Parallel()
	packets, err := LoadPromotionPackets(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, packets)
}

func TestSaveThenLoadPromotionPackets_RoundTrips(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	created := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	want := []domain.PromotionPacket{
		{
			PacketID:           "runbook-candidate-foo",
			OriginMissionID:    "20260905-refined-implementation-audit",
			TriggerReason:      "recurring pattern observed across 3 missions",
			Procedure:          []string{"step 1", "step 2"},
			VerificationChecks: []string{"check 1"},
			SuggestedOwner:     "unassigned",
			CreatedAt:          created,
			Status:             domain.PromotionPacketStatusPending,
		},
	}

	require.NoError(t, SavePromotionPackets(root, want))
	assert.FileExists(t, filepath.Join(root, "promotion-packets.yaml"))

	got, err := LoadPromotionPackets(root)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, want[0].PacketID, got[0].PacketID)
	assert.Equal(t, want[0].OriginMissionID, got[0].OriginMissionID)
	assert.Equal(t, want[0].TriggerReason, got[0].TriggerReason)
	assert.Equal(t, want[0].Procedure, got[0].Procedure)
	assert.Equal(t, want[0].VerificationChecks, got[0].VerificationChecks)
	assert.Equal(t, want[0].SuggestedOwner, got[0].SuggestedOwner)
	assert.True(t, want[0].CreatedAt.Equal(got[0].CreatedAt))
	assert.Equal(t, want[0].Status, got[0].Status)
	assert.Nil(t, got[0].ExpiresAt)
}

func TestSavePromotionPackets_ReplacesExistingContent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	first := []domain.PromotionPacket{{PacketID: "a", Status: domain.PromotionPacketStatusPending}}
	second := []domain.PromotionPacket{{PacketID: "b", Status: domain.PromotionPacketStatusPending}}

	require.NoError(t, SavePromotionPackets(root, first))
	require.NoError(t, SavePromotionPackets(root, second))

	got, err := LoadPromotionPackets(root)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "b", got[0].PacketID)
}

func TestLoadPromotionPackets_RejectsUnsupportedSchemaVersion(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, SavePromotionPackets(root, nil))
	// Overwrite with an unsupported schema_version.
	path := filepath.Join(root, "promotion-packets.yaml")
	require.NoError(t, writeFileAtomic(path, []byte("schema_version: \"99\"\npackets: []\n"), 0o644))

	_, err := LoadPromotionPackets(root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema_version")
}

func TestAddPromotionPacket_AppendsNewPacket(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, AddPromotionPacket(root, domain.PromotionPacket{PacketID: "a", Status: domain.PromotionPacketStatusPending}))
	require.NoError(t, AddPromotionPacket(root, domain.PromotionPacket{PacketID: "b", Status: domain.PromotionPacketStatusPending}))

	got, err := LoadPromotionPackets(root)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestAddPromotionPacket_RejectsDuplicatePacketID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, AddPromotionPacket(root, domain.PromotionPacket{PacketID: "dup", Status: domain.PromotionPacketStatusPending}))

	err := AddPromotionPacket(root, domain.PromotionPacket{PacketID: "dup", Status: domain.PromotionPacketStatusPending})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dup")

	got, err := LoadPromotionPackets(root)
	require.NoError(t, err)
	assert.Len(t, got, 1, "a rejected duplicate must not be appended")
}

func TestPromotionPacketExpiresAt_RoundTrips(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	expires := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	require.NoError(t, SavePromotionPackets(root, []domain.PromotionPacket{
		{PacketID: "a", Status: domain.PromotionPacketStatusPending, ExpiresAt: &expires},
	}))

	got, err := LoadPromotionPackets(root)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.NotNil(t, got[0].ExpiresAt)
	assert.True(t, expires.Equal(*got[0].ExpiresAt))
}
