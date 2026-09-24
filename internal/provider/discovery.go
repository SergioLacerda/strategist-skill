package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// DiscoveryArtifactSchemaVersion identifies the Ranger-owned normalized
// artifact envelope. The provider payload remains opaque Markdown inside it.
const DiscoveryArtifactSchemaVersion = "strategist-ranger-discovery/v1"

const maxDiscoveryArtifactBytes = 4 << 20

// DiscoveryWeaponRequest is the trusted invocation context supplied by
// Ranger. A Weapon cannot replace these identity fields through its response.
type DiscoveryWeaponRequest struct {
	MissionID    string
	Role         string
	Slot         string
	ProviderID   string
	ArtifactPath string
}

// DiscoveryWeaponResponse is the untrusted result returned by a host skill
// loader. InvocationEvidence is supplied by that host boundary; a provider
// response alone never certifies that invocation occurred.
type DiscoveryWeaponResponse struct {
	ProviderID         string
	InvocationEvidence string
	Artifact           []byte
}

// DiscoveryWeaponInvoker is the narrow host integration point. The CLI does
// not implement it or start a provider process; an authorized host supplies
// the adapter when it can actually invoke the selected Weapon.
type DiscoveryWeaponInvoker func(context.Context, DiscoveryWeaponRequest) (DiscoveryWeaponResponse, error)

// NormalizedDiscoveryArtifact is Ranger-owned output ready for the pending
// analysis write boundary. Its content has canonical identity metadata and
// preserves the provider's Markdown body as untrusted content.
type NormalizedDiscoveryArtifact struct {
	MissionID          string
	ProviderID         string
	ArtifactPath       string
	InvocationEvidence string
	Content            []byte
}

// InvokeAndNormalizeDiscovery invokes the selected Weapon through the host
// adapter and normalizes its untrusted result. It never substitutes native
// Ranger behavior when the adapter is missing, fails, or returns an
// incompatible result.
func InvokeAndNormalizeDiscovery(ctx context.Context, request DiscoveryWeaponRequest, invoke DiscoveryWeaponInvoker) (NormalizedDiscoveryArtifact, error) {
	return invokeAndNormalizeDiscovery(ctx, request, invoke, nil, "")
}

// InvokeAndNormalizeDiscoveryWithTelemetry is the host-facing variant that
// records the Ranger/Weapon boundary outcome. Telemetry is optional for
// compatibility, but when supplied its delivery is part of the fail-closed
// boundary: a successful artifact is not returned if its evidence event cannot
// be persisted.
func InvokeAndNormalizeDiscoveryWithTelemetry(ctx context.Context, request DiscoveryWeaponRequest, invoke DiscoveryWeaponInvoker, sink telemetry.EventSink, runID string) (NormalizedDiscoveryArtifact, error) {
	return invokeAndNormalizeDiscovery(ctx, request, invoke, sink, runID)
}

func invokeAndNormalizeDiscovery(ctx context.Context, request DiscoveryWeaponRequest, invoke DiscoveryWeaponInvoker, sink telemetry.EventSink, runID string) (NormalizedDiscoveryArtifact, error) {
	if runID == "" {
		runID = request.MissionID
	}
	fail := func(normalizationStatus string, err error) (NormalizedDiscoveryArtifact, error) {
		if telemetryErr := emitDiscoveryTelemetry(ctx, sink, runID, request, telemetry.DiscoveryInvocationFailed, normalizationStatus, ""); telemetryErr != nil {
			return NormalizedDiscoveryArtifact{}, invocationFailure(fmt.Errorf("emit discovery telemetry: %w", telemetryErr))
		}
		return NormalizedDiscoveryArtifact{}, invocationFailure(err)
	}
	response, status, err := callDiscoveryWeapon(ctx, request, invoke)
	if err != nil {
		return fail(status, err)
	}
	evidence := response.InvocationEvidence
	content, err := normalizeDiscoveryArtifact(request, response)
	if err != nil {
		return fail(telemetry.DiscoveryNormalizationRejected, err)
	}
	if telemetryErr := emitDiscoveryTelemetry(ctx, sink, runID, request, telemetry.DiscoveryInvocationInvoked, telemetry.DiscoveryNormalizationNormalized, response.InvocationEvidence); telemetryErr != nil {
		return NormalizedDiscoveryArtifact{}, invocationFailure(fmt.Errorf("emit discovery telemetry: %w", telemetryErr))
	}
	return NormalizedDiscoveryArtifact{
		MissionID:          request.MissionID,
		ProviderID:         request.ProviderID,
		ArtifactPath:       request.ArtifactPath,
		InvocationEvidence: evidence,
		Content:            content,
	}, nil
}

// callDiscoveryWeapon validates the request, invokes the host adapter and checks
// the response identity and evidence. The returned status is the normalization
// status to record if the call fails.
func callDiscoveryWeapon(ctx context.Context, request DiscoveryWeaponRequest, invoke DiscoveryWeaponInvoker) (DiscoveryWeaponResponse, string, error) {
	if err := validateDiscoveryRequest(request); err != nil {
		return DiscoveryWeaponResponse{}, telemetry.DiscoveryNormalizationNotAttempted, err
	}
	if invoke == nil {
		return DiscoveryWeaponResponse{}, telemetry.DiscoveryNormalizationNotAttempted, fmt.Errorf("discovery Weapon invoker is unavailable")
	}
	response, err := invoke(ctx, request)
	if err != nil {
		return DiscoveryWeaponResponse{}, telemetry.DiscoveryNormalizationNotAttempted, err
	}
	if response.ProviderID != request.ProviderID {
		return DiscoveryWeaponResponse{}, telemetry.DiscoveryNormalizationRejected, fmt.Errorf("provider identity mismatch: expected %q, got %q", request.ProviderID, response.ProviderID)
	}
	response.InvocationEvidence = strings.TrimSpace(response.InvocationEvidence)
	if response.InvocationEvidence == "" {
		return DiscoveryWeaponResponse{}, telemetry.DiscoveryNormalizationRejected, fmt.Errorf("invocation evidence is required")
	}
	return response, "", nil
}

func emitDiscoveryTelemetry(ctx context.Context, sink telemetry.EventSink, runID string, request DiscoveryWeaponRequest, invocationStatus, normalizationStatus, evidence string) error {
	if sink == nil {
		return nil
	}
	reason := "role_invocation_failed"
	if invocationStatus == telemetry.DiscoveryInvocationInvoked && normalizationStatus == telemetry.DiscoveryNormalizationNormalized {
		reason = ""
	}
	event := telemetry.NewDiscoveryWeaponEvent(runID, request.ProviderID, request.ArtifactPath, invocationStatus, normalizationStatus, evidence, reason)
	if err := sink.Emit(ctx, event); err != nil {
		return fmt.Errorf("emit discovery telemetry: %w", err)
	}
	return nil
}
