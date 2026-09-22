package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func labelTestPolicy(t *testing.T) leveling.Policy {
	t.Helper()
	raw, err := embed.Extractor{}.ReadFile("leveling.yaml")
	require.NoError(t, err)
	policy, err := leveling.Parse(raw)
	require.NoError(t, err)
	return policy
}

func TestLabelRolePersistsAndReusesPhaseLevel(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	policy := labelTestPolicy(t)
	opts := levelingLabelOptions{Role: "ranger", Mission: "m1", HostModel: "sonnet", HostEffort: "high"}

	first, err := labelRole(loaderOf(policy), domain.LevelingConfig{}, ledger, opts)
	require.NoError(t, err)
	assert.Equal(t, "Sonnet-High", first.Level.Label())
	assert.False(t, first.Reused)

	// A later line of the same phase must not drift, even if the host input changes.
	opts.HostModel = "opus"
	second, err := labelRole(loaderOf(policy), domain.LevelingConfig{}, ledger, opts)
	require.NoError(t, err)
	assert.True(t, second.Reused)
	assert.Equal(t, "Sonnet-High", second.Level.Label())
}

func TestLabelRoleEscalationRecordsNewTupleWithReason(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	policy := labelTestPolicy(t)
	base := levelingLabelOptions{Role: "archivist", Mission: "m1", HostModel: "sonnet", HostEffort: "medium"}
	_, err := labelRole(loaderOf(policy), domain.LevelingConfig{}, ledger, base)
	require.NoError(t, err)

	esc := levelingLabelOptions{Role: "archivist", Mission: "m1", HostModel: "opus", HostEffort: "high", Reason: "escalated"}
	got, err := labelRole(loaderOf(policy), domain.LevelingConfig{}, ledger, esc)
	require.NoError(t, err)
	assert.False(t, got.Reused)
	assert.Equal(t, "Opus-High", got.Level.Label())

	latest, ok, err := leveling.LatestRecord(ledger, "m1", "archivist")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "escalated", latest.Reason)
}

func TestLabelRoleWithoutMissionDoesNotWriteLedger(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	got, err := labelRole(loaderOf(labelTestPolicy(t)), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "scout", Provider: "CLAUDE"})
	require.NoError(t, err)
	assert.Equal(t, leveling.SourcePolicy, got.Level.Source)
	_, ok, err := leveling.LatestRecord(ledger, "", "scout")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestLabelRoleDegradesOnPolicyErrorWithoutBlocking(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	got, err := labelRole(loaderOf(labelTestPolicy(t)), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "ranger", Mission: "m1", Provider: "CLAUDE", Risk: "bogus"})
	require.NoError(t, err)
	assert.True(t, got.Level.Unknown())
	assert.Contains(t, got.Warning, "leveling_signal_unknown")

	latest, ok, err := leveling.LatestRecord(ledger, "m1", "ranger")
	require.NoError(t, err)
	require.True(t, ok, "unknown level is still recorded as null for telemetry")
	assert.True(t, latest.Unknown())
}

func TestLabelRoleUnknownIsNotReusedOnceALevelIsKnown(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	policy := labelTestPolicy(t)
	_, err := labelRole(loaderOf(policy), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "sniper", Mission: "m1"})
	require.NoError(t, err)
	got, err := labelRole(loaderOf(policy), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "sniper", Mission: "m1", HostModel: "haiku", HostEffort: "low"})
	require.NoError(t, err)
	assert.False(t, got.Reused)
	assert.Equal(t, "Haiku-Low", got.Level.Label())
}

func TestLevelingLabelCmdRegistered(t *testing.T) {
	found, _, err := levelingCmd.Find([]string{"label"})
	require.NoError(t, err)
	assert.Equal(t, "label", found.Name())
}

func loaderOf(policy leveling.Policy) leveling.PolicyLoader {
	return func() (leveling.Policy, error) { return policy, nil }
}

type countingPolicyLoader struct{ calls int }

func (c *countingPolicyLoader) load() (leveling.Policy, error) {
	c.calls++
	return leveling.Policy{}, assert.AnError
}

func TestLabelRoleManualCompleteNeverLoadsPolicyAndRecordsManualSource(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	loader := &countingPolicyLoader{}
	cfg := domain.LevelingConfig{Mode: domain.LevelingModeManual, Roles: map[string]domain.LevelingRoleChoice{"ranger": {Model: "Sonnet", Effort: "high"}}}

	got, err := labelRole(loader.load, cfg, ledger, levelingLabelOptions{Role: "ranger", Mission: "m1", Provider: "CLAUDE", HostModel: "opus", HostEffort: "low"})
	require.NoError(t, err)
	assert.Equal(t, "Sonnet-High", got.Level.Label(), "manual wins over the host report")
	assert.Empty(t, got.Warning)
	assert.Zero(t, loader.calls, "leveling.yaml is never read for a complete manual choice")

	record, ok, err := leveling.LatestRecord(ledger, "m1", "ranger")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, leveling.SourceManual, record.Source)
}

