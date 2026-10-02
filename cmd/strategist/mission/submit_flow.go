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
	return submitPreflight{gateDigest: gateDigest, outcome: outcome, packageDigest: packageDigest}, nil
}

func repairPackageDigest(basePath, missionID string, event domain.MissionEngineEvent) (string, error) {
	if event != domain.MissionEventRefinementArtifactInvalid {
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

func finishSubmit(root, basePath, missionID string, evt domain.MissionEngineEvent, outcome *handoff.Outcome) error {
	if err := livemission.SealSideQuestOnGateApproval(root, missionID, evt, time.Now()); err != nil {
		return fmt.Errorf("seal side quest: %w", err)
	}
	return commitExecutionEntry(root, basePath, missionID, evt, outcome)
}

func commitExecutionEntry(root, basePath, missionID string, event domain.MissionEngineEvent, outcome *handoff.Outcome) error {
	if outcome != nil {
		if err := livemission.ConsumeHandoffOutcome(root, *outcome); err != nil {
			return fmt.Errorf("consume handoff outcome: %w", err)
		}
	}
	return recordSniperClaimsOnHandoffPassed(root, basePath, missionID, event)
}
