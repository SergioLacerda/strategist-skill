package mission

import (
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
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
	before := engine.Status()
	status, err := applySubmit(engine, evt, pre.gateDigest)
	if err != nil {
		return domain.MissionEngineStatus{}, err
	}
	if err := commitSubmit(deps, root, basePath, missionID, evt, before, status, pre); err != nil {
		return domain.MissionEngineStatus{}, err
	}
	return status, nil
}

func commitSubmit(deps LifecycleDependencies, root, basePath, missionID string, evt domain.MissionEngineEvent, before, status domain.MissionEngineStatus, pre submitPreflight) error {
	if err := invalidateRepairEvidence(root, missionID, evt); err != nil {
		return fmt.Errorf("mission submit: %w", err)
	}
	if err := retainRepairEvidence(basePath, before, status, pre.packageDigest, evt); err != nil {
		return fmt.Errorf("mission submit: %w", err)
	}
	analysisPath := filepath.Join(basePath, "refined", missionID, "analysis.md")
	original, changed, err := acceptGateAnalysis(analysisPath, evt, pre.gateDigest)
	if err != nil {
		return fmt.Errorf("mission submit: %w", err)
	}
	if err := persistSubmitState(deps, root, status, analysisPath, original, changed); err != nil {
		return err
	}
	if err := finishSubmit(root, basePath, missionID, evt, pre.outcome); err != nil {
		return fmt.Errorf("mission submit: %w", err)
	}
	return nil
}
