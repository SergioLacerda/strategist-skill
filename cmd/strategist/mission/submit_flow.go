package mission

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
)

func invalidateRepairEvidence(root, missionID string, event domain.MissionEngineEvent) error {
	if event != domain.MissionEventRefinementArtifactInvalid {
		return nil
	}
	if _, err := handoff.NewOutcomeStore(root).InvalidateLatest(missionID, string(event)); err != nil {
		return fmt.Errorf("invalidate handoff outcome: %w", err)
	}
	return nil
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

type submitPreflight struct {
	gateDigest    string
	outcome       *handoff.Outcome
	packageDigest string
	claimTargets  []string
}

func preflightSubmit(root, basePath, missionID string, status domain.MissionEngineStatus, evt domain.MissionEngineEvent) (submitPreflight, error) {
	if err := livemission.RequireNoAcceptedSideQuest(root, missionID, evt); err != nil {
		return submitPreflight{}, fmt.Errorf("mission submit: rejected: %w", err)
	}
	if err := requireAuthoredPackageRepair(basePath, missionID, evt); err != nil {
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
	packageDigest, err := repairPackageDigest(basePath, missionID, evt)
	if err != nil {
		return submitPreflight{}, err
	}
	claimTargets, err := preflightSniperClaims(basePath, missionID, evt)
	if err != nil {
		return submitPreflight{}, err
	}
	return submitPreflight{gateDigest: gateDigest, outcome: outcome, packageDigest: packageDigest, claimTargets: claimTargets}, nil
}

// preflightSniperClaims validates every claim target before the FSM is
// advanced. RecordSniperClaims repeats the extraction when it appends the
// telemetry records, but this earlier pass is the transactional boundary for
// deterministic package errors: malformed targets cannot leave a mission in
// execution merely because claim recording would reject them later.
func preflightSniperClaims(basePath, missionID string, event domain.MissionEngineEvent) ([]string, error) {
	if event != domain.MissionEventHandoffSatisfied {
		return nil, nil
	}
	refined := filepath.Join(basePath, "refined", missionID)
	pkg, err := handoff.LoadRefinedPackageForGate(refined, missionID)
	if err != nil {
		return nil, fmt.Errorf("mission submit: rejected: preflight sniper claims: %w", err)
	}
	return pkg.DocumentationTargets, nil
}

func repairPackageDigest(basePath, missionID string, event domain.MissionEngineEvent) (string, error) {
	if event != domain.MissionEventRefinementArtifactInvalid && event != domain.MissionEventHandoffSatisfied {
		return "", nil
	}
	digest, err := handoff.PackageDigest(filepath.Join(basePath, "refined", missionID))
	if err != nil {
		return "", fmt.Errorf("mission submit: rejected: read repair package digest: %w", err)
	}
	return digest, nil
}

func retainRepairEvidence(basePath string, before, after domain.MissionEngineStatus, packageDigest string, event domain.MissionEngineEvent) error {
	if event != domain.MissionEventRefinementArtifactInvalid {
		return nil
	}
	if err := refinement.RetainRepairEvidence(basePath, before, after, packageDigest, string(event), time.Now().UTC()); err != nil {
		return fmt.Errorf("retain repair evidence: %w", err)
	}
	return nil
}

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

func finishSubmit(root, basePath, missionID string, evt domain.MissionEngineEvent, outcome *handoff.Outcome, entry *executionEntry) error {
	if err := livemission.SealSideQuestOnGateApproval(root, missionID, evt, time.Now()); err != nil {
		return fmt.Errorf("seal side quest: %w", err)
	}
	if entry != nil {
		entry.SideQuestSealed = true
		if err := saveExecutionEntry(root, entry); err != nil {
			return err
		}
	}
	return commitExecutionEntry(root, basePath, missionID, evt, outcome, entry)
}

func consumeExecutionOutcome(root string, outcome handoff.Outcome) error {
	consumed, err := handoff.NewOutcomeStore(root).Consumed(outcome)
	if err != nil {
		return fmt.Errorf("read handoff outcome: %w", err)
	}
	if consumed {
		return nil
	}
	if err := executionEntryConsume(root, outcome); err != nil {
		return fmt.Errorf("consume handoff outcome: %w", err)
	}
	return nil
}

func recordExecutionEntryClaims(root, basePath, missionID string, entry *executionEntry) error {
	if _, err := executionEntryRecordClaims(root, basePath, missionID, entry.ID, entry.Targets, time.Now().UTC()); err != nil {
		return fmt.Errorf("record sniper claims: %w", err)
	}
	entry.ClaimsRecorded, entry.Completed = true, true
	return saveExecutionEntry(root, entry)
}

func persistExecutionOutcome(root string, outcome *handoff.Outcome, entry *executionEntry) error {
	if outcome == nil {
		return nil
	}
	if err := consumeExecutionOutcome(root, *outcome); err != nil {
		return err
	}
	if entry == nil {
		return nil
	}
	entry.OutcomeConsumed = true
	if err := saveExecutionEntry(root, entry); err != nil {
		return err
	}
	return nil
}

func commitExecutionEntry(root, basePath, missionID string, event domain.MissionEngineEvent, outcome *handoff.Outcome, entry *executionEntry) error {
	if err := persistExecutionOutcome(root, outcome, entry); err != nil {
		return err
	}
	if entry != nil && event == domain.MissionEventHandoffSatisfied {
		return recordExecutionEntryClaims(root, basePath, missionID, entry)
	}
	return recordSniperClaimsOnHandoffPassed(root, basePath, missionID, event)
}
