package governance

import (
	"context"
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/governancebridge"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// Bridge adapts any explicit governance.Source to the generic
// governancebridge.GovernanceBridge contract. It performs a dry-run only and
// never mutates the Strategist manifest while evaluating a policy question.
type Bridge struct {
	SkillRoot     string
	GovernanceDir string
	Source        Source
}

// NewBridge creates a provider-neutral bridge over source.
func NewBridge(skillRoot, governanceDir string, source Source) Bridge {
	return Bridge{SkillRoot: skillRoot, GovernanceDir: governanceDir, Source: source}
}

// Evaluate returns the decision produced by the explicitly selected source.
func (b Bridge) Evaluate(_ context.Context, request governancebridge.GovernanceRequest) (governancebridge.GovernanceDecision, error) {
	report, err := RunSync(b.SkillRoot, b.Source, b.GovernanceDir, true)
	if err != nil {
		return governancebridge.GovernanceDecision{}, fmt.Errorf("governance bridge: evaluate: %w", err)
	}

	correlationID := request.CorrelationID
	if correlationID == "" {
		correlationID = b.Source.Name() + ":" + report.GovernanceFingerprint
	}
	decision := governancebridge.GovernanceDecision{
		Allowed:       len(report.MandatesMissing) == 0,
		PolicyID:      report.GovernanceFingerprint,
		Authority:     telemetry.AuthorityExternal(b.Source.Name()),
		CorrelationID: correlationID,
	}
	if !decision.Allowed {
		decision.Reason = "missing mandates: " + strings.Join(report.MandatesMissing, ", ")
	}
	return decision, nil
}

var _ governancebridge.GovernanceBridge = Bridge{}
