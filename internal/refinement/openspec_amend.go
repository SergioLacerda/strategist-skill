package refinement

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// AmendInput identifies a post-publication amendment of a refined package: a new
// completed OpenSpec change that replaces proposal.md, design.md and tasks.md while
// analysis.md, the mission status and the original provider_change_id stay as they
// are. BasePath and RuntimeRoot must be resolved absolute paths.
type AmendInput struct {
	MissionID   string
	BasePath    string
	RuntimeRoot string
	// ChangeID is the amending change; Amends is the change it supersedes (the
	// package's provider_change_id, or the previous amendment's change id).
	ChangeID string
	Amends   string
	// AuthorizationRef is a human's authorization, recorded verbatim; the bridge
	// cannot verify a human, like `metrics gate-outcome --ref`.
	AuthorizationRef string
	// GateLabel is the mission's gate outcome label ("" when there is none).
	GateLabel string
	// PersistedStatus is the mission FSM snapshot read from missions/<id>.json.
	// Amendment eligibility fails closed when it is absent or disagrees with the
	// package frontmatter and gate telemetry.
	PersistedStatus domain.MissionEngineStatus
	// Reason is the operator's correction rationale. Empty uses the derived
	// same-mission repair/amendment disposition.
	Reason string
	// SupersedesMissionID links this amendment to a prior mission when the
	// operator is reconciling an equivalent replacement lineage.
	SupersedesMissionID string
	// Now is injected so tests are deterministic; nil uses the wall clock.
	Now func() time.Time

	// replaceHook, when set, runs before each file replacement and may fail it.
	replaceHook func(name string) error
}

// AmendResult describes the amendment that was applied.
type AmendResult struct {
	RefinedPath      string
	AmendmentDir     string
	Amendment        int
	ProviderChangeID string
	Status           string
}

// AmendOpenSpec applies an amendment after every precondition holds; the default
// NormalizeOpenSpec path is untouched. Nothing is written until the whole plan is
// valid. The previous three files are snapshotted under <package>/.amendments/NNN
// before they are replaced, and restored from it if a replacement fails.
func AmendOpenSpec(input AmendInput) (AmendResult, error) {
	plan, err := planAmendment(input)
	if err != nil {
		return AmendResult{}, err
	}
	if err := plan.apply(); err != nil {
		return AmendResult{}, err
	}
	if err := cleanupOpenSpecScratch(plan.changeDir); err != nil {
		return AmendResult{}, fmt.Errorf("openspec amend: amendment %03d applied but provider scratch cleanup failed: %w", plan.number, err)
	}
	return AmendResult{RefinedPath: plan.refined, AmendmentDir: plan.snapshotDir, Amendment: plan.number, ProviderChangeID: plan.original, Status: plan.status}, nil
}
