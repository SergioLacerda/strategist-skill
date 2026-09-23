package initiative

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeEnterRoleReusesAdviceAndKeepsLevelingSeparate(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(root, DefaultPolicy())
	require.NoError(t, err)
	input := AdviceInput{
		MissionID: "m-runtime", Role: "ranger", RunID: "run-1", Trigger: TriggerInitial,
		Observed: Observation{State: ObservationKnown, Model: "host-model", Provider: "host", Effort: EffortMedium, LevelSource: "host"},
	}

	first, reused, err := runtime.EnterRole(input)
	require.NoError(t, err)
	require.False(t, reused)
	second, reused, err := runtime.EnterRole(input)
	require.NoError(t, err)
	require.True(t, reused)
	require.Equal(t, first.AdviceID, second.AdviceID)

	records, err := ReadRecords(LedgerPath(root))
	require.NoError(t, err)
	require.Len(t, records, 1)
	_, err = os.Stat(filepath.Join(root, "memory", "initiative-records.jsonl"))
	require.NoError(t, err)
}

func TestRuntimeReevaluateSupersedesWithoutRewritingPriorAdvice(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(root, DefaultPolicy())
	require.NoError(t, err)
	first, _, err := runtime.EnterRole(AdviceInput{MissionID: "m", Role: "archivist", RunID: "r", Trigger: TriggerInitial})
	require.NoError(t, err)
	second, err := runtime.Reevaluate(AdviceInput{MissionID: "m", Role: "archivist", RunID: "r", Trigger: TriggerHandoffChallenged})
	require.NoError(t, err)
	require.NotEqual(t, first.AdviceID, second.AdviceID)
	require.Equal(t, first.AdviceID, second.Supersedes)

	records, err := ReadRecords(LedgerPath(root))
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Equal(t, first.AdviceID, records[0].Advice.AdviceID)
	require.Equal(t, first.AdviceID, records[1].Supersedes)
}

func TestRuntimeRecordResultReturnsAdvisoryChallenge(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(root, DefaultPolicy())
	require.NoError(t, err)
	advice, _, err := runtime.EnterRole(AdviceInput{MissionID: "m", Role: "sniper", RunID: "r", Trigger: TriggerInitial})
	require.NoError(t, err)
	assessment, err := runtime.RecordResult(advice, Result{
		AdviceID: advice.AdviceID, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID,
		GateIndependent: true,
		Checks:          []ObligationCheck{{ID: "verify_scope", Status: CheckBlocked}},
	})
	require.NoError(t, err)
	require.True(t, assessment.Challenge)
	require.Contains(t, assessment.Reasons, "blocked_obligation:verify_scope")
	require.Contains(t, assessment.Reasons, "missing_result_evidence")

	records, err := ReadRecords(LedgerPath(root))
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Equal(t, RecordKindAdvice, records[0].Kind)
	require.Equal(t, RecordKindResult, records[1].Kind)
}

func TestRuntimeRejectsStaleAdviceAndSupersessionMismatch(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(root, DefaultPolicy())
	require.NoError(t, err)
	first, _, err := runtime.EnterRole(AdviceInput{MissionID: "m", Role: "ranger", RunID: "r", Trigger: TriggerInitial})
	require.NoError(t, err)
	_, err = runtime.Reevaluate(AdviceInput{MissionID: "m", Role: "ranger", RunID: "r", Trigger: TriggerHandoffChallenged, Supersedes: "other"})
	require.ErrorContains(t, err, "does not match latest advice")
	second, err := runtime.Reevaluate(AdviceInput{MissionID: "m", Role: "ranger", RunID: "r", Trigger: TriggerHandoffChallenged})
	require.NoError(t, err)
	_, err = runtime.RecordResult(first, validResultForAdvice(first))
	require.ErrorContains(t, err, "does not match persisted latest advice")
	_, err = runtime.RecordResult(second, validResultForAdvice(second))
	require.NoError(t, err)

	records, err := ReadRecords(LedgerPath(root))
	require.NoError(t, err)
	require.Len(t, records, 3)
}

func TestRuntimeConcurrentInitialStartsConverge(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(root, DefaultPolicy())
	require.NoError(t, err)
	const callers = 12
	ids := make(chan string, callers)
	errs := make(chan error, callers)
	var group sync.WaitGroup
	for range callers {
		group.Add(1)
		go func() {
			defer group.Done()
			advice, _, callErr := runtime.EnterRole(AdviceInput{MissionID: "m-concurrent", Role: "ranger", RunID: "run", Trigger: TriggerInitial})
			if callErr != nil {
				errs <- callErr
				return
			}
			ids <- advice.AdviceID
		}()
	}
	group.Wait()
	close(ids)
	close(errs)
	for callErr := range errs {
		require.NoError(t, callErr)
	}
	first := <-ids
	for id := range ids {
		require.Equal(t, first, id)
	}
	records, err := ReadRecords(LedgerPath(root))
	require.NoError(t, err)
	require.Len(t, records, 1)
}

func TestRuntimeConcurrentReevaluationsKeepSupersessionLineage(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(root, DefaultPolicy())
	require.NoError(t, err)
	_, _, err = runtime.EnterRole(AdviceInput{MissionID: "m-revisions", Role: "ranger", RunID: "run", Trigger: TriggerInitial})
	require.NoError(t, err)
	const callers = 8
	adviceIDs := make(chan string, callers)
	errs := make(chan error, callers)
	var group sync.WaitGroup
	for range callers {
		group.Add(1)
		go func() {
			defer group.Done()
			advice, callErr := runtime.Reevaluate(AdviceInput{MissionID: "m-revisions", Role: "ranger", RunID: "run", Trigger: TriggerConflictingEvidence})
			if callErr != nil {
				errs <- callErr
				return
			}
			adviceIDs <- advice.AdviceID
		}()
	}
	group.Wait()
	close(adviceIDs)
	close(errs)
	for callErr := range errs {
		require.NoError(t, callErr)
	}
	records, err := ReadRecords(LedgerPath(root))
	require.NoError(t, err)
	require.Len(t, records, callers+1)
	seen := map[string]bool{records[0].AdviceID: true}
	for _, record := range records[1:] {
		require.True(t, seen[record.Supersedes], "supersession must point to a committed predecessor")
		require.False(t, seen[record.AdviceID], "advice ids must remain unique")
		seen[record.AdviceID] = true
	}
}

func validResultForAdvice(advice Advice) Result {
	return Result{
		AdviceID: advice.AdviceID, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID,
		GateIndependent: true,
		Checks:          []ObligationCheck{{ID: "inspect_evidence", Status: CheckSatisfied, EvidenceRefs: []EvidenceRef{{ID: "e-1", Class: "explicit"}}}},
		EvidenceRefs:    []EvidenceRef{{ID: "e-1", Class: "explicit"}},
	}
}