func TestLabelRoleManualForOtherRoleFallsBackToAutomatic(t *testing.T) {
	loader := &countingPolicyLoader{}
	cfg := domain.LevelingConfig{Mode: domain.LevelingModeManual, Roles: map[string]domain.LevelingRoleChoice{"ranger": {Model: "Sonnet", Effort: "high"}}}
	got, err := labelRole(loader.load, cfg, filepath.Join(t.TempDir(), "l.jsonl"), levelingLabelOptions{Role: "sniper", Provider: "CLAUDE"})
	require.NoError(t, err)
	assert.Equal(t, 1, loader.calls, "an unconfigured role loads the policy on demand")
	assert.True(t, got.Level.Unknown(), "a loader failure degrades to an unlabelled level")
	assert.NotEmpty(t, got.Warning)
}

func TestLabelRoleAutomaticHostCompleteNeverLoadsPolicy(t *testing.T) {
	loader := &countingPolicyLoader{}
	got, err := labelRole(loader.load, domain.LevelingConfig{Mode: domain.LevelingModeAutomatic}, filepath.Join(t.TempDir(), "l.jsonl"),
		levelingLabelOptions{Role: "ranger", Provider: "CLAUDE", HostModel: "opus", HostEffort: "low"})
	require.NoError(t, err)
	assert.Equal(t, "Opus-Low", got.Level.Label())
	assert.Zero(t, loader.calls)
}

func TestReadActiveLevelingConfig(t *testing.T) {
	dir := t.TempDir()
	cfg, warn := readActiveLevelingConfig(dir)
	assert.Equal(t, domain.LevelingModeAutomatic, cfg.EffectiveMode(), "missing active.yaml is automatic")
	assert.Empty(t, warn)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: epic\nleveling:\n  mode: manual\n  roles:\n    ranger: {model: Sonnet, effort: high}\n"), 0o600))
	cfg, warn = readActiveLevelingConfig(dir)
	assert.Empty(t, warn)
	choice, ok := cfg.Choice("ranger")
	require.True(t, ok)
	assert.True(t, choice.Complete())

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("leveling:\n  mode: manual\n  roles:\n    ranger: {model: S, effort: turbo}\n"), 0o600))
	cfg, warn = readActiveLevelingConfig(dir)
	assert.Contains(t, warn, "leveling_mapping_invalid")
	assert.Equal(t, domain.LevelingModeAutomatic, cfg.EffectiveMode(), "an invalid block degrades to automatic")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(":\n - [broken"), 0o600))
	_, warn = readActiveLevelingConfig(dir)
	assert.NotEmpty(t, warn)
}

func TestLevelingLabelCmdManualWorksWithoutPolicyFile(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, ".strategist")
	testutil.MinimalRoot(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: full\nbase_path: .analysis\nleveling:\n  mode: manual\n  roles:\n    archivist: {model: Opus, effort: medium}\n"), 0o600))
	require.NoFileExists(t, filepath.Join(root, "leveling.yaml"))
	t.Chdir(tmp)

	prev := levelingLabelOpts
	t.Cleanup(func() { levelingLabelOpts = prev })
	levelingLabelOpts = levelingLabelOptions{Role: "archivist", Mission: "m9", Message: "hello", Width: 80}

	var out, errOut bytes.Buffer
	levelingLabelCmd.SetOut(&out)
	levelingLabelCmd.SetErr(&errOut)
	require.NoError(t, levelingLabelCmd.RunE(levelingLabelCmd, nil))
	assert.Equal(t, "Archivist(Opus-Medium) - hello\n", out.String())
	assert.Empty(t, errOut.String(), "no policy warning: leveling.yaml was never needed")
	assert.FileExists(t, filepath.Join(root, "memory", roleLevelLedger))
}

type recordingHandler struct{ records []slog.Record }

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(string) slog.Handler      { return h }

func attrsOf(r slog.Record) map[string]any {
	out := map[string]any{}
	r.Attrs(func(a slog.Attr) bool { out[a.Key] = a.Value.Any(); return true })
	return out
}

