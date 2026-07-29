package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/governance"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// writeSddFixtures creates .sdd/metadata.json and .sdd/source/governance-core.json
// under dir with the given mandate IDs, all marked type=MANDATE status=required.
func writeSddFixtures(t *testing.T, dir string, mandateIDs []string) {
	t.Helper()
	sddDir := filepath.Join(dir, ".sdd")
	require.NoError(t, os.MkdirAll(filepath.Join(sddDir, "source"), 0o755))

	meta := map[string]any{"fingerprints": map[string]any{"combined": "abc123"}}
	metaRaw, _ := json.Marshal(meta)
	require.NoError(t, os.WriteFile(filepath.Join(sddDir, "metadata.json"), metaRaw, 0o644))

	items := make([]map[string]any, 0, len(mandateIDs))
	for _, id := range mandateIDs {
		items = append(items, map[string]any{"id": id, "type": "MANDATE", "status": "required"})
	}
	coreRaw, _ := json.Marshal(map[string]any{"items": items})
	require.NoError(t, os.WriteFile(filepath.Join(sddDir, "source", "governance-core.json"), coreRaw, 0o644))
}

// writeSkillYAML marshals data as YAML and writes it to dir/.strategist/skill.yaml (or subpath).
func writeSkillYAML(t *testing.T, dir, subpath string, data map[string]any) string {
	t.Helper()
	full := filepath.Join(dir, subpath)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	raw, err := yaml.Marshal(data)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(full, raw, 0o644))
	return full
}

// --- printSyncReport ---

func TestPrintSyncReport_Compliant(t *testing.T) {
	out := captureStdout(t, func() {
		printSyncReport(governance.SyncReport{GovernanceFingerprint: "fp1"}, nil)
	})
	assert.Contains(t, out, "fp1")
	assert.Contains(t, out, "status=ok")
}

func TestPrintSyncReport_WithMissing(t *testing.T) {
	out := captureStdout(t, func() {
		printSyncReport(governance.SyncReport{
			GovernanceFingerprint: "fp2",
			MandatesActive:        []string{"M001", "M002"},
			MandatesMissing:       []string{"M002"},
		}, nil)
	})
	assert.Contains(t, out, "M002")
}

func TestPrintSyncReport_DryRun(t *testing.T) {
	out := captureStdout(t, func() {
		printSyncReport(governance.SyncReport{FieldsApplied: []string{"validation_policy"}, DryRun: true}, nil)
	})
	assert.Contains(t, out, "dry-run")
	assert.Contains(t, out, "validation_policy")
}

func TestPrintSyncReport_Applied(t *testing.T) {
	out := captureStdout(t, func() {
		printSyncReport(governance.SyncReport{FieldsApplied: []string{"budget_policy"}, DryRun: false}, nil)
	})
	assert.Contains(t, out, "applied")
	assert.Contains(t, out, "budget_policy")
}

// --- runSyncGovernanceCmd (Cobra RunE) ---

func TestSyncGovernanceCmd_Success(t *testing.T) {
	dir := t.TempDir()
	writeSddFixtures(t, dir, []string{"M001"})
	writeSkillYAML(t, dir, ".strategist/skill.yaml", map[string]any{
		"compliance":        map[string]any{"mandates": []any{"M001"}},
		"validation_policy": map[string]any{},
		"budget_policy":     map[string]any{},
		"telemetry_policy":  map[string]any{},
	})
	origRoot, origSdd := syncGovernanceRoot, syncGovernanceSddDir
	t.Cleanup(func() { syncGovernanceRoot = origRoot; syncGovernanceSddDir = origSdd })
	syncGovernanceRoot = filepath.Join(dir, ".strategist")
	syncGovernanceSddDir = filepath.Join(dir, ".sdd")

	_ = captureStdout(t, func() {
		require.NoError(t, syncGovernanceCmd.RunE(syncGovernanceCmd, nil))
	})
}

func TestSyncGovernanceCmd_WithMissionRunDoesNotError(t *testing.T) {
	dir := t.TempDir()
	writeSddFixtures(t, dir, []string{"M001"})
	writeSkillYAML(t, dir, ".strategist/skill.yaml", map[string]any{
		"compliance":        map[string]any{"mandates": []any{"M001"}},
		"validation_policy": map[string]any{},
		"budget_policy":     map[string]any{},
		"telemetry_policy":  map[string]any{},
	})
	origRoot, origSdd := syncGovernanceRoot, syncGovernanceSddDir
	t.Cleanup(func() { syncGovernanceRoot = origRoot; syncGovernanceSddDir = origSdd })
	syncGovernanceRoot = filepath.Join(dir, ".strategist")
	syncGovernanceSddDir = filepath.Join(dir, ".sdd")
	attachMissionRun(t, syncGovernanceCmd)

	_ = captureStdout(t, func() {
		require.NoError(t, syncGovernanceCmd.RunE(syncGovernanceCmd, nil))
	})
}

func TestSyncGovernanceCmd_ErrorPath(t *testing.T) {
	origRoot, origSdd := syncGovernanceRoot, syncGovernanceSddDir
	t.Cleanup(func() { syncGovernanceRoot = origRoot; syncGovernanceSddDir = origSdd })
	syncGovernanceRoot = filepath.Join(t.TempDir(), ".strategist")
	syncGovernanceSddDir = filepath.Join(t.TempDir(), ".sdd")

	err := syncGovernanceCmd.RunE(syncGovernanceCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sync-governance")
}

func TestSyncGovernanceCmd_DefaultFlags(t *testing.T) {
	// When root/sdd are empty the RunE sets them to defaults; since no .sdd
	// exists in the temp CWD the command should return an error.
	origRoot, origSdd := syncGovernanceRoot, syncGovernanceSddDir
	t.Cleanup(func() { syncGovernanceRoot = origRoot; syncGovernanceSddDir = origSdd })
	syncGovernanceRoot = ""
	syncGovernanceSddDir = ""

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	err = syncGovernanceCmd.RunE(syncGovernanceCmd, nil)
	require.Error(t, err)
	assert.Equal(t, ".strategist", syncGovernanceRoot)
	assert.Equal(t, ".sdd", syncGovernanceSddDir)
}

// --- addLine ---

func TestAddLine_NilRun(t *testing.T) {
	t.Parallel()
	// addLine must not panic when run is nil.
	assert.NotPanics(t, func() { addLine(nil) })
}

func TestAddLine_NonNilRun(t *testing.T) {
	t.Parallel()
	run := telemetry.NewMissionRun("test-add-line")
	// addLine must not panic and must update snapshot metrics.
	assert.NotPanics(t, func() { addLine(run) })
	snap := run.Snapshot()
	assert.Equal(t, int64(1), snap.LinesEmitted)
}

// --- go-file-size-report (Makefile contract) ---
