package metrics

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeClaimFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "claim.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

func ledgerLines(t *testing.T, root string) int {
	t.Helper()
	records, err := telemetry.ReadConfidenceRecords(telemetry.ConfidenceHistoryPath(root))
	require.NoError(t, err)
	return len(records)
}

// A flat claim file used to decode into an empty claim and fail with the
// unrelated "rejected observation requires claim_id".
func TestRecordRejectsAClaimFileWithoutTheClaimKey(t *testing.T) {
	root := testRoot(t)
	cmd, _ := testCommand("metrics")
	flat := writeClaimFile(t, "id: A-1\nstatement: s\nclaim_kind: assertion\nconfidence_percent: 90\n")

	err := RunRecord(cmd, testDependencies(), RecordOptions{Root: root, Mission: "m", Agent: "ranger", ClaimFile: flat})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown top-level key")
	assert.Contains(t, err.Error(), "`claim:`")
	assert.NotContains(t, err.Error(), "claim_id")
	assert.Zero(t, ledgerLines(t, root), "nothing is recorded for a mis-shaped file")

	onlyEvidence := writeClaimFile(t, "evidence: []\n")
	err = RunRecord(cmd, testDependencies(), RecordOptions{Root: root, Mission: "m", Agent: "ranger", ClaimFile: onlyEvidence})
	require.ErrorContains(t, err, "no `claim` mapping")
}

// A claim that decodes but fails normalization is still recorded as rejected
// for the metrics, and the command reports the real cause and fails.
func TestRecordFailsAndNamesTheCauseForARejectedClaim(t *testing.T) {
	root := testRoot(t)
	cmd, out := testCommand("metrics")
	bad := writeClaimFile(t, "claim: {id: A-1, statement: s, agent: ranger, correlation_key: k, claim_kind: assertion, confidence_percent: 90, evidence_ids: [E-1], evidence_classes: [source]}\nevidence:\n  - {id: E-1, source_ref: a.go, class: explicit}\n")

	err := RunRecord(cmd, testDependencies(), RecordOptions{Root: root, Mission: "m", Agent: "ranger", ClaimFile: bad})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"source" is not allowed`)
	assert.Contains(t, err.Error(), "explicit, corroborated_inference, weak_inference, unknown")
	assert.Contains(t, out.String(), "coverage_status: rejected")
	assert.Equal(t, 1, ledgerLines(t, root), "the rejected observation is still persisted")
}

func TestRecordFailsWithTheNormalizationErrorForAClaimWithoutID(t *testing.T) {
	root := testRoot(t)
	cmd, _ := testCommand("metrics")
	noID := writeClaimFile(t, "claim: {statement: s, claim_kind: assertion, confidence_percent: 90}\n")

	err := RunRecord(cmd, testDependencies(), RecordOptions{Root: root, Mission: "m", Agent: "ranger", ClaimFile: noID})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no id; nothing recorded")
	assert.NotContains(t, err.Error(), "rejected observation requires claim_id")
	assert.Zero(t, ledgerLines(t, root))
}

// There is no `critic` alias: the documented name is response_critic, and the
// rejection lists the accepted agents so the fix is obvious.
func TestRecordRejectsCriticAndListsTheAcceptedAgents(t *testing.T) {
	root := testRoot(t)
	cmd, _ := testCommand("metrics")
	err := RunRecord(cmd, testDependencies(), RecordOptions{Root: root, Mission: "m", Agent: "critic", Missing: true, CorrelationKey: "k", Reason: "r"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported agent "critic"`)
	assert.Contains(t, err.Error(), "response_critic")
	assert.Zero(t, ledgerLines(t, root))

	require.NoError(t, RunRecord(cmd, testDependencies(), RecordOptions{Root: root, Mission: "m", Agent: "response_critic", Missing: true, CorrelationKey: "k", Reason: "r"}))
}
