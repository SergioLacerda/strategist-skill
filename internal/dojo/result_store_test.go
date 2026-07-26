package dojo_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersistResult_WritesResultJSON(t *testing.T) {
	base := t.TempDir()
	result := domain.DojoCheckResult{
		Scenario: "quick-draw",
		Items: []domain.DojoCheckItem{
			{Label: "files_created todo/geral.md", Passed: true},
		},
	}
	started := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	finished := started.Add(2 * time.Second)

	require.NoError(t, dojo.PersistResult(base, result, started, finished))

	raw, err := os.ReadFile(filepath.Join(base, "dojo", ".last-run", "quick-draw", "result.json"))
	require.NoError(t, err)
	var record dojo.ResultRecord
	require.NoError(t, json.Unmarshal(raw, &record))
	assert.Equal(t, "quick-draw", record.Scenario)
	assert.True(t, record.Passed)
	assert.Equal(t, 0, record.FailCount)
	assert.Equal(t, "2026-07-25T10:00:00Z", record.StartedAt)
	assert.Len(t, record.Items, 1)
}

func TestPersistResult_AppendsHistory(t *testing.T) {
	base := t.TempDir()
	failing := domain.DojoCheckResult{
		Scenario: "critical-hit",
		Items: []domain.DojoCheckItem{
			{Label: "timing total_wall_time_ms", Passed: false, Detail: "too slow"},
		},
	}
	now := time.Now()
	require.NoError(t, dojo.PersistResult(base, failing, now, now))
	require.NoError(t, dojo.PersistResult(base, failing, now, now))

	raw, err := os.ReadFile(filepath.Join(base, "dojo", ".history.jsonl"))
	require.NoError(t, err)
	lines := splitNonEmptyLines(string(raw))
	require.Len(t, lines, 2)

	var entry dojo.RunRecord
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &entry))
	assert.Equal(t, "critical-hit", entry.Scenario)
	assert.False(t, entry.Passed)
	assert.Equal(t, []dojo.FailureReason{dojo.FailureTiming}, entry.Reasons)
}

func splitNonEmptyLines(s string) []string {
	var out []string
	for _, line := range splitLines(s) {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, r := range s {
		if r == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func TestPersistResult_ScenarioIsolation(t *testing.T) {
	base := t.TempDir()
	result := domain.DojoCheckResult{Scenario: "quick-draw"}
	require.NoError(t, dojo.PersistResult(base, result, time.Now(), time.Now()))

	// Persistence must never touch anything outside <base>/dojo/.
	entries, err := os.ReadDir(base)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "dojo", entries[0].Name())
}
