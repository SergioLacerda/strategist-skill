package leveling

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	internal "github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type budgetEmission struct {
	budget domain.EffortBudget
	kind   internal.BudgetEventKind
	reason string
}

func budgetLabelDeps(t *testing.T, root string, emitted *[]budgetEmission) Dependencies {
	t.Helper()
	var levels []internal.Level
	deps := testDeps(root, &levels)
	policy := inferTestPolicy(t)
	deps.LoadPolicy = func() (internal.Policy, string, error) { return policy, "", nil }
	deps.EmitBudgetEvent = func(_ context.Context, budget domain.EffortBudget, kind internal.BudgetEventKind, reason string) {
		*emitted = append(*emitted, budgetEmission{budget: budget, kind: kind, reason: reason})
	}
	return deps
}

func runBudgetLabel(t *testing.T, deps Dependencies, opts LabelOptions) (out bytes.Buffer) {
	t.Helper()
	opts.JSON = true
	cmd := NewLabel(deps, &opts)
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.RunE(cmd, nil))
	return out
}

// TestAvailabilityContractPaths pins what each way of labelling a role does to
// the effort budget (ADR-0068 DEC-001): only an authoritative ranked provider
// mapping issues one, and nothing else fabricates it.
func TestAvailabilityContractPaths(t *testing.T) {
	t.Parallel()
	policy := inferTestPolicy(t)
	load := func() (internal.Policy, error) { return policy, nil }
	cases := []struct {
		name       string
		cfg        domain.LevelingConfig
		opts       LabelOptions
		wantBudget bool
		wantReused bool
	}{
		{"ranked provider issues a budget and replay reuses it", domain.LevelingConfig{Mode: domain.LevelingModeAutomatic},
			LabelOptions{Role: "ranger", Mission: "m", Provider: "CLAUDE", HostModel: "claude-sonnet-5-5", HostEffort: "high"}, true, true},
		{"unknown provider issues none and is not reused", domain.LevelingConfig{Mode: domain.LevelingModeAutomatic},
			LabelOptions{Role: "ranger", Mission: "m", Provider: "NOPE", HostModel: "m1", HostEffort: "high"}, false, false},
		{"host model without a provider issues none", domain.LevelingConfig{Mode: domain.LevelingModeAutomatic},
			LabelOptions{Role: "ranger", Mission: "m", HostModel: "claude-sonnet-5-5", HostEffort: "high"}, false, true},
		{"manual mode is host passthrough and issues none", domain.LevelingConfig{Mode: domain.LevelingModeManual},
			LabelOptions{Role: "ranger", Mission: "m", Provider: "CLAUDE", HostModel: "claude-sonnet-5-5", HostEffort: "high"}, false, true},
		{"nothing resolved issues none", domain.LevelingConfig{Mode: domain.LevelingModeAutomatic},
			LabelOptions{Role: "ranger", Mission: "m"}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
			first, err := LabelRoleWith(domain.DefaultRoleRegistry(), load, tc.cfg, ledger, tc.opts)
			require.NoError(t, err)
			replay, err := LabelRoleWith(domain.DefaultRoleRegistry(), load, tc.cfg, ledger, tc.opts)
			require.NoError(t, err)

			assert.Equal(t, tc.wantBudget, first.Level.Budget != nil, "budget issued")
			assert.Equal(t, tc.wantReused, replay.Reused, "replay reuse")
			if tc.wantBudget {
				require.NotNil(t, replay.Level.Budget)
				assert.Equal(t, first.Level.Budget.ResolutionID, replay.Level.Budget.ResolutionID)
			}
		})
	}
}

func TestLabelEscalatedReasonRecordsAnEscalationEventAndEmitsIt(t *testing.T) {
	root := t.TempDir()
	var emitted []budgetEmission
	deps := budgetLabelDeps(t, root, &emitted)
	base := LabelOptions{Role: "ranger", Mission: "m1", Provider: "CLAUDE", HostModel: "claude-sonnet-5-5", HostEffort: "medium"}
	runBudgetLabel(t, deps, base)
	assert.Empty(t, emitted, "a plain label emits no budget event")

	base.Reason = "repeated_failure"
	out := runBudgetLabel(t, deps, base)

	var payload struct {
		Budget     *domain.EffortBudget `json:"effort_budget"`
		Escalation *struct {
			Attempted bool   `json:"attempted"`
			Granted   bool   `json:"granted"`
			Reason    string `json:"reason"`
			Parent    string `json:"parent_resolution_id"`
		} `json:"escalation"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &payload))
	require.NotNil(t, payload.Escalation)
	assert.True(t, payload.Escalation.Granted)
	require.NotNil(t, payload.Budget)
	assert.Equal(t, payload.Escalation.Parent, payload.Budget.ParentResolutionID)
	assert.Equal(t, 1, payload.Budget.EscalationLevel)

	events, err := internal.ReadBudgetEvents(filepath.Join(root, domain.DefaultBudgetLedgerPath))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, internal.BudgetEventEscalation, events[0].Event)
	assert.Equal(t, "repeated_failure", events[0].Reason)
	assert.Equal(t, payload.Budget.ResolutionID, events[0].ResolutionID)
	assert.Equal(t, payload.Budget.ParentResolutionID, events[0].ParentResolutionID)
	require.Len(t, emitted, 1)
	assert.Equal(t, internal.BudgetEventEscalation, emitted[0].kind)
}

func TestLabelEscalationBeyondThePolicyLimitIsRecordedAsExhaustion(t *testing.T) {
	root := t.TempDir()
	var emitted []budgetEmission
	deps := budgetLabelDeps(t, root, &emitted)
	opts := LabelOptions{Role: "ranger", Mission: "m1", Provider: "CLAUDE", HostModel: "claude-sonnet-5-5", HostEffort: "medium"}
	runBudgetLabel(t, deps, opts)
	opts.Reason = "escalated"
	runBudgetLabel(t, deps, opts)
	out := runBudgetLabel(t, deps, opts)

	assert.Contains(t, out.String(), `"granted":false`)
	assert.Contains(t, out.String(), internal.ReasonLevelingBudgetEscalationExhausted)
	events, err := internal.ReadBudgetEvents(filepath.Join(root, domain.DefaultBudgetLedgerPath))
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, internal.BudgetEventEscalation, events[0].Event)
	assert.Equal(t, internal.BudgetEventExhaustion, events[1].Event)
	assert.Equal(t, internal.ReasonLevelingBudgetEscalationExhausted, events[1].Reason)
}

func TestLabelEscalationWithoutAProviderBudgetIsReportedAsUnavailable(t *testing.T) {
	root := t.TempDir()
	var emitted []budgetEmission
	deps := budgetLabelDeps(t, root, &emitted)
	runBudgetLabel(t, deps, LabelOptions{Role: "ranger", Mission: "m1", Provider: "CLAUDE", HostModel: "claude-sonnet-5-5", HostEffort: "medium"})

	out := runBudgetLabel(t, deps, LabelOptions{Role: "ranger", Mission: "m1", HostModel: "claude-sonnet-5-5", HostEffort: "medium", Reason: "escalated"})

	assert.Contains(t, out.String(), internal.ReasonLevelingBudgetUnavailable)
	events, err := internal.ReadBudgetEvents(filepath.Join(root, domain.DefaultBudgetLedgerPath))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, internal.BudgetEventDenial, events[0].Event)
	assert.Equal(t, internal.ReasonLevelingBudgetUnavailable, events[0].Reason)
}
