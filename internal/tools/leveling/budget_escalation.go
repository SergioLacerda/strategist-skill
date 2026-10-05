package leveling

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// EscalateBudget creates a new bounded envelope. The parent remains unchanged
// and the child records the parent identity and explicit escalation reason.
func EscalateBudget(parent domain.EffortBudget, parentResolutionID, reason string) (domain.EffortBudget, error) {
	if err := parent.Validate(); err != nil {
		return domain.EffortBudget{}, fmt.Errorf("%s: %w", ReasonLevelingBudgetInvalid, err)
	}
	if !parent.EscalationAllowed {
		return domain.EffortBudget{}, fmt.Errorf("%s: escalation is disabled", ReasonLevelingBudgetEscalationDenied)
	}
	if parent.EscalationLevel >= parent.MaxEscalations {
		return domain.EffortBudget{}, fmt.Errorf("%s: maximum escalations reached", ReasonLevelingBudgetEscalationExhausted)
	}
	parentResolutionID = strings.TrimSpace(parentResolutionID)
	if parentResolutionID == "" {
		return domain.EffortBudget{}, fmt.Errorf("%s: parent resolution id is required", ReasonLevelingBudgetEscalationDenied)
	}
	child := parent
	child.EscalationLevel++
	child.ParentResolutionID = parentResolutionID
	child.EscalationReason = strings.TrimSpace(reason)
	child.ResolutionID = budgetResolutionID(child, parentResolutionID)
	child.MaxDelegations++
	if child.MaxDepth > 0 {
		child.MaxDepth++
	}
	return child, nil
}

func budgetResolutionID(budget domain.EffortBudget, parentID string) string {
	material := fmt.Sprintf("%s|%s|%d|%s", parentID, budget.Role, budget.EscalationLevel, budget.EscalationReason)
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])
}
