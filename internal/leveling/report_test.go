package leveling_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedLedger(t *testing.T, records ...leveling.Record) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "role-levels.jsonl")
	for _, record := range records {
		require.NoError(t, leveling.AppendRecord(path, record))
	}
	return path
}

func rec(mission, role, model, effort, source string) leveling.Record {
	return leveling.Record{MissionID: mission, Level: leveling.Level{Role: role, Model: model, Effort: effort, Source: source}}
}

func TestReadRecordsSkipsMalformedLinesAndMissingLedger(t *testing.T) {
	got, err := leveling.ReadRecords(filepath.Join(t.TempDir(), "absent.jsonl"))
	require.NoError(t, err)
	assert.Empty(t, got)

	path := seedLedger(t, rec("m1", "ranger", "Sonnet", "high", "host"))
	appendRaw(t, path, "{broken\n")
	got, err = leveling.ReadRecords(path)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestSummarizeGroupsByRoleAndLevel(t *testing.T) {
	esc := rec("m1", "ranger", "Opus", "xhigh", "manual")
	esc.Reason = "escalated"
	report := leveling.Summarize([]leveling.Record{
		rec("m1", "ranger", "Sonnet", "high", "host"),
		esc,
		rec("m2", "ranger", "Sonnet", "high", "host"),
		rec("m1", "archivist", "Opus", "medium", "policy"),
		rec("m1", "scout", "", "", ""),
	})
	assert.Equal(t, 5, report.Records)
	assert.Equal(t, 2, report.Missions)
	assert.Equal(t, 1, report.Escalations)
	assert.Equal(t, 1, report.Unknown)

	byRole := map[string]leveling.RoleSummary{}
	for _, role := range report.Roles {
		byRole[role.Role] = role
	}
	require.Contains(t, byRole, "ranger")
	assert.Equal(t, 3, byRole["ranger"].Records)
	assert.Equal(t, 2, byRole["ranger"].Levels["Sonnet-High"])
	assert.Equal(t, 1, byRole["ranger"].Levels["Opus-Xhigh"])
	assert.Equal(t, 1, byRole["ranger"].Escalations)
	assert.Equal(t, map[string]int{"host": 2, "manual": 1}, byRole["ranger"].Sources)
	assert.Equal(t, 1, byRole["scout"].Unknown)
	assert.Equal(t, []string{"archivist", "ranger", "scout"}, []string{report.Roles[0].Role, report.Roles[1].Role, report.Roles[2].Role}, "roles are sorted for stable output")
}

func TestSummarizeEmptyLedger(t *testing.T) {
	report := leveling.Summarize(nil)
	assert.Zero(t, report.Records)
	assert.Empty(t, report.Roles)
}

func TestRotateLedgerKeepsLatestTuplePerMissionRoleRun(t *testing.T) {
	path := seedLedger(t)
	for i := 0; i < 6; i++ {
		require.NoError(t, leveling.AppendRecord(path, rec("m1", "ranger", "Old", "low", "host")))
	}
	require.NoError(t, leveling.AppendRecord(path, rec("m1", "ranger", "Newest", "high", "host")))
	require.NoError(t, leveling.AppendRecord(path, rec("m1", "archivist", "Arch", "medium", "policy")))
	revision := rec("m1", "archivist", "Rev", "high", "host")
	revision.Run = "2"
	require.NoError(t, leveling.AppendRecord(path, revision))

	dropped, err := leveling.RotateLedger(path, 3)
	require.NoError(t, err)
	assert.Equal(t, 6, dropped)

	records, err := leveling.ReadRecords(path)
	require.NoError(t, err)
	require.Len(t, records, 3)
	got, ok, err := leveling.LatestRecord(path, "m1", "ranger")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "Newest", got.Model, "reuse still resolves the latest tuple after rotation")
	got, ok, err = leveling.LatestRunRecord(path, "m1", "archivist", "2")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "Rev", got.Model)
}

func TestRotateLedgerNoopWhenWithinLimitAndOnMissingFile(t *testing.T) {
	path := seedLedger(t, rec("m1", "ranger", "A", "low", "host"), rec("m1", "ranger", "B", "low", "host"))
	dropped, err := leveling.RotateLedger(path, 10)
	require.NoError(t, err)
	assert.Zero(t, dropped)
	before, err := os.ReadFile(path) //nolint:gosec // test temp file
	require.NoError(t, err)
	assert.Equal(t, 2, strings.Count(string(before), "\n"))

	dropped, err = leveling.RotateLedger(filepath.Join(t.TempDir(), "absent.jsonl"), 1)
	require.NoError(t, err)
	assert.Zero(t, dropped)
}

func TestRotateLedgerNeverDropsBelowOneTuplePerKey(t *testing.T) {
	path := seedLedger(t, rec("m1", "ranger", "A", "low", "host"), rec("m1", "archivist", "B", "low", "host"), rec("m1", "sniper", "C", "low", "host"))
	dropped, err := leveling.RotateLedger(path, 1)
	require.NoError(t, err)
	assert.Zero(t, dropped, "the latest tuple of every key is always kept, even above the limit")
	records, err := leveling.ReadRecords(path)
	require.NoError(t, err)
	assert.Len(t, records, 3)
}

func TestRotateLedgerIfLargeOnlyRotatesPastTheSizeThreshold(t *testing.T) {
	path := seedLedger(t)
	for i := 0; i < 8; i++ {
		require.NoError(t, leveling.AppendRecord(path, rec("m1", "ranger", "Old", "low", "host")))
	}
	require.NoError(t, leveling.AppendRecord(path, rec("m1", "ranger", "Newest", "high", "host")))

	dropped, err := leveling.RotateLedgerIfLarge(path, 1<<20, 2)
	require.NoError(t, err)
	assert.Zero(t, dropped, "a small ledger is left alone without even being read")

	dropped, err = leveling.RotateLedgerIfLarge(path, 100, 2)
	require.NoError(t, err)
	assert.Equal(t, 7, dropped)

	dropped, err = leveling.RotateLedgerIfLarge(filepath.Join(t.TempDir(), "absent.jsonl"), 1, 1)
	require.NoError(t, err)
	assert.Zero(t, dropped)
}
