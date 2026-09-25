package metrics

import (
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const batchClaimsYAML = `claims:
  - {id: C-1, statement: s1, agent: ranger, correlation_key: k1, claim_kind: assertion, confidence_percent: 90, evidence_ids: [E-1], evidence_classes: [explicit]}
  - {id: C-2, statement: s2, agent: ranger, correlation_key: k2, claim_kind: assertion, confidence_percent: 70, evidence_ids: [E-1], evidence_classes: [explicit]}
  - {id: Q-1, statement: "is it so", agent: ranger, correlation_key: k3, claim_kind: question, confidence_percent: 30}
evidence:
  - {id: E-1, source_ref: a.go, class: explicit, confidence: high}
`

func recordBatch(t *testing.T, root, agent, body string) (string, error) {
	t.Helper()
	cmd, out := testCommand("metrics")
	cmd.SetIn(strings.NewReader(body))
	err := RunRecord(cmd, testDependencies(), RecordOptions{Root: root, Mission: "m", Agent: agent, ClaimFile: "-"})
	return out.String(), err
}

// The drift this closes: ten declared claims used to reach the ledger one
// command at a time, so a role that recorded one claim looked like a sample
// of one. A batch document records every declared claim, questions included.
func TestRecordBatchRecordsEveryDeclaredClaim(t *testing.T) {
	root := testRoot(t)
	out, err := recordBatch(t, root, "ranger", batchClaimsYAML)
	require.NoError(t, err)
	assert.Equal(t, 3, ledgerLines(t, root))
	for _, id := range []string{"C-1", "C-2", "Q-1"} {
		assert.Contains(t, out, "claim: "+id)
	}
	assert.Contains(t, out, "recorded: 3 (reported 3, rejected 0)")

	records, err := telemetry.ReadConfidenceRecords(telemetry.ConfidenceHistoryPath(root))
	require.NoError(t, err)
	metrics := telemetry.ComputeConfidenceMetrics(records)
	assert.Equal(t, 3, metrics.SampleSize)
	assert.Equal(t, 1, metrics.ClaimKinds["question"], "the question is preserved, not dropped")
}

func TestRecordBatchRejectsMixingClaimAndClaims(t *testing.T) {
	root := testRoot(t)
	mixed := validClaimYAML + "claims:\n  - {id: C-9, statement: s, claim_kind: assertion, confidence_percent: 90}\n"
	_, err := recordBatch(t, root, "ranger", mixed)
	require.ErrorContains(t, err, "either `claim:` or `claims:`")
	assert.Zero(t, ledgerLines(t, root))
}

func TestRecordBatchRejectsAnEmptyOrMisShapedClaimsList(t *testing.T) {
	root := testRoot(t)
	_, err := recordBatch(t, root, "ranger", "claims: []\n")
	require.ErrorContains(t, err, "`claims` list is empty")
	_, err = recordBatch(t, root, "ranger", "claims: {id: C-1}\n")
	require.ErrorContains(t, err, "`claims` must be a list")
	assert.Zero(t, ledgerLines(t, root))
}

// A claim declared by another agent would be silently attributed to the
// producing agent, so the whole batch fails before anything is written.
func TestRecordBatchRejectsAClaimDeclaredByAnotherAgent(t *testing.T) {
	root := testRoot(t)
	foreign := strings.Replace(batchClaimsYAML, "id: C-2, statement: s2, agent: ranger", "id: C-2, statement: s2, agent: archivist", 1)
	_, err := recordBatch(t, root, "ranger", foreign)
	require.ErrorContains(t, err, `claim C-2 is declared by agent "archivist"`)
	assert.Zero(t, ledgerLines(t, root), "nothing is recorded for a batch with a foreign claim")
}

func TestRecordBatchRejectsDuplicateClaimIDs(t *testing.T) {
	root := testRoot(t)
	dup := strings.Replace(batchClaimsYAML, "id: C-2,", "id: C-1,", 1)
	_, err := recordBatch(t, root, "ranger", dup)
	require.ErrorContains(t, err, "duplicate claim id C-1")
	assert.Zero(t, ledgerLines(t, root))
}

// Per-claim rejection keeps its meaning: the invalid claim is persisted as a
// rejected observation, the valid ones are recorded, and the command fails
// naming every rejected claim.
func TestRecordBatchPersistsRejectionsAndFailsNamingThem(t *testing.T) {
	root := testRoot(t)
	bad := strings.Replace(batchClaimsYAML, "id: C-2, statement: s2, agent: ranger, correlation_key: k2, claim_kind: assertion, confidence_percent: 70, evidence_ids: [E-1], evidence_classes: [explicit]", "id: C-2, statement: s2, agent: ranger, correlation_key: k2, claim_kind: assertion, confidence_percent: 70, evidence_ids: [E-1], evidence_classes: [source]", 1)
	out, err := recordBatch(t, root, "ranger", bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 of 3 claims rejected")
	assert.Contains(t, err.Error(), "C-2")
	assert.Contains(t, out, "recorded: 3 (reported 2, rejected 1)")
	assert.Equal(t, 3, ledgerLines(t, root), "the rejected observation is persisted too")
}

// The single-claim document keeps working unchanged.
func TestRecordBatchLeavesTheSingleClaimFormUntouched(t *testing.T) {
	root := testRoot(t)
	out, err := recordBatch(t, root, "ranger", validClaimYAML)
	require.NoError(t, err)
	assert.Equal(t, 1, ledgerLines(t, root))
	assert.NotContains(t, out, "recorded:")
}

// A claim without an id would abort the batch after earlier claims were
// written, so it is refused up front and nothing is recorded.
func TestRecordBatchRejectsAClaimWithoutIDBeforeWritingAnything(t *testing.T) {
	root := testRoot(t)
	noID := strings.Replace(batchClaimsYAML, "id: Q-1, ", "", 1)
	_, err := recordBatch(t, root, "ranger", noID)
	require.ErrorContains(t, err, "claim #3 has no id; nothing recorded")
	assert.Zero(t, ledgerLines(t, root))
}
