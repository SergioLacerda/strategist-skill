package mission

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
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
	pre, err := preflightSubmit(root, basePath, missionID, engine.Status(), evt)
	if err != nil {
		return domain.MissionEngineStatus{}, err
	}
	status, err := applySubmit(engine, evt, pre.gateDigest)
	if err != nil {
		return domain.MissionEngineStatus{}, err
	}
	analysisPath := filepath.Join(basePath, "refined", missionID, "analysis.md")
	original, changed, err := acceptGateAnalysis(analysisPath, evt, pre.gateDigest)
	if err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission submit: %w", err)
	}
	if err := persistSubmitState(deps, root, status, analysisPath, original, changed); err != nil {
		return domain.MissionEngineStatus{}, err
	}
	if err := finishSubmit(root, basePath, missionID, evt, pre.outcome); err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission submit: %w", err)
	}
	return status, nil
}

func persistSubmitState(deps LifecycleDependencies, root string, status domain.MissionEngineStatus, analysisPath string, original []byte, changed bool) error {
	if err := deps.Save(root, status); err != nil {
		if changed {
			if restoreErr := handoff.RestoreAnalysis(analysisPath, original); restoreErr != nil {
				return fmt.Errorf("mission submit: %v; rollback failed: %w", err, restoreErr)
			}
		}
		return fmt.Errorf("mission submit: %w", err)
	}
	return nil
}

func acceptGateAnalysis(path string, evt domain.MissionEngineEvent, gateDigest string) ([]byte, bool, error) {
	if evt != domain.MissionEventGateApproved || gateDigest == "" {
		return nil, false, nil
	}
	original, changed, err := handoff.AcceptAnalysisAtGate(path)
	if err != nil {
		return nil, false, fmt.Errorf("accept approval-gate analysis: %w", err)
	}
	return original, changed, nil
}

// submitPreflight is what the guards derive before the event is applied.
type submitPreflight struct {
	gateDigest string
	outcome    *handoff.Outcome
}

// preflightSubmit runs every guard that must pass before the transition: the
// accepted-side-quest conflict, the approval gate package digest, and the
// execution-entry evidence.
func preflightSubmit(root, basePath, missionID string, status domain.MissionEngineStatus, evt domain.MissionEngineEvent) (submitPreflight, error) {
	if err := livemission.RequireNoAcceptedSideQuest(root, missionID, evt); err != nil {
		return submitPreflight{}, fmt.Errorf("mission submit: rejected: %w", err)
	}
	if err := validateSubmitArtifacts(basePath, missionID, evt); err != nil {
		return submitPreflight{}, fmt.Errorf("mission submit: rejected: %w", err)
	}
	gateDigest, err := approvalGatePackageDigest(basePath, status, evt)
	if err != nil {
		return submitPreflight{}, fmt.Errorf("mission submit: rejected: %w", err)
	}
	outcome, err := requireExecutionEvidence(root, basePath, status, evt)
	if err != nil {
		return submitPreflight{}, err
	}
	return submitPreflight{gateDigest: gateDigest, outcome: outcome}, nil
}

// applySubmit applies the event and, when the gate bound a package digest,
// records it on the new status.
func applySubmit(engine *domain.MissionEngine, evt domain.MissionEngineEvent, gateDigest string) (domain.MissionEngineStatus, error) {
	status, err := engine.Submit(evt)
	if err == nil && gateDigest != "" {
		status, err = engine.RecordApprovalGatePackageDigest(gateDigest)
	}
	if err != nil {
		return domain.MissionEngineStatus{}, fmt.Errorf("mission submit: rejected: %w", err)
	}
	return status, nil
}

// finishSubmit runs the steps that follow a saved transition: sealing the
// OA-ADR record and, on execution entry, consuming the handoff outcome and
// recording the Sniper claims.
func finishSubmit(root, basePath, missionID string, evt domain.MissionEngineEvent, outcome *handoff.Outcome) error {
	if err := livemission.SealSideQuestOnGateApproval(root, missionID, evt, time.Now()); err != nil {
		return fmt.Errorf("seal side quest: %w", err)
	}
	return commitExecutionEntry(root, basePath, missionID, evt, outcome)
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
