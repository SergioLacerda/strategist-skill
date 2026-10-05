package leveling

import (
	"context"
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// DelegationKind is the closed set of work classes that a budget can permit.
type DelegationKind string

const (
	// DelegationSpecialist permits specialist work when the envelope allows it.
	DelegationSpecialist DelegationKind = "specialist"
	// DelegationSideQuest permits side-quest work when the envelope allows it.
	DelegationSideQuest DelegationKind = "side_quest"
	// DelegationDeepHistory permits deep-history work when the envelope allows it.
	DelegationDeepHistory DelegationKind = "deep_history"
)

// DelegationRequest is the synchronous input to the budget boundary. The
// boundary intentionally does not start goroutines or choose a Role owner.
type DelegationRequest struct {
	MissionID     string
	RunID         string
	Role          string
	Kind          DelegationKind
	Depth         int
	ContextBudget string
	EvidenceDepth string
}

// DelegationDecision makes grant, denial, and remaining capacity observable.
type DelegationDecision struct {
	Granted              bool
	Reason               string
	RemainingDelegations int
	ActiveDelegations    int
}

// BudgetState is the mutable consumption ledger for one immutable envelope.
// Callers must use it synchronously; it deliberately has no implicit
// concurrency or side-effecting provider invocation.
type BudgetState struct {
	Budget            domain.EffortBudget
	UsedDelegations   int
	ActiveDelegations int
	telemetryContext  context.Context
	telemetrySink     telemetry.EventSink
}

// NewBudgetState validates and snapshots a resolved envelope.
func NewBudgetState(budget domain.EffortBudget) (BudgetState, error) {
	return newBudgetState(budget, nil)
}

// NewBudgetStateWithTelemetry attaches an optional sink to the same
// synchronous boundary. Sink failures never turn a deterministic grant into
// an implicit provider invocation; the caller can use a strict sink wrapper if
// telemetry delivery itself is a hard requirement.
func NewBudgetStateWithTelemetry(budget domain.EffortBudget, sink telemetry.EventSink) (BudgetState, error) {
	return newBudgetState(budget, sink)
}

func newBudgetState(budget domain.EffortBudget, sink telemetry.EventSink) (BudgetState, error) {
	if err := budget.Validate(); err != nil {
		return BudgetState{}, fmt.Errorf("%s: %w", ReasonLevelingBudgetInvalid, err)
	}
	return BudgetState{Budget: budget, telemetryContext: context.Background(), telemetrySink: sink}, nil
}

// Begin attempts one synchronous delegation and consumes capacity only when
// every identity, permission, depth, context, evidence, and limit check passes.
func (s *BudgetState) Begin(request DelegationRequest) DelegationDecision {
	if s == nil {
		return DelegationDecision{Reason: ReasonLevelingBudgetInvalid}
	}
	if reason := budgetRequestError(s.Budget, request); reason != "" {
		decision := s.denied(reason)
		s.emit(telemetry.LevelingBudgetDenied, reason, decision)
		return decision
	}
	if s.UsedDelegations >= s.Budget.MaxDelegations {
		decision := s.denied(ReasonLevelingBudgetExhausted)
		s.emit(telemetry.LevelingBudgetExhausted, ReasonLevelingBudgetExhausted, decision)
		return decision
	}
	if s.ActiveDelegations >= s.Budget.Parallelism {
		decision := s.denied(ReasonLevelingBudgetParallelism)
		s.emit(telemetry.LevelingBudgetDenied, ReasonLevelingBudgetParallelism, decision)
		return decision
	}
	s.UsedDelegations++
	s.ActiveDelegations++
	decision := DelegationDecision{
		Granted: true, Reason: "granted",
		RemainingDelegations: s.Budget.MaxDelegations - s.UsedDelegations,
		ActiveDelegations:    s.ActiveDelegations,
	}
	s.emit(telemetry.LevelingBudgetGranted, decision.Reason, decision)
	return decision
}

// End releases one synchronous active delegation. It does not restore used
// capacity: a completed delegation remains consumed for deterministic replay.
func (s *BudgetState) End() error {
	if s == nil || s.ActiveDelegations == 0 {
		return fmt.Errorf("%s: no active delegation", ReasonLevelingBudgetInvalid)
	}
	s.ActiveDelegations--
	decision := DelegationDecision{Granted: true, Reason: "completed", RemainingDelegations: s.Budget.MaxDelegations - s.UsedDelegations, ActiveDelegations: s.ActiveDelegations}
	s.emit(telemetry.LevelingBudgetConsumed, decision.Reason, decision)
	return nil
}

func (s *BudgetState) emit(kind telemetry.LevelingBudgetEventKind, reason string, decision DelegationDecision) {
	if s == nil || s.telemetrySink == nil {
		return
	}
	event := telemetry.NewLevelingBudgetEvent(kind, s.Budget.RunID, s.Budget, telemetry.LevelingBudgetEventInput{
		MissionID: s.Budget.MissionID, Role: s.Budget.Role, ResolutionID: s.Budget.ResolutionID,
		ParentResolutionID: s.Budget.ParentResolutionID, Reason: reason,
		Remaining: decision.RemainingDelegations, Active: decision.ActiveDelegations,
	})
	if err := s.telemetrySink.Emit(s.telemetryContext, event); err != nil {
		return
	}
}

func (s *BudgetState) denied(reason string) DelegationDecision {
	remaining := s.Budget.MaxDelegations - s.UsedDelegations
	if remaining < 0 {
		remaining = 0
	}
	return DelegationDecision{Reason: reason, RemainingDelegations: remaining, ActiveDelegations: s.ActiveDelegations}
}

func budgetRequestError(budget domain.EffortBudget, request DelegationRequest) string {
	if reason := budgetIdentityError(budget, request); reason != "" {
		return reason
	}
	if request.Depth < 0 || request.Depth > budget.MaxDepth {
		return ReasonLevelingBudgetDepthExceeded
	}
	if reason := budgetPermissionError(budget, request.Kind); reason != "" {
		return reason
	}
	if budgetRank(request.ContextBudget) > budgetRank(budget.ContextBudget) {
		return ReasonLevelingBudgetContextExceeded
	}
	if evidenceRank(request.EvidenceDepth) > evidenceRank(budget.EvidenceDepth) {
		return ReasonLevelingBudgetEvidenceExceeded
	}
	return ""
}

func budgetIdentityError(budget domain.EffortBudget, request DelegationRequest) string {
	if request.MissionID != "" && budget.MissionID != "" && request.MissionID != budget.MissionID {
		return ReasonLevelingBudgetIdentityMismatch
	}
	if request.RunID != "" && budget.RunID != "" && request.RunID != budget.RunID {
		return ReasonLevelingBudgetIdentityMismatch
	}
	if request.Role != "" && budget.Role != "" && !strings.EqualFold(strings.TrimSpace(request.Role), strings.TrimSpace(budget.Role)) {
		return ReasonLevelingBudgetIdentityMismatch
	}
	return ""
}

func budgetPermissionError(budget domain.EffortBudget, kind DelegationKind) string {
	allowed := (kind == DelegationSpecialist && budget.Permissions.Specialists) ||
		(kind == DelegationSideQuest && budget.Permissions.SideQuests) ||
		(kind == DelegationDeepHistory && budget.Permissions.DeepHistory)
	if !allowed {
		return ReasonLevelingBudgetPermissionDenied
	}
	return ""
}
