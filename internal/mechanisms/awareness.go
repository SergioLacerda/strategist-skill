package mechanisms

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// BriefForStage produces the awareness view for a Role situated in one
// canonical Stage. The legacy Brief methods remain role-only compatibility
// adapters; new consumers should bind awareness to both Role and Stage.
func (r Registry) BriefForStage(role string, stage domain.Stage) (string, error) {
	return r.renderStage(role, stage, false)
}

// BriefFullForStage is the detailed Role+Stage awareness view.
func (r Registry) BriefFullForStage(role string, stage domain.Stage) (string, error) {
	return r.renderStage(role, stage, true)
}

func (r Registry) renderStage(role string, stage domain.Stage, full bool) (string, error) {
	if err := stage.Validate(); err != nil {
		return "", fmt.Errorf("validate Stage for mechanisms awareness: %w", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Tools, Mechanisms, and Feats available to %s in %s:\n", role, stage)
	for _, row := range r.ForRole(role) {
		if !rowAppliesToStage(row, stage) {
			continue
		}
		b.WriteString(briefLine(row, full))
	}
	return b.String(), nil
}

func rowAppliesToStage(row Row, stage domain.Stage) bool {
	if len(row.PhaseScope) == 0 {
		return true
	}
	allowed := stagePhaseScopes(stage)
	for _, phase := range row.PhaseScope {
		if phase == "all" || allowed[phase] {
			return true
		}
	}
	return false
}

func stagePhaseScopes(stage domain.Stage) map[string]bool {
	switch stage {
	case domain.StageFull:
		return map[string]bool{
			"bootstrap": true, "intake": true, "discovery": true,
			"refinement": true, "approval_gate": true, "execution": true,
		}
	case domain.StageShort:
		return map[string]bool{"bootstrap": true, "intake": true, "execution": true}
	case domain.StageRoster:
		return map[string]bool{"bootstrap": true, "roster": true}
	default:
		return nil
	}
}
