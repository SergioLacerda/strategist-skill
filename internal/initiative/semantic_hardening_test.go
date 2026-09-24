package initiative

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRejectsUnknownFieldsAndConfidenceTiers(t *testing.T) {
	base := "version: '1'\nprofiles:\n  ranger:\n    recommended_capability: reasoning\n    recommended_effort: high\n    diligence: [inspect]\n    confidence_ceiling: %s\nreevaluation_triggers: [scope_changed]\n"
	_, err := Parse([]byte("version: '1'\nunknown: true\n"))
	require.ErrorContains(t, err, "field unknown not found")
	_, err = Parse([]byte(fmt.Sprintf(base, "trusted")))
	require.ErrorContains(t, err, "initiative_confidence_tier_invalid")
	_, err = Parse([]byte(fmt.Sprintf(base, "HIGH")))
	require.ErrorContains(t, err, "initiative_confidence_tier_invalid")
	_, err = Parse([]byte(fmt.Sprintf(base+"---\nversion: '1'\n", "high")))
	require.ErrorContains(t, err, "multiple YAML documents")
}

func TestValidateAgainstRequiresExactObligationsAndReasons(t *testing.T) {
	advice := validAdvice()
	base := Result{AdviceID: advice.AdviceID, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID, GateIndependent: true}
	valid := base
	valid.Checks = []ObligationCheck{{ID: "inspect", Status: CheckSatisfied, EvidenceRefs: []EvidenceRef{{ID: "e-1", Class: "explicit"}}}}
	valid.EvidenceRefs = []EvidenceRef{{ID: "e-1", Class: "explicit"}}
	require.NoError(t, valid.ValidateAgainst(advice))
	for _, checks := range [][]ObligationCheck{
		{},
		{{ID: "inspect", Status: CheckSatisfied, EvidenceRefs: []EvidenceRef{{ID: "e-1", Class: "explicit"}}}, {ID: "inspect", Status: CheckSatisfied, EvidenceRefs: []EvidenceRef{{ID: "e-2", Class: "explicit"}}}},
		{{ID: "unknown", Status: CheckBlocked}},
		{{ID: "inspect", Status: CheckNotApplicable}},
	} {
		result := base
		result.Checks = checks
		result.EvidenceRefs = []EvidenceRef{{ID: "e-1", Class: "explicit"}, {ID: "e-2", Class: "explicit"}}
		require.Error(t, result.ValidateAgainst(advice))
	}
}

func TestValidateAgainstRejectsStructurallyEmptyEvidence(t *testing.T) {
	advice := validAdvice()
	result := Result{
		AdviceID: advice.AdviceID, MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID,
		GateIndependent: true,
		Checks:          []ObligationCheck{{ID: "inspect", Status: CheckSatisfied, EvidenceRefs: []EvidenceRef{{ID: "", Class: "explicit"}}}},
		EvidenceRefs:    []EvidenceRef{{ID: "", Class: "explicit"}},
	}
	require.ErrorContains(t, result.ValidateAgainst(advice), "evidence id is required")
}

func TestRuntimeRequiresExplicitResultRevisions(t *testing.T) {
	runtime, err := NewRuntime(t.TempDir(), DefaultPolicy())
	require.NoError(t, err)
	advice, _, err := runtime.EnterRole(initialInput("ranger"))
	require.NoError(t, err)
	_, err = runtime.RecordResult(advice, validResultForAdvice(advice))
	require.NoError(t, err)
	current, found, err := LatestResult(runtime.LedgerFile, advice.MissionID, advice.Role, advice.RunID, advice.AdviceID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 1, current.Sequence)
	require.NotEmpty(t, current.ResultID)

	second := validResultForAdvice(advice)
	_, err = runtime.RecordResult(advice, second)
	require.ErrorContains(t, err, "initiative_result_revision_conflict")

	second.ResultID = "res-explicit-2"
	second.Sequence = 2
	second.Supersedes = current.ResultID
	_, err = runtime.RecordResult(advice, second)
	require.NoError(t, err)
	latest, found, err := LatestResult(runtime.LedgerFile, advice.MissionID, advice.Role, advice.RunID, advice.AdviceID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, second.ResultID, latest.ResultID)
}

func TestLatestResultIgnoresOtherAdvice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.jsonl")
	result := Result{AdviceID: "a", MissionID: "m", Role: "ranger", RunID: "r", ResultID: "res-1", Sequence: 1, GateIndependent: true}
	record := Record{Kind: RecordKindResult, MissionID: "m", Role: "ranger", RunID: "r", AdviceID: "a", Result: &result}
	// The low-level record validator intentionally accepts only revision shape;
	// obligation correlation belongs to Runtime where Advice is available.
	require.NoError(t, appendRecordUnlocked(path, record))
	_, found, err := LatestResult(path, "m", "ranger", "r", "other")
	require.NoError(t, err)
	require.False(t, found)
}

func TestLedgerMigratesLegacyRecordsAndSkipsUnsupportedVersions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.jsonl")
	advice := validAdvice()
	legacy := fmt.Sprintf(`{"kind":"advice","mission_id":"%s","role":"ranger","run_id":"r","advice_id":"%s","advice":%s}`+"\n", advice.MissionID, advice.AdviceID, mustJSON(t, advice))
	unsupported := `{"schema_version":"99","kind":"advice","mission_id":"m","role":"ranger","run_id":"r","advice_id":"other"}` + "\n"
	require.NoError(t, os.WriteFile(path, []byte(legacy+unsupported), 0o600))
	records, err := ReadRecords(path)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, LedgerSchemaVersion, records[0].SchemaVersion)
}
