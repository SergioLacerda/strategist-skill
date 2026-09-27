package mission_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	livemission "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

func writeTasksMD(t *testing.T, basePath, missionID, body string) {
	t.Helper()
	dir := filepath.Join(basePath, "refined", missionID)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(body), 0o644))
}

func TestRecordSniperClaims_OneClaimPerDocumentationTarget(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, ".analysis")
	missionID := "m-claims"
	writeTasksMD(t, basePath, missionID,
		"- [ ] 1.1 [documentation_target] Write `docs/adr/0099-example.md` about the thing.\n"+
			"- [ ] 2.1 [implementation_handoff] Edit `internal/foo.go`.\n")

	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	n, err := livemission.RecordSniperClaims(root, basePath, missionID, now)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	records, err := telemetry.ReadRecentSniperClaims(telemetry.SniperClaimHistoryPath(root), now, telemetry.SniperClaimWindow)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, missionID, records[0].MissionID)
	require.Equal(t, "docs/adr/0099-example.md", records[0].TargetPath)
	require.Equal(t, basePath, records[0].BasePath)
}

func TestRecordSniperClaims_NoTasksFileIsNotAnError(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, ".analysis")
	n, err := livemission.RecordSniperClaims(root, basePath, "m-none", time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, n)
}

func TestRecordSniperClaims_NoDocumentationTargetsRecordsNothing(t *testing.T) {
	root := t.TempDir()
	basePath := filepath.Join(root, ".analysis")
	missionID := "m-no-targets"
	writeTasksMD(t, basePath, missionID, "- [ ] 1.1 [implementation_handoff] Edit `internal/foo.go`.\n")

	n, err := livemission.RecordSniperClaims(root, basePath, missionID, time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, n)
}

func TestReadMissionPhase(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "missions"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(root, "missions", "m-1.json"),
		[]byte(`{"mission_id":"m-1","phase":"EXECUTION","state":"EXECUTION"}`),
		0o644,
	))

	phase, found := livemission.ReadMissionPhase(root, "m-1")
	require.True(t, found)
	require.Equal(t, domain.PhaseExecution, phase)

	_, found = livemission.ReadMissionPhase(root, "missing")
	require.False(t, found)
}

func TestReadMissionPhase_UnparseableStateReportsNotFound(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "missions"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "missions", "m-bad.json"), []byte("not json"), 0o644))

	_, found := livemission.ReadMissionPhase(root, "m-bad")
	require.False(t, found)
}

func TestTerminal(t *testing.T) {
	require.True(t, livemission.Terminal(domain.PhaseDone))
	require.False(t, livemission.Terminal(domain.PhaseExecution))
	require.False(t, livemission.Terminal(domain.PhaseBlocked), "a blocked mission may still resume; its claim keeps counting")
}
