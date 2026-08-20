package governance

import (
	"context"
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/governancebridge"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// SDDBridge adapts the existing .sdd/ sync logic (RunSync/SyncReport) to the
// governancebridge.GovernanceBridge interface (UNC-02 resolution: this
// package is not renamed or generalized — it stays the concrete .sdd/
// adapter; the interface itself lives in the new internal/governancebridge
// package). RunSync and SyncReport are unchanged by this file — Evaluate is
// purely additive, built on top of the existing, stable public contract.
type SDDBridge struct {
	// SkillRoot is the same skillRoot argument RunSync already takes.
	SkillRoot string
	// SDDDir is the same sddDir argument RunSync already takes.
	SDDDir string
}

// NewSDDBridge returns an SDDBridge reading governance state from sddDir and
// reconciling skillRoot/skill.yaml against it.
func NewSDDBridge(skillRoot, sddDir string) SDDBridge {
	return SDDBridge{SkillRoot: skillRoot, SDDDir: sddDir}
}

// Evaluate answers request by running a dry-run sync against .sdd/ — Evaluate
// never writes skill.yaml (RunSync is always called with dryRun=true here),
// satisfying GovernanceBridge's read-only contract (acceptance check 6.7: no
// concurrent auto-correction between Strategist and external governance).
// Allowed is true only when every active mandate is already compliant or
// partially compliant per SyncReport.MandatesMissing being empty.
func (b SDDBridge) Evaluate(_ context.Context, request governancebridge.GovernanceRequest) (governancebridge.GovernanceDecision, error) {
	report, err := RunSync(b.SkillRoot, b.SDDDir, true)
	if err != nil {
		return governancebridge.GovernanceDecision{}, fmt.Errorf("sdd bridge: evaluate: %w", err)
	}

	decision := governancebridge.GovernanceDecision{
		Allowed:       len(report.MandatesMissing) == 0,
		PolicyID:      report.GovernanceFingerprint,
		Authority:     telemetry.AuthorityExternal("sdd"),
		CorrelationID: request.CorrelationID,
	}
	if !decision.Allowed {
		decision.Reason = "missing mandates: " + strings.Join(report.MandatesMissing, ", ")
	}
	return decision, nil
}

var _ governancebridge.GovernanceBridge = SDDBridge{}
