package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedLevelsRoot(t *testing.T, records ...leveling.Record) string {
	t.Helper()
	root := t.TempDir()
	for _, record := range records {
		require.NoError(t, leveling.AppendRecord(filepath.Join(root, "memory", roleLevelLedger), record))
	}
	return root
}

func levelRec(mission, role, model, effort, source string) leveling.Record {
	return leveling.Record{MissionID: mission, Level: leveling.Level{Role: role, Model: model, Effort: effort, Source: source}}
}

func TestMetricsLevelsReportsSeededLedger(t *testing.T) {
	root := seedLevelsRoot(t,
		levelRec("m1", "ranger", "Sonnet", "high", "host"),
		levelRec("m2", "ranger", "Sonnet", "high", "manual"),
		levelRec("m1", "archivist", "Opus", "medium", "policy"),
		levelRec("m1", "scout", "", "", ""),
	)
	var out bytes.Buffer
	require.NoError(t, writeLevelsReport(&out, root, metricsLevelsOptions{MaxRecords: 2000}))
	text := out.String()
	assert.Contains(t, text, "records: 4\n")
	assert.Contains(t, text, "missions: 2\n")
	assert.Contains(t, text, "unknown: 1\n")
	assert.Contains(t, text, "role.ranger.records: 2\n")
	assert.Contains(t, text, "role.ranger.level.Sonnet-High: 2\n")
	assert.Contains(t, text, "role.ranger.source.manual: 1\n")
	assert.Contains(t, text, "role.archivist.level.Opus-Medium: 1\n")
}

func TestMetricsLevelsEmptyLedgerReportsZero(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, writeLevelsReport(&out, t.TempDir(), metricsLevelsOptions{MaxRecords: 2000}))
	assert.Equal(t, "records: 0\nmissions: 0\nunknown: 0\nescalations: 0\n", out.String())
}

func TestMetricsLevelsJSONAndMissionFilter(t *testing.T) {
	root := seedLevelsRoot(t, levelRec("m1", "ranger", "Sonnet", "high", "host"), levelRec("m2", "sniper", "Haiku", "low", "policy"))
	var out bytes.Buffer
	require.NoError(t, writeLevelsReport(&out, root, metricsLevelsOptions{Mission: "m2", JSON: true, MaxRecords: 2000}))
	var report leveling.Report
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	assert.Equal(t, 1, report.Records)
	require.Len(t, report.Roles, 1)
	assert.Equal(t, "sniper", report.Roles[0].Role)
}

func TestMetricsLevelsRotateCompactsLedgerAndKeepsLatestTuples(t *testing.T) {
	root := seedLevelsRoot(t)
	path := filepath.Join(root, "memory", roleLevelLedger)
	for i := 0; i < 5; i++ {
		require.NoError(t, leveling.AppendRecord(path, levelRec("m1", "ranger", "Old", "low", "host")))
	}
	require.NoError(t, leveling.AppendRecord(path, levelRec("m1", "ranger", "Newest", "high", "host")))

	var out bytes.Buffer
	require.NoError(t, writeLevelsReport(&out, root, metricsLevelsOptions{Rotate: true, MaxRecords: 2}))
	assert.Contains(t, out.String(), "rotated: dropped 4 record(s), max_records=2\n")
	got, ok, err := leveling.LatestRecord(path, "m1", "ranger")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "Newest", got.Model)
	raw, err := os.ReadFile(path) //nolint:gosec // test temp file
	require.NoError(t, err)
	assert.Equal(t, 2, bytes.Count(raw, []byte("\n")))
}

func TestMetricsLevelsCmdRegistered(t *testing.T) {
	found, _, err := metricsCmd.Find([]string{"levels"})
	require.NoError(t, err)
	assert.Equal(t, "levels", found.Name())
}
