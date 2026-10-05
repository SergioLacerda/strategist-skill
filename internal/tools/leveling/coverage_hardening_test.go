package leveling

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/feats/initiative"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func coverageBudget() domain.EffortBudget {
	return domain.EffortBudget{
		SchemaVersion: domain.EffortBudgetSchemaVersion,
		MissionID:     "mission-1", RunID: "run-1", Role: "ranger",
		MaxDelegations: 2, MaxDepth: 1, Parallelism: 1,
		ContextBudget: "medium", EvidenceDepth: "standard",
		Permissions:       domain.BudgetPermissions{Specialists: true},
		EscalationAllowed: true, MaxEscalations: 1,
	}
}

func TestCoverageBudgetRanksAndEscalationBranches(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"low", "medium", "high", "unknown"} {
		assert.GreaterOrEqual(t, budgetRank(value), 0)
	}
	for _, value := range []string{"minimal", "standard", "deep", "unknown"} {
		assert.GreaterOrEqual(t, evidenceRank(value), 0)
	}

	parent := coverageBudget()
	child, err := EscalateBudget(parent, "resolution-1", " repeated_failure ")
	require.NoError(t, err)
	assert.Equal(t, "repeated_failure", child.EscalationReason)
	assert.Equal(t, 2, child.MaxDepth)
	assert.Equal(t, ReasonLevelingBudgetEscalationDenied, EscalationDenialReason(errors.New(ReasonLevelingBudgetEscalationDenied+": denied")))
	assert.Empty(t, EscalationDenialReason(nil))

	for name, invalid := range map[string]domain.EffortBudget{
		"invalid schema": {Role: "ranger"},
		"disabled":       func() domain.EffortBudget { b := parent; b.EscalationAllowed = false; b.MaxEscalations = 0; return b }(),
		"exhausted":      func() domain.EffortBudget { b := parent; b.EscalationLevel = 1; return b }(),
	} {
		_, err := EscalateBudget(invalid, "resolution-1", "reason")
		require.Error(t, err, name)
	}
	_, err = EscalateBudget(parent, "", "reason")
	require.ErrorContains(t, err, ReasonLevelingBudgetEscalationDenied)
}

func TestCoverageBudgetAdmissionRejectsBoundaryFailures(t *testing.T) {
	t.Parallel()
	_, err := NewBudgetState(domain.EffortBudget{})
	require.Error(t, err)
	var nilState *BudgetState
	assert.Equal(t, ReasonLevelingBudgetInvalid, nilState.Begin(DelegationRequest{}).Reason)
	require.Error(t, nilState.End())

	state, err := NewBudgetState(coverageBudget())
	require.NoError(t, err)
	cases := []DelegationRequest{
		{MissionID: "other", Kind: DelegationSpecialist},
		{RunID: "other", Kind: DelegationSpecialist},
		{Role: "other", Kind: DelegationSpecialist},
		{Depth: -1, Kind: DelegationSpecialist},
		{Depth: 2, Kind: DelegationSpecialist},
		{Kind: DelegationSideQuest},
		{Kind: DelegationSpecialist, ContextBudget: "high"},
		{Kind: DelegationSpecialist, EvidenceDepth: "deep"},
	}
	for _, request := range cases {
		decision := state.Begin(request)
		assert.False(t, decision.Granted)
	}

	require.True(t, state.Begin(DelegationRequest{Kind: DelegationSpecialist}).Granted)
	parallel := state.Begin(DelegationRequest{Kind: DelegationSpecialist})
	assert.Equal(t, ReasonLevelingBudgetParallelism, parallel.Reason)
	require.NoError(t, state.End())
	require.True(t, state.Begin(DelegationRequest{Kind: DelegationSpecialist}).Granted)
	require.NoError(t, state.End())
	exhausted := state.Begin(DelegationRequest{Kind: DelegationSpecialist})
	assert.Equal(t, ReasonLevelingBudgetExhausted, exhausted.Reason)

	assert.ErrorContains(t, (&BudgetState{}).End(), ReasonLevelingBudgetInvalid)
}

