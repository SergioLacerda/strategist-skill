package initiative

import (
	"fmt"
	"strings"
)

// PRECISE-SHOT is the stable mechanism identifier. TIRO PRECISO is its
// Portuguese presentation label; the identifier remains stable in storage
// and telemetry so localization never changes correlation keys.
const (
	PreciseShotMechanism               = "PRECISE-SHOT"
	PreciseShotLabelPTBR               = "TIRO PRECISO"
	PreciseShotAlgorithmVersion        = "v1"
	PreciseShotSchemaVersion           = "1"
	PreciseShotCalibrationNoSample     = "no_sample"
	PreciseShotCalibrationUncalibrated = "uncalibrated"
)

// EscalationPolicy bounds advisory requests for one role/run.
type EscalationPolicy struct {
	MaxRequests       int `json:"max_requests" yaml:"max_requests"`
	CooldownSequences int `json:"cooldown_sequences" yaml:"cooldown_sequences"`
}

// EscalationState is the minimal persisted/request-loop state required to
// suppress replayed requests without making LEVELING state writable here.
type EscalationState struct {
	RequestCount       int
	LastResultSequence int
	LastEffective      ConfidenceTier
	LastAdviceID       string
	LastIdempotencyKey string
	MaximumEffort      bool
}

// DefaultEscalationPolicy returns the bounded provider-neutral default.
func DefaultEscalationPolicy() EscalationPolicy {
	return EscalationPolicy{MaxRequests: 3, CooldownSequences: 0}
}

// Validate reports whether the policy is bounded and self-consistent.
func (p EscalationPolicy) Validate() error {
	if p.MaxRequests < 1 {
		return fmt.Errorf("initiative_escalation_invalid: max_requests must be positive")
	}
	if p.CooldownSequences < 0 {
		return fmt.Errorf("initiative_escalation_invalid: cooldown_sequences must not be negative")
	}
	return nil
}

// BoundEscalation applies replay, cooldown, request-count, and maximum-effort
// controls. It returns an observational status; it never selects an effort or
// provider and never mutates the supplied state.
func BoundEscalation(request EscalationRequest, state EscalationState, policy EscalationPolicy) (EscalationRequest, error) {
	if err := policy.Validate(); err != nil {
		return EscalationRequest{}, err
	}
	if request.Mechanism != PreciseShotMechanism || request.IdempotencyKey == "" {
		return EscalationRequest{}, fmt.Errorf("initiative_escalation_invalid: mechanism and idempotency key are required")
	}
	bounded := request
	bounded.Reasons = canonicalReasons(request.Reasons)
	bounded.Status = escalationStatus(request, state, policy)
	return bounded, nil
}

// escalationStatus picks the first matching control: replay, request budget,
// maximum-effort critical reasons, then cooldown.
func escalationStatus(request EscalationRequest, state EscalationState, policy EscalationPolicy) string {
	switch {
	case state.LastIdempotencyKey == request.IdempotencyKey:
		return EscalationSuppressed
	case state.RequestCount >= policy.MaxRequests:
		return EscalationHumanReview
	case state.MaximumEffort && hasCriticalReason(request.Reasons):
		return EscalationHumanReview
	case state.LastAdviceID == request.AdviceID && request.ResultSequence <= state.LastResultSequence+policy.CooldownSequences && state.LastEffective == request.Effective:
		return EscalationSuppressed
	}
	return EscalationRequested
}

// PreciseShotLabel returns the presentation label for a supported locale.
// Unknown locales retain the stable identifier rather than inventing a label.
func PreciseShotLabel(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), "pt-BR") || strings.EqualFold(strings.TrimSpace(locale), "pt_BR") {
		return PreciseShotLabelPTBR
	}
	return PreciseShotMechanism
}

// EvidenceSummary is a bounded summary of validated result coverage.
type EvidenceSummary struct {
	Expected    int `json:"expected" yaml:"expected"`
	Satisfied   int `json:"satisfied" yaml:"satisfied"`
	Partial     int `json:"partial" yaml:"partial"`
	Blocked     int `json:"blocked" yaml:"blocked"`
	Verified    int `json:"verified" yaml:"verified"`
	Missing     int `json:"missing" yaml:"missing"`
	Invalid     int `json:"invalid" yaml:"invalid"`
	Conflicting int `json:"conflicting" yaml:"conflicting"`
}

// EscalationSignals is the provider-neutral signal projection consumed by
// LEVELING. It intentionally contains no model, provider, capability, effort,
// or level-source authority.
type EscalationSignals struct {
	Ambiguity           string `json:"ambiguity,omitempty" yaml:"ambiguity,omitempty"`
	Risk                string `json:"risk,omitempty" yaml:"risk,omitempty"`
	Scope               string `json:"scope,omitempty" yaml:"scope,omitempty"`
	Evidence            string `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	ArchitecturalChange bool   `json:"architectural_change,omitempty" yaml:"architectural_change,omitempty"`
	SecuritySensitive   bool   `json:"security_sensitive,omitempty" yaml:"security_sensitive,omitempty"`
	ConflictingEvidence bool   `json:"conflicting_evidence,omitempty" yaml:"conflicting_evidence,omitempty"`
	RepeatedFailures    int    `json:"repeated_failures,omitempty" yaml:"repeated_failures,omitempty"`
}

// EscalationRequest is an advisory INITIATIVE-to-LEVELING request. It is
// idempotent and carries only bounded categorical signals.
type EscalationRequest struct {
	SchemaVersion  string            `json:"schema_version" yaml:"schema_version"`
	Mechanism      string            `json:"mechanism" yaml:"mechanism"`
	RequestID      string            `json:"request_id" yaml:"request_id"`
	IdempotencyKey string            `json:"idempotency_key" yaml:"idempotency_key"`
	MissionID      string            `json:"mission_id" yaml:"mission_id"`
	Role           string            `json:"role" yaml:"role"`
	RunID          string            `json:"run_id" yaml:"run_id"`
	AdviceID       string            `json:"advice_id" yaml:"advice_id"`
	ResultID       string            `json:"result_id" yaml:"result_id"`
	ResultSequence int               `json:"result_sequence" yaml:"result_sequence"`
	AssessmentID   string            `json:"assessment_id" yaml:"assessment_id"`
	Trigger        Trigger           `json:"trigger" yaml:"trigger"`
	Effective      ConfidenceTier    `json:"effective" yaml:"effective"`
	Reasons        []string          `json:"reasons" yaml:"reasons"`
	Signals        EscalationSignals `json:"signals" yaml:"signals"`
	Status         string            `json:"status" yaml:"status"`
}

// Escalation request statuses.
const (
	EscalationRequested   = "requested"
	EscalationSuppressed  = "suppressed"
	EscalationHumanReview = "human_review"
	EscalationBlocked     = "blocked"
)
