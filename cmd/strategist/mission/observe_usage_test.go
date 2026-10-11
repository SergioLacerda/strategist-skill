package mission_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mission "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const hostSecret = "PRIVATE-CONVERSATION-TEXT"

func domainStatusFor(id string) domain.MissionEngineStatus {
	return domain.MissionEngineStatus{MissionID: id, Phase: domain.PhaseRefinement, State: domain.StateRefinement}
}

func observeDeps(artifactDir string) mission.ReportUsageDependencies {
	return mission.ReportUsageDependencies{
		RootFlag:        cliutil.FlagRoot,
		ResolveBasePath: cliutil.ResolveActiveBasePath,
		MissionKnown: func(_, id string) bool {
			_, err := os.Stat(filepath.Join(artifactDir, id))
			return err == nil
		},
	}
}

func observeFixture(t *testing.T) (root, artifacts string) {
	t.Helper()
	root = setupViewRoot(t, domainStatusFor("m-obs"))
	artifacts = filepath.Join(filepath.Dir(root), ".analysis", "refined")
	require.NoError(t, os.MkdirAll(filepath.Join(artifacts, "m-obs"), 0o755))
	return root, artifacts
}

func claudeLine(id, ts string, in, out, cacheRead int) string {
	return fmt.Sprintf(`{"type":"assistant","timestamp":%q,"sessionId":"sess-9","isSidechain":false,`+
		`"message":{"id":%q,"model":"model-a","content":[{"type":"text","text":%q}],`+
		`"usage":{"input_tokens":%d,"output_tokens":%d,"cache_read_input_tokens":%d,"cache_creation_input_tokens":0}}}`,
		ts, id, hostSecret, in, out, cacheRead)
}

func writeLines(t *testing.T, path string, lines ...string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600))
}

func observeArgs(root, transcript string, extra ...string) []string {
	return append([]string{"--root", root, "--mission-id", "m-obs", "--host", "claude", "--transcript", transcript,
		"--since", "2026-10-10T10:00:00Z", "--until", "2026-10-10T11:00:00Z", "--phase", "discovery", "--role", "ranger"}, extra...)
}

func TestObserveUsage_RecordsAHostMeasuredWindowFromAClaudeTranscript(t *testing.T) {
	root, artifacts := observeFixture(t)
	transcript := filepath.Join(t.TempDir(), "sess.jsonl")
	writeLines(t, transcript,
		claudeLine("msg_1", "2026-10-10T10:05:00Z", 10, 500, 1000),
		claudeLine("msg_1", "2026-10-10T10:05:01Z", 10, 500, 1000), // content-block repeat
		claudeLine("msg_2", "2026-10-10T10:30:00Z", 5, 100, 2000),
		claudeLine("msg_3", "2026-10-10T12:00:00Z", 99, 99, 99), // outside the window
	)
	// a sub-agent file next to the transcript
	writeLines(t, filepath.Join(strings.TrimSuffix(transcript, ".jsonl"), "subagents", "agent-1.jsonl"),
		claudeLine("msg_sub", "2026-10-10T10:40:00Z", 1, 50, 0))

	out, err := execCmd(t, mission.NewObserveUsage(observeDeps(artifacts)), "", observeArgs(root, transcript)...)
	require.NoError(t, err)
	assert.Contains(t, out, "calls=3")
	assert.Contains(t, out, "tokens_out=650")
	assert.Contains(t, out, "quality=host_measured")
	assert.Contains(t, out, "recorded")
	assert.NotContains(t, out, hostSecret)

	recs, err := telemetry.ReadMissionUsageFor(telemetry.MissionUsagePath(root), "m-obs")
	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Equal(t, "host_counter", recs[0].MeasurementSource)
	assert.Equal(t, "host_measured", recs[0].MeasurementQuality)
	assert.Equal(t, "window", recs[0].UsageScope)
	assert.Equal(t, 3, recs[0].Calls)
	assert.Equal(t, "DISCOVERY", recs[0].Phase)
	assert.Equal(t, int64(3000), *recs[0].TokensCache)
	raw, err := os.ReadFile(telemetry.MissionUsagePath(root))
	require.NoError(t, err)
	assert.NotContains(t, string(raw), hostSecret, "the ledger never holds conversation content")
}

func TestObserveUsage_RerunIsADuplicateAndChangedCountsAreAConflict(t *testing.T) {
	root, artifacts := observeFixture(t)
	transcript := filepath.Join(t.TempDir(), "sess.jsonl")
	writeLines(t, transcript, claudeLine("msg_1", "2026-10-10T10:05:00Z", 10, 500, 0))
	run := func() (string, error) {
		return execCmd(t, mission.NewObserveUsage(observeDeps(artifacts)), "", observeArgs(root, transcript, "--include-subagents=false")...)
	}
	_, err := run()
	require.NoError(t, err)
	out, err := run()
	require.NoError(t, err)
	assert.Contains(t, out, "duplicate")

	// the host file grows inside the same window: same key, different counts
	writeLines(t, transcript,
		claudeLine("msg_1", "2026-10-10T10:05:00Z", 10, 500, 0),
		claudeLine("msg_2", "2026-10-10T10:10:00Z", 10, 300, 0))
	_, err = run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicts")
}

