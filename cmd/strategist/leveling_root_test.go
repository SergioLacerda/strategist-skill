package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	levelingadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var levelingLabelOpts levelingadapter.LabelOptions
var levelingLabelCmd = levelingadapter.NewLabel(levelingAdapterDependencies(), &levelingLabelOpts)

func rootLevelingPolicy(t *testing.T) leveling.Policy {
	t.Helper()
	raw, err := (embed.Extractor{}).ReadFile("leveling.yaml")
	require.NoError(t, err)
	policy, err := leveling.Parse(raw)
	require.NoError(t, err)
	return policy
}

func TestReadActiveLevelingConfig(t *testing.T) {
	dir := t.TempDir()
	cfg, warn := readActiveLevelingConfig(dir)
	assert.Equal(t, domain.LevelingModeAutomatic, cfg.EffectiveMode())
	assert.Empty(t, warn)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: epic\nleveling:\n  mode: manual\n"), 0o600))
	cfg, warn = readActiveLevelingConfig(dir)
	assert.Empty(t, warn)
	assert.True(t, cfg.HostPassthrough())

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("leveling:\n  mode: smart\n"), 0o600))
	cfg, warn = readActiveLevelingConfig(dir)
	assert.Contains(t, warn, "leveling_mapping_invalid")
	assert.Equal(t, domain.LevelingModeAutomatic, cfg.EffectiveMode())

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(":\n - [broken"), 0o600))
	_, warn = readActiveLevelingConfig(dir)
	assert.NotEmpty(t, warn)
}

func TestLevelingLabelCmdManualWorksWithoutPolicyFile(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, ".strategist")
	testutil.MinimalRoot(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: full\nbase_path: .analysis\nleveling:\n  mode: manual\n"), 0o600))
	require.NoFileExists(t, filepath.Join(root, "leveling.yaml"))
	t.Chdir(tmp)

	prev := levelingLabelOpts
	t.Cleanup(func() { levelingLabelOpts = prev })
	levelingLabelOpts = levelingadapter.LabelOptions{Role: "archivist", Mission: "m9", Message: "hello", Width: 80, HostModel: "Opus", HostEffort: "medium"}
	var out, errOut bytes.Buffer
	levelingLabelCmd.SetOut(&out)
	levelingLabelCmd.SetErr(&errOut)
	require.NoError(t, levelingLabelCmd.RunE(levelingLabelCmd, nil))
	assert.Equal(t, "Archivist(Opus-Medium) - hello\n", out.String())
	assert.Empty(t, errOut.String())
	assert.FileExists(t, filepath.Join(root, "memory", roleLevelLedger))
}

type rootRecordingHandler struct{ records []slog.Record }

func (h *rootRecordingHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *rootRecordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *rootRecordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *rootRecordingHandler) WithGroup(string) slog.Handler      { return h }

func rootAttrsOf(r slog.Record) map[string]any {
	out := map[string]any{}
	r.Attrs(func(a slog.Attr) bool { out[a.Key] = a.Value.Any(); return true })
	return out
}

func TestEmitRoleLevelCarriesLevelFieldsForTelemetry(t *testing.T) {
	handler := &rootRecordingHandler{}
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(previous) })

	emitRoleLevel(context.Background(), "m1", "2", leveling.Level{Role: "ranger", Model: "Sonnet", Effort: "high", Source: leveling.SourceHost}, "escalated")
	require.Len(t, handler.records, 1)
	attrs := rootAttrsOf(handler.records[0])
	assert.Equal(t, "m1", attrs[telemetry.AttrMissionID])
	assert.Equal(t, "ranger", attrs[telemetry.AttrRole])
	assert.Equal(t, "Sonnet", attrs[telemetry.AttrModel])
	assert.Equal(t, "high", attrs[telemetry.AttrEffort])
	assert.Equal(t, "host", attrs[telemetry.AttrLevelSource])
	assert.Equal(t, "escalated", attrs[telemetry.AttrReason])
	assert.Equal(t, "2", attrs[telemetry.AttrRoleRun])
}

