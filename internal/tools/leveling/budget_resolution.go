package leveling

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// BaseResolutionID is the canonical identity of the resolution an envelope came
// out of: mission, run, role, provider, policy digest, model and capability. It
// is deterministic, so a replayed label and an independent resolution of the same
// facts agree, and it deliberately ignores the limits and escalation fields so the
// id names the resolution rather than the budget derived from it. An escalated
// child gets its own id (see EscalateBudget) and names this one as its parent.
func BaseResolutionID(budget domain.EffortBudget) string {
	material := fmt.Sprintf("base|%s|%s|%s|%s|%s|%s|%s",
		budget.MissionID, budget.RunID, NormalizeRole(budget.Role), budget.Provider,
		budget.PolicyDigest, budget.Model, budget.Capability)
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])
}

// EscalationDenialReason returns the stable reason code of an error returned by
// EscalateBudget, so callers can report denied, exhausted and invalid apart from
// each other and from a provider failure.
func EscalationDenialReason(err error) string {
	if err == nil {
		return ""
	}
	code, _, _ := strings.Cut(err.Error(), ":")
	return strings.TrimSpace(code)
}
