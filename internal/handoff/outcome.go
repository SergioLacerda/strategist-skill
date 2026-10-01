package handoff

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// OutcomeSchemaVersion identifies a durable Archivist-to-Sniper outcome record.
const OutcomeSchemaVersion = "strategist-handoff-outcome/v1"

// Terminal outcome results. Only passed and policy-authorized skipped may
// authorize execution entry; failed never does.
const (
	OutcomePassed  = "passed"
	OutcomeFailed  = "failed"
	OutcomeSkipped = "skipped"
)

// SignalsRecord is the evaluated RiskSignals, stored with the outcome.
type SignalsRecord struct {
	MandatoryConstraintsPresent  bool `json:"mandatory_constraints_present"`
	UnresolvedQuestionsPresent   bool `json:"unresolved_questions_present"`
	ForbiddenScopePresent        bool `json:"forbidden_scope_present"`
	ImplementationHandoffPresent bool `json:"implementation_handoff_present"`
	DestructiveOperationPossible bool `json:"destructive_operation_possible"`
	SecuritySensitiveTask        bool `json:"security_sensitive_task"`
	InformationalOnly            bool `json:"informational_only"`
}

// SignalsRecordOf projects RiskSignals onto the persisted shape.
func SignalsRecordOf(s RiskSignals) SignalsRecord {
	return SignalsRecord{
		MandatoryConstraintsPresent: s.MandatoryConstraintsPresent, UnresolvedQuestionsPresent: s.UnresolvedQuestionsPresent,
		ForbiddenScopePresent: s.ForbiddenScopePresent, ImplementationHandoffPresent: s.ImplementationHandoffPresent,
		DestructiveOperationPossible: s.DestructiveOperationPossible, SecuritySensitiveTask: s.SecuritySensitiveTask,
		InformationalOnly: s.InformationalOnly,
	}
}

// Outcome is the durable, tamper-evident terminal record of one evaluation of
// the Archivist-to-Sniper handoff. It is correlated by mission, transition,
// package content identity and attempt; entering execution consumes it.
type Outcome struct {
	SchemaVersion string `json:"schema_version"`
	MissionID     string `json:"mission_id"`
	Transition    string `json:"transition"`
	PackageDigest string `json:"package_digest"`
	Attempt       int    `json:"attempt"`
	Result        string `json:"result"`
	// PolicyID identifies the rules the outcome was decided under.
	PolicyID string `json:"policy_id"`
	// Required reports whether the policy required the semantic challenge.
	Required   bool               `json:"required"`
	Signals    SignalsRecord      `json:"signals"`
	Provenance []SignalProvenance `json:"provenance,omitempty"`
	// GateObserved is the mission state the Approval Gate evidence was read from.
	GateObserved     string `json:"gate_observed"`
	ChallengeStatus  string `json:"challenge_status,omitempty"`
	CriticalFailures int    `json:"critical_failures,omitempty"`
	CreatedAt        string `json:"created_at"`
	// Integrity is the digest of every other field; any edit breaks it.
	Integrity string `json:"integrity"`
}

// Seal stamps the schema version and creation time when absent and computes
// the integrity digest.
func (o Outcome) Seal(now time.Time) (Outcome, error) {
	o.SchemaVersion = OutcomeSchemaVersion
	if o.CreatedAt == "" {
		o.CreatedAt = now.UTC().Format(time.RFC3339Nano)
	}
	if err := o.validateShape(); err != nil {
		return Outcome{}, err
	}
	digest, err := o.digest()
	if err != nil {
		return Outcome{}, err
	}
	o.Integrity = digest
	return o, nil
}

func (o Outcome) digest() (string, error) {
	o.Integrity = ""
	raw, err := json.Marshal(o)
	if err != nil {
		return "", fmt.Errorf("handoff_outcome_invalid: encode outcome: %w", err)
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(raw)), nil
}

// VerifyIntegrity reports whether the record still matches its digest.
func (o Outcome) VerifyIntegrity() error {
	want, err := o.digest()
	if err != nil {
		return err
	}
	if o.Integrity == "" || o.Integrity != want {
		return fmt.Errorf("handoff_outcome_tampered: integrity digest does not match the record for mission %q attempt %d", o.MissionID, o.Attempt)
	}
	return nil
}

func (o Outcome) validateShape() error {
	switch {
	case o.MissionID == "" || o.Transition == "" || o.PackageDigest == "" || o.PolicyID == "":
		return fmt.Errorf("handoff_outcome_invalid: mission, transition, package digest and policy identity are required")
	case o.Attempt < 1:
		return fmt.Errorf("handoff_outcome_invalid: attempt must be positive, got %d", o.Attempt)
	case o.GateObserved == "":
		return fmt.Errorf("handoff_outcome_invalid: gate observation is required")
	case !knownOutcomeResult(o.Result):
		return fmt.Errorf("handoff_outcome_unknown_result: %q is not passed, failed or skipped", o.Result)
	case o.Result == OutcomeSkipped && o.Required:
		return fmt.Errorf("handoff_outcome_invalid: a skipped outcome cannot be a required challenge")
	}
	return nil
}

func knownOutcomeResult(result string) bool {
	return result == OutcomePassed || result == OutcomeFailed || result == OutcomeSkipped
}
