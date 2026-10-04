package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
)

func preflightSideQuestGuard(root, missionID string, event domain.MissionEngineEvent) error {
	if err := livemission.RequireNoAcceptedSideQuest(root, missionID, event); err != nil {
		return fmt.Errorf("mission submit: rejected: %w", err)
	}
	return nil
}

func preflightRepairGuard(basePath, missionID string, event domain.MissionEngineEvent) error {
	if err := requireAuthoredPackageRepair(basePath, missionID, event); err != nil {
		return fmt.Errorf("mission submit: rejected: %w", err)
	}
	return nil
}

func preflightArtifactGuard(basePath, missionID string, event domain.MissionEngineEvent) error {
	if err := validateSubmitArtifacts(basePath, missionID, event); err != nil {
		return fmt.Errorf("mission submit: rejected: %w", err)
	}
	return nil
}

func preflightGateDigest(basePath string, status domain.MissionEngineStatus, event domain.MissionEngineEvent) (string, error) {
	gateDigest, err := approvalGatePackageDigest(basePath, status, event)
	if err != nil {
		return "", fmt.Errorf("mission submit: rejected: %w", err)
	}
	return gateDigest, nil
}
