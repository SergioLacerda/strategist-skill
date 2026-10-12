package mission_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	mission "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	telemetryverify "github.com/SergioLacerda/strategist-skill/internal/telemetry/verify"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedEvidence(t *testing.T, root string) {
	t.Helper()
	writeLifecycleLedger(t, root,
		lcRec("1", "mission_started", "BOOTSTRAP", "", "", "2026-10-10T10:00:00Z"),
		lcRec("2", "phase_started", "REFINEMENT", "r1", "", "2026-10-10T10:24:00Z"),
	)
	in, out := int64(1000), int64(200)
	_, err := telemetry.AppendMissionUsageRecord(telemetry.MissionUsagePath(root), telemetry.MissionUsageRecord{
		SchemaVersion: telemetry.MissionUsageSchemaVersion, RecordID: "inv|m-usage|inv_1|1|invocation", MissionID: "m-usage",
		Phase: "REFINEMENT", Role: "archivist", InvocationID: "inv_1", Attempt: 1, UsageScope: "invocation",
		RecordKind: "incremental", TokensInput: &in, TokensOutput: &out,
		MeasurementSource: "agent_report", MeasurementQuality: "reported_unverified", ObservedAt: "2026-10-10T10:30:00Z",
	})
	require.NoError(t, err)
}

func exportEvents(t *testing.T, root, path string) {
	t.Helper()
	lifecycle, err := telemetry.ReadMissionLifecycleFor(telemetry.MissionLifecyclePath(root), "m-usage")
	require.NoError(t, err)
	for _, r := range lifecycle {
		e, err := telemetry.NewMissionLifecycleEvent(r)
		require.NoError(t, err)
		require.NoError(t, telemetry.AppendEventLine(path, e))
	}
	usage, err := telemetry.ReadMissionUsageFor(telemetry.MissionUsagePath(root), "m-usage")
	require.NoError(t, err)
	for _, r := range usage {
		require.NoError(t, telemetry.AppendEventLine(path, telemetry.NewMissionUsageEvent(r)))
	}
}

func usageMissionRoot(t *testing.T) string {
	t.Helper()
	return setupViewRoot(t, domainStatusFor("m-usage"))
}

func TestVerifyTelemetry_HealthyEvidenceWithItsEventStream(t *testing.T) {
	root := usageMissionRoot(t)
	seedEvidence(t, root)
	events := filepath.Join(t.TempDir(), "events.jsonl")
	exportEvents(t, root, events)

	out, err := execCmd(t, mission.NewVerifyTelemetry(usageDeps()), "", "--root", root, "--mission-id", "m-usage", "--events", events, "--json")
	require.NoError(t, err)
	var v struct {
		Schema string `json:"schema"`
		telemetryverify.Report
	}
	require.NoError(t, json.Unmarshal([]byte(out), &v))
	assert.Equal(t, "strategist-telemetry-verify/v1", v.Schema)
	assert.True(t, v.OK)
	assert.Equal(t, 3, v.Checked.Events)
	assert.Empty(t, v.Findings)
}

