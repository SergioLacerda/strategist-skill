package mission

import "strings"

// BudgetReason identifies a deterministic admission decision. Estimates are
// local approximations and are never provider-reported usage.
type BudgetReason string

const (
	// BudgetReasonAdmitted means the local estimate fits the declared budget.
	BudgetReasonAdmitted BudgetReason = "budget_admitted"
	// BudgetReasonOverflow means the local estimate exceeds the declared budget.
	BudgetReasonOverflow BudgetReason = "budget_overflow"
	// BudgetReasonMalformed means the phase, role, or budget is invalid.
	BudgetReasonMalformed BudgetReason = "budget_malformed"
	// BudgetReasonObservationUnavailable means no provider usage was reported.
	BudgetReasonObservationUnavailable BudgetReason = "observation_unavailable"
)

// BudgetInput describes one phase or role admission attempt.
type BudgetInput struct {
	Phase            string
	Role             string
	Content          string
	Limit            int64
	ObservedProvider *int64
}

// BudgetOutcome is a redacted admission decision. It intentionally contains
// counts only; neither content nor provider payloads are retained.
type BudgetOutcome struct {
	Admitted             bool
	Reason               BudgetReason
	EstimatedTokens      int64
	ObservedProvider     *int64
	RemainingTokens      int64
	ObservationAvailable bool
}

// EstimateWhitespaceTokens returns a deterministic local approximation. It is
// not a model tokenizer and must not be presented as provider token usage.
func EstimateWhitespaceTokens(content string) int64 {
	return int64(len(strings.Fields(content)))
}

// EvaluateBudget applies a per-phase/role limit without an unbounded fallback.
func EvaluateBudget(input BudgetInput) BudgetOutcome {
	estimate := EstimateWhitespaceTokens(input.Content)
	if input.Phase == "" || input.Role == "" || input.Limit < 0 {
		return BudgetOutcome{Reason: BudgetReasonMalformed, EstimatedTokens: estimate}
	}
	if estimate > input.Limit {
		return BudgetOutcome{
			Reason: BudgetReasonOverflow, EstimatedTokens: estimate,
			RemainingTokens: input.Limit - estimate,
		}
	}
	return BudgetOutcome{
		Admitted: true, Reason: BudgetReasonAdmitted, EstimatedTokens: estimate,
		ObservedProvider: input.ObservedProvider, RemainingTokens: input.Limit - estimate,
		ObservationAvailable: input.ObservedProvider != nil,
	}
}

// TelemetryAttributes exposes redacted accounting facts for telemetry sinks.
func (o BudgetOutcome) TelemetryAttributes() map[string]any {
	return map[string]any{
		"strategist.budget.admitted":              o.Admitted,
		"strategist.budget.reason":                string(o.Reason),
		"strategist.budget.estimated_tokens":      o.EstimatedTokens,
		"strategist.budget.remaining_tokens":      o.RemainingTokens,
		"strategist.budget.observation_available": o.ObservationAvailable,
		"strategist.budget.observed_provider":     o.ObservedProvider,
	}
}
