package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
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
	targets, err := verifySniperLifecycleAndTasks(refined, record.Request.MissionID)
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

func verifySniperAuthorization(input missionadapter.InvocationCompleteInput, record domain.MissionInvocationRecord) error {
	if input.Adapter != domain.ExecutionAdapterCurrentHost {
		return fmt.Errorf("invocation_adapter_mismatch: Sniper execution requires the current-host adapter")
	}
	expectedReportPath, err := sniperReportRelativePath(input.Root, input.BasePath, record.Request.MissionID)
	if err != nil {
		return err
	}
	if err := verifySniperCompletionSignal(input.Completion.Result, expectedReportPath); err != nil {
		return err
	}
	_, status, err := loadMission(input.Root, record.Request.MissionID)
	if err != nil {
		return fmt.Errorf("role_invocation_failed: %w", err)
	}
	acceptedDigest, ok := record.Request.Input["approval_gate_package_digest"].(string)
	if !ok {
		acceptedDigest = ""
	}
	if acceptedDigest == "" || status.ApprovalGatePackageDigest != acceptedDigest || status.State != domain.StateExecution {
		return fmt.Errorf("role_invocation_failed: approval_gate_package_changed: Sniper request is no longer authorized")
	}
	return nil
}

func sniperReportRelativePath(root, basePath, missionID string) (string, error) {
	report := filepath.Join(basePath, "archived", missionID+"-report.md")
	relative, err := filepath.Rel(filepath.Dir(root), report)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("role_invocation_failed: Sniper report path escapes workspace")
	}
	return filepath.ToSlash(relative), nil
}

// verifySniperCompletionSignal accepts exactly the protocol line declared by
// Sniper. Substring checks allow frontmatter, a second status, or a forged
// report_path to masquerade as a completion signal; the canonical report and
// materialization ledger are verified separately after this parser succeeds.
func verifySniperCompletionSignal(raw, expectedReportPath string) error {
	line := strings.TrimSpace(raw)
	expected := "sniper: done | report_path: " + expectedReportPath + " | mission_status: documentation_applied"
	if line == expected || line == "Return: "+expected {
		return nil
	}
	return fmt.Errorf("role_invocation_failed: Sniper completion signal is invalid")
}

func verifySniperLifecycleAndTasks(refined, missionID string) ([]string, error) {
	lifecycle, err := handoff.ReadAnalysisLifecycle(filepath.Join(refined, "analysis.md"))
	if err != nil {
		return nil, fmt.Errorf("role_invocation_failed: %w", err)
	}
	if lifecycle.MissionID != missionID || lifecycle.Status != "documentation_applied" || strings.TrimSpace(lifecycle.ClaimedBy) == "" {
		return nil, fmt.Errorf("role_invocation_failed: Sniper lifecycle proof is incomplete")
	}
	targets, err := refinement.DocumentationTargetPaths(filepath.Join(refined, "tasks.md"))
	if err != nil || len(targets) == 0 {
		return nil, fmt.Errorf("role_invocation_failed: Sniper documentation targets are missing")
	}
	if err := requireCompletedDocumentationTargets(filepath.Join(refined, "tasks.md"), targets); err != nil {
		return nil, err
	}
	return targets, nil
}

func requireCompletedDocumentationTargets(tasksPath string, targets []string) error {
	raw, err := os.ReadFile(tasksPath) //nolint:gosec // tasks path is derived from the refined package
	if err != nil {
		return fmt.Errorf("role_invocation_failed: read Sniper tasks: %w", err)
	}
	for _, target := range targets {
		if !documentationTargetCompleted(raw, target) {
			return fmt.Errorf("role_invocation_failed: documentation target %q is not checked complete", target)
		}
	}
	return nil
}

func documentationTargetCompleted(raw []byte, target string) bool {
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(line, "`"+target+"`") && (strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]")) {
			return true
		}
	}
	return false
}

func requireSniperMaterializations(root string, record domain.MissionInvocationRecord, targets []string) error {
	records, err := telemetry.ReadRecentSniperMaterializations(telemetry.SniperMaterializationHistoryPath(root), time.Now().UTC(), telemetry.SniperMaterializationWindow)
	if err != nil {
		return fmt.Errorf("role_invocation_failed: read Sniper materializations: %w", err)
	}
	seen := sniperMaterializationTargets(records, record)
	for _, target := range targets {
		if !seen[target] {
			return fmt.Errorf("role_invocation_failed: documentation target %q has no materialization ledger proof for this request", target)
		}
	}
	return nil
}

func sniperMaterializationTargets(records []telemetry.SniperMaterializationRecord, record domain.MissionInvocationRecord) map[string]bool {
	seen := make(map[string]bool, len(records))
	expectedDigest := requestedPackageDigest(record)
	for _, materialization := range records {
		if materializationMatchesRequest(materialization, record, expectedDigest) {
			seen[materialization.TargetPath] = true
		}
	}
	return seen
}

func requestedPackageDigest(record domain.MissionInvocationRecord) string {
	if raw, ok := record.Request.Input["approval_gate_package_digest"]; ok {
		if digest, ok := raw.(string); ok {
			return digest
		}
	}
	return ""
}

func materializationMatchesRequest(materialization telemetry.SniperMaterializationRecord, record domain.MissionInvocationRecord, expectedDigest string) bool {
	return materialization.MissionID == record.Request.MissionID &&
		(materialization.PackageDigest == "" || materialization.PackageDigest == expectedDigest) &&
		!materialization.MaterializedAt.Before(record.CreatedAt)
}
