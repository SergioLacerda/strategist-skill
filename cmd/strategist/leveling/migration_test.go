package leveling

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/embed"
	core "github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func migrationPolicy(t *testing.T) core.Policy {
	t.Helper()
	raw, err := readMigrationDefaults()
	require.NoError(t, err)
	policy, err := core.Parse(raw)
	require.NoError(t, err)
	return policy
}

func migrationLoader(policy core.Policy) core.PolicyLoader {
	return func() (core.Policy, error) { return policy, nil }
}

func TestMigrationLabelRolePersistsAndReuses(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	policy := migrationPolicy(t)
	opts := LabelOptions{Role: "ranger", Mission: "m1", HostModel: "sonnet", HostEffort: "high"}

	first, err := LabelRole(migrationLoader(policy), domain.LevelingConfig{}, ledger, opts)
	require.NoError(t, err)
	assert.Equal(t, "Sonnet-High", first.Level.Label())
	assert.False(t, first.Reused)

	opts.HostModel = "opus"
	second, err := LabelRole(migrationLoader(policy), domain.LevelingConfig{}, ledger, opts)
	require.NoError(t, err)
	assert.True(t, second.Reused)
	assert.Equal(t, "Sonnet-High", second.Level.Label())
}

func TestMigrationLabelRoleEscalationRecordsReason(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	policy := migrationPolicy(t)
	base := LabelOptions{Role: "archivist", Mission: "m1", HostModel: "sonnet", HostEffort: "medium"}
	_, err := LabelRole(migrationLoader(policy), domain.LevelingConfig{}, ledger, base)
	require.NoError(t, err)

	escalated := base
	escalated.HostModel, escalated.HostEffort, escalated.Reason = "opus", "high", "escalated"
	got, err := LabelRole(migrationLoader(policy), domain.LevelingConfig{}, ledger, escalated)
	require.NoError(t, err)
	assert.False(t, got.Reused)
	assert.Equal(t, "Opus-High", got.Level.Label())

	latest, ok, err := core.LatestRecord(ledger, "m1", "archivist")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "escalated", latest.Reason)
}

