package connectors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// EmbeddedPromptPayload is the immutable prompt payload selected by the
// compiled registry. It contains bytes, never a filesystem path.
type EmbeddedPromptPayload struct {
	WeaponID     string
	Content      []byte
	SourceDigest string
}

// EmbeddedPromptPayloadSource resolves only catalog-approved embedded
// payloads. Implementations must not read a host skill loader or provider
// directory.
type EmbeddedPromptPayloadSource func(ctx context.Context, weaponID, version string) (EmbeddedPromptPayload, error)

// EmbeddedPromptRequest is the internal bridge contract for prompt-oriented
// Ranked Weapons. The host may provide model execution, but it receives only
// the payload and identity selected by Strategist's compiled registry.
type EmbeddedPromptRequest struct {
	Envelope      InvocationEnvelope
	Payload       []byte
	SourceDigest  string
	BindingDigest string
	Entrypoint    string
}

// EmbeddedPromptBridge executes one embedded prompt payload through an
// explicitly registered internal adapter.
type EmbeddedPromptBridge func(context.Context, EmbeddedPromptRequest) ConnectorResult

// EmbeddedInvocationReceipt records internal execution without claiming that
// a host-issued receipt or host capability isolation exists.
type EmbeddedInvocationReceipt struct {
	SchemaVersion string
	MissionID     string
	Role          string
	Slot          string
	WeaponID      string
	Entrypoint    string
	BindingDigest string
	SourceDigest  string
	// RequestID ties the receipt to the single-use mission invocation request
	// it completes. Optional: in-process embedded invocations have no request.
	RequestID string
	// Nonce is the request's prompt-delimiter nonce, echoed as non-authenticating
	// correlation evidence. Optional for the same reason as RequestID.
	Nonce string
	// ExecutionAdapter and ChildPolicyID carry Strategist-committed adapter
	// provenance. Neither is a capability-isolation claim.
	ExecutionAdapter domain.MissionExecutionAdapter
	ChildPolicyID    string
	IssuedAt         time.Time
}

// EmbeddedInvocationReceiptSchemaVersion identifies the supported receipt schema.
const EmbeddedInvocationReceiptSchemaVersion = "strategist-embedded-invocation-receipt/v1"

// Validate fails closed unless the receipt matches the supported schema and
// carries the required invocation evidence.
func (r EmbeddedInvocationReceipt) Validate() error {
	if r.SchemaVersion != EmbeddedInvocationReceiptSchemaVersion {
		return fmt.Errorf("unsupported embedded invocation receipt schema")
	}
	fields := []struct {
		name  string
		value string
	}{
		{name: "mission", value: r.MissionID},
		{name: "role", value: r.Role},
		{name: "slot", value: r.Slot},
		{name: "Weapon", value: r.WeaponID},
		{name: "entrypoint", value: r.Entrypoint},
		{name: "binding digest", value: r.BindingDigest},
		{name: "source digest", value: r.SourceDigest},
	}
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("embedded invocation receipt %s is required", field.name)
		}
	}
	if r.IssuedAt.IsZero() {
		return fmt.Errorf("embedded invocation receipt issued time is required")
	}
	return r.validateAdapterProvenance()
}

// validateAdapterProvenance accepts an absent adapter (in-process embedded
// invocations have none) but rejects an unknown mode, and requires a policy
// identity exactly when Strategist launched a child.
func (r EmbeddedInvocationReceipt) validateAdapterProvenance() error {
	if r.ExecutionAdapter == "" {
		if r.ChildPolicyID != "" {
			return fmt.Errorf("embedded invocation receipt child policy requires an execution adapter")
		}
		return nil
	}
	if !r.ExecutionAdapter.Known() {
		return fmt.Errorf("embedded invocation receipt execution adapter %q is unknown", r.ExecutionAdapter)
	}
	if r.ExecutionAdapter.IsChild() != (r.ChildPolicyID != "") {
		return fmt.Errorf("embedded invocation receipt child policy must be present exactly for child adapters")
	}
	return nil
}

