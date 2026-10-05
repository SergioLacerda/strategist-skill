package leveling

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBudgetCoverageAdditionalValidationAndRotationBranches(t *testing.T) {
	for _, value := range []string{"low", " medium ", "HIGH", "unknown"} {
		assert.GreaterOrEqual(t, budgetRank(value), 0)
	}
	for _, value := range []string{"minimal", " standard ", "DEEP", "unknown"} {
		assert.GreaterOrEqual(t, evidenceRank(value), 0)
	}

	root := t.TempDir()
	require.Error(t, ValidateLegacyCompatibility(root))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".leveling-compat.yaml"), []byte("version: 2\nmode: legacy\n"), 0o600))
	require.Error(t, ValidateLegacyCompatibility(root))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".leveling-compat.yaml"), []byte("version: 1\nmode: LEGACY\n"), 0o600))
	assert.NoError(t, ValidateLegacyCompatibility(root))

	valid := domain.EffortBudget{
		SchemaVersion: domain.EffortBudgetSchemaVersion, MissionID: "mission", RunID: "run", Role: "ranger",
		MaxDelegations: 1, MaxDepth: 1, Parallelism: 1, ContextBudget: "medium", EvidenceDepth: "standard",
		EscalationAllowed: false, MaxEscalations: 0,
	}
	_, err := EscalateBudget(domain.EffortBudget{}, "parent", "reason")
	require.Error(t, err)
	_, err = EscalateBudget(valid, "parent", "reason")
	require.ErrorContains(t, err, ReasonLevelingBudgetEscalationDenied)
	valid.EscalationAllowed = true
	valid.MaxEscalations = 1
	valid.EscalationLevel = valid.MaxEscalations
	_, err = EscalateBudget(valid, "parent", "reason")
	require.ErrorContains(t, err, ReasonLevelingBudgetEscalationExhausted)
	valid.EscalationLevel = 0
	_, err = EscalateBudget(valid, " ", "reason")
	require.ErrorContains(t, err, ReasonLevelingBudgetEscalationDenied)

	path := filepath.Join(root, "events.jsonl")
	_, err = RotateBudgetEvents(path, 0)
	require.Error(t, err)
	parentFile := filepath.Join(root, "parent-file")
	require.NoError(t, os.WriteFile(parentFile, []byte("x"), 0o600))
	_, err = RotateBudgetEvents(filepath.Join(parentFile, "events.jsonl"), 1)
	require.Error(t, err)
	record := BudgetEventRecord{SchemaVersion: BudgetLedgerSchemaVersion, Timestamp: "now", Event: BudgetEventGrant, MissionID: "mission", RunID: "run", Role: "ranger"}
	require.NoError(t, AppendBudgetEvent(path, record))
	dropped, err := RotateBudgetEvents(path, 2)
	require.NoError(t, err)
	assert.Zero(t, dropped)
	_, err = ReadBudgetEvents(filepath.Join(parentFile, "events.jsonl"))
	require.Error(t, err)

	for _, invalid := range []BudgetEventRecord{
		{},
		{SchemaVersion: "other", MissionID: "mission", RunID: "run", Role: "ranger", Event: BudgetEventGrant},
		{SchemaVersion: BudgetLedgerSchemaVersion, MissionID: "mission", RunID: "run", Role: "ranger", Event: "unknown"},
	} {
		assert.Error(t, AppendBudgetEvent(filepath.Join(root, "invalid.jsonl"), invalid))
	}
}

func TestBudgetCoverageRequestAndTelemetryErrorBranches(t *testing.T) {
	budget := domain.EffortBudget{
		MissionID: "mission", RunID: "run", Role: "ranger", MaxDepth: 1,
		ContextBudget: "medium", EvidenceDepth: "standard", Permissions: domain.BudgetPermissions{Specialists: true},
	}
	cases := []struct {
		name    string
		request DelegationRequest
		want    string
	}{
		{"identity", DelegationRequest{MissionID: "other"}, ReasonLevelingBudgetIdentityMismatch},
		{"depth", DelegationRequest{Depth: 2}, ReasonLevelingBudgetDepthExceeded},
		{"permission", DelegationRequest{Kind: DelegationSideQuest}, ReasonLevelingBudgetPermissionDenied},
		{"context", DelegationRequest{Kind: DelegationSpecialist, ContextBudget: "high"}, ReasonLevelingBudgetContextExceeded},
		{"evidence", DelegationRequest{Kind: DelegationSpecialist, EvidenceDepth: "deep"}, ReasonLevelingBudgetEvidenceExceeded},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, budgetRequestError(budget, test.request))
		})
	}
	assert.Empty(t, budgetRequestError(budget, DelegationRequest{Kind: DelegationSpecialist}))
	assert.Equal(t, ReasonLevelingBudgetIdentityMismatch, budgetIdentityError(budget, DelegationRequest{RunID: "other"}))
	assert.Equal(t, ReasonLevelingBudgetIdentityMismatch, budgetIdentityError(budget, DelegationRequest{Role: "archivist"}))
	assert.Equal(t, ReasonLevelingBudgetPermissionDenied, budgetPermissionError(budget, DelegationDeepHistory))

	require.ErrorContains(t, (&BudgetState{}).End(), ReasonLevelingBudgetInvalid)
	state := BudgetState{Budget: domain.EffortBudget{MaxDelegations: 0}, UsedDelegations: 2}
	assert.Zero(t, state.denied("denied").RemainingDelegations)
	state.telemetrySink = errorBudgetSink{}
	state.emit(telemetry.LevelingBudgetDenied, "denied", DelegationDecision{})
	assert.Error(t, errorBudgetSink{}.Emit(context.Background(), telemetry.Event{}))
}

type errorBudgetSink struct{}

func (errorBudgetSink) Emit(context.Context, telemetry.Event) error { return errors.New("sink failed") }