func TestObserveUsage_CodexUsesCounterDeltas(t *testing.T) {
	root, artifacts := observeFixture(t)
	rollout := filepath.Join(t.TempDir(), "rollout.jsonl")
	count := func(ts string, in, out int) string {
		return fmt.Sprintf(`{"type":"event_msg","timestamp":%q,"payload":{"type":"token_count","info":{"total_token_usage":`+
			`{"input_tokens":%d,"cached_input_tokens":0,"output_tokens":%d,"reasoning_output_tokens":0}}}}`, ts, in, out)
	}
	writeLines(t, rollout,
		`{"type":"session_meta","timestamp":"2026-10-10T09:00:00Z","payload":{"id":"thread-7"}}`,
		count("2026-10-10T09:30:00Z", 1000, 100),
		count("2026-10-10T10:30:00Z", 4000, 700))
	args := []string{"--root", root, "--mission-id", "m-obs", "--host", "codex", "--transcript", rollout,
		"--since", "2026-10-10T10:00:00Z", "--until", "2026-10-10T11:00:00Z", "--phase", "execution", "--role", "sniper"}
	out, err := execCmd(t, mission.NewObserveUsage(observeDeps(artifacts)), "", args...)
	require.NoError(t, err)
	assert.Contains(t, out, "tokens_in=3000")
	assert.Contains(t, out, "tokens_out=600")

	recs, err := telemetry.ReadMissionUsageFor(telemetry.MissionUsagePath(root), "m-obs")
	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Contains(t, recs[0].EventID, "thread-7")
	assert.Equal(t, "openai", recs[0].Provider)
}

func TestObserveUsage_DryRunWritesNothingAndEmptyWindowRecordsNothing(t *testing.T) {
	root, artifacts := observeFixture(t)
	transcript := filepath.Join(t.TempDir(), "sess.jsonl")
	writeLines(t, transcript, claudeLine("msg_1", "2026-10-10T10:05:00Z", 10, 500, 0))

	out, err := execCmd(t, mission.NewObserveUsage(observeDeps(artifacts)), "", observeArgs(root, transcript, "--dry-run")...)
	require.NoError(t, err)
	assert.Contains(t, out, "dry-run")
	recs, _ := telemetry.ReadMissionUsageFor(telemetry.MissionUsagePath(root), "m-obs")
	assert.Empty(t, recs)

	args := observeArgs(root, transcript)
	args[9], args[11] = "2026-10-10T13:00:00Z", "2026-10-10T14:00:00Z"
	_, err = execCmd(t, mission.NewObserveUsage(observeDeps(artifacts)), "", args...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing is recorded as zero")
}

func withFlag(args []string, name, value string) []string {
	out := append([]string(nil), args...)
	for i := 0; i+1 < len(out); i++ {
		if out[i] == name {
			out[i+1] = value
			return out
		}
	}
	return append(out, name, value)
}

func TestObserveUsage_RejectsBadInput(t *testing.T) {
	root, artifacts := observeFixture(t)
	transcript := filepath.Join(t.TempDir(), "sess.jsonl")
	writeLines(t, transcript, claudeLine("msg_1", "2026-10-10T10:05:00Z", 10, 500, 0))
	future := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	base := observeArgs(root, transcript)

	cases := map[string][]string{
		"future window":   withFlag(base, "--until", future),
		"bad time":        withFlag(base, "--since", "yesterday"),
		"missing file":    withFlag(base, "--transcript", filepath.Join(t.TempDir(), "nope.jsonl")),
		"unknown host":    withFlag(base, "--host", "gemini"),
		"gate phase":      withFlag(base, "--phase", "approval_gate"),
		"unknown mission": withFlag(base, "--mission-id", "m-missing"),
		"inverted window": withFlag(withFlag(base, "--since", "2026-10-10T12:00:00Z"), "--until", "2026-10-10T11:00:00Z"),
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := execCmd(t, mission.NewObserveUsage(observeDeps(artifacts)), "", args...)
			require.Error(t, err)
		})
	}
}

func TestObserveUsage_UnrecognizedFormatFailsClosed(t *testing.T) {
	root, artifacts := observeFixture(t)
	transcript := filepath.Join(t.TempDir(), "sess.jsonl")
	writeLines(t, transcript, `{"type":"assistant","timestamp":"2026-10-10T10:05:00Z","message":{"id":"m","usage":{"tokens":5}}}`)
	_, err := execCmd(t, mission.NewObserveUsage(observeDeps(artifacts)), "", observeArgs(root, transcript)...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "recognized shape")
}

func TestObserveUsage_CodexNeedsExactlyOneFile(t *testing.T) {
	root, artifacts := observeFixture(t)
	args := []string{"--root", root, "--mission-id", "m-obs", "--host", "codex", "--transcript", "a.jsonl", "--transcript", "b.jsonl",
		"--since", "2026-10-10T10:00:00Z", "--until", "2026-10-10T11:00:00Z", "--phase", "execution", "--role", "sniper"}
	_, err := execCmd(t, mission.NewObserveUsage(observeDeps(artifacts)), "", args...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one")
}