func TestEmitRoleLevelCarriesLevelFieldsForTelemetry(t *testing.T) {
	handler := &recordingHandler{}
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(prev) })

	emitRoleLevel(context.Background(), "m1", "2", leveling.Level{Role: "ranger", Model: "Sonnet", Effort: "high", Source: leveling.SourceManual}, "escalated")
	require.Len(t, handler.records, 1)
	assert.Contains(t, handler.records[0].Message, "role_level_resolved")
	attrs := attrsOf(handler.records[0])
	assert.Equal(t, "m1", attrs[telemetry.AttrMissionID])
	assert.Equal(t, "ranger", attrs[telemetry.AttrRole])
	assert.Equal(t, "Sonnet", attrs[telemetry.AttrModel])
	assert.Equal(t, "high", attrs[telemetry.AttrEffort])
	assert.Equal(t, "manual", attrs[telemetry.AttrLevelSource])
	assert.Equal(t, "escalated", attrs[telemetry.AttrReason])
	assert.Equal(t, "2", attrs[telemetry.AttrRoleRun])
}

func TestEmitRoleLevelUnknownLevelEmitsNulls(t *testing.T) {
	handler := &recordingHandler{}
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(prev) })

	emitRoleLevel(context.Background(), "m1", "", leveling.Level{Role: "scout"}, "")
	require.Len(t, handler.records, 1)
	attrs := attrsOf(handler.records[0])
	assert.Empty(t, attrs[telemetry.AttrModel])
	assert.Empty(t, attrs[telemetry.AttrEffort])
	assert.Empty(t, attrs[telemetry.AttrLevelSource])
	assert.NotContains(t, attrs, telemetry.AttrReason, "no reason attribute when none was given")
	assert.NotContains(t, attrs, telemetry.AttrRoleRun, "no run attribute for the default run")
}

func TestLabelRoleReportsWhetherItRecordedANewTuple(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "l.jsonl")
	opts := levelingLabelOptions{Role: "ranger", Mission: "m1", HostModel: "sonnet", HostEffort: "high"}
	first, err := labelRole(loaderOf(labelTestPolicy(t)), domain.LevelingConfig{}, ledger, opts)
	require.NoError(t, err)
	assert.True(t, first.Recorded)
	second, err := labelRole(loaderOf(labelTestPolicy(t)), domain.LevelingConfig{}, ledger, opts)
	require.NoError(t, err)
	assert.False(t, second.Recorded, "a reused tuple is not emitted again")
	noMission, err := labelRole(loaderOf(labelTestPolicy(t)), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "ranger", HostModel: "sonnet", HostEffort: "high"})
	require.NoError(t, err)
	assert.False(t, noMission.Recorded)
}

func TestLabelRoleWithUsesTheRoleLevelingKey(t *testing.T) {
	reg, err := domain.NewRoleRegistry([]domain.Role{{ID: "scout"}, {ID: "ranger", Slot: "discovery", Phase: 1, Pluggable: true, Leveling: "sniper"}})
	require.NoError(t, err)
	policy := labelTestPolicy(t)
	ledger := filepath.Join(t.TempDir(), "l.jsonl")

	got, err := labelRoleWith(reg, loaderOf(policy), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "ranger", Provider: "CLAUDE"})
	require.NoError(t, err)
	want, err := leveling.Suggest(policy, "CLAUDE", "sniper", leveling.Signals{})
	require.NoError(t, err)
	assert.Equal(t, want.Effort, got.Level.Effort, "the ranger resolves with the sniper policy role")
	assert.Equal(t, "ranger", got.Level.Role, "the label still names the real role")
}

func TestLevelingLabelCmdReadsRolesFromTheWorkspace(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, ".strategist")
	testutil.MinimalRoot(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "auditor.yaml"), []byte("role: auditor\nphase: 5\npluggable: false\n"), 0o600))
	t.Chdir(tmp)

	prev := levelingLabelOpts
	t.Cleanup(func() { levelingLabelOpts = prev })
	levelingLabelOpts = levelingLabelOptions{Role: "auditor", HostModel: "Sonnet", HostEffort: "high", Message: "hi"}

	var out, errOut bytes.Buffer
	levelingLabelCmd.SetOut(&out)
	levelingLabelCmd.SetErr(&errOut)
	require.NoError(t, levelingLabelCmd.RunE(levelingLabelCmd, nil))
	assert.Equal(t, "Fase: 05/05\nAuditor\nSonnet-High\nhi\n", out.String())
	assert.Empty(t, errOut.String())
}

func TestLevelingLabelCmdFallsBackToBuiltInRolesOnABrokenRoleFile(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, ".strategist")
	testutil.MinimalRoot(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "broken.yaml"), []byte(":\n - [nope"), 0o600))
	t.Chdir(tmp)

	prev := levelingLabelOpts
	t.Cleanup(func() { levelingLabelOpts = prev })
	levelingLabelOpts = levelingLabelOptions{Role: "ranger", HostModel: "Sonnet", HostEffort: "high", Message: "hi"}

	var out, errOut bytes.Buffer
	levelingLabelCmd.SetOut(&out)
	levelingLabelCmd.SetErr(&errOut)
	require.NoError(t, levelingLabelCmd.RunE(levelingLabelCmd, nil), "a broken role file never blocks labelling")
	assert.Equal(t, "Fase: 01/04\nRanger\nSonnet-High\nhi\n", out.String())
	assert.Contains(t, errOut.String(), "broken.yaml")
}

