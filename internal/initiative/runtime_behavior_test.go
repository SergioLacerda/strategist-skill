package initiative

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func blockedRevision(advice Advice, sequence int, previous string) Result {
	result := validResultForAdvice(advice)
	result.Checks[0].Status = CheckBlocked
	result.ResultID, result.Sequence, result.Supersedes = fmt.Sprintf("res-%d", sequence), sequence, previous
	return result
}

func TestRuntimeBoundsEscalationAcrossResultRevisions(t *testing.T) {
	runtime := newTestRuntime(t)
	advice, _, err := runtime.EnterRole(initialInput("ranger"))
	require.NoError(t, err)
	var statuses []string
	previous := ""
	for sequence := 1; sequence <= 4; sequence++ {
		result := blockedRevision(advice, sequence, previous)
		assessment, err := runtime.RecordResult(advice, result)
		require.NoError(t, err)
		require.NotNil(t, assessment.Escalation)
		statuses = append(statuses, assessment.Escalation.Status)
		previous = result.ResultID
	}
	require.Equal(t, []string{EscalationRequested, EscalationRequested, EscalationRequested, EscalationHumanReview}, statuses)
	latest, found, err := LatestAssessment(runtime.LedgerFile, advice.MissionID, advice.Role, advice.RunID, advice.AdviceID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 4, latest.ResultSequence)
}

func TestFirstResultAssignsIdentityAndRejectsMisplacedRevisions(t *testing.T) {
	advice := validAdvice()
	filled, err := firstResult(advice, Result{})
	require.NoError(t, err)
	require.Equal(t, 1, filled.Sequence)
	require.Equal(t, resultID(advice, 1), filled.ResultID)
	_, err = firstResult(advice, Result{Sequence: 2})
	require.ErrorContains(t, err, "initiative_result_revision_conflict")
	_, err = firstResult(advice, Result{Sequence: 1, Supersedes: "prior"})
	require.ErrorContains(t, err, "without supersession")
}

func TestReevaluationRequiresAFreshValidLevelingEvent(t *testing.T) {
	runtime := newTestRuntime(t)
	first := initialInput("ranger")
	first.Leveling = &LevelingResolution{EventID: "ev-1", Role: "ranger", State: ObservationKnown}
	_, _, err := runtime.EnterRole(first)
	require.NoError(t, err)
	stale := reevalInput("ranger")
	stale.Leveling = first.Leveling
	_, err = runtime.Reevaluate(stale)
	require.ErrorContains(t, err, "new LEVELING event")
	missing := reevalInput("ranger")
	missing.Leveling = &LevelingResolution{Role: "ranger", State: ObservationKnown}
	_, err = runtime.Reevaluate(missing)
	require.ErrorContains(t, err, "event_id is required")
}

func TestLevelingResolutionValidatesRoleAndState(t *testing.T) {
	require.ErrorContains(t, LevelingResolution{Role: "sniper", State: ObservationKnown}.ValidateFor("ranger"), "does not match")
	require.ErrorContains(t, LevelingResolution{Role: "ranger", State: "bogus"}.ValidateFor("ranger"), "invalid resolution state")
	require.NoError(t, LevelingResolution{EventID: "e", Role: "ranger", State: ObservationUnknown}.ValidateForRole("ranger"))
}

func TestPolicyProfilesRejectIncompleteDiligence(t *testing.T) {
	valid := Profile{RecommendedCapability: "reasoning", RecommendedEffort: EffortHigh, Diligence: []string{"inspect"}, ConfidenceCeiling: ConfidenceHigh}
	require.NoError(t, validateProfile("ranger", valid))
	for _, tc := range []struct {
		name   string
		mutate func(*Profile)
		want   string
	}{
		{"capability", func(p *Profile) { p.RecommendedCapability = " " }, "no recommended capability"},
		{"effort", func(p *Profile) { p.RecommendedEffort = "bogus" }, "invalid recommended effort"},
		{"empty diligence", func(p *Profile) { p.Diligence = nil }, "requires diligence"},
		{"blank check", func(p *Profile) { p.Diligence = []string{" "} }, "empty diligence check"},
		{"duplicate check", func(p *Profile) { p.Diligence = []string{"a", "a"} }, "duplicate diligence check"},
		{"ceiling", func(p *Profile) { p.ConfidenceCeiling = "bogus" }, "ranger"},
	} {
		profile := valid
		tc.mutate(&profile)
		require.ErrorContains(t, validateProfile("ranger", profile), tc.want, tc.name)
	}
	require.False(t, uniqueNonEmpty([]string{"a", " "}))
	require.False(t, uniqueNonEmpty([]string{"a", "a"}))
	require.True(t, uniqueNonEmpty([]string{"a", "b"}))
}

func TestParseRejectsExtraYAMLDocuments(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "embed", "defaults", "initiative.yaml"))
	require.NoError(t, err)
	_, err = Parse(raw)
	require.NoError(t, err)
	_, err = Parse(append(append([]byte{}, raw...), []byte("\n---\nextra: true\n")...))
	require.ErrorContains(t, err, "multiple YAML documents")
	_, err = Parse(append(append([]byte{}, raw...), []byte("\n---\n: : :\n")...))
	require.ErrorContains(t, err, "trailing document")
	require.NotEmpty(t, DefaultPolicy().Digest())
}

func TestRecordValidationRejectsMalformedRecords(t *testing.T) {
	advice, result, assessment := blockedAssessment(t)
	base := Record{
		Kind: RecordKindResult, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID,
		AdviceID: advice.AdviceID, Result: &result, Assessment: &assessment,
	}
	require.NoError(t, validateRecord(base))
	for _, tc := range []struct {
		name   string
		mutate func(*Record)
		want   string
	}{
		{"kind", func(r *Record) { r.Kind = "bogus" }, "unknown kind"},
		{"identity", func(r *Record) { r.RunID = "" }, "identity are required"},
		{"advice payload", func(r *Record) { r.Kind, r.Result = RecordKindAdvice, nil }, "advice record payload mismatch"},
		{"result payload", func(r *Record) { r.Result = nil }, "result record payload mismatch"},
		{"revision", func(r *Record) { r.Result = &Result{AdviceID: advice.AdviceID} }, "positive sequence"},
		{"assessment", func(r *Record) { bad := assessment; bad.AdviceID = "other"; r.Assessment = &bad }, "assessment payload mismatch"},
	} {
		record := base
		tc.mutate(&record)
		require.ErrorContains(t, validateRecord(record), tc.want, tc.name)
	}
}

func TestLedgerAppendRejectsResultsAndUnwritablePaths(t *testing.T) {
	advice, result, assessment := blockedAssessment(t)
	record := Record{
		Kind: RecordKindResult, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID,
		AdviceID: advice.AdviceID, Result: &result, Assessment: &assessment,
	}
	dir := t.TempDir()
	require.ErrorContains(t, AppendRecord(filepath.Join(dir, "l.jsonl"), record), "validated runtime")
	require.ErrorContains(t, appendRecordUnlocked(filepath.Join(dir, "l.jsonl"), Record{Kind: "bogus"}), "unknown kind")
	blocker := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))
	require.ErrorContains(t, appendRecordUnlocked(filepath.Join(blocker, "sub", "l.jsonl"), record), "create ledger directory")
}
