package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
)

func isCriticalHitInvocation(root, missionID string) bool {
	_, status, err := loadMission(root, missionID)
	return err == nil && status.Stage == domain.StageShort && status.StageFeat == "critical_hit" && status.StageGateApproved
}

func verifyCriticalHitCompletion(input missionadapter.InvocationCompleteInput, record domain.MissionInvocationRecord) (string, []byte, error) {
	expectedReportPath, err := sniperReportRelativePath(input.Root, input.BasePath, record.Request.MissionID)
	if err != nil {
		return "", nil, err
	}
	if err := verifySniperCompletionSignal(input.Completion.Result, expectedReportPath); err != nil {
		return "", nil, err
	}
	reportPath := filepath.Join(input.BasePath, "archived", record.Request.MissionID+"-report.md")
	report, err := os.ReadFile(reportPath) //nolint:gosec // report path is derived from the active mission base path
	if err != nil || len(strings.TrimSpace(string(report))) == 0 {
		return "", nil, fmt.Errorf("role_invocation_failed: Critical Hit completion report is missing or empty")
	}
	return reportPath, report, nil
}

func completeCriticalHitMission(root, missionID string) error {
	if !isCriticalHitInvocation(root, missionID) {
		return nil
	}
	if err := missionruntime.Lock(root, missionID, func() error {
		engine, status, err := loadMission(root, missionID)
		if err != nil {
			return fmt.Errorf("load Critical Hit mission for completion: %w", err)
		}
		return completeCriticalHitMissionLocked(root, engine, status)
	}); err != nil {
		return fmt.Errorf("lock Critical Hit mission for completion: %w", err)
	}
	return nil
}

func completeCriticalHitMissionLocked(root string, engine *domain.MissionEngine, status domain.MissionEngineStatus) error {
	if status.State == domain.StateDoneDelivery {
		return nil
	}
	if status.State != domain.StateExecution || !status.StageGateApproved {
		return fmt.Errorf("role_invocation_failed: Critical Hit completion requires approved execution state")
	}
	next, err := engine.Submit(domain.MissionEventSniperDone)
	if err != nil {
		return fmt.Errorf("complete Critical Hit mission: %w", err)
	}
	if err := saveMission(root, next); err != nil {
		return fmt.Errorf("persist Critical Hit completion: %w", err)
	}
	return nil
}