// NewEmbeddedPromptInvoker adapts the internal bridge to the common connector
// invoker callback. It verifies the embedded payload before the bridge runs.
func NewEmbeddedPromptInvoker(weaponID, weaponVersion, entrypoint, bindingDigest, sourceDigest string, source EmbeddedPromptPayloadSource, bridge EmbeddedPromptBridge) EmbeddedWeaponInvoker {
	return func(ctx context.Context, envelope InvocationEnvelope) ConnectorResult {
		return invokeEmbeddedPrompt(ctx, envelope, weaponID, weaponVersion, entrypoint, bindingDigest, sourceDigest, source, bridge)
	}
}

func invokeEmbeddedPrompt(ctx context.Context, envelope InvocationEnvelope, weaponID, weaponVersion, entrypoint, bindingDigest, sourceDigest string, source EmbeddedPromptPayloadSource, bridge EmbeddedPromptBridge) ConnectorResult {
	if result := validateEmbeddedPromptRequest(envelope, weaponID, weaponVersion, source, bridge); result != nil {
		return *result
	}
	payload, err := source(ctx, weaponID, weaponVersion)
	if err != nil {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "embedded_prompt_payload_unavailable", Detail: err.Error()}
	}
	if payload.WeaponID != weaponID || payload.SourceDigest != sourceDigest || len(payload.Content) == 0 {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "embedded_prompt_payload_mismatch"}
	}
	result := bridge(ctx, EmbeddedPromptRequest{
		Envelope: envelope, Payload: append([]byte(nil), payload.Content...), SourceDigest: sourceDigest,
		BindingDigest: bindingDigest, Entrypoint: entrypoint,
	})
	if result.Status != domain.ReadinessReady {
		return result
	}
	if strings.TrimSpace(result.ProviderID) == "" || strings.TrimSpace(result.InvocationEvidence) == "" {
		return ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: "embedded_prompt_result_incomplete"}
	}
	result.EmbeddedInvocationReceipt = embeddedInvocationReceipt(envelope, weaponID, entrypoint, bindingDigest, sourceDigest)
	return result
}

func validateEmbeddedPromptRequest(envelope InvocationEnvelope, weaponID, weaponVersion string, source EmbeddedPromptPayloadSource, bridge EmbeddedPromptBridge) *ConnectorResult {
	if bridge == nil {
		return blockedEmbeddedPrompt("embedded_prompt_bridge_unavailable")
	}
	if source == nil {
		return blockedEmbeddedPrompt("embedded_prompt_payload_unavailable")
	}
	if envelope.WeaponID != "" && envelope.WeaponID != weaponID {
		return blockedEmbeddedPrompt("embedded_prompt_weapon_mismatch")
	}
	if envelope.WeaponVersion != "" && envelope.WeaponVersion != weaponVersion {
		return blockedEmbeddedPrompt("embedded_prompt_weapon_mismatch")
	}
	if envelope.HostAPI != "" {
		return blockedEmbeddedPrompt("embedded_host_api_forbidden")
	}
	return nil
}

func blockedEmbeddedPrompt(reason string) *ConnectorResult {
	return &ConnectorResult{Status: domain.ReadinessBlocked, ReasonCode: reason}
}

func embeddedInvocationReceipt(envelope InvocationEnvelope, weaponID, entrypoint, bindingDigest, sourceDigest string) EmbeddedInvocationReceipt {
	return EmbeddedInvocationReceipt{
		SchemaVersion: EmbeddedInvocationReceiptSchemaVersion,
		MissionID:     envelope.MissionID, Role: envelope.Role, Slot: envelope.Slot, WeaponID: weaponID,
		Entrypoint: entrypoint, BindingDigest: bindingDigest, SourceDigest: sourceDigest, IssuedAt: time.Now(),
	}
}
