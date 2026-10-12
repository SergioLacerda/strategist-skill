package mission_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	mission "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionview "github.com/SergioLacerda/strategist-skill/internal/mission/view"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedMarkers struct {
	calls    [][]domain.LifecycleMarker
	failWith error
	warnings []string
}

func (r *recordedMarkers) attach(deps *mission.LifecycleDependencies) {
	deps.RecordLifecycle = func(_, _ string, markers []domain.LifecycleMarker) error {
		r.calls = append(r.calls, markers)
		return r.failWith
	}
	deps.LifecycleWarn = func(missionID string, err error) {
		r.warnings = append(r.warnings, missionID+": "+err.Error())
	}
}

func execCmd(t *testing.T, cmd *cobra.Command, stdin string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func kinds(markers []domain.LifecycleMarker) []domain.LifecycleMarkerKind {
	var out []domain.LifecycleMarkerKind
	for _, m := range markers {
		out = append(out, m.Kind)
	}
	return out
}

func TestLifecycleMarkers_StartRecordsMissionAndFirstPhase(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	deps := lifecycleDeps(t)
	var rec recordedMarkers
	rec.attach(&deps)

	_, err := execCmd(t, mission.NewStart(deps), "", "--root", root, "--mission-id", "m-life")
	require.NoError(t, err)

	require.Len(t, rec.calls, 1)
	assert.Equal(t, []domain.LifecycleMarkerKind{domain.MarkerMissionStarted, domain.MarkerPhaseStarted}, kinds(rec.calls[0]))
}

func TestLifecycleMarkers_SubmitRecordsTheTransitionAfterPersistence(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{MissionID: "m-life", Phase: domain.PhaseBootstrap, State: domain.StateInit})
	deps := lifecycleDeps(t)
	var rec recordedMarkers
	rec.attach(&deps)
	deps.RecordLifecycle = func(gotRoot, id string, markers []domain.LifecycleMarker) error {
		// the state file already reflects the new phase when markers are written
		_, status, err := testLoadMission(gotRoot, id)
		require.NoError(t, err)
		assert.Equal(t, domain.PhaseIntake, status.Phase)
		rec.calls = append(rec.calls, markers)
		return nil
	}

	_, err := execCmd(t, mission.NewSubmit(deps), "", "--root", root, "--mission-id", "m-life", "--event", string(domain.MissionEventBootstrapDone))
	require.NoError(t, err)

	require.Len(t, rec.calls, 1)
	assert.Equal(t, []domain.LifecycleMarkerKind{domain.MarkerPhaseEnded, domain.MarkerPhaseStarted}, kinds(rec.calls[0]))
	assert.Equal(t, domain.PhaseBootstrap, rec.calls[0][0].Phase)
	assert.Equal(t, domain.PhaseIntake, rec.calls[0][1].Phase)
}

func TestLifecycleMarkers_RejectedSubmitWritesNothing(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{MissionID: "m-life", Phase: domain.PhaseBootstrap, State: domain.StateInit})
	deps := lifecycleDeps(t)
	var rec recordedMarkers
	rec.attach(&deps)

	_, err := execCmd(t, mission.NewSubmit(deps), "", "--root", root, "--mission-id", "m-life", "--event", string(domain.MissionEventSniperDone))
	require.Error(t, err)
	assert.Empty(t, rec.calls)
}

func TestLifecycleMarkers_WriteFailureDoesNotFailTheSubmitAndIsObservable(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{MissionID: "m-life", Phase: domain.PhaseBootstrap, State: domain.StateInit})
	deps := lifecycleDeps(t)
	rec := recordedMarkers{failWith: errors.New("ledger not writable")}
	rec.attach(&deps)

	out, err := execCmd(t, mission.NewSubmit(deps), "", "--root", root, "--mission-id", "m-life", "--event", string(domain.MissionEventBootstrapDone))
	require.NoError(t, err)
	assert.Equal(t, domain.PhaseIntake, decodeStatus(t, out).Phase)
	require.Len(t, rec.warnings, 1)
	assert.Contains(t, rec.warnings[0], "m-life")
	assert.Contains(t, rec.warnings[0], "ledger not writable")
}

