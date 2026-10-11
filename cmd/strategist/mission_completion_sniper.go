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

func completeSniperInvocation(store missionruntime.InvocationStore, input missionadapter.InvocationCompleteInput, record domain.MissionInvocationRecord) (domain.MissionInvocationOutcome, error) {
	reportPath, report, err := verifySniperCompletion(input, record)
	if err != nil {
		return domain.MissionInvocationOutcome{}, err
	}
	relative, err := filepath.Rel(filepath.Dir(input.Root), reportPath)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return domain.MissionInvocationOutcome{}, fmt.Errorf("role_invocation_failed: Sniper report path escapes workspace")
	}
	digest := contentDigest(report)
	artifactPath := filepath.ToSlash(relative)
	if err := store.BeginProcessing(record.Request.RequestID, artifactPath, digest); err != nil {
		return domain.MissionInvocationOutcome{}, fmt.Errorf("begin mission invocation: %w", err)
	}
	if err := store.Complete(record.Request.RequestID, digest); err != nil {
		return domain.MissionInvocationOutcome{}, fmt.Errorf("complete mission invocation: %w", err)
	}
	if err := completeCriticalHitMission(input.Root, record.Request.MissionID); err != nil {
		return domain.MissionInvocationOutcome{}, err
	}
	return domain.MissionInvocationOutcome{RequestID: input.RequestID, MissionID: record.Request.MissionID, Status: "verified", ArtifactPath: artifactPath, BindingDigest: record.Request.BindingDigest, SourceDigest: record.Request.SourceDigest}, nil
}

func verifySniperCompletion(input missionadapter.InvocationCompleteInput, record domain.MissionInvocationRecord) (string, []byte, error) {
	if err := verifySniperAuthorization(input, record); err != nil {
		return "", nil, err
	}
	if isCriticalHitInvocation(input.Root, record.Request.MissionID) {
		return verifyCriticalHitCompletion(input, record)
	}
	refined := filepath.Join(input.BasePath, "refined", record.Request.MissionID)
	executionSnapshot, err := missionadapter.LoadExecutionSnapshot(input.Root, input.BasePath, record.Request.MissionID)
	if err != nil {
		return "", nil, fmt.Errorf("role_invocation_failed: load execution targets: %w", err)
	}
	targets, err := verifySniperLifecycleAndSnapshot(refined, record.Request.MissionID, executionSnapshot)
	if err != nil {
		return "", nil, err
	}
	if err := requireSniperMaterializations(input.Root, record, targets); err != nil {
		return "", nil, err
	}
	reportPath := filepath.Join(input.BasePath, "archived", record.Request.MissionID+"-report.md")
	report, err := os.ReadFile(reportPath) //nolint:gosec // report path is derived from the active mission base path
	if err != nil || len(strings.TrimSpace(string(report))) == 0 {
		return "", nil, fmt.Errorf("role_invocation_failed: Sniper report is missing or empty")
	}
	return reportPath, report, nil
}
