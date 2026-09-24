package mission

import (
	"context"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/provider"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// InvokeRangerDiscovery is the mission boundary for the selected discovery
// Weapon. The mission supplies only identity and write scope; the provider
// adapter performs host invocation, Ranger normalization, and fail-closed
// evidence checks.
func InvokeRangerDiscovery(ctx context.Context, missionID, providerID, artifactPath string, instance domain.InstalledInstance, connector connectors.RuntimeConnector, sink telemetry.EventSink, runID string) (provider.NormalizedDiscoveryArtifact, error) {
	artifact, err := provider.InvokeDiscoveryViaConnector(ctx, provider.DiscoveryWeaponRequest{
		MissionID: missionID, Role: "ranger", Slot: string(domain.SlotDiscovery),
		ProviderID: providerID, ArtifactPath: artifactPath,
	}, instance, connector, sink, runID)
	if err != nil {
		return provider.NormalizedDiscoveryArtifact{}, fmt.Errorf("invoke Ranger discovery Weapon: %w", err)
	}
	return artifact, nil
}
