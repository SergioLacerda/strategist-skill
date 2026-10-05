package levelingapp

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	leveling "github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
)

// EscalationOutcome is the explicit result of a requested budget escalation.
// Granted, denied, exhausted and unavailable are different reasons; none of
// them is a provider failure, which surfaces as an unresolved level instead.
type EscalationOutcome struct {
	Attempted bool `json:"attempted"`
	Granted   bool `json:"granted"`
	// Reason is "granted" or the stable leveling_budget_* denial code.
	Reason string `json:"reason"`
	// ParentResolutionID names the envelope the request escalated from.
	ParentResolutionID string `json:"parent_resolution_id,omitempty"`
	// ResolutionID is the envelope in force after the request: the new child
	// when granted, the unchanged effective one otherwise.
	ResolutionID string `json:"resolution_id,omitempty"`
	// Budget is the envelope whose escalation decision must be audited. It is
	// retained for an unavailable re-resolution where the effective Level has
	// no budget, and is intentionally excluded from CLI JSON.
	Budget *domain.EffortBudget `json:"-"`
}

// escalateBudget applies an escalation request to the freshly resolved level.
// It only acts when a reason was given and the latest tuple of the run carries
// an issued budget; the prior record is never modified (the ledger is
// append-only) and a refusal keeps the effective envelope in force.
func escalateBudget(ledgerPath string, input LevelingInput, level *leveling.Level) (EscalationOutcome, error) {
	if input.Reason == "" || input.Mission == "" {
		return EscalationOutcome{}, nil
	}
	prior, found, err := leveling.LatestRunRecord(ledgerPath, input.Mission, input.Role, input.Run)
	if err != nil {
		return EscalationOutcome{}, fmt.Errorf("leveling: read role level ledger: %w", err)
	}
	if !found || prior.Budget == nil {
		return EscalationOutcome{}, nil
	}
	return escalateFrom(*prior.Budget, level, input.Reason), nil
}

func escalateFrom(prior domain.EffortBudget, level *leveling.Level, reason string) EscalationOutcome {
	parentID := effectiveResolutionID(prior)
	outcome := EscalationOutcome{Attempted: true, ParentResolutionID: parentID, ResolutionID: parentID, Budget: &prior}
	if level.Budget == nil {
		outcome.Reason = leveling.ReasonLevelingBudgetUnavailable
		return outcome
	}
	fresh := *level.Budget
	fresh.EscalationLevel = prior.EscalationLevel
	child, err := leveling.EscalateBudget(fresh, parentID, reason)
	if err != nil {
		outcome.Reason = leveling.EscalationDenialReason(err)
		effective := keepEffectiveBudget(prior, fresh, parentID)
		level.Budget = &effective
		outcome.Budget = &effective
		return outcome
	}
	level.Budget = &child
	outcome.Granted, outcome.Reason, outcome.ResolutionID, outcome.Budget = true, "granted", child.ResolutionID, &child
	return outcome
}

// keepEffectiveBudget carries the prior limits and escalation identity and only
// refreshes the descriptors that describe the level being recorded.
func keepEffectiveBudget(prior, fresh domain.EffortBudget, resolutionID string) domain.EffortBudget {
	effective := prior
	effective.ResolutionID = resolutionID
	effective.Provider, effective.Model, effective.Capability = fresh.Provider, fresh.Model, fresh.Capability
	effective.PolicyVersion, effective.PolicyDigest = fresh.PolicyVersion, fresh.PolicyDigest
	return effective
}

// effectiveResolutionID is the id of a stored budget; records written before
// the base id existed get the id the same facts would have received.
func effectiveResolutionID(budget domain.EffortBudget) string {
	if budget.ResolutionID != "" {
		return budget.ResolutionID
	}
	return leveling.BaseResolutionID(budget)
}
