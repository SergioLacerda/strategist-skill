package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestReadinessCheckReady(t *testing.T) {
	t.Parallel()

	assert.True(t, domain.ReadinessCheck{Status: domain.ReadinessReady}.Ready())
	assert.False(t, domain.ReadinessCheck{Status: domain.ReadinessBlocked}.Ready())
}

func allReadyVector() domain.PluginReadinessVector {
	ready := domain.ReadinessCheck{Status: domain.ReadinessReady}
	return domain.PluginReadinessVector{
		Descriptor:          ready,
		Source:              ready,
		Trust:               ready,
		Dependencies:        ready,
		HostAPI:             ready,
		Connector:           ready,
		Entrypoint:          ready,
		PermissionGrant:     ready,
		EnforcementCoverage: ready,
		ActiveBinding:       ready,
	}
}

func TestPluginReadinessVectorReadyRequiresEveryDimension(t *testing.T) {
	t.Parallel()

	vector := allReadyVector()
	assert.True(t, vector.Ready())

	vector.Trust = domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "trust_blocked"}
	assert.False(t, vector.Ready())
}

func TestPluginReadinessVectorReasonCodesCollectsNonReadyOnly(t *testing.T) {
	t.Parallel()

	vector := allReadyVector()
	vector.Trust = domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "trust_blocked"}
	vector.HostAPI = domain.ReadinessCheck{Status: domain.ReadinessUnsupported}
	vector.Connector = domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "connector_unknown"}

	assert.Equal(t, []string{"trust_blocked", "connector_unknown"}, vector.ReasonCodes())
}

func TestPluginReadinessVectorReasonCodesEmptyWhenAllReady(t *testing.T) {
	t.Parallel()

	assert.Empty(t, allReadyVector().ReasonCodes())
}
