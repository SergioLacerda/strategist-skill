package handoff

import (
	"fmt"
	"slices"
)

// Check identifiers a handoff outcome can name. Only the delegable ones may be
// satisfied by an external provider; the rest stay local and deterministic.
const (
	// CheckInputOutputConformance is the semantic conformance of a handoff's
	// input and output to its contract criteria: what the agent does today.
	CheckInputOutputConformance = "input_output_conformance"
	// CheckArtifactIntegrity covers digests, tamper evidence, replay and
	// cross-mission binding.
	CheckArtifactIntegrity = "artifact_integrity"
	// CheckApprovalGate is the Approval Gate evidence, the user's decision.
	CheckApprovalGate = "approval_gate"
	// CheckAuthorization is the execution authorization.
	CheckAuthorization = "authorization"
	// CheckPolicyFacts is the derivation of typed policy facts.
	CheckPolicyFacts = "policy_facts"
	// CheckReceiverComprehension is the receiver's acknowledgment of what it
	// was handed; a provider evaluates the text it is shown, not the receiver.
	CheckReceiverComprehension = "receiver_comprehension"
)

// DelegableChecks lists the checks a delegated result may satisfy.
func DelegableChecks() []string { return []string{CheckInputOutputConformance} }

// NeverDelegatedChecks lists the checks no provider may satisfy.
func NeverDelegatedChecks() []string {
	return []string{CheckArtifactIntegrity, CheckApprovalGate, CheckAuthorization, CheckPolicyFacts, CheckReceiverComprehension}
}

// Delegation is the provenance of a delegated check result. It is part of the
// outcome record and covered by its integrity digest. The decision stays local:
// Approved is derived here from the recorded confidence and threshold, never
// read from the provider, and a delegation records only an approval; a call
// that did not approve leaves the handoff on the main path with no delegation.
type Delegation struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	BindingDigest string `json:"binding_digest"`
	Capability    string `json:"capability"`
	Criterion     string `json:"criterion"`
	// Subject is the digest of the artifact the provider evaluated. It must
	// equal the digest the outcome is bound to, so a result obtained for one
	// revision can never vouch for another.
	Subject      string   `json:"subject"`
	Checks       []string `json:"checks"`
	Confidence   float64  `json:"confidence"`
	Threshold    float64  `json:"threshold"`
	InputTokens  int      `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
}

// Approved applies the fixed local rule: confidence strictly above the threshold.
func (d Delegation) Approved() bool { return d.Confidence > d.Threshold }

// Validate fails closed on a delegation without full provenance, one that names
// a check that may not be delegated, or one the local rule does not approve.
func (d Delegation) Validate() error {
	if err := d.validateProvenance(); err != nil {
		return err
	}
	if err := validateDelegatedChecks(d.Checks); err != nil {
		return err
	}
	return d.validateDecision()
}

func (d Delegation) validateProvenance() error {
	for name, value := range map[string]string{
		"provider": d.Provider, "model": d.Model, "binding_digest": d.BindingDigest,
		"capability": d.Capability, "criterion": d.Criterion, "subject": d.Subject,
	} {
		if value == "" {
			return fmt.Errorf("handoff_delegation_invalid: provenance field %s is required", name)
		}
	}
	if d.InputTokens < 0 || d.OutputTokens < 0 {
		return fmt.Errorf("handoff_delegation_invalid: usage cannot be negative")
	}
	return nil
}

func validateDelegatedChecks(checks []string) error {
	if len(checks) == 0 {
		return fmt.Errorf("handoff_delegation_invalid: a delegation names at least one check")
	}
	for _, check := range checks {
		if !slices.Contains(DelegableChecks(), check) {
			return fmt.Errorf("handoff_delegation_forbidden_check: %q may not be satisfied by a delegated result", check)
		}
	}
	return nil
}

func (d Delegation) validateDecision() error {
	if d.Confidence < 0 || d.Confidence > 1 || d.Threshold <= 0 || d.Threshold > 1 {
		return fmt.Errorf("handoff_delegation_invalid: confidence must be in [0,1] and threshold in (0,1]")
	}
	if !d.Approved() {
		return fmt.Errorf("handoff_delegation_invalid: confidence %v is not strictly above threshold %v", d.Confidence, d.Threshold)
	}
	return nil
}

// validateDelegation applies the delegation rules to an outcome: the delegation
// must be valid and must have evaluated the very revision the outcome is bound
// to. Conformance is checked on every handoff, so a skipped outcome may carry one.
func (o Outcome) validateDelegation() error {
	if o.Delegation == nil {
		return nil
	}
	if err := o.Delegation.Validate(); err != nil {
		return err
	}
	if o.Delegation.Subject != o.revisionDigest() {
		return fmt.Errorf("handoff_delegation_invalid: the delegation evaluated another revision than the one the outcome is bound to")
	}
	return nil
}

func (o Outcome) revisionDigest() string {
	if o.Transition == TransitionRangerToArchivist {
		return o.ArtifactDigest
	}
	return o.PackageDigest
}

// WithDelegation attaches a delegation only when it is valid for this outcome.
// A delegation that does not fit is dropped, never an error: the provider can
// never block a handoff, and without it the outcome is exactly the main path's.
func (o Outcome) WithDelegation(delegation *Delegation) Outcome {
	if delegation == nil {
		return o
	}
	candidate := o
	candidate.Delegation = delegation
	if candidate.validateDelegation() != nil {
		return o
	}
	return candidate
}