func TestCoverageBudgetEventAndRotationBranches(t *testing.T) {
	t.Parallel()
	valid := NewBudgetEventRecord(coverageBudget(), BudgetEventGrant, "granted", "unverified", 1, 0)
	for _, invalid := range []BudgetEventRecord{
		{SchemaVersion: "other", MissionID: "m", RunID: "r", Role: "r", Event: BudgetEventGrant},
		{SchemaVersion: BudgetLedgerSchemaVersion, Event: BudgetEventGrant},
		func() BudgetEventRecord { record := valid; record.Event = "unknown"; return record }(),
	} {
		require.Error(t, validateBudgetEvent(invalid))
	}

	path := filepath.Join(t.TempDir(), "nested", "budget.jsonl")
	require.NoError(t, AppendBudgetEvent(path, valid))
	records, err := ReadBudgetEvents(path)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "ranger", records[0].Role)
	assert.Equal(t, 0, mustRotateBudgetEvents(t, filepath.Join(t.TempDir(), "missing.jsonl"), 2))
	_, err = RotateBudgetEvents(filepath.Join(t.TempDir(), "events.jsonl"), 0)
	require.Error(t, err)

	fileRoot := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(fileRoot, []byte("file"), 0o600))
	_, err = ReadBudgetEvents(filepath.Join(fileRoot, "events.jsonl"))
	require.Error(t, err)

	encoded, err := json.Marshal(valid)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, append(encoded, '\n'), 0o600))
}

func mustRotateBudgetEvents(t *testing.T, path string, maxRecords int) int {
	t.Helper()
	dropped, err := RotateBudgetEvents(path, maxRecords)
	require.NoError(t, err)
	return dropped
}

func TestCoverageBudgetMergeOverrides(t *testing.T) {
	t.Parallel()
	base := domain.EffortBudgetPolicy{MaxDelegations: 1, MaxDepth: 1, Parallelism: 1, ContextBudget: "low", EvidenceDepth: "minimal"}
	override := domain.EffortBudgetPolicy{
		MaxDelegations: 2, MaxDepth: 2, Parallelism: 2, ContextBudget: "high", EvidenceDepth: "deep",
		Permissions:       domain.BudgetPermissions{Specialists: true, SideQuests: true, DeepHistory: true},
		EscalationAllowed: true, MaxEscalations: 2,
	}
	merged := mergeBudgetPolicy(base, override)
	assert.Equal(t, override, merged)

	maxDelegations, maxDepth, parallelism, contextBudget, evidenceDepth, escalationAllowed, maxEscalations := 3, 3, 3, "medium", "standard", false, 0
	permissions := &budgetPermissionOverrides{Specialists: boolPtr(false), SideQuests: boolPtr(true), DeepHistory: boolPtr(true)}
	overrideValues := budgetOverride{
		MaxDelegations: &maxDelegations, MaxDepth: &maxDepth, Parallelism: &parallelism,
		ContextBudget: &contextBudget, EvidenceDepth: &evidenceDepth, Permissions: permissions,
		EscalationAllowed: &escalationAllowed, MaxEscalations: &maxEscalations,
	}
	policy := Policy{Defaults: Defaults{Roles: map[string]Role{"ranger": {}}}}
	document := budgetOverrideDocument{}
	document.Defaults.Budget = &overrideValues
	document.Defaults.Roles = map[string]struct {
		Budget *budgetOverride `yaml:"budget"`
	}{"ranger": {Budget: &overrideValues}}
	applyExplicitBudgetOverrides(&policy, document)
	assert.Equal(t, 3, policy.Defaults.Budget.MaxDelegations)
	assert.Equal(t, 3, policy.Defaults.Roles["ranger"].Budget.MaxDepth)

	_, err := parseBudgetOverrides([]byte("defaults: ["))
	require.Error(t, err)
	parsed, err := parseBudgetOverrides([]byte("defaults:\n  budget:\n    max_depth: 2\n"))
	require.NoError(t, err)
	require.NotNil(t, parsed.Defaults.Budget)
}

func boolPtr(value bool) *bool { return &value }

