package domain

import (
	"fmt"
	"strings"
	"time"
)

// MissionInvocationProtocolVersion identifies the host bridge contract used
// by `mission invoke` and `mission complete`.
const MissionInvocationProtocolVersion = "strategist-invocation/v1"

// MissionInvocationRequest is the immutable request emitted to a host agent.
// Payload is deliberately carried as content, never as a host filesystem path.
type MissionInvocationRequest struct {
	Protocol      string                `json:"protocol"`
	RequestID     string                `json:"request_id"`
	MissionID     string                `json:"mission_id"`
	Role          string                `json:"role"`
	Slot          string                `json:"slot"`
	Weapon        MissionWeaponIdentity `json:"weapon"`
	BindingDigest string                `json:"binding_digest"`
	SourceDigest  string                `json:"source_digest"`
	ExecutionMode string                `json:"execution_mode"`
	Entrypoint    string                `json:"entrypoint"`
	Payload       string                `json:"payload"`
	Input         map[string]any        `json:"input,omitempty"`
	// Nonce is non-authenticating correlation evidence: it delimits the host
	// prompt and is echoed in the embedded receipt. It never proves that a
	// host ran the Weapon or that capabilities were isolated.
	Nonce string `json:"nonce,omitempty"`
}

// MissionWeaponIdentity binds a request to one compiled Weapon version.
type MissionWeaponIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

// MissionInvocationCompletion is the only response accepted from a host.
// The host supplies raw model output and no authority-bearing metadata.
type MissionInvocationCompletion struct {
	RequestID string `json:"request_id"`
	Result    string `json:"result"`
}

// MissionInvocationOutcome is the Ranger-owned result returned after a raw
// host response has been validated, normalized, and persisted.
type MissionInvocationOutcome struct {
	RequestID     string `json:"request_id"`
	MissionID     string `json:"mission_id"`
	Status        string `json:"status"`
	ArtifactPath  string `json:"artifact_path"`
	BindingDigest string `json:"binding_digest"`
	SourceDigest  string `json:"source_digest"`
}

// MissionInvocationState is the durable completion state of one request.
type MissionInvocationState string

const (
	// InvocationStatePending is an issued request that no completion has started.
	InvocationStatePending MissionInvocationState = "pending"
	// InvocationStateProcessing means a completion committed to publish ArtifactDigest at TargetPath.
	InvocationStateProcessing MissionInvocationState = "processing"
	// InvocationStateCompleted is terminal: the artifact is published and the request is replay-rejected.
	InvocationStateCompleted MissionInvocationState = "completed"
)

// CanTransitionTo reports whether the state machine allows next. Processing may
// re-enter itself so crash recovery can retry; nothing leaves completed.
func (s MissionInvocationState) CanTransitionTo(next MissionInvocationState) bool {
	switch s {
	case InvocationStatePending:
		return next == InvocationStateProcessing
	case InvocationStateProcessing:
		return next == InvocationStateProcessing || next == InvocationStateCompleted
	case InvocationStateCompleted:
		return false
	default:
		return false
	}
}

// MissionInvocationRecord is the durable single-use request state. Records
// written before State existed carry only Consumed; EffectiveState maps them.
type MissionInvocationRecord struct {
	Request   MissionInvocationRequest `json:"request"`
	CreatedAt time.Time                `json:"created_at"`
	ExpiresAt time.Time                `json:"expires_at"`
	State     MissionInvocationState   `json:"state,omitempty"`
	// ExecutionAdapter is committed by Strategist-owned dispatch, never by a
	// completion. Records written before the field existed read as
	// ExecutionAdapterCurrentHostUnverified; a child mode is never inferred.
	ExecutionAdapter MissionExecutionAdapter `json:"execution_adapter,omitempty"`
	// ChildPolicyID is the versioned identity of the restrictions Strategist
	// configured for a child. It proves what was requested, not what was enforced.
	ChildPolicyID  string     `json:"child_policy_id,omitempty"`
	TargetPath     string     `json:"target_path,omitempty"`
	ArtifactDigest string     `json:"artifact_digest,omitempty"`
	ProcessingAt   *time.Time `json:"processing_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	// Consumed mirrors State == completed so an older binary still rejects replay.
	Consumed   bool       `json:"consumed"`
	ConsumedAt *time.Time `json:"consumed_at,omitempty"`
}

// EffectiveAdapter returns the committed mode, or the explicit compatibility
// interpretation for a record that predates the field.
func (r MissionInvocationRecord) EffectiveAdapter() MissionExecutionAdapter {
	if r.ExecutionAdapter == "" {
		return ExecutionAdapterCurrentHostUnverified
	}
	return r.ExecutionAdapter
}

// EffectiveState returns State, mapping a pre-State record to pending or completed.
func (r MissionInvocationRecord) EffectiveState() MissionInvocationState {
	switch {
	case r.State != "":
		return r.State
	case r.Consumed:
		return InvocationStateCompleted
	default:
		return InvocationStatePending
	}
}

// Validate checks the immutable request contract before it reaches a host.
func (r MissionInvocationRequest) Validate() error {
	fields := []struct {
		name  string
		value string
	}{
		{"protocol", r.Protocol}, {"request_id", r.RequestID}, {"mission_id", r.MissionID},
		{"role", r.Role}, {"slot", r.Slot}, {"weapon id", r.Weapon.ID},
		{"weapon version", r.Weapon.Version}, {"weapon digest", r.Weapon.Digest},
		{"binding digest", r.BindingDigest}, {"source digest", r.SourceDigest},
		{"execution mode", r.ExecutionMode}, {"entrypoint", r.Entrypoint}, {"payload", r.Payload},
	}
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("mission invocation: %s is required", field.name)
		}
	}
	if r.Protocol != MissionInvocationProtocolVersion {
		return fmt.Errorf("mission invocation: unsupported protocol %q", r.Protocol)
	}
	return nil
}

// Validate checks the host response without accepting host-controlled identity.
func (c MissionInvocationCompletion) Validate() error {
	if strings.TrimSpace(c.RequestID) == "" {
		return fmt.Errorf("mission invocation completion: request_id is required")
	}
	if strings.TrimSpace(c.Result) == "" {
		return fmt.Errorf("mission invocation completion: result is required")
	}
	return nil
}
