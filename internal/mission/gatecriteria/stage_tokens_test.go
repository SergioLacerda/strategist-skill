package gatecriteria

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tok(v int64) *int64 { return &v }

func usageRec(id, phase, inv string, in, out *int64) telemetry.MissionUsageRecord {
	return telemetry.MissionUsageRecord{
		SchemaVersion: telemetry.MissionUsageSchemaVersion, RecordID: id, MissionID: "m-1", Phase: phase, Role: "ranger",
		InvocationID: inv, Attempt: 1, UsageScope: "invocation", RecordKind: "incremental",
		TokensInput: in, TokensOutput: out, MeasurementSource: "agent_report", MeasurementQuality: "reported_unverified",
		ObservedAt: "2026-10-10T10:00:00Z",
	}
}

func seed(t *testing.T, recs ...telemetry.MissionUsageRecord) Input {
	t.Helper()
	root := t.TempDir()
	for _, r := range recs {
		_, err := telemetry.AppendMissionUsageRecord(filepath.Join(root, "memory", "mission-usage.jsonl"), r)
		require.NoError(t, err)
	}
	return Input{Root: root, BasePath: filepath.Join(root, ".analysis"), MissionID: "m-1"}
}

func stageByName(t *testing.T, st StageTokens, name string) StageUsage {
	t.Helper()
	for _, s := range st.Stages {
		if s.Stage == name {
			return s
		}
	}
	require.Failf(t, "stage not found", name)
	return StageUsage{}
}

func TestTokenStagesReportsDiscoveryRefiningAndAPendingDelivery(t *testing.T) {
	in := seed(t,
		usageRec("a", "DISCOVERY", "inv_1", tok(100), tok(11)),
		usageRec("b", "REFINEMENT", "inv_2", tok(200), tok(22)),
		usageRec("c", "REFINEMENT", "inv_3", tok(0), tok(0)),
	)
	st := TokenStages(in)

	assert.Equal(t, "available", st.Availability)
	require.Len(t, st.Stages, 3, "intake is shown only when it has observations")
	assert.Equal(t, int64(111), *stageByName(t, st, "discovery").Total)
	assert.Equal(t, int64(222), *stageByName(t, st, "refining").Total)
	delivering := stageByName(t, st, "delivering")
	assert.Nil(t, delivering.Total)
	assert.True(t, delivering.Pending, "delivery has not run at the gate")
	require.NotNil(t, st.Total.Tokens.Total)
	assert.Equal(t, int64(333), *st.Total.Tokens.Total)
	assert.Equal(t, "reported_unverified", st.Total.Quality)
}

func TestTokenStagesRenderingFollowsTheRequestedLayout(t *testing.T) {
	in := seed(t,
		usageRec("a", "DISCOVERY", "inv_1", tok(100), tok(11)),
		usageRec("b", "REFINEMENT", "inv_2", tok(200), tok(22)),
	)
	text := RenderTokenStages(TokenStages(in))

	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, "tokens estimados:", lines[0])
	assert.Contains(t, lines[1], "discovery")
	assert.Contains(t, lines[1], "(111)")
	assert.Contains(t, lines[1], "-> refining")
	assert.Contains(t, lines[1], "(222)")
	assert.Contains(t, lines[1], "-> delivering")
	assert.Contains(t, lines[1], "(—)")
	assert.Contains(t, lines[2], "total")
	assert.Contains(t, lines[2], "(333)")
	assert.Contains(t, lines[2], "reported_unverified")
}

func TestTokenStagesShowsIntakeAndUnattributedWhenObserved(t *testing.T) {
	in := seed(t,
		usageRec("i", "INTAKE", "inv_0", tok(9), tok(1)),
		usageRec("b", "BOOTSTRAP", "inv_b", tok(5), tok(0)),
		usageRec("d", "DISCOVERY", "inv_1", tok(100), tok(11)),
		usageRec("u", "UNATTRIBUTED", "inv_u", tok(40), tok(4)),
	)
	st := TokenStages(in)
	assert.Equal(t, int64(15), *stageByName(t, st, "intake").Total, "bootstrap and intake form one stage")
	assert.Equal(t, int64(44), *st.Unattributed.Tokens.Total)
	assert.Equal(t, int64(15+111+44), *st.Total.Tokens.Total, "unattributed stays in the total")

	text := RenderTokenStages(st)
	assert.Contains(t, text, "intake")
	assert.Contains(t, text, "não atribuído (44)")
}

func TestTokenStagesNeverInventsNumbers(t *testing.T) {
	st := TokenStages(seed(t))
	assert.Equal(t, "not_applicable", st.Availability)
	assert.Nil(t, st.Total.Tokens.Total)
	text := RenderTokenStages(st)
	assert.Contains(t, text, "discovery")
	assert.Contains(t, text, "total")
	assert.Equal(t, 0, strings.Count(text, "(0)"), "an unobserved stage is a dash, never zero")
	assert.Contains(t, text, "nenhum uso reportado")
}

func TestTokenStagesPartialStageIsALowerBoundNotATotal(t *testing.T) {
	in := seed(t, usageRec("a", "DISCOVERY", "inv_1", tok(100), nil))
	text := RenderTokenStages(TokenStages(in))
	assert.Contains(t, text, "(≥100)")
	assert.NotContains(t, text, "(100)")
}

func TestTokenStagesRowIsEvaluatedOnlyWithObservations(t *testing.T) {
	withUsage := Build(seed(t, usageRec("a", "DISCOVERY", "inv_1", tok(100), tok(11))))
	row := rowByCriterion(t, withUsage, "token_usage.by_stage")
	assert.Equal(t, StateEvaluated, row.State)
	assert.Contains(t, row.Value, "discovery 111")
	assert.Contains(t, row.Value, "total 111")
	assert.NotEmpty(t, withUsage.TokenStages.Stages)

	empty := Build(seed(t))
	assert.Equal(t, StateNotEvaluated, rowByCriterion(t, empty, "token_usage.by_stage").State)
}

func TestTokenStagesUnreadableLedgerIsUnknownNotZero(t *testing.T) {
	root := t.TempDir()
	// the memory path exists as a directory, so the ledger cannot be read as a file
	require.NoError(t, osMkdir(filepath.Join(root, "memory", "mission-usage.jsonl")))
	rep := Build(Input{Root: root, MissionID: "m-1"})
	assert.Equal(t, StateUnknown, rowByCriterion(t, rep, "token_usage.by_stage").State)
	assert.Equal(t, "unavailable", rep.TokenStages.Availability)
}

func rowByCriterion(t *testing.T, r Report, criterion string) Row {
	t.Helper()
	for _, row := range r.Rows {
		if row.Criterion == criterion {
			return row
		}
	}
	require.Failf(t, "row not found", criterion)
	return Row{}
}

func osMkdir(path string) error { return os.MkdirAll(path, 0o755) }