func TestCoveragePolicyValidationBranches(t *testing.T) {
	t.Parallel()
	for _, value := range []struct{ path, input string }{{"capability", ""}, {"whitespace", " capability "}} {
		require.Error(t, validateCapability(value.path, value.input))
	}
	require.Error(t, validateEffortTiers("tiers", nil))
	require.Error(t, validateEffortTiers("tiers", []string{"invalid"}))
	require.Error(t, validateEffortTiers("tiers", []string{"low", "low"}))

	for _, policy := range []domain.EffortBudgetPolicy{
		{MaxDelegations: -1, Parallelism: 1, ContextBudget: "low", EvidenceDepth: "minimal"},
		{Parallelism: 0, ContextBudget: "low", EvidenceDepth: "minimal"},
		{Parallelism: 1, ContextBudget: "invalid", EvidenceDepth: "minimal"},
		{Parallelism: 1, ContextBudget: "low", EvidenceDepth: "invalid"},
		{Parallelism: 1, ContextBudget: "low", EvidenceDepth: "minimal", MaxEscalations: 1},
	} {
		require.Error(t, validateBudgetPolicy("budget", policy))
	}

	require.Error(t, validateProvider("PROVIDER", Provider{}))
	require.Error(t, validateProviderModels("PROVIDER", map[string]string{" capability ": "model"}))
	require.Error(t, validateProviderDisplay("PROVIDER", map[string]string{"model": "line\nname"}))
	require.NoError(t, validateProviderDisplay("PROVIDER", map[string]string{"model": "Model"}))
	require.Error(t, validateRoleBasics("", Role{}, []string{"low"}))
	require.Error(t, validateRoleBasics("ranger", Role{Capability: "cap"}, []string{"low"}))
	require.Error(t, validateRoleBasics("ranger", Role{Capability: "cap", Effort: "invalid"}, []string{"low"}))
	require.Error(t, validateEscalationEffort("ranger", Escalation{Capability: "cap", Effort: "invalid"}, []string{"low"}))
	require.Error(t, validateEscalationEffort("ranger", Escalation{Effort: "low"}, []string{"low"}))
	require.Error(t, validateCriteria("ranger", Criteria{Ambiguity: "invalid"}))
	require.Error(t, validateProviderCapabilities("PROVIDER", Provider{Models: map[string]string{"unknown": "model"}}, map[string]bool{"known": true}))
}

func coverageDefaults(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "embed", "defaults", "leveling.yaml"))
	require.NoError(t, err)
	return raw
}

func TestCoveragePolicyAuthorityAndResolutionBranches(t *testing.T) {
	raw := coverageDefaults(t)
	policy, err := Parse(raw)
	require.NoError(t, err)

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, ".leveling-compat.yaml"), []byte("version: 1\nmode: legacy\n"), 0o600))
	require.NoError(t, ValidateLegacyCompatibility(root))
	require.NoError(t, VerifyDefaultAuthority(raw, policy.Version, policy.Digest()))
	require.Error(t, VerifyDefaultAuthority(raw, policy.Version+1, policy.Digest()))
	require.Error(t, VerifyDefaultAuthority([]byte("version: ["), policy.Version, policy.Digest()))

	effective, decision, err := LoadAuthorized(root, raw, raw, "test")
	require.NoError(t, err)
	assert.Equal(t, AuthorityCurrent, decision.Class)
	assert.Equal(t, policy.Digest(), effective.DefaultDigest)
	_, _, err = LoadAuthorized(root, raw, []byte("version: ["), "test")
	require.Error(t, err)
	badMarkerRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(badMarkerRoot, ".leveling-compat.yaml"), []byte("mode: legacy\n"), 0o600))
	require.Error(t, ValidateLegacyCompatibility(badMarkerRoot))

	artifact, err := ResolveLevelArtifact(policy, "CODEX", "ranger", Signals{}, Host{})
	require.NoError(t, err)
	assert.Equal(t, "ranger", artifact.Role)

	loader := func() (Policy, error) { return policy, nil }
	level, err := ResolveLevelInferred(loader, "archivist", Signals{}, Host{Model: "claude-general"})
	require.NoError(t, err)
	assert.Equal(t, ProviderMatchExact, ProviderMatch(level.ProviderMatch))

	_, err = ResolveLevelInferred(func() (Policy, error) { return Policy{}, errors.New("loader failed") }, "ranger", Signals{}, Host{Model: "claude-general"})
	require.Error(t, err)

	assert.Equal(t, ProviderMatchUnknown, policy.MatchProvider("").Match)
	assert.Equal(t, ProviderMatchUnknown, policy.MatchProvider("gemini-2").Match)
}

