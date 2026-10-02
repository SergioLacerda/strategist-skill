package mission

import (
	"context"
	"fmt"
	"path/filepath"

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
	return InvokeRangerDiscoveryWithVerification(ctx, missionID, providerID, artifactPath, instance, connector, provider.FileReceiptNonceStore{Root: filepath.Join(filepath.Dir(artifactPath), ".invocation-receipts")}, filepath.Join(".strategist", "plugins", "catalog.yaml"), sink, runID)
}

// InvokeRangerDiscoveryWithReceiptStore permits the mission runtime to supply
// durable nonce storage while keeping hermetic tests independent of the host.
func InvokeRangerDiscoveryWithReceiptStore(ctx context.Context, missionID, providerID, artifactPath string, instance domain.InstalledInstance, connector connectors.RuntimeConnector, receiptStore provider.ReceiptNonceStore, sink telemetry.EventSink, runID string) (provider.NormalizedDiscoveryArtifact, error) {
	return InvokeRangerDiscoveryWithVerification(ctx, missionID, providerID, artifactPath, instance, connector, receiptStore, "", sink, runID)
}

// InvokeRangerDiscoveryWithVerification admits an explicit catalog path for
// hosts whose active binding is materialized outside the workspace default.
func InvokeRangerDiscoveryWithVerification(ctx context.Context, missionID, providerID, artifactPath string, instance domain.InstalledInstance, connector connectors.RuntimeConnector, receiptStore provider.ReceiptNonceStore, catalogPath string, sink telemetry.EventSink, runID string) (provider.NormalizedDiscoveryArtifact, error) {
	artifact, err := provider.InvokeDiscoveryViaConnector(ctx, provider.DiscoveryWeaponRequest{
		MissionID: missionID, Role: "ranger", Slot: string(domain.SlotDiscovery),
		ProviderID: providerID, ArtifactPath: artifactPath, ReceiptStore: receiptStore, CatalogPath: catalogPath,
	}, instance, connector, sink, runID)
	if err != nil {
		return provider.NormalizedDiscoveryArtifact{}, fmt.Errorf("invoke Ranger discovery Weapon: %w", err)
	}
	return artifact, nil
}

// InvokeRangerDiscoveryWithRoleWeaponBinding is the binding-aware mission
// entrypoint. Ranked execution can only use the supplied compiled Embedded
// dispatch; Custom execution can only use its explicit connector.
func InvokeRangerDiscoveryWithRoleWeaponBinding(ctx context.Context, request provider.DiscoveryWeaponRequest, instance domain.InstalledInstance, binding domain.RoleWeaponBinding, embedded connectors.EmbeddedWeaponDispatch, custom connectors.RuntimeConnector, sink telemetry.EventSink, runID string) (provider.NormalizedDiscoveryArtifact, error) {
	artifact, err := provider.InvokeDiscoveryViaRoleWeaponBinding(ctx, request, instance, binding, embedded, custom, sink, runID)
	if err != nil {
		return provider.NormalizedDiscoveryArtifact{}, fmt.Errorf("invoke Ranger discovery via Role Weapon binding: %w", err)
	}
	return artifact, nil
}
