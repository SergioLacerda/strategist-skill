package domain

import (
	"errors"
	"fmt"
)

// Decision is a mission-scoped choice with stable identity, evidence
// backing, and confidence — the ledger entry critique_skill.txt (item 2) and
// critique_skill_3.txt (§ Skill/agent gaps item 1) ask for, so contradictions
// can be detected, decisions can be superseded, and Treasure Chest/ADR
// generation has something durable to reuse instead of re-deriving decisions
// from prose. See
// .analysis/done/20260803-critique-skill-affinity-review/design.md §
// "Consolidated Decision/Evidence Model".
type Decision struct {
	ID                   string   `yaml:"id"`
	Statement            string   `yaml:"statement"`
	Status               string   `yaml:"status"`
	EvidenceIDs          []string `yaml:"evidence"`
	AlternativesRejected []string `yaml:"alternatives_rejected,omitempty"`
	Supersedes           []string `yaml:"supersedes,omitempty"`
	Confidence           string   `yaml:"confidence"`
	ConfidencePercent    *int     `yaml:"confidence_percent,omitempty"`
	ClaimKind            string   `yaml:"claim_kind,omitempty"`
	ApprovedAt           string   `yaml:"approved_at,omitempty"`
}

// Decision status values. "open" is the only non-terminal status — a
// decision still awaiting resolution. See mission_quality.go's
// unresolved_questions_preserved check, which treats "open" as the status
// that must never be silently dropped between revisions.
const (
	DecisionStatusOpen       = "open"
	DecisionStatusApproved   = "approved"
	DecisionStatusRejected   = "rejected"
	DecisionStatusSuperseded = "superseded"
)

var allowedDecisionStatuses = stringSet(
	DecisionStatusOpen,
	DecisionStatusApproved,
	DecisionStatusRejected,
	DecisionStatusSuperseded,
)

// Confidence levels shared by Decision and Evidence.
const (
	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"
)

var allowedConfidenceLevels = stringSet(ConfidenceLow, ConfidenceMedium, ConfidenceHigh)

// ValidateDecision checks a Decision against the required fields and allowed
// values documented in schemas/decision.schema.yaml. It does not check that
// EvidenceIDs resolve to real Evidence records — that cross-reference check
// belongs to EvaluateMissionQuality, which has the full Evidence set to
// check against.
func ValidateDecision(d Decision) error {
	errs := validateNamedValue("decision_invalid", "id", d.ID, nil)
	if d.Statement == "" {
		errs = append(errs, errors.New("decision_invalid: statement is required"))
	}
	errs = append(errs, validateNamedValue("decision_invalid", "status", d.Status, allowedDecisionStatuses)...)
	errs = append(errs, validateNamedValue("decision_invalid", "confidence", d.Confidence, allowedConfidenceLevels)...)
	if err := validateConfidenceCompatibility(d.Confidence, d.ConfidencePercent); err != nil {
		errs = append(errs, err)
	}
	if d.ClaimKind != "" {
		if d.ConfidencePercent == nil {
			errs = append(errs, errors.New("decision_invalid: claim_kind requires confidence_percent"))
		}
		if _, ok := allowedClaimKinds[d.ClaimKind]; !ok {
			errs = append(errs, fmt.Errorf("decision_invalid: claim_kind %q is not allowed", d.ClaimKind))
		}
	}
	return errors.Join(errs...)
}

// ValidateDecisionWithEvidence applies the claim-kind and evidence rules once
// the caller has the mission's complete evidence set. ValidateDecision remains
// the scalar/schema validator for backward-compatible callers.
func ValidateDecisionWithEvidence(d Decision, evidence []Evidence) error {
	if err := ValidateDecision(d); err != nil {
		return err
	}
	if d.ClaimKind == "" && d.ConfidencePercent == nil {
		return nil
	}
	if d.ConfidencePercent == nil {
		return errors.New("decision_invalid: confidence_percent is required for claim validation")
	}
	claim := ConfidenceClaim{
		ID: d.ID, Statement: d.Statement, ClaimKind: d.ClaimKind,
		ConfidencePercent: *d.ConfidencePercent, EvidenceIDs: append([]string(nil), d.EvidenceIDs...),
		CorrelationKey: d.ID,
	}
	return ValidateConfidenceClaim(claim, evidence)
}

func validateNamedValue(token, field, value string, allowed map[string]struct{}) []error {
	if value == "" {
		return []error{fmt.Errorf("%s: %s is required", token, field)}
	}
	if _, ok := allowed[value]; allowed != nil && !ok {
		return []error{fmt.Errorf("%s: %s %q is not an allowed value", token, field, value)}
	}
	return nil
}
