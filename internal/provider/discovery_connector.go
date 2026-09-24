package provider

import (
	"context"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// InvokeDiscoveryViaConnector adapts the governed RuntimeConnector SPI to the
// Ranger normalization boundary. The connector owns host/process invocation;
// Ranger still owns identity checks, normalization, write scope and evidence.
func InvokeDiscoveryViaConnector(ctx context.Context, request DiscoveryWeaponRequest, instance domain.InstalledInstance, connector connectors.RuntimeConnector, sink telemetry.EventSink, runID string) (NormalizedDiscoveryArtifact, error) {
	if connector == nil {
		return InvokeAndNormalizeDiscoveryWithTelemetry(ctx, request, nil, sink, runID)
	}
	return InvokeAndNormalizeDiscoveryWithTelemetry(ctx, request, func(invokeCtx context.Context, invokeRequest DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error) {
		if instance.ID != invokeRequest.ProviderID {
			return DiscoveryWeaponResponse{}, fmt.Errorf("connector instance %q does not match selected Weapon %q", instance.ID, invokeRequest.ProviderID)
		}
		capabilities := connector.Capabilities(invokeCtx)
		if !capabilities.CanInvoke {
			return DiscoveryWeaponResponse{}, fmt.Errorf("connector %q cannot invoke discovery Weapon", capabilities.ConnectorID)
		}
		result := connector.Invoke(invokeCtx, connectors.InvocationEnvelope{
			Instance:     instance,
			Role:         invokeRequest.Role,
			Slot:         invokeRequest.Slot,
			Entrypoint:   "discover",
			MissionID:    invokeRequest.MissionID,
			ArtifactPath: invokeRequest.ArtifactPath,
		})
		if result.Status != domain.ReadinessReady {
			return DiscoveryWeaponResponse{}, fmt.Errorf("connector invocation blocked: status=%s reason=%s", result.Status, result.ReasonCode)
		}
		return DiscoveryWeaponResponse{
			ProviderID:         result.ProviderID,
			InvocationEvidence: result.InvocationEvidence,
			Artifact:           result.Artifact,
		}, nil
	}, sink, runID)
}
