package main

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ledgerRecords(t *testing.T, ledger string) []leveling.Record {
	t.Helper()
	records, err := leveling.ReadRecords(ledger)
	require.NoError(t, err)
	return records
}

// B2: `--role Ranger` and `--role ranger` are one role for ledger reuse.
func TestLabelRoleReusesAcrossRoleSpellings(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	policy := labelTestPolicy(t)
	first, err := labelRole(loaderOf(policy), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "Ranger", Mission: "m1", HostModel: "Opus", HostEffort: "high"})
	require.NoError(t, err)
	assert.Equal(t, "ranger", first.Level.Role)

	second, err := labelRole(loaderOf(policy), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "ranger", Mission: "m1"})
	require.NoError(t, err)
	assert.True(t, second.Reused, "a different spelling must reuse the recorded level")
	assert.Len(t, ledgerRecords(t, ledger), 1)
}

// H3: the null tuple is the telemetry signal of a failed resolution, so it is
// recorded, but only once per mission, role and run.
func TestLabelRoleRecordsAnUnknownLevelOnlyOnce(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	opts := levelingLabelOptions{Role: "archivist", Mission: "m1"}
	for range 3 {
		_, err := labelRole(loaderOf(labelTestPolicy(t)), domain.LevelingConfig{}, ledger, opts)
		require.NoError(t, err)
	}
	records := ledgerRecords(t, ledger)
	require.Len(t, records, 1, "repeated unknown resolutions must not accumulate")
	assert.True(t, records[0].Unknown())

	opts.Run = "2"
	_, err := labelRole(loaderOf(labelTestPolicy(t)), domain.LevelingConfig{}, ledger, opts)
	require.NoError(t, err)
	assert.Len(t, ledgerRecords(t, ledger), 2, "a new run still records its own null tuple")
}

func TestLabelRoleEscalationRecordsEvenWhenUnknown(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	opts := levelingLabelOptions{Role: "archivist", Mission: "m1"}
	_, err := labelRole(loaderOf(labelTestPolicy(t)), domain.LevelingConfig{}, ledger, opts)
	require.NoError(t, err)
	opts.Reason = "escalated"
	_, err = labelRole(loaderOf(labelTestPolicy(t)), domain.LevelingConfig{}, ledger, opts)
	require.NoError(t, err)
	assert.Len(t, ledgerRecords(t, ledger), 2, "an explicit reason always records a new tuple")
}

// H5: the Approval Gate is not a role and is never written to the ledger, but
// a gate line still renders with its phase header.
func TestLabelRoleNeverRecordsTheGate(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	got, err := labelRole(loaderOf(labelTestPolicy(t)), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "Gate", Mission: "m1", HostModel: "Opus", HostEffort: "high"})
	require.NoError(t, err)
	assert.False(t, got.Recorded)
	assert.Empty(t, ledgerRecords(t, ledger))
	assert.Contains(t, leveling.RenderWith(domain.DefaultRoleRegistry(), got.Level, "x", 0), "Fase: 03/04")
}

// H1 + H2: running the on_start hook verbatim, or with an effort outside the
// tier catalog, never records the bad value.
func TestLabelRoleRejectsPlaceholderAndUnknownEffort(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "role-levels.jsonl")
	got, err := labelRole(loaderOf(labelTestPolicy(t)), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "ranger", Mission: "m1", HostModel: "<your-model>", HostEffort: "<your-effort>"})
	require.NoError(t, err)
	assert.True(t, got.Level.Unknown(), "placeholders must not become a level")
	assert.Contains(t, got.Warning, "placeholder")
	for _, record := range ledgerRecords(t, ledger) {
		assert.NotContains(t, record.Model, "<")
		assert.NotContains(t, record.Effort, "<")
	}

	got, err = labelRole(loaderOf(labelTestPolicy(t)), domain.LevelingConfig{}, ledger, levelingLabelOptions{Role: "sniper", Mission: "m1", HostModel: "Opus", HostEffort: "banana"})
	require.NoError(t, err)
	assert.Equal(t, "Opus", got.Level.Label(), "the valid model is kept, the bad effort dropped")
	assert.Contains(t, got.Warning, "banana")
}