func TestLifecycleMarkers_NilPortsKeepTheSubmitWorking(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{MissionID: "m-life", Phase: domain.PhaseBootstrap, State: domain.StateInit})
	_, err := execCmd(t, mission.NewSubmit(lifecycleDeps(t)), "", "--root", root, "--mission-id", "m-life", "--event", string(domain.MissionEventBootstrapDone))
	require.NoError(t, err)
}

func TestLifecycleMarkers_CriticalHitActivationRecordsTheJump(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{MissionID: "m-ch", Phase: domain.PhaseIntake, State: domain.StateInit})
	deps := lifecycleDeps(t)
	var rec recordedMarkers
	rec.attach(&deps)
	deps.RecordRoute = func(context.Context, string, string, []byte) (bool, error) { return true, nil }
	deps.ActivateCriticalHit = func(string, string) (domain.MissionEngineStatus, error) {
		return domain.MissionEngineStatus{MissionID: "m-ch", Phase: domain.PhaseApprovalGate, State: domain.StateApprovalGate, Stage: domain.StageShort}, nil
	}

	decision := `{"selected_route":"critical_hit"}`
	_, err := execCmd(t, mission.NewRoute(deps), decision, "--root", root, "--mission-id", "m-ch")
	require.NoError(t, err)

	require.Len(t, rec.calls, 1)
	assert.Equal(t, domain.PhaseIntake, rec.calls[0][0].Phase)
	assert.Equal(t, domain.MarkerPhaseEnded, rec.calls[0][0].Kind)
	assert.Equal(t, domain.PhaseApprovalGate, rec.calls[0][1].Phase)
}

func usageDeps() mission.ViewDependencies {
	return testDeps()
}

func writeLifecycleLedger(t *testing.T, root string, recs ...telemetry.MissionLifecycleRecord) {
	t.Helper()
	_, err := telemetry.AppendMissionLifecycleRecords(telemetry.MissionLifecyclePath(root), recs)
	require.NoError(t, err)
}

func lcRec(id, kind, phase, run, reason, at string) telemetry.MissionLifecycleRecord {
	return telemetry.MissionLifecycleRecord{
		SchemaVersion: telemetry.MissionLifecycleSchemaVersion, MarkerID: id, MarkerKind: kind,
		MissionID: "m-usage", Phase: phase, PhaseRunID: run, EndReason: reason, ObservedAt: at,
	}
}

func TestRunUsage_JSONAndViewShareTheSameTimeline(t *testing.T) {
	status := domain.MissionEngineStatus{MissionID: "m-usage", Phase: domain.PhaseRefinement, State: domain.StateRefinement}
	root := setupViewRoot(t, status)
	writeLifecycleLedger(t, root,
		lcRec("1", "mission_started", "BOOTSTRAP", "", "", "2026-10-10T10:00:00Z"),
		lcRec("2", "phase_started", "REFINEMENT", "r1", "", "2026-10-10T10:24:00Z"),
	)

	usageOut, err := execCmd(t, mission.NewUsage(usageDeps()), "", "--root", root, "--mission-id", "m-usage", "--json")
	require.NoError(t, err)
	viewOut, err := execCmd(t, mission.NewView(testDeps()), "", "--root", root, "--mission-id", "m-usage", "--json")
	require.NoError(t, err)

	var usage struct {
		Schema        string          `json:"schema"`
		PhaseTimeline json.RawMessage `json:"phase_timeline"`
	}
	var view struct {
		PhaseTimeline json.RawMessage `json:"phase_timeline"`
	}
	require.NoError(t, json.Unmarshal([]byte(usageOut), &usage))
	require.NoError(t, json.Unmarshal([]byte(viewOut), &view))
	assert.Equal(t, missionview.UsageSchemaVersion, usage.Schema)
	assert.JSONEq(t, string(view.PhaseTimeline), string(usage.PhaseTimeline), "shared section must be equal")
	assert.Contains(t, string(usage.PhaseTimeline), `"status":"running"`)
}