func TestLabelRoleRepeatedRoleResolvesANewLevelPerRun(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "l.jsonl")
	policy := labelTestPolicy(t)
	base := levelingLabelOptions{Role: "archivist", Mission: "m1", HostModel: "sonnet", HostEffort: "medium"}

	first, err := labelRole(loaderOf(policy), domain.LevelingConfig{}, ledger, base)
	require.NoError(t, err)
	assert.Equal(t, "Sonnet-Medium", first.Level.Label())

	sameRun, err := labelRole(loaderOf(policy), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "archivist", Mission: "m1", HostModel: "opus", HostEffort: "high"})
	require.NoError(t, err)
	assert.True(t, sameRun.Reused, "the same run keeps its level")
	assert.Equal(t, "Sonnet-Medium", sameRun.Level.Label())

	revision, err := labelRole(loaderOf(policy), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "archivist", Mission: "m1", Run: "2", HostModel: "opus", HostEffort: "high"})
	require.NoError(t, err)
	assert.False(t, revision.Reused, "a revision loop is a new run")
	assert.Equal(t, "Opus-High", revision.Level.Label())

	again, err := labelRole(loaderOf(policy), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "archivist", Mission: "m1", Run: "2", HostModel: "haiku", HostEffort: "low"})
	require.NoError(t, err)
	assert.True(t, again.Reused)
	assert.Equal(t, "Opus-High", again.Level.Label())
}

// A role's on_start hook is a real command: it must parse to `leveling label`
// with valid flags, and running it must resolve and record the role's level.
func TestRoleStartHookResolvesAndRecordsTheLevelForEveryRole(t *testing.T) {
	reg := domain.DefaultRoleRegistry()
	policy := labelTestPolicy(t)
	for _, id := range reg.IDs() {
		commands := reg.StartCommands(id, "m-start")
		require.Len(t, commands, 1, id)
		fields := strings.Fields(commands[0])
		require.Equal(t, "strategist", fields[0], id)

		found, rest, err := rootCmd.Find(fields[1:])
		require.NoError(t, err, id)
		require.Same(t, levelingLabelCmd, found, "%s hook must invoke `leveling label`", id)

		prev := levelingLabelOpts
		t.Cleanup(func() { levelingLabelOpts = prev })
		levelingLabelOpts = levelingLabelOptions{}
		require.NoError(t, found.ParseFlags(rest), id)
		opts := levelingLabelOpts
		opts.HostModel, opts.HostEffort = "Sonnet", "high"

		ledger := filepath.Join(t.TempDir(), "l.jsonl")
		result, err := labelRoleWith(reg, loaderOf(policy), domain.LevelingConfig{}, ledger, opts)
		require.NoError(t, err, id)
		assert.True(t, result.Recorded, id)
		record, ok, err := leveling.LatestRecord(ledger, "m-start", id)
		require.NoError(t, err, id)
		require.True(t, ok, "%s: the start hook must record its level", id)
		assert.Equal(t, "Sonnet-High", record.Label(), id)
	}
}

func TestLabelJSONCarriesTheInlineTag(t *testing.T) {
	var out bytes.Buffer
	cmd := *levelingLabelCmd
	cmd.SetOut(&out)
	require.NoError(t, writeJSONLabel(&cmd, labelResult{Level: leveling.Level{Role: "ranger", Model: "Sonnet", Effort: "high", Source: leveling.SourceHost}}, "r"))
	var payload map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &payload))
	assert.Equal(t, "(Sonnet-High)", payload["tag"])
	assert.Equal(t, "Sonnet-High", payload["label"])

	out.Reset()
	require.NoError(t, writeJSONLabel(&cmd, labelResult{Level: leveling.Level{Role: "scout"}}, "r"))
	require.NoError(t, json.Unmarshal(out.Bytes(), &payload))
	assert.Empty(t, payload["tag"], "an unknown level has an empty tag")
}

func TestNewLevelingCommand_HasIsolatedCompleteSubcommandTree(t *testing.T) {
	cmd := newLevelingCommand()

	assert.NotSame(t, levelingCmd, cmd)
	assert.Len(t, cmd.Commands(), 3)
	for _, name := range []string{"validate", "suggest", "label"} {
		subcommand, _, err := cmd.Find([]string{name})
		require.NoError(t, err)
		assert.NotSame(t, levelingCmd, subcommand)
	}
}
