package mission

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/initiative"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	jsonlsink "github.com/SergioLacerda/strategist-skill/internal/telemetry/sink/jsonl"
)

// InitiativeRuntime wires the existing INITIATIVE domain to role boundaries,
// its independent append-only ledger, and a diagnostic telemetry stream.
type InitiativeRuntime struct {
	core      initiative.Runtime
	eventSink telemetry.EventSink
}

// NewInitiativeRuntime creates the internal runtime with the supplied policy.
func NewInitiativeRuntime(strategistRoot string, policy initiative.Policy) (InitiativeRuntime, error) {
	return NewInitiativeRuntimeWithSink(strategistRoot, policy, jsonlsink.New(telemetry.InitiativeEventHistoryPath(strategistRoot)))
}

// NewInitiativeRuntimeWithSink creates the runtime with an explicit
// authoritative event sink. Tests and host orchestration can inject a sink;
// the default constructor uses the independent local INITIATIVE JSONL stream.
func NewInitiativeRuntimeWithSink(strategistRoot string, policy initiative.Policy, eventSink telemetry.EventSink) (InitiativeRuntime, error) {
	core, err := initiative.NewRuntime(strategistRoot, policy)
	if err != nil {
		return InitiativeRuntime{}, fmt.Errorf("initiative runtime: create core: %w", err)
	}
	if eventSink == nil {
		return InitiativeRuntime{}, fmt.Errorf("initiative runtime: event sink is required")
	}
	return InitiativeRuntime{core: core, eventSink: eventSink}, nil
}

// NewDefaultInitiativeRuntime creates the runtime using the built-in policy.
func NewDefaultInitiativeRuntime(strategistRoot string) (InitiativeRuntime, error) {
	raw, err := (embed.Extractor{}).ReadFile("initiative.yaml")
	if err != nil {
		return InitiativeRuntime{}, fmt.Errorf("initiative runtime: read canonical policy: %w", err)
	}
	policy, err := initiative.Parse(raw)
	if err != nil {
		return InitiativeRuntime{}, fmt.Errorf("initiative runtime: parse canonical policy: %w", err)
	}
	mirrorPath := filepath.Join(strategistRoot, "initiative.yaml")
	if mirror, readErr := os.ReadFile(mirrorPath); readErr == nil { //nolint:gosec // mirrorPath is rooted at the selected Strategist workspace
		if !bytes.Equal(bytes.TrimSpace(raw), bytes.TrimSpace(mirror)) {
			return InitiativeRuntime{}, fmt.Errorf("initiative runtime: canonical policy mirror drift at %s", mirrorPath)
		}
	} else if !os.IsNotExist(readErr) {
		return InitiativeRuntime{}, fmt.Errorf("initiative runtime: read canonical policy mirror: %w", readErr)
	}
	return NewInitiativeRuntime(strategistRoot, policy)
}

// EnterRole resolves one advice envelope at role entry and emits one
// diagnostic event. Repeated entry for the same mission/role/run reuses the
// existing advice identity and emits a reuse marker without appending a new
// advice record.
func (r InitiativeRuntime) EnterRole(input InitiativeRoleEntry) (initiative.Advice, error) {
	advice, reused, err := r.core.EnterRole(initiative.AdviceInput{
		MissionID: input.MissionID, Role: input.Role, RunID: input.RunID,
		Trigger: initiative.TriggerInitial, Observed: input.Observed, Leveling: input.Leveling,
	})
	if err != nil {
		return initiative.Advice{}, fmt.Errorf("initiative runtime: enter role: %w", err)
	}
	if err := r.emitAdvice(advice, reused); err != nil {
		return initiative.Advice{}, err
	}
	return advice, nil
}

// Reevaluate records one explicit advice supersession and emits its
// diagnostic event. The previous record is never rewritten.
func (r InitiativeRuntime) Reevaluate(input InitiativeRoleEntry, trigger initiative.Trigger) (initiative.Advice, error) {
	advice, err := r.core.Reevaluate(initiative.AdviceInput{
		MissionID: input.MissionID, Role: input.Role, RunID: input.RunID,
		Trigger: trigger, Observed: input.Observed, Leveling: input.Leveling,
	})
	if err != nil {
		return initiative.Advice{}, fmt.Errorf("initiative runtime: re-evaluate: %w", err)
	}
	if err := r.emitAdvice(advice, false); err != nil {
		return initiative.Advice{}, err
	}
	return advice, nil
}

// CompleteRole validates and persists a role result, returning an additive
// handoff envelope for the next consumer. A challenge is observable but does
// not authorize or reject the Approval Gate.
func (r InitiativeRuntime) CompleteRole(advice initiative.Advice, result initiative.Result, toRole string) (InitiativeHandoff, error) {
	if strings.TrimSpace(toRole) == "" {
		return InitiativeHandoff{}, fmt.Errorf("initiative handoff: destination role is required")
	}
	if err := validateInitiativeTransition(advice.Role, toRole); err != nil {
		return InitiativeHandoff{}, err
	}
	assessment, err := r.core.RecordResult(advice, result)
	if err != nil {
		return InitiativeHandoff{}, fmt.Errorf("initiative runtime: record result: %w", err)
	}
	// RecordResult assigns deterministic identity metadata for first revisions;
	// expose that same persisted revision in the handoff envelope so replay
	// validation compares the exact record that was assessed.
	if result.ResultID == "" {
		result.ResultID = assessment.ResultID
	}
	if result.Sequence == 0 {
		result.Sequence = assessment.ResultSequence
	}
	handoff := InitiativeHandoff{
		FromRole: advice.Role, ToRole: toRole, Advice: advice,
		Result: result, Assessment: assessment,
	}
	if err := handoff.Validate(); err != nil {
		return InitiativeHandoff{}, err
	}
	if err := r.emitResult(advice, result, assessment); err != nil {
		return InitiativeHandoff{}, err
	}
	return handoff, nil
}