func TestEmitRoleLevelUnknownLevelEmitsNulls(t *testing.T) {
	handler := &rootRecordingHandler{}
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(previous) })

	emitRoleLevel(context.Background(), "m1", "", leveling.Level{Role: "scout"}, "")
	require.Len(t, handler.records, 1)
	attrs := rootAttrsOf(handler.records[0])
	assert.Empty(t, attrs[telemetry.AttrModel])
	assert.Empty(t, attrs[telemetry.AttrEffort])
	assert.Empty(t, attrs[telemetry.AttrLevelSource])
	assert.NotContains(t, attrs, telemetry.AttrReason)
	assert.NotContains(t, attrs, telemetry.AttrRoleRun)
}

func TestLevelingLabelCmdReadsRolesFromWorkspace(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, ".strategist")
	testutil.MinimalRoot(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "auditor.yaml"), []byte("role: auditor\nphase: 5\npluggable: false\n"), 0o600))
	t.Chdir(tmp)

	prev := levelingLabelOpts
	t.Cleanup(func() { levelingLabelOpts = prev })
	levelingLabelOpts = levelingadapter.LabelOptions{Role: "auditor", HostModel: "Sonnet", HostEffort: "high", Message: "hi"}
	var out, errOut bytes.Buffer
	levelingLabelCmd.SetOut(&out)
	levelingLabelCmd.SetErr(&errOut)
	require.NoError(t, levelingLabelCmd.RunE(levelingLabelCmd, nil))
	assert.Equal(t, "Fase: 05/05\nAuditor\nSonnet-High\nhi\n", out.String())
	assert.Empty(t, errOut.String())
}

func TestLevelingLabelCmdFallsBackToBuiltInRolesOnBrokenFile(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, ".strategist")
	testutil.MinimalRoot(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "broken.yaml"), []byte(":\n - [nope"), 0o600))
	t.Chdir(tmp)

	prev := levelingLabelOpts
	t.Cleanup(func() { levelingLabelOpts = prev })
	levelingLabelOpts = levelingadapter.LabelOptions{Role: "ranger", HostModel: "Sonnet", HostEffort: "high", Message: "hi"}
	var out, errOut bytes.Buffer
	levelingLabelCmd.SetOut(&out)
	levelingLabelCmd.SetErr(&errOut)
	require.NoError(t, levelingLabelCmd.RunE(levelingLabelCmd, nil))
	assert.Equal(t, "Fase: 01/04\nRanger\nSonnet-High\nhi\n", out.String())
	assert.Contains(t, errOut.String(), "broken.yaml")
}

func TestRoleStartHookResolvesAndRecordsEveryRole(t *testing.T) {
	reg := domain.DefaultRoleRegistry()
	policy := rootLevelingPolicy(t)
	for _, id := range reg.IDs() {
		commands := reg.StartCommands(id, "m-start")
		require.Len(t, commands, 1, id)
		fields := strings.Fields(commands[0])
		require.Equal(t, "strategist", fields[0], id)

		found, rest, err := rootCmd.Find(fields[1:])
		require.NoError(t, err, id)
		require.Equal(t, "label", found.Name(), "%s hook must invoke `leveling label`", id)

		opts := levelingadapter.LabelOptions{}
		cmd := levelingadapter.NewLabel(levelingAdapterDependencies(), &opts)
		require.NoError(t, cmd.ParseFlags(rest), id)
		opts.HostModel, opts.HostEffort = "Sonnet", "high"
		ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
		result, err := levelingadapter.LabelRoleWith(reg, func() (leveling.Policy, error) { return policy, nil }, domain.LevelingConfig{}, ledger, opts)
		require.NoError(t, err, id)
		assert.True(t, result.Recorded, id)
		record, ok, err := leveling.LatestRecord(ledger, "m-start", id)
		require.NoError(t, err, id)
		require.True(t, ok, "%s: the start hook must record its level", id)
		assert.Equal(t, "Sonnet-High", record.Label(), id)
	}
}
