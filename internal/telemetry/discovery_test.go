package telemetry_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiscoveryWeaponEventRecordsSuccessfulBoundary(t *testing.T) {
	t.Parallel()

	event := telemetry.NewDiscoveryWeaponEvent(
		"mission-1", "brainstorming", ".analysis/pending/mission-1-analysis.md",
		telemetry.DiscoveryInvocationInvoked, telemetry.DiscoveryNormalizationNormalized,
		"host-run-42", "", "authenticated", "match", "unverified",
	)

	require.NoError(t, event.Validate())
	assert.Equal(t, telemetry.DiscoveryWeaponEventName, event.Name)
	assert.Equal(t, telemetry.SeverityInfo, event.SeverityNumber)
	assert.Equal(t, "done", event.Attributes[telemetry.AttrStatus])
	assert.Equal(t, telemetry.DiscoveryInvocationInvoked, event.Attributes[telemetry.AttrDiscoveryInvocationStatus])
	assert.Equal(t, telemetry.DiscoveryNormalizationNormalized, event.Attributes[telemetry.AttrDiscoveryNormalization])
	assert.Equal(t, "host-run-42", event.Attributes[telemetry.AttrInvocationEvidence])
	assert.Equal(t, "authenticated", event.Attributes[telemetry.AttrReceiptAuthentication])
	assert.Equal(t, "match", event.Attributes[telemetry.AttrReceiptPinStatus])
	assert.Equal(t, "unverified", event.Attributes[telemetry.AttrCapabilityIsolation])
}

func TestNewDiscoveryWeaponEventRecordsFailClosedBoundaryWithoutPayload(t *testing.T) {
	t.Parallel()

	event := telemetry.NewDiscoveryWeaponEvent(
		"mission-1", "brainstorming", "/secret/mission.md",
		telemetry.DiscoveryInvocationFailed, telemetry.DiscoveryNormalizationNotAttempted,
		"", "role_invocation_failed", "failed", "not_checked", "unverified",
	)

	require.NoError(t, event.Validate())
	assert.Equal(t, telemetry.SeverityError, event.SeverityNumber)
	assert.Equal(t, "blocked", event.Attributes[telemetry.AttrStatus])
	assert.Equal(t, "<redacted-path>", event.Attributes[telemetry.AttrArtifactPath])
	assert.Equal(t, "role_invocation_failed", event.Attributes[telemetry.AttrReason])
	assert.NotContains(t, event.Attributes, telemetry.AttrInvocationEvidence)
}
