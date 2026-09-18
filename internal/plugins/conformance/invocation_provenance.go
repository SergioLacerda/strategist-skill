package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// InvocationStatus is the result of a provider invocation attempt. Static
// binding and readiness evidence intentionally do not use InvocationInvoked.
type InvocationStatus string

const (
	// InvocationStatusInvoked records a provider invocation with runtime evidence.
	InvocationStatusInvoked InvocationStatus = "invoked"
	// InvocationStatusFailed records an invocation that returned an error.
	InvocationStatusFailed InvocationStatus = "failed"
	// InvocationStatusBlocked records an invocation prevented by policy or readiness.
	InvocationStatusBlocked InvocationStatus = "blocked"
	// InvocationStatusUnsupported records a provider that cannot support the request.
	InvocationStatusUnsupported InvocationStatus = "unsupported"
	// InvocationStatusUnknown records an invocation whose outcome is unavailable.
	InvocationStatusUnknown InvocationStatus = "unknown"
)

// InvocationProvenanceInput is the evidence collected at one role/provider
// invocation boundary. RuntimeEvidence must come from the host invocation
// path; catalog, lock, or manifest metadata alone cannot set it to true.
type InvocationProvenanceInput struct {
	MissionID         string
	Role              string
	Slot              string
	Provider          string
	Mode              string
	BindingDigest     string
	EnvelopeDigest    string
	SchemaResult      string
	Status            InvocationStatus
	RuntimeEvidence   bool
	FallbackDecision  string
	FailureReasonCode string
}

// InvocationProvenanceEvent adapts validated provenance to the existing
// standalone-safe telemetry envelope. It records evidence, not a new
// authority or a provider invocation itself.
func InvocationProvenanceEvent(provenance InvocationProvenance) (telemetry.Event, error) {
	if err := provenance.Validate(); err != nil {
		return telemetry.Event{}, err
	}
	return telemetry.Event{
		Name:           "strategist.invocation." + string(provenance.Status),
		Timestamp:      time.Now().UTC(),
		SeverityNumber: telemetry.SeverityInfo,
		Body:           provenance.FailureReasonCode,
		Attributes: map[string]any{
			"strategist.invocation.mission_id":          provenance.MissionID,
			"strategist.invocation.role":                provenance.Role,
			"strategist.invocation.slot":                provenance.Slot,
			"strategist.invocation.provider":            provenance.Provider,
			"strategist.invocation.mode":                provenance.Mode,
			"strategist.invocation.binding_digest":      provenance.BindingDigest,
			"strategist.invocation.envelope_digest":     provenance.EnvelopeDigest,
			"strategist.invocation.schema_result":       provenance.SchemaResult,
			"strategist.invocation.runtime_evidence":    provenance.RuntimeEvidence,
			"strategist.invocation.fallback_decision":   provenance.FallbackDecision,
			"strategist.invocation.failure_reason_code": provenance.FailureReasonCode,
			"strategist.invocation.fingerprint":         provenance.Fingerprint,
		},
	}, nil
}

// InvocationProvenance is immutable, fingerprinted evidence for one
// invocation boundary. The fingerprint covers every field except itself.
type InvocationProvenance struct {
	MissionID         string           `json:"mission_id"`
	Role              string           `json:"role"`
	Slot              string           `json:"slot"`
	Provider          string           `json:"provider"`
	Mode              string           `json:"mode"`
	BindingDigest     string           `json:"binding_digest"`
	EnvelopeDigest    string           `json:"envelope_digest"`
	SchemaResult      string           `json:"schema_result"`
	Status            InvocationStatus `json:"status"`
	RuntimeEvidence   bool             `json:"runtime_evidence"`
	FallbackDecision  string           `json:"fallback_decision"`
	FailureReasonCode string           `json:"failure_reason_code,omitempty"`
	Fingerprint       string           `json:"fingerprint"`
}

// NewInvocationProvenance validates and fingerprints invocation evidence.
func NewInvocationProvenance(input InvocationProvenanceInput) (InvocationProvenance, error) {
	provenance := InvocationProvenance{
		MissionID: input.MissionID, Role: input.Role, Slot: input.Slot,
		Provider: input.Provider, Mode: input.Mode,
		BindingDigest: input.BindingDigest, EnvelopeDigest: input.EnvelopeDigest,
		SchemaResult: input.SchemaResult, Status: input.Status,
		RuntimeEvidence: input.RuntimeEvidence, FallbackDecision: input.FallbackDecision,
		FailureReasonCode: input.FailureReasonCode,
	}
	if err := provenance.Validate(); err != nil {
		return InvocationProvenance{}, err
	}
	data, err := json.Marshal(provenance)
	if err != nil {
		return InvocationProvenance{}, fmt.Errorf("invocation provenance: fingerprint: %w", err)
	}
	sum := sha256.Sum256(data)
	provenance.Fingerprint = "sha256:" + hex.EncodeToString(sum[:])
	return provenance, nil
}

// Validate enforces that successful invocation evidence is runtime-backed and
// that no provider substitution was hidden as a fallback.
func (p InvocationProvenance) Validate() error {
	if err := p.validateRequiredFields(); err != nil {
		return err
	}
	if !validInvocationStatus(p.Status) {
		return fmt.Errorf("invocation provenance invalid: unknown status %q", p.Status)
	}
	if p.FallbackDecision != "none" {
		return fmt.Errorf("invocation provenance invalid: fallback decision %q is not permitted", p.FallbackDecision)
	}
	if p.Status == InvocationStatusInvoked {
		return p.validateInvoked()
	}
	return p.validateNonInvoked()
}

func (p InvocationProvenance) validateRequiredFields() error {
	fields := map[string]string{
		"mission_id": p.MissionID, "role": p.Role, "slot": p.Slot,
		"provider": p.Provider, "mode": p.Mode, "binding_digest": p.BindingDigest,
		"envelope_digest": p.EnvelopeDigest, "schema_result": p.SchemaResult,
		"fallback_decision": p.FallbackDecision,
	}
	for name, value := range fields {
		if value == "" {
			return fmt.Errorf("invocation provenance invalid: %s is required", name)
		}
	}
	return nil
}

func (p InvocationProvenance) validateInvoked() error {
	if !p.RuntimeEvidence {
		return fmt.Errorf("invocation provenance invalid: invoked status requires runtime evidence")
	}
	if p.SchemaResult != "accepted" {
		return fmt.Errorf("invocation provenance invalid: invoked status requires accepted schema result")
	}
	if p.FailureReasonCode != "" {
		return fmt.Errorf("invocation provenance invalid: invoked status cannot have a failure reason")
	}
	return nil
}

func (p InvocationProvenance) validateNonInvoked() error {
	if p.RuntimeEvidence {
		return fmt.Errorf("invocation provenance invalid: non-invoked status cannot claim runtime evidence")
	}
	if p.FailureReasonCode == "" {
		return fmt.Errorf("invocation provenance invalid: %s status requires a failure reason", p.Status)
	}
	return nil
}

func validInvocationStatus(status InvocationStatus) bool {
	switch status {
	case InvocationStatusInvoked, InvocationStatusFailed, InvocationStatusBlocked,
		InvocationStatusUnsupported, InvocationStatusUnknown:
		return true
	default:
		return false
	}
}
