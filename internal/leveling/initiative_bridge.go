package leveling

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/initiative"
)

// SignalsFromEscalation converts the closed PRECISE-SHOT advisory contract to
// LEVELING's existing provider-neutral signal vocabulary. It does not resolve
// a suggestion and cannot mutate LEVELING policy or its ledger.
func SignalsFromEscalation(request initiative.EscalationRequest) (Signals, error) {
	if request.Mechanism != initiative.PreciseShotMechanism {
		return Signals{}, fmt.Errorf("leveling_precise_shot_invalid: unsupported mechanism %q", request.Mechanism)
	}
	if strings.TrimSpace(request.AssessmentID) == "" || strings.TrimSpace(request.IdempotencyKey) == "" {
		return Signals{}, fmt.Errorf("leveling_precise_shot_invalid: assessment and idempotency identities are required")
	}
	signals := Signals{
		Ambiguity:           request.Signals.Ambiguity,
		Risk:                request.Signals.Risk,
		Scope:               request.Signals.Scope,
		Evidence:            request.Signals.Evidence,
		ArchitecturalChange: request.Signals.ArchitecturalChange,
		SecuritySensitive:   request.Signals.SecuritySensitive,
		ConflictingEvidence: request.Signals.ConflictingEvidence,
		RepeatedFailures:    request.Signals.RepeatedFailures,
	}
	if request.Effective == initiative.ConfidenceLow && signals.Evidence == "" {
		signals.Evidence = "insufficient"
	}
	if err := validateSignals(signals); err != nil {
		return Signals{}, fmt.Errorf("leveling_precise_shot_invalid: %w", err)
	}
	return signals, nil
}
