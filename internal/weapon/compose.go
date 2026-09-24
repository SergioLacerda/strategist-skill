package weapon

import (
	"context"
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// ConnectorResolver selects the runtime connector declared for one component.
// It is supplied by the host/runtime layer, not discovered through fallback.
type ConnectorResolver func(domain.WeaponManifest) (connectors.RuntimeConnector, error)

// InvocationRequest is the Role-owned context propagated to every component.
type InvocationRequest struct {
	MissionID    string
	Role         string
	Slot         string
	ArtifactPath string
	Entrypoint   string
	RunID        string
	Sink         telemetry.EventSink
}

// ComponentOutcome records untrusted component output and evidence. The Role
// must normalize artifacts before treating the aggregate as a handoff.
type ComponentOutcome struct {
	ID                 string
	Required           bool
	Status             domain.ReadinessStatus
	ReasonCode         string
	Artifact           []byte
	InvocationEvidence string
}

// CompositeOutcome is an evidence-bearing aggregate, never a handoff itself.
type CompositeOutcome struct {
	WeaponID           string
	ParentInvocationID string
	Components         []ComponentOutcome
	Degraded           bool
}

// Invoke resolves each component through its declared connector in the
// deterministic order produced by Resolve. Required failures block; optional
// failures degrade explicitly, except incompatibility which always blocks.
func Invoke(ctx context.Context, resolved ResolvedWeapon, request InvocationRequest, resolveConnector ConnectorResolver) (CompositeOutcome, error) {
	if resolveConnector == nil {
		return CompositeOutcome{}, fmt.Errorf("weapon invocation connector resolver is unavailable")
	}
	parentID := request.MissionID + "/" + resolved.Manifest.ID
	outcome := CompositeOutcome{WeaponID: resolved.Manifest.ID, ParentInvocationID: parentID}
	for _, component := range invocationComponents(resolved) {
		componentOutcome, err := invokeComponent(ctx, resolved.Manifest.ID, parentID, component, request, resolveConnector)
		outcome.record(component, componentOutcome, err)
		if telemetryErr := emitComponentEvent(ctx, request, resolved.Manifest.ID, parentID, component, componentOutcome, outcome.Degraded); telemetryErr != nil {
			return outcome, telemetryErr
		}
		if blocksInvocation(component, componentOutcome, err) {
			return outcome, fmt.Errorf("weapon %q component %q blocked: %w", resolved.Manifest.ID, component.Manifest.ID, err)
		}
	}
	return outcome, nil
}

// invocationComponents treats an atomic Weapon as its own single required
// component.
func invocationComponents(resolved ResolvedWeapon) []ResolvedComponent {
	if resolved.Manifest.Kind == domain.WeaponKindAtomic {
		return []ResolvedComponent{{Manifest: resolved.Manifest, Required: true}}
	}
	return resolved.Components
}

func isIncompatible(outcome ComponentOutcome) bool {
	return strings.HasPrefix(outcome.ReasonCode, "incompatible")
}

// record appends a component outcome; an optional, compatible failure marks
// the aggregate degraded.
func (o *CompositeOutcome) record(component ResolvedComponent, outcome ComponentOutcome, err error) {
	o.Components = append(o.Components, outcome)
	if err != nil && !component.Required && !isIncompatible(outcome) {
		o.Degraded = true
	}
}

// blocksInvocation reports a failure that stops the composite: any required
// component failure, or incompatibility of any component.
func blocksInvocation(component ResolvedComponent, outcome ComponentOutcome, err error) bool {
	return err != nil && (isIncompatible(outcome) || component.Required)
}

func emitComponentEvent(ctx context.Context, request InvocationRequest, weaponID, parentID string, component ResolvedComponent, outcome ComponentOutcome, degraded bool) error {
	if request.Sink == nil {
		return nil
	}
	runID := request.RunID
	if runID == "" {
		runID = request.MissionID
	}
	event := telemetry.NewCompositeWeaponEvent(runID, request.Role, request.Slot, weaponID, component.Manifest.ID, parentID, string(outcome.Status), outcome.ReasonCode, outcome.InvocationEvidence, component.Required, degraded)
	if err := request.Sink.Emit(ctx, event); err != nil {
		return fmt.Errorf("emit Weapon component telemetry: %w", err)
	}
	return nil
}

func invokeComponent(ctx context.Context, weaponID, parentID string, component ResolvedComponent, request InvocationRequest, resolveConnector ConnectorResolver) (ComponentOutcome, error) {
	result := ComponentOutcome{ID: component.Manifest.ID, Required: component.Required, Status: domain.ReadinessBlocked}
	connector, err := invocableConnector(ctx, component.Manifest, resolveConnector)
	if err != nil {
		result.ReasonCode = "runtime_unavailable"
		return result, err
	}
	connectorResult := connector.Invoke(ctx, invocationEnvelope(weaponID, parentID, component, request))
	result.Status = connectorResult.Status
	result.ReasonCode = connectorResult.ReasonCode
	result.Artifact = connectorResult.Artifact
	result.InvocationEvidence = strings.TrimSpace(connectorResult.InvocationEvidence)
	return result, checkInvocationResult(&result, connectorResult, component.Manifest.ID)
}

func invocableConnector(ctx context.Context, manifest domain.WeaponManifest, resolveConnector ConnectorResolver) (connectors.RuntimeConnector, error) {
	connector, err := resolveConnector(manifest)
	if err != nil {
		return nil, err
	}
	if connector == nil {
		return nil, fmt.Errorf("connector is nil")
	}
	if capabilities := connector.Capabilities(ctx); !capabilities.CanInvoke {
		return nil, fmt.Errorf("connector %q cannot invoke", capabilities.ConnectorID)
	}
	return connector, nil
}

func invocationEnvelope(weaponID, parentID string, component ResolvedComponent, request InvocationRequest) connectors.InvocationEnvelope {
	entrypoint := request.Entrypoint
	if entrypoint == "" {
		entrypoint = "invoke"
	}
	return connectors.InvocationEnvelope{
		SchemaVersion: "weapon-invocation/v1",
		Instance:      domain.InstalledInstance{ID: component.Manifest.ID, State: "active"},
		WeaponID:      weaponID, ComponentID: component.Manifest.ID, ParentInvocationID: parentID,
		HostAPI: component.Manifest.Runtime.HostAPI,
		Role:    request.Role, Slot: request.Slot, Entrypoint: entrypoint,
		MissionID: request.MissionID, ArtifactPath: request.ArtifactPath, WriteScope: request.ArtifactPath,
	}
}

// checkInvocationResult rejects a non-ready status, missing evidence, or a
// result that does not carry the component's identity, setting the reason
// code on result.
func checkInvocationResult(result *ComponentOutcome, connectorResult connectors.ConnectorResult, componentID string) error {
	if connectorResult.Status != domain.ReadinessReady {
		if result.ReasonCode == "" {
			result.ReasonCode = "invocation_failed"
		}
		return fmt.Errorf("status=%s reason=%s", connectorResult.Status, result.ReasonCode)
	}
	if result.InvocationEvidence == "" {
		result.ReasonCode = "role_invocation_failed"
		return fmt.Errorf("invocation evidence is required")
	}
	if connectorResult.ProviderID != "" && connectorResult.ProviderID != componentID {
		result.ReasonCode = "incompatible_identity"
		return fmt.Errorf("result identity %q does not match component %q", connectorResult.ProviderID, componentID)
	}
	return nil
}