func TestRunUsage_HumanDetailsAndNoLedger(t *testing.T) {
	status := domain.MissionEngineStatus{MissionID: "m-usage", Phase: domain.PhaseRefinement, State: domain.StateRefinement}
	root := setupViewRoot(t, status)

	out, err := execCmd(t, mission.NewUsage(usageDeps()), "", "--root", root, "--mission-id", "m-usage")
	require.NoError(t, err)
	assert.Contains(t, out, "no lifecycle recorded")
	assert.Contains(t, out, "MISSION TOTAL")

	writeLifecycleLedger(t, root, lcRec("1", "phase_started", "REFINEMENT", "r1", "", "2026-10-10T10:24:00Z"))
	out, err = execCmd(t, mission.NewUsage(usageDeps()), "", "--root", root, "--mission-id", "m-usage", "--details")
	require.NoError(t, err)
	assert.Contains(t, out, "run REFINEMENT r1")
}

func TestRunUsage_RejectsUnknownMissionAndMissingID(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	_, err := execCmd(t, mission.NewUsage(usageDeps()), "", "--root", root, "--mission-id", "nope")
	require.Error(t, err)
	_, err = execCmd(t, mission.NewUsage(usageDeps()), "", "--root", root)
	require.Error(t, err)
}

func TestRunUsage_ShowsLegacyLedgerOnItsOwnRow(t *testing.T) {
	status := domain.MissionEngineStatus{MissionID: "m-usage", Phase: domain.PhaseRefinement, State: domain.StateRefinement}
	root := setupViewRoot(t, status)
	require.NoError(t, telemetry.AppendMissionTokenUsage(telemetry.MissionTokenUsageHistoryPath(root),
		telemetry.MissionTokenUsageRecord{MissionID: "m-usage", TokensIn: 10, TokensOut: 2, Source: "agent_report", ReportedAt: "2026-10-10T10:00:00Z"}))

	out, err := execCmd(t, mission.NewUsage(usageDeps()), "", "--root", root, "--mission-id", "m-usage", "--json")
	require.NoError(t, err)
	var v missionview.UsageView
	require.NoError(t, json.Unmarshal([]byte(out), &v))
	assert.Equal(t, 1, v.Legacy.RecordCount)
	assert.Nil(t, v.PhaseTimeline.Total.Tokens.Total)
}

func TestRunUsage_TimelineAddsTheMergedChronology(t *testing.T) {
	status := domain.MissionEngineStatus{MissionID: "m-usage", Phase: domain.PhaseRefinement, State: domain.StateRefinement}
	root := setupViewRoot(t, status)
	writeLifecycleLedger(t, root, lcRec("1", "phase_started", "REFINEMENT", "r1", "", "2026-10-10T10:24:00Z"))
	in, out := int64(1000), int64(200)
	_, err := telemetry.AppendMissionUsageRecord(telemetry.MissionUsagePath(root), telemetry.MissionUsageRecord{
		SchemaVersion: telemetry.MissionUsageSchemaVersion, RecordID: "inv|m-usage|inv_1|1|invocation", MissionID: "m-usage",
		Phase: "REFINEMENT", Role: "archivist", InvocationID: "inv_1", Attempt: 1, UsageScope: "invocation",
		RecordKind: "incremental", TokensInput: &in, TokensOutput: &out,
		MeasurementSource: "agent_report", MeasurementQuality: "reported_unverified", ObservedAt: "2026-10-10T10:30:00Z",
	})
	require.NoError(t, err)

	text, err := execCmd(t, mission.NewUsage(usageDeps()), "", "--root", root, "--mission-id", "m-usage", "--timeline")
	require.NoError(t, err)
	assert.Contains(t, text, "Timeline")
	assert.Contains(t, text, "usage_observation")
	assert.Contains(t, text, "MISSION TOTAL   in=1000 out=200 total=1200")

	jsonOut, err := execCmd(t, mission.NewUsage(usageDeps()), "", "--root", root, "--mission-id", "m-usage", "--json")
	require.NoError(t, err)
	var v missionview.UsageView
	require.NoError(t, json.Unmarshal([]byte(jsonOut), &v))
	assert.Empty(t, v.Chronology, "the chronology is only included on request")
	assert.Equal(t, int64(1200), *v.PhaseTimeline.Total.Tokens.Total)
}