func TestCoverageLedgerAndBudgetPersistenceBranches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	record := Record{MissionID: "mission", Run: "run", Level: Level{Role: "Ranger", Provider: "CODEX", Model: "model", Effort: "low"}}
	require.NoError(t, AppendRecord(path, record))
	latest, found, err := LatestRunRecord(path, "mission", "RANGER", "run")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "ranger", latest.Role)
	// The parent is created as a file to exercise the directory error path.
	parent := filepath.Join(t.TempDir(), "parent")
	require.NoError(t, os.WriteFile(parent, []byte("x"), 0o600))
	_, _, err = LatestRunRecord(filepath.Join(parent, "ledger.jsonl"), "m", "r", "")
	require.Error(t, err)
	require.Error(t, AppendRecord(filepath.Join(parent, "ledger.jsonl"), record))

	parsed, ok := parseLedgerLine([]byte(`{"role":" Ranger "}`))
	require.True(t, ok)
	assert.Equal(t, LedgerSchemaVersion, parsed.SchemaVersion)
	assert.Equal(t, "ranger", parsed.Role)
	assert.False(t, func() bool { _, ok := parseLedgerLine([]byte(`{"schema_version":"other"}`)); return ok }())
	require.Error(t, scanRecords(bytes.NewBufferString(strings.Repeat("x", maxLedgerRecordBytes+1)), func(string, Record) {}))
	require.Error(t, withLedgerFile(t.TempDir(), func(*os.File) error { return errors.New("use failed") }))
	require.Error(t, appendJSONLine(t.TempDir(), []byte("{}"), "test"))

	eventPath := filepath.Join(t.TempDir(), "events.jsonl")
	event := NewBudgetEventRecord(coverageBudget(), BudgetEventGrant, "grant", "verified", 1, 0)
	require.Error(t, AppendBudgetEvent(filepath.Join(parent, "events.jsonl"), event))
	require.NoError(t, AppendBudgetEvent(eventPath, event))
	event.Event = BudgetEventConsumption
	require.NoError(t, AppendBudgetEvent(eventPath, event))
	dropped, err := RotateBudgetEvents(eventPath, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, dropped)
	got, err := ReadBudgetEvents(eventPath)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.NoError(t, writeBudgetEventsUnlocked(eventPath, []BudgetEventRecord{event}))
	require.NoError(t, encodeBudgetEvents(bufio.NewWriter(io.Discard), []BudgetEventRecord{event}))
	_, err = ResolveLevelArtifact(Policy{}, "CODEX", "ranger", Signals{}, Host{})
	require.Error(t, err)
}