func TestMigrationLabelRoleWithoutMissionDoesNotWrite(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	got, err := LabelRole(migrationLoader(migrationPolicy(t)), domain.LevelingConfig{}, ledger, LabelOptions{Role: "scout", Provider: "CLAUDE"})
	require.NoError(t, err)
	assert.Equal(t, core.SourcePolicy, got.Level.Source)
	_, ok, err := core.LatestRecord(ledger, "", "scout")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestMigrationLabelRoleDegradesOnPolicyError(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	got, err := LabelRole(migrationLoader(migrationPolicy(t)), domain.LevelingConfig{}, ledger, LabelOptions{Role: "ranger", Mission: "m1", Provider: "CLAUDE", Risk: "bogus"})
	require.NoError(t, err)
	assert.True(t, got.Level.Unknown())
	assert.Contains(t, got.Warning, "leveling_signal_unknown")

	latest, ok, err := core.LatestRecord(ledger, "m1", "ranger")
	require.NoError(t, err)
	require.True(t, ok)
	assert.True(t, latest.Unknown())
}

func TestMigrationLabelRoleUnknownIsNotReusedOnceKnown(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	_, err := LabelRole(migrationLoader(migrationPolicy(t)), domain.LevelingConfig{}, ledger, LabelOptions{Role: "sniper", Mission: "m1"})
	require.NoError(t, err)
	got, err := LabelRole(migrationLoader(migrationPolicy(t)), domain.LevelingConfig{}, ledger, LabelOptions{Role: "sniper", Mission: "m1", HostModel: "haiku", HostEffort: "low"})
	require.NoError(t, err)
	assert.False(t, got.Reused)
	assert.Equal(t, "Haiku-Low", got.Level.Label())
}

func TestMigrationManualModeUsesHostOnly(t *testing.T) {
	loaderCalls := 0
	loader := func() (core.Policy, error) { loaderCalls++; return core.Policy{}, assert.AnError }
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	cfg := domain.LevelingConfig{Mode: domain.LevelingModeManual}

	got, err := LabelRole(loader, cfg, ledger, LabelOptions{Role: "ranger", Mission: "m1", Provider: "CLAUDE", HostModel: "claude-opus-5", HostEffort: "low"})
	require.NoError(t, err)
	assert.Equal(t, "Claude-opus-5-Low", got.Level.Label())
	assert.Zero(t, loaderCalls)

	got, err = LabelRole(loader, cfg, filepath.Join(t.TempDir(), "partial.jsonl"), LabelOptions{Role: "sniper", HostEffort: "high"})
	require.NoError(t, err)
	assert.Equal(t, "High", got.Level.Label())

	got, err = LabelRole(loader, cfg, filepath.Join(t.TempDir(), "unknown.jsonl"), LabelOptions{Role: "sniper"})
	require.NoError(t, err)
	assert.True(t, got.Level.Unknown())
}

func TestMigrationAutomaticCompleteHostNeverLoadsPolicy(t *testing.T) {
	loaderCalls := 0
	loader := func() (core.Policy, error) { loaderCalls++; return core.Policy{}, assert.AnError }
	got, err := LabelRole(loader, domain.LevelingConfig{Mode: domain.LevelingModeAutomatic}, filepath.Join(t.TempDir(), "role-levels.jsonl"), LabelOptions{Role: "ranger", Provider: "CLAUDE", HostModel: "opus", HostEffort: "low"})
	require.NoError(t, err)
	assert.Equal(t, "Opus-Low", got.Level.Label())
	assert.Zero(t, loaderCalls)
}

func TestMigrationLabelRoleReportsRecordingAndRuns(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	opts := LabelOptions{Role: "archivist", Mission: "m1", HostModel: "sonnet", HostEffort: "medium"}
	first, err := LabelRole(migrationLoader(migrationPolicy(t)), domain.LevelingConfig{}, ledger, opts)
	require.NoError(t, err)
	assert.True(t, first.Recorded)

	sameRun, err := LabelRole(migrationLoader(migrationPolicy(t)), domain.LevelingConfig{}, ledger, LabelOptions{Role: "archivist", Mission: "m1", HostModel: "opus", HostEffort: "high"})
	require.NoError(t, err)
	assert.True(t, sameRun.Reused)

	revision, err := LabelRole(migrationLoader(migrationPolicy(t)), domain.LevelingConfig{}, ledger, LabelOptions{Role: "archivist", Mission: "m1", Run: "2", HostModel: "opus", HostEffort: "high"})
	require.NoError(t, err)
	assert.False(t, revision.Reused)
	assert.Equal(t, "Opus-High", revision.Level.Label())

	noMission, err := LabelRole(migrationLoader(migrationPolicy(t)), domain.LevelingConfig{}, ledger, LabelOptions{Role: "ranger", HostModel: "sonnet", HostEffort: "high"})
	require.NoError(t, err)
	assert.False(t, noMission.Recorded)
}

func TestMigrationLabelRoleUsesRoleLevelingKey(t *testing.T) {
	reg, err := domain.NewRoleRegistry([]domain.Role{{ID: "scout"}, {ID: "ranger", Slot: "discovery", Phase: 1, Pluggable: true, Leveling: "sniper"}})
	require.NoError(t, err)
	policy := migrationPolicy(t)
	got, err := LabelRoleWith(reg, migrationLoader(policy), domain.LevelingConfig{}, filepath.Join(t.TempDir(), "role-levels.jsonl"), LabelOptions{Role: "ranger", Provider: "CLAUDE"})
	require.NoError(t, err)
	want, err := core.Suggest(policy, "CLAUDE", "sniper", core.Signals{})
	require.NoError(t, err)
	assert.Equal(t, want.Effort, got.Level.Effort)
	assert.Equal(t, "ranger", got.Level.Role)
}

func TestMigrationLabelRoleNormalizesRunsAndGate(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	policy := migrationPolicy(t)
	first, err := LabelRole(migrationLoader(policy), domain.LevelingConfig{}, ledger, LabelOptions{Role: "Ranger", Mission: "m1", HostModel: "Opus", HostEffort: "high"})
	require.NoError(t, err)
	assert.Equal(t, "ranger", first.Level.Role)
	second, err := LabelRole(migrationLoader(policy), domain.LevelingConfig{}, ledger, LabelOptions{Role: "ranger", Mission: "m1"})
	require.NoError(t, err)
	assert.True(t, second.Reused)

	gate, err := LabelRole(migrationLoader(policy), domain.LevelingConfig{}, ledger, LabelOptions{Role: "Gate", Mission: "m1", HostModel: "Opus", HostEffort: "high"})
	require.NoError(t, err)
	assert.False(t, gate.Recorded)
	records := mustReadRecords(t, ledger)
	require.Len(t, records, 1)
	assert.Equal(t, "ranger", records[0].Role)
	assert.Contains(t, core.RenderWith(domain.DefaultRoleRegistry(), gate.Level, "x", 0), "Fase: 03/04")
}

func TestMigrationLabelRoleRecordsUnknownOnlyOnceAndEscalates(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	opts := LabelOptions{Role: "archivist", Mission: "m1"}
	for range 3 {
		_, err := LabelRole(migrationLoader(migrationPolicy(t)), domain.LevelingConfig{}, ledger, opts)
		require.NoError(t, err)
	}
	assert.Len(t, mustReadRecords(t, ledger), 1)
	opts.Run = "2"
	_, err := LabelRole(migrationLoader(migrationPolicy(t)), domain.LevelingConfig{}, ledger, opts)
	require.NoError(t, err)
	assert.Len(t, mustReadRecords(t, ledger), 2)

	opts.Run = ""
	opts.Reason = "escalated"
	_, err = LabelRole(migrationLoader(migrationPolicy(t)), domain.LevelingConfig{}, ledger, opts)
	require.NoError(t, err)
	assert.Len(t, mustReadRecords(t, ledger), 3)
}

func TestMigrationLabelRoleRejectsPlaceholdersAndUnknownEffort(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	got, err := LabelRole(migrationLoader(migrationPolicy(t)), domain.LevelingConfig{}, ledger, LabelOptions{Role: "ranger", Mission: "m1", HostModel: "<your-model>", HostEffort: "<your-effort>"})
	require.NoError(t, err)
	assert.True(t, got.Level.Unknown())
	assert.Contains(t, got.Warning, "placeholder")

	got, err = LabelRole(migrationLoader(migrationPolicy(t)), domain.LevelingConfig{}, ledger, LabelOptions{Role: "sniper", Mission: "m1", HostModel: "Opus", HostEffort: "banana"})
	require.NoError(t, err)
	assert.Equal(t, "Opus", got.Level.Label())
	assert.Contains(t, got.Warning, "banana")
}

func TestMigrationWriteJSONLabelCarriesInlineTag(t *testing.T) {
	var out bytes.Buffer
	cmd := NewLabel(testDeps(t.TempDir(), nil), &LabelOptions{})
	cmd.SetOut(&out)
	require.NoError(t, WriteJSONLabel(cmd, LabelResult{Level: core.Level{Role: "ranger", Model: "Sonnet", Effort: "high", Source: core.SourceHost}}, "r"))
	var payload map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &payload))
	assert.Equal(t, "(Sonnet-High)", payload["tag"])
	assert.Equal(t, "Sonnet-High", payload["label"])
}

