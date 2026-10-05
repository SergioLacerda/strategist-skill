package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
)

type rangerBudgetAdmission struct {
	state  leveling.BudgetState
	path   string
	budget domain.EffortBudget
}

func admitRangerBudget(ctx context.Context, root, missionID string) (rangerBudgetAdmission, bool) {
	path, configErr := rangerBudgetLedgerPath(root)
	if configErr != nil {
		return denyRangerBudget(ctx, path, missionID, leveling.ReasonLevelingBudgetUnavailable)
	}
	budget, found := loadRangerBudget(root, missionID)
	if !found {
		return denyRangerBudget(ctx, path, missionID, leveling.ReasonLevelingBudgetUnavailable)
	}
	normalizeRangerBudget(&budget, missionID)
	state, err := leveling.NewBudgetState(budget)
	if err != nil {
		appendRangerBudgetEvent(ctx, path, budget, leveling.BudgetEventDenial, leveling.ReasonLevelingBudgetInvalid, "unavailable", 0, 0)
		return rangerBudgetAdmission{}, false
	}
	decision := state.Begin(leveling.DelegationRequest{
		MissionID: missionID, RunID: missionID, Role: "ranger", Kind: leveling.DelegationSpecialist,
		Depth: 0, ContextBudget: budget.ContextBudget, EvidenceDepth: budget.EvidenceDepth,
	})
	event := rangerBudgetEvent(decision)
	if !decision.Granted {
		appendRangerBudgetEvent(ctx, path, budget, event, decision.Reason, "unverified", decision.RemainingDelegations, decision.ActiveDelegations)
		return rangerBudgetAdmission{}, false
	}
	if !appendRangerBudgetEvent(ctx, path, budget, event, decision.Reason, "unverified", decision.RemainingDelegations, decision.ActiveDelegations) {
		return rangerBudgetAdmission{}, false
	}
	return rangerBudgetAdmission{state: state, path: path, budget: budget}, true
}

func denyRangerBudget(ctx context.Context, path, missionID, reason string) (rangerBudgetAdmission, bool) {
	appendRangerBudgetEvent(ctx, path, unavailableRangerBudget(missionID), leveling.BudgetEventDenial, reason, "unavailable", 0, 0)
	return rangerBudgetAdmission{}, false
}

func loadRangerBudget(root, missionID string) (domain.EffortBudget, bool) {
	record, found, err := leveling.LatestRunRecord(filepath.Join(root, "memory", roleLevelLedger), missionID, "ranger", missionID)
	if err != nil || !found || record.Budget == nil {
		return domain.EffortBudget{}, false
	}
	return *record.Budget, true
}

func rangerBudgetEvent(decision leveling.DelegationDecision) leveling.BudgetEventKind {
	if decision.Reason == leveling.ReasonLevelingBudgetExhausted {
		return leveling.BudgetEventExhaustion
	}
	if decision.Granted {
		return leveling.BudgetEventGrant
	}
	return leveling.BudgetEventDenial
}

func rangerBudgetLedgerPath(root string) (string, error) {
	config, warning := readActiveLevelingConfig(root)
	if warning != "" {
		return filepath.Join(root, domain.DefaultBudgetLedgerPath), fmt.Errorf("leveling config unavailable: %s", warning)
	}
	path, err := leveling.ResolveBudgetLedgerPath(root, config)
	if err != nil {
		return "", fmt.Errorf("resolve budget ledger path: %w", err)
	}
	return path, nil
}

func normalizeRangerBudget(budget *domain.EffortBudget, missionID string) {
	if budget.MissionID == "" {
		budget.MissionID = missionID
	}
	if budget.RunID == "" {
		budget.RunID = missionID
	}
	if budget.Role == "" {
		budget.Role = "ranger"
	}
}

func unavailableRangerBudget(missionID string) domain.EffortBudget {
	return domain.EffortBudget{SchemaVersion: domain.EffortBudgetSchemaVersion, MissionID: missionID, RunID: missionID, Role: "ranger"}
}

func appendRangerBudgetEvent(ctx context.Context, path string, budget domain.EffortBudget, event leveling.BudgetEventKind, reason, evidence string, remaining, active int) bool {
	if err := leveling.AppendBudgetEvent(path, leveling.NewBudgetEventRecord(budget, event, reason, evidence, remaining, active)); err != nil {
		return false
	}
	emitRangerBudgetTelemetry(ctx, budget, event, reason, evidence, remaining, active)
	return true
}

func emitRangerBudgetTelemetry(ctx context.Context, budget domain.EffortBudget, event leveling.BudgetEventKind, reason, evidence string, remaining, active int) {
	if err := selectDiscoveryTelemetrySink().Emit(ctx, telemetry.NewLevelingBudgetEvent(
		telemetry.LevelingBudgetEventKind(event), budget.RunID, budget, telemetry.LevelingBudgetEventInput{
			MissionID: budget.MissionID, Role: budget.Role, ResolutionID: budget.ResolutionID,
			ParentResolutionID: budget.ParentResolutionID, Reason: reason, Remaining: remaining,
			Active: active, ProviderEvidence: evidence,
		},
	)); err != nil {
		return
	}
}

func finishRangerBudget(ctx context.Context, admission rangerBudgetAdmission, report handoffconsumer.Report) {
	if err := admission.state.End(); err != nil {
		return
	}
	remaining := admission.state.Budget.MaxDelegations - admission.state.UsedDelegations
	active := admission.state.ActiveDelegations
	appendRangerBudgetEvent(ctx, admission.path, admission.budget, leveling.BudgetEventConsumption, "completed", providerEvidence(report), remaining, active)
	appendRangerBudgetEvent(ctx, admission.path, admission.budget, leveling.BudgetEventProviderUsage, string(report.Reason), providerEvidence(report), remaining, active)
}

func providerEvidence(report handoffconsumer.Report) string {
	if report.Attempted {
		return "unverified"
	}
	return "unavailable"
}

func rangerBudgetUnavailableReport() handoffconsumer.Report {
	return handoffconsumer.Report{
		Status: handoffconsumer.StatusFallback, Reason: integration.StateUnavailable,
		BudgetState: "unavailable", BudgetReason: leveling.ReasonLevelingBudgetUnavailable,
	}
}
