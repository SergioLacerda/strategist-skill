package mission

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
	"github.com/spf13/cobra"
)

// NewSubmit builds `mission submit`.
func NewSubmit(deps LifecycleDependencies) *cobra.Command {
	var event string
	cmd := &cobra.Command{Use: "submit", Short: "Submit one authoritative mission event"}
	f := bindLifecycleFlags(cmd, deps)
	cmd.Flags().StringVar(&event, "event", "", "mission event to submit (required)")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return RunSubmit(cmd, deps, f.root, f.missionID, event, f.asJSON)
	}
	return cmd
}

// RunSubmit applies one event through the restored domain.MissionEngine and
// persists the resulting status.
//
// The Load, execution-evidence check, Submit and Save all run inside one
// lock acquisition (deps.Lock, when configured) keyed by (root, missionID),
// so two concurrent `mission submit` calls for the same mission can never
// both read the same prior state and each save an event the other one lost —
// the read-modify-write race ADR-0057 § D2 closes. Pre-checks that do not
// read or write this mission's own state file (flag validation, the
// analysis-only-package guard) run before the lock is acquired.
func RunSubmit(cmd *cobra.Command, deps LifecycleDependencies, rootInput, missionID, event string, asJSON bool) error {
	if err := deps.RequireMissionID(missionID); err != nil {
		return err
	}
	evt := domain.MissionEngineEvent(event)
	if evt == "" {
		return fmt.Errorf("mission submit: --event is required")
	}
	root, basePath, err := deps.ResolveBasePath(rootInput)
	if err != nil {
		return fmt.Errorf("mission submit: %w", err)
	}
	if err := requireAnalysisOnlyPackage(basePath, missionID, evt); err != nil {
		return err
	}

	var status domain.MissionEngineStatus
	err = withMissionLock(deps, root, missionID, func() error {
		s, lockedErr := submitLocked(deps, root, basePath, missionID, evt)
		if lockedErr != nil {
			return lockedErr
		}
		status = s
		return nil
	})
	if err != nil {
		return err
	}
	return deps.WriteResult(cmd, asJSON, status)
}

// submitLocked is RunSubmit's read-modify-write critical section: Load,
// the execution-evidence check, Submit, Save, and the handoff_challenge_satisfied
// claim recording. Extracted out of RunSubmit's own lock closure so each step
// is a single, flat sequence rather than nested inside an anonymous function
// (which gocognit weighs more heavily for nesting).
func submitLocked(deps LifecycleDependencies, root, basePath, missionID string, evt domain.MissionEngineEvent) (domain.MissionEngineStatus, error) {
	engine, _, err := deps.Load(root, missionID)
	if err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission submit: %w", err)
	}
	outcome, err := requireExecutionEvidence(root, basePath, engine.Status(), evt)
	if err != nil {
		return domain.MissionEngineStatus{}, err
	}
	status, err := engine.Submit(evt)
	if err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission submit: rejected: %w", err)
	}
	if err := deps.Save(root, status); err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission submit: %w", err)
	}
	if err := commitExecutionEntry(root, basePath, missionID, evt, outcome); err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission submit: %w", err)
	}
	return status, nil
}

// commitExecutionEntry runs once the transition into execution is saved: it
// consumes the handoff outcome that authorized it, then records the Sniper
// claims. Both stay inside the mission lock.
func commitExecutionEntry(root, basePath, missionID string, event domain.MissionEngineEvent, outcome *handoff.Outcome) error {
	if outcome != nil {
		if err := livemission.ConsumeHandoffOutcome(root, *outcome); err != nil {
			return fmt.Errorf("consume handoff outcome: %w", err)
		}
	}
	return recordSniperClaimsOnHandoffPassed(root, basePath, missionID, event)
}

// requireAnalysisOnlyPackage rejects an analysis-only terminal event when the
// mission's refined tasks.md declares a documentation_target: those items must
// reach Sniper through the handoff challenge instead of being dropped.
func requireAnalysisOnlyPackage(basePath, missionID string, event domain.MissionEngineEvent) error {
	if !domain.MissionEventRequiresNoDocumentationTargets(event) {
		return nil
	}
	tasks := filepath.Join(basePath, "refined", missionID, "tasks.md")
	has, err := refinement.HasDocumentationTargets(tasks)
	if err != nil {
		return fmt.Errorf("mission submit: %w", err)
	}
	if has {
		return fmt.Errorf("mission submit: rejected: %s requires a package with no documentation_target, but %s declares one — use gate_approved, then verify the handoff and submit handoff_challenge_satisfied so Sniper materializes it", event, tasks)
	}
	return nil
}

// recordSniperClaimsOnHandoffPassed records a Sniper claim on each declared
// documentation_target once the mission has committed its transition into
// execution (event == handoff_challenge_satisfied, called after deps.Save
// succeeds and still inside the mission lock). Deriving claims here, rather
// than through a separate command the agent would have to remember to call,
// is the design.md Batch A decision for ADR-0057 § A1: the mechanism that
// must not be forgotten should not depend on being remembered. Every other
// event records nothing.
func recordSniperClaimsOnHandoffPassed(root, basePath, missionID string, event domain.MissionEngineEvent) error {
	if event != domain.MissionEventHandoffSatisfied {
		return nil
	}
	if _, err := livemission.RecordSniperClaims(root, basePath, missionID, time.Now().UTC()); err != nil {
		return fmt.Errorf("record sniper claims: %w", err)
	}
	return nil
}

// requireExecutionEvidence is the live execution boundary: entering Sniper
// execution is rejected as pipeline_bypass_detected unless the evidence the
// mission's Scout route requires is present (see
// internal/mission.EvaluateExecutionEntry), and unless a durable, correlated
// handoff outcome authorizes this package: passed, or a skip the package's own
// facts authorize. The event name alone proves nothing. It returns the outcome
// to consume once the transition is committed. Every other event is unguarded,
// including gate_approved_analysis_only, which never enters execution.
func requireExecutionEvidence(root, basePath string, status domain.MissionEngineStatus, event domain.MissionEngineEvent) (*handoff.Outcome, error) {
	if event != domain.MissionEventHandoffSatisfied {
		return nil, nil
	}
	decision, err := livemission.EvaluateExecutionEntry(root, basePath, status)
	if err != nil {
		return nil, fmt.Errorf("mission submit: %w", err)
	}
	if !decision.Allowed {
		return nil, fmt.Errorf("mission submit: rejected: %w", decision)
	}
	outcome, err := livemission.AuthorizeHandoffExecution(root, basePath, status)
	if err != nil {
		return nil, fmt.Errorf("mission submit: rejected: %w", err)
	}
	return &outcome, nil
}