func TestMigrationValidateChecksDigestAndWritesSummary(t *testing.T) {
	policy := migrationPolicy(t)
	deps := Dependencies{LoadPolicy: func() (core.Policy, string, error) {
		return policy, ".strategist/leveling.yaml", nil
	}}
	cmd := NewValidate(deps)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--expected-digest", policy.Digest()})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "policy=.strategist/leveling.yaml")
	assert.Contains(t, out.String(), "providers=")

	cmd = NewValidate(deps)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--expected-digest", "bad-digest"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "verify digest")
}

func TestMigrationSuggestRendersPlainAndJSON(t *testing.T) {
	policy := migrationPolicy(t)
	deps := Dependencies{LoadPolicy: func() (core.Policy, string, error) {
		return policy, "", nil
	}}

	plain := NewSuggest(deps)
	var plainOut bytes.Buffer
	plain.SetOut(&plainOut)
	plain.SetArgs([]string{"--provider", "CLAUDE", "--role", "ranger"})
	require.NoError(t, plain.Execute())
	assert.Contains(t, plainOut.String(), "provider=CLAUDE")
	assert.Contains(t, plainOut.String(), "role=ranger")

	jsonCmd := NewSuggest(deps)
	var jsonOut bytes.Buffer
	jsonCmd.SetOut(&jsonOut)
	jsonCmd.SetArgs([]string{"--provider", "CLAUDE", "--role", "ranger", "--json"})
	require.NoError(t, jsonCmd.Execute())
	var suggestion map[string]any
	require.NoError(t, json.Unmarshal(jsonOut.Bytes(), &suggestion))
	assert.Equal(t, "CLAUDE", suggestion["provider"])
}