func TestCoverageRemainingPolicyAndErrorBranches(t *testing.T) {
	raw := coverageDefaults(t)
	policy, err := Parse(raw)
	require.NoError(t, err)
	require.NoError(t, policy.VerifyDigest(policy.Digest()))
	require.Error(t, policy.VerifyDigest(""))
	require.Error(t, policy.VerifyDigest("stale"))
	require.NoError(t, CheckDefaultParity(raw, raw))
	require.Error(t, CheckDefaultParity([]byte("version: ["), raw))
	require.Error(t, CheckDefaultParity(raw, []byte("version: [")))
	_, err = LoadEffective(raw, []byte("version: ["), "test")
	require.Error(t, err)

	base := domain.EffortBudgetPolicy{}
	override := budgetOverride{
		MaxDelegations: boolInt(2), MaxDepth: boolInt(1), Parallelism: boolInt(1),
		ContextBudget: stringPtr("high"), EvidenceDepth: stringPtr("deep"),
		Permissions:       &budgetPermissionOverrides{Specialists: boolPtr(true), SideQuests: boolPtr(true), DeepHistory: boolPtr(true)},
		EscalationAllowed: boolPtr(true), MaxEscalations: boolInt(2),
	}
	merged := applyBudgetOverride(base, override)
	assert.Equal(t, 2, merged.MaxDelegations)
	assert.True(t, merged.Permissions.DeepHistory)
	assert.Equal(t, 2, merged.MaxEscalations)
	assert.Equal(t, base, applyBudgetOverride(base, budgetOverride{}))

	require.NoError(t, validateDefaultFallback(Defaults{EffortTiers: []string{"low"}, Fallback: Fallback{Capability: "cap", Effort: "low", Reason: "reason"}}))
	require.Error(t, validateDefaultFallback(Defaults{EffortTiers: []string{"low"}, Fallback: Fallback{Capability: "cap", Effort: "high", Reason: "reason"}}))
	require.NoError(t, validateRoleEscalation("ranger", Escalation{Capability: "cap", Effort: "low"}, []string{"low"}))
	require.Error(t, validateRoleEscalation("ranger", Escalation{Effort: "low"}, []string{"low"}))
	require.Error(t, validateRoleEscalation("ranger", Escalation{Capability: " invalid ", Effort: "low"}, []string{"low"}))
	require.NoError(t, validateProviders(map[string]Provider{"P": {Ranked: true, Models: map[string]string{"cap": "model"}, EffortTiers: []string{"low"}}}))
	require.Error(t, validateProviders(nil))
	require.Error(t, validateSignals(Signals{RepeatedFailures: -1}))
	_, err = SignalsFromEscalation(initiative.EscalationRequest{Mechanism: initiative.PreciseShotMechanism})
	require.Error(t, err)
	_, err = SignalsFromEscalation(initiative.EscalationRequest{Mechanism: initiative.PreciseShotMechanism, AssessmentID: "a", IdempotencyKey: "i", Signals: initiative.EscalationSignals{Evidence: "invalid"}})
	require.Error(t, err)
	assert.False(t, providerOwnsExactModel(Provider{}, "model"))
	assert.True(t, exceedsLevel("high", ""))
	assert.False(t, exceedsLevel("low", "high"))

	state := BudgetState{Budget: domain.EffortBudget{MaxDelegations: 1}, UsedDelegations: 2}
	assert.Equal(t, 0, state.denied("denied").RemainingDelegations)
	f, err := os.CreateTemp(t.TempDir(), "closed-*")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	var latest Record
	found := false
	closeLatestRecordLedger(f, &latest, &found, new(error))
	tmp, err := os.CreateTemp(t.TempDir(), "cleanup-*")
	require.NoError(t, err)
	name := tmp.Name()
	cleanupBudgetRotation(tmp, name)
}

func boolInt(value int) *int { return &value }

func stringPtr(value string) *string { return &value }

type coverageFailSink struct{}

func (coverageFailSink) Emit(context.Context, telemetry.Event) error {
	return errors.New("sink failed")
}

func TestCoverageTelemetryAndFilesystemFailures(t *testing.T) {
	state, err := NewBudgetStateWithTelemetry(coverageBudget(), coverageFailSink{})
	require.NoError(t, err)
	decision := state.Begin(DelegationRequest{Kind: DelegationSpecialist})
	assert.True(t, decision.Granted)
	require.NoError(t, state.End())

	parent := filepath.Join(t.TempDir(), "parent")
	require.NoError(t, os.WriteFile(parent, []byte("x"), 0o600))
	require.Error(t, scanLedger(filepath.Join(parent, "ledger.jsonl"), func(string, Record) {}))
	_, err = RotateLedgerIfLarge(filepath.Join(parent, "ledger.jsonl"), 1, 1)
	require.Error(t, err)
	require.Error(t, normalizeProviders(&Policy{Providers: map[string]Provider{" ": {}}}))
	spacedProviderID := strings.Join([]string{" ", "P", " "}, "")
	require.Error(t, normalizeProviders(&Policy{Providers: map[string]Provider{"p": {}, spacedProviderID: {}}}))
	var roles map[string]Role
	mergeRoles(&roles, map[string]Role{"ranger": {Capability: "cap"}})
	assert.Equal(t, "cap", roles["ranger"].Capability)
	var providers map[string]Provider
	mergeProviders(&providers, map[string]Provider{"P": {}}, nil)
	assert.Contains(t, providers, "P")
}
