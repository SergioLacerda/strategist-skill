package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	"github.com/spf13/cobra"
)

// rangerPrecheck checks the Ranger analysis artifact of a mission.
func rangerPrecheck(ctx context.Context, root, basePath, missionID string) handoffconsumer.Report {
	admission, granted := admitRangerBudget(ctx, root, missionID)
	if !granted {
		return rangerBudgetUnavailableReport()
	}
	report := rangerArtifactPrecheck(ctx, root, basePath, missionID)
	report.BudgetState = "granted"
	finishRangerBudget(ctx, admission, report)
	return report
}

func rangerArtifactPrecheck(ctx context.Context, root, basePath, missionID string) handoffconsumer.Report {
	artifact := filepath.Join(basePath, "pending", missionID+"-analysis.md")
	subject, err := handoff.RangerArtifactDigest(artifact)
	if err != nil {
		return unreadableReport()
	}
	source, err := handoffconsumer.NewRangerSource(artifact)
	if err != nil {
		return unreadableReport()
	}
	return precheckHandoff(ctx, root, missionID, handoff.TransitionRangerToArchivist, subject, source)
}

// archivistPrecheck checks the refined package of a mission.
func archivistPrecheck(ctx context.Context, root, basePath, missionID string) handoffconsumer.Report {
	refined := filepath.Join(basePath, "refined", missionID)
	subject, err := handoff.PackageDigest(refined)
	if err != nil {
		return unreadableReport()
	}
	source, err := handoffconsumer.NewPackageSource(refined)
	if err != nil {
		return unreadableReport()
	}
	report := precheckHandoff(ctx, root, missionID, handoff.TransitionArchivistToSniper, subject, source)
	report.BudgetState, report.BudgetReason = "not_applicable", "ranger_precheck_only"
	return report
}

// gatePrecheck reads the refined package with the provider before the Approval
// Gate. It is the advisory Archivist to Gate boundary: it records no lifecycle
// outcome, and the later Archivist to Sniper evaluation reuses it for the same
// package digest.
func gatePrecheck(ctx context.Context, root, basePath, missionID string) handoffconsumer.Report {
	refined := filepath.Join(basePath, "refined", missionID)
	subject, err := handoff.PackageDigest(refined)
	if err != nil {
		return unreadableReport()
	}
	source, err := handoffconsumer.NewPackageSource(refined)
	if err != nil {
		return unreadableReport()
	}
	report := precheckHandoff(ctx, root, missionID, handoffconsumer.TransitionArchivistToGate, subject, source)
	report.BudgetState, report.BudgetReason = "not_applicable", "ranger_precheck_only"
	return report
}

// unreadableReport is the silent case: the artifact cannot be read here, and the
// main path will report that precisely, so the pre-check simply does not apply.
func unreadableReport() handoffconsumer.Report {
	return handoffconsumer.Report{Status: handoffconsumer.StatusDisabled, Reason: integration.StateDisabled}
}

// rangerDelegate is the callback for the automatic Ranger skip: it pre-checks
// the artifact only when the skip is about to be recorded.
func rangerDelegate(ctx context.Context, root, basePath, missionID string) func(subject string) *handoff.Delegation {
	return func(string) *handoff.Delegation {
		return rangerPrecheck(ctx, root, basePath, missionID).Delegation
	}
}

// printDelegation tells the agent what the pre-check decided. It prints nothing
// when no integration is enabled, so existing output is unchanged.
func printDelegation(cmd *cobra.Command, report handoffconsumer.Report) {
	if report.Status == handoffconsumer.StatusDisabled {
		return
	}
	var line strings.Builder
	fmt.Fprintf(&line, "delegation: %s", report.Status)
	line.WriteString(delegationDetails(report))
	appendDelegationMetadata(&line, report)
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), line.String()); err != nil {
		return
	}
}

func delegationDetails(report handoffconsumer.Report) string {
	var details strings.Builder
	switch report.Status { //nolint:exhaustive // approved and signal print their detail; every other status prints its reason
	case handoffconsumer.StatusApproved:
		fmt.Fprintf(&details, " provider=%s model=%s confidence=%.2f threshold=%.2f", report.Delegation.Provider, report.Delegation.Model, report.Confidence, report.Threshold)
	case handoffconsumer.StatusSignal:
		fmt.Fprintf(&details, " confidence=%.2f threshold=%.2f path=main hints=%s", report.Confidence, report.Threshold, hintList(report.Hints))
	case handoffconsumer.StatusShadow:
		fmt.Fprintf(&details, " path=main reading=recorded would_approve=%t", report.WouldApprove)
	default:
		fmt.Fprintf(&details, " reason=%s path=main", report.Reason)
	}
	return details.String()
}

func appendDelegationMetadata(line *strings.Builder, report handoffconsumer.Report) {
	if report.Decision.Suspect {
		line.WriteString(" integrity=suspect")
	}
	if report.Reused {
		line.WriteString(" reused=true")
	}
	if report.BudgetState == "" {
		return
	}
	fmt.Fprintf(line, " budget=%s", report.BudgetState)
	if report.BudgetReason != "" {
		fmt.Fprintf(line, " reason=%s", report.BudgetReason)
	}
}

// commandContext returns the command's context, or a background one when the
// command was built without it (tests, direct calls).
func commandContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

func hintList(hints []handoffconsumer.Hint) string {
	names := make([]string, 0, len(hints))
	for _, hint := range hints {
		names = append(names, hint.Criterion)
	}
	return strings.Join(names, ",")
}