func TestMigrationLabelUsesDefaultDependencyValues(t *testing.T) {
	root := t.TempDir()
	policy := migrationPolicy(t)
	deps := Dependencies{
		LoadPolicy:    func() (core.Policy, string, error) { return policy, "", nil },
		WorkspaceRoot: func() (string, error) { return root, nil },
	}
	cmd := NewLabel(deps, &LabelOptions{})
	cmd.SetArgs([]string{"--role", "ranger", "--mission", "m1", "--host-model", "sonnet", "--host-effort", "high"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	require.NoError(t, cmd.Execute())
	assert.Equal(t, "Sonnet-High\n", out.String())
	assert.FileExists(t, filepath.Join(root, "memory", defaultLedgerName))
}

func TestMigrationSuggestAndValidatePropagatePolicyErrors(t *testing.T) {
	deps := Dependencies{LoadPolicy: func() (core.Policy, string, error) {
		return core.Policy{}, "", assert.AnError
	}}

	validate := NewValidate(deps)
	validate.SetErr(&bytes.Buffer{})
	require.ErrorIs(t, validate.Execute(), assert.AnError)

	suggest := NewSuggest(deps)
	suggest.SetErr(&bytes.Buffer{})
	assert.ErrorIs(t, suggest.Execute(), assert.AnError)
}

func TestMigrationSuggestRejectsInvalidPolicyInput(t *testing.T) {
	policy := migrationPolicy(t)
	deps := Dependencies{LoadPolicy: func() (core.Policy, string, error) {
		return policy, "", nil
	}}
	cmd := NewSuggest(deps)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--provider", "UNKNOWN", "--role", "ranger", "--risk", "bogus"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "leveling: suggest")
}

func TestMigrationLabelPropagatesWorkspaceErrors(t *testing.T) {
	deps := Dependencies{WorkspaceRoot: func() (string, error) { return "", assert.AnError }}
	cmd := NewLabel(deps, &LabelOptions{})
	cmd.SetErr(&bytes.Buffer{})
	assert.ErrorIs(t, cmd.Execute(), assert.AnError)
}

func TestMigrationLabelOutputWritesWarningsAndHandlesWriterErrors(t *testing.T) {
	cmd := NewLabel(testDeps(t.TempDir(), nil), &LabelOptions{})
	var errOut bytes.Buffer
	cmd.SetErr(&errOut)
	require.NoError(t, WriteLabelWarning(cmd, "degraded"))
	assert.Equal(t, "warning: degraded\n", errOut.String())

	var out bytes.Buffer
	cmd.SetOut(&out)
	require.NoError(t, WritePlainLabel(cmd, "Ranger", "rendered", false))
	require.NoError(t, WritePlainLabel(cmd, "Ranger", "rendered", true))
	assert.Equal(t, "Ranger\nrendered\n", out.String())

	failing := NewLabel(testDeps(t.TempDir(), nil), &LabelOptions{})
	failing.SetOut(migrationFailWriter{})
	failing.SetErr(migrationFailWriter{})
	require.Error(t, WriteLabelWarning(failing, "degraded"))
	require.Error(t, WritePlainLabel(failing, "Ranger", "rendered", false))
	assert.Error(t, WriteJSONLabel(failing, LabelResult{Level: core.Level{Role: "ranger"}}, "rendered"))
}

type migrationFailWriter struct{}

func (migrationFailWriter) Write([]byte) (int, error) { return 0, assert.AnError }

func mustReadRecords(t *testing.T, path string) []core.Record {
	t.Helper()
	records, err := core.ReadRecords(path)
	require.NoError(t, err)
	return records
}

func readMigrationDefaults() ([]byte, error) {
	return (embed.Extractor{}).ReadFile("leveling.yaml")
}
