package mission

import (
	"fmt"
	"slices"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/initiative"
)

// InitiativeRoleEntry is the role-boundary input for the consultative
// INITIATIVE ability. Observed execution facts are copied from the caller;
// INITIATIVE never resolves or changes them.
type InitiativeRoleEntry struct {
	MissionID string
	Role      string
	RunID     string
	Observed  initiative.Observation
}

// InitiativeHandoff carries advisory metadata across a role boundary. It is
// additive and optional so legacy handoffs remain readable without advice.
type InitiativeHandoff struct {
	FromRole   string
	ToRole     string
	Advice     initiative.Advice
	Result     initiative.Result
	Assessment initiative.ResultAssessment
}

// Validate checks correlation and the advisory boundary for a handoff. It
// does not inspect or authorize Approval Gate or LEVELING state.
func (h InitiativeHandoff) Validate() error {
	if strings.TrimSpace(h.FromRole) == "" || strings.TrimSpace(h.ToRole) == "" {
		return fmt.Errorf("initiative handoff: source and destination roles are required")
	}
	if err := validateInitiativeTransition(h.FromRole, h.ToRole); err != nil {
		return err
	}
	if h.Advice.Role != strings.ToLower(strings.TrimSpace(h.FromRole)) {
		return fmt.Errorf("initiative handoff: advice role does not match source role")
	}
	return h.validateResult()
}

func (h InitiativeHandoff) validateResult() error {
	if err := h.Result.ValidateAgainst(h.Advice); err != nil {
		return fmt.Errorf("initiative handoff: validate result: %w", err)
	}
	assessment, err := initiative.AssessResult(h.Advice, h.Result)
	if err != nil {
		return fmt.Errorf("initiative handoff: assess result: %w", err)
	}
	if assessment.Challenge != h.Assessment.Challenge || assessment.ConfidenceCeiling != h.Assessment.ConfidenceCeiling || !slices.Equal(assessment.Reasons, h.Assessment.Reasons) {
		return fmt.Errorf("initiative handoff: assessment does not match result")
	}
	return nil
}

var initiativeTransitions = map[string]string{
	"scout":     "ranger",
	"ranger":    "archivist",
	"archivist": "sniper",
}

func validateInitiativeTransition(fromRole, toRole string) error {
	from := strings.ToLower(strings.TrimSpace(fromRole))
	to := strings.ToLower(strings.TrimSpace(toRole))
	if initiativeTransitions[from] != to {
		return fmt.Errorf("initiative handoff: transition %q to %q is not allowed", from, to)
	}
	return nil
}