func TestVerifyTelemetry_CorruptedEvidenceExitsNonZeroAndNamesTheFindings(t *testing.T) {
	root := usageMissionRoot(t)
	seedEvidence(t, root)
	f, err := os.OpenFile(telemetry.MissionLifecyclePath(root), os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = f.WriteString("{not json}\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	// a stream with a usage event whose counts differ from the ledger
	events := filepath.Join(t.TempDir(), "events.jsonl")
	exportEvents(t, root, events)
	bad := int64(31337)
	require.NoError(t, telemetry.AppendEventLine(events, telemetry.NewMissionUsageEvent(telemetry.MissionUsageRecord{
		SchemaVersion: "1", RecordID: "inv|m-usage|inv_1|1|invocation", MissionID: "m-usage", Phase: "REFINEMENT", Role: "archivist",
		Attempt: 1, UsageScope: "invocation", RecordKind: "incremental", TokensInput: &bad,
		MeasurementSource: "agent_report", MeasurementQuality: "reported_unverified", ObservedAt: "2026-10-10T10:31:00Z",
	})))

	out, err := execCmd(t, mission.NewVerifyTelemetry(usageDeps()), "", "--root", root, "--mission-id", "m-usage", "--events", events)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "integrity error")
	assert.Contains(t, out, "unreadable_lines")
	assert.Contains(t, out, "event_conflicting_duplicate")
}

func TestVerifyTelemetry_LedgersOnlyAndUnknownMission(t *testing.T) {
	root := usageMissionRoot(t)
	out, err := execCmd(t, mission.NewVerifyTelemetry(usageDeps()), "", "--root", root, "--mission-id", "m-usage")
	require.NoError(t, err)
	assert.Contains(t, out, "ok=true")

	_, err = execCmd(t, mission.NewVerifyTelemetry(usageDeps()), "", "--root", root, "--mission-id", "nope")
	require.Error(t, err)
	_, err = execCmd(t, mission.NewVerifyTelemetry(usageDeps()), "", "--root", root)
	require.Error(t, err)
}

func TestUsageFromEvents_ReconstructsTheSameReportAsTheLedgers(t *testing.T) {
	root := usageMissionRoot(t)
	seedEvidence(t, root)
	events := filepath.Join(t.TempDir(), "events.jsonl")
	exportEvents(t, root, events)

	timeline := func(extra ...string) json.RawMessage {
		out, err := execCmd(t, mission.NewUsage(usageDeps()), "", append([]string{"--root", root, "--mission-id", "m-usage", "--json"}, extra...)...)
		require.NoError(t, err)
		var v struct {
			Source        string          `json:"source"`
			PhaseTimeline json.RawMessage `json:"phase_timeline"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &v))
		if len(extra) > 0 {
			assert.Equal(t, "events", v.Source)
		} else {
			assert.Equal(t, "ledgers", v.Source)
		}
		return v.PhaseTimeline
	}
	assert.JSONEq(t, string(timeline()), string(timeline("--from-events", events)))
}

func TestUsageFromEvents_ReportsAnUnreadableStreamAsADiagnostic(t *testing.T) {
	root := usageMissionRoot(t)
	events := filepath.Join(t.TempDir(), "events.jsonl")
	require.NoError(t, os.WriteFile(events, []byte("{broken\n"), 0o600))
	out, err := execCmd(t, mission.NewUsage(usageDeps()), "", "--root", root, "--mission-id", "m-usage", "--from-events", events, "--json")
	require.NoError(t, err)
	assert.Contains(t, out, "mission_view_usage_unavailable")
}

type failingSink struct{ err error }

func (f failingSink) Emit(context.Context, telemetry.Event) error { return f.err }

type captureSink struct{ events []telemetry.Event }

func (c *captureSink) Emit(_ context.Context, e telemetry.Event) error {
	c.events = append(c.events, e)
	return nil
}

func TestReportUsage_ExportsOneEventPerNewRecordAndNoneForADuplicate(t *testing.T) {
	root, artifacts := observeFixture(t)
	sink := &captureSink{}
	deps := observeDeps(artifacts)
	deps.Validate = func(_ *cobra.Command, _ mission.ReportUsageOptions) error { return nil }
	deps.TelemetrySink = func() telemetry.EventSink { return sink }
	args := []string{"--root", root, "--mission-id", "m-obs", "--phase", "discovery", "--role", "ranger",
		"--invocation-id", "inv_x", "--tokens-in", "10", "--tokens-out", "5"}

	for i := 0; i < 2; i++ {
		_, err := execCmd(t, mission.NewReportUsage(deps), "", args...)
		require.NoError(t, err)
	}
	require.Len(t, sink.events, 1, "the duplicate delivery is not exported twice")
	assert.Equal(t, telemetry.EventUsageRecorded, sink.events[0].Name)
	assert.Equal(t, "m-obs", sink.events[0].Attributes[telemetry.AttrCorrelationID])
}

func TestReportUsage_StrictExportFailureStillLeavesTheLedgerWritten(t *testing.T) {
	root, artifacts := observeFixture(t)
	deps := observeDeps(artifacts)
	deps.Validate = func(_ *cobra.Command, _ mission.ReportUsageOptions) error { return nil }
	deps.TelemetrySink = func() telemetry.EventSink { return failingSink{err: errors.New("collector down")} }
	args := []string{"--root", root, "--mission-id", "m-obs", "--phase", "discovery", "--role", "ranger",
		"--invocation-id", "inv_y", "--tokens-in", "10"}

	_, err := execCmd(t, mission.NewReportUsage(deps), "", args...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "telemetry export failed")
	recs, readErr := telemetry.ReadMissionUsageFor(telemetry.MissionUsagePath(root), "m-obs")
	require.NoError(t, readErr)
	assert.Len(t, recs, 1, "the ledger is the source of truth and was written")
}
