package metrics

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

const validClaimYAML = `claim:
  id: C-1
  statement: a claim
  agent: ranger
  correlation_key: k
  claim_kind: assertion
  confidence_percent: 90
  confidence_level: high
  evidence_ids: [E-1]
  evidence_classes: [explicit]
evidence:
  - {id: E-1, source_ref: a.go, class: explicit, confidence: high}
`

func recordFromStdin(t *testing.T, root, body string) error {
	t.Helper()
	cmd, _ := testCommand("metrics")
	cmd.SetIn(strings.NewReader(body))
	return RunRecord(cmd, testDependencies(), RecordOptions{Root: root, Mission: "m", Agent: "ranger", ClaimFile: "-"})
}

func TestRecordReadsTheClaimFromStandardInput(t *testing.T) {
	root := testRoot(t)
	require.NoError(t, recordFromStdin(t, root, validClaimYAML))
	assert.Equal(t, 1, ledgerLines(t, root))
}

func TestRecordRejectsEmptyStandardInput(t *testing.T) {
	root := testRoot(t)
	err := recordFromStdin(t, root, "  \n")
	require.ErrorContains(t, err, "standard input is empty")
	assert.Zero(t, ledgerLines(t, root))
}

func TestRecordStdinAndFileFormsProduceTheSameEntry(t *testing.T) {
	stdinRoot, fileRoot := testRoot(t), testRoot(t)
	require.NoError(t, recordFromStdin(t, stdinRoot, validClaimYAML))
	cmd, _ := testCommand("metrics")
	require.NoError(t, RunRecord(cmd, testDependencies(), RecordOptions{Root: fileRoot, Mission: "m", Agent: "ranger", ClaimFile: writeClaimFile(t, validClaimYAML)}))

	fromStdin, err := telemetry.ReadConfidenceRecords(telemetry.ConfidenceHistoryPath(stdinRoot))
	require.NoError(t, err)
	fromFile, err := telemetry.ReadConfidenceRecords(telemetry.ConfidenceHistoryPath(fileRoot))
	require.NoError(t, err)
	require.Len(t, fromStdin, 1)
	require.Len(t, fromFile, 1)
	fromStdin[0].Timestamp, fromFile[0].Timestamp = "", ""
	assert.Equal(t, fromFile[0], fromStdin[0])
}

func depsWithBasePath(base string, err error) Dependencies {
	deps := testDependencies()
	deps.ResolveBasePath = func(string) (string, error) { return base, err }
	return deps
}

func recordFile(t *testing.T, deps Dependencies, root, path string) error {
	t.Helper()
	cmd, _ := testCommand("metrics")
	return RunRecord(cmd, deps, RecordOptions{Root: root, Mission: "m", Agent: "ranger", ClaimFile: path})
}

func TestRecordRejectsAClaimFileInsideBasePath(t *testing.T) {
	root, base := testRoot(t), t.TempDir()
	pending := filepath.Join(base, "pending")
	require.NoError(t, os.MkdirAll(pending, 0o755))
	inside := filepath.Join(pending, "m-ranger-confidence.yaml")
	require.NoError(t, os.WriteFile(inside, []byte(validClaimYAML), 0o600))

	err := recordFile(t, depsWithBasePath(base, nil), root, inside)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workspace artifact tree")
	assert.Contains(t, err.Error(), "--claim-file -")
	assert.Zero(t, ledgerLines(t, root), "nothing is persisted for a rejected location")
}

func TestRecordGuardResolvesDotDotAndSymlinks(t *testing.T) {
	root, base, outside := testRoot(t), t.TempDir(), t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, "pending"), 0o755))
	realPath := filepath.Join(base, "pending", "claim.yaml")
	require.NoError(t, os.WriteFile(realPath, []byte(validClaimYAML), 0o600))

	dotdot := filepath.Join(outside, "..", filepath.Base(base), "pending", "claim.yaml")
	require.ErrorContains(t, recordFile(t, depsWithBasePath(base, nil), root, dotdot), "workspace artifact tree")

	link := filepath.Join(outside, "link.yaml")
	if err := os.Symlink(realPath, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	require.ErrorContains(t, recordFile(t, depsWithBasePath(base, nil), root, link), "workspace artifact tree")
	assert.Zero(t, ledgerLines(t, root))
}

func TestRecordAllowsAClaimFileOutsideBasePath(t *testing.T) {
	root := testRoot(t)
	require.NoError(t, recordFile(t, depsWithBasePath(t.TempDir(), nil), root, writeClaimFile(t, validClaimYAML)))
	assert.Equal(t, 1, ledgerLines(t, root))
}

func TestRecordGuardFailsClosedWhenBasePathCannotBeResolved(t *testing.T) {
	root := testRoot(t)
	err := recordFile(t, depsWithBasePath("", errors.New("active.yaml: base_path is empty")), root, writeClaimFile(t, validClaimYAML))
	require.ErrorContains(t, err, "resolve base_path")
	assert.Zero(t, ledgerLines(t, root))
}

func TestRecordGuardIsSkippedWithoutAResolver(t *testing.T) {
	root := testRoot(t)
	require.NoError(t, recordFile(t, testDependencies(), root, writeClaimFile(t, validClaimYAML)))
	assert.Equal(t, 1, ledgerLines(t, root))
}

func TestRecordStandardInputBypassesTheGuardByConstruction(t *testing.T) {
	root := testRoot(t)
	cmd, _ := testCommand("metrics")
	cmd.SetIn(strings.NewReader(validClaimYAML))
	require.NoError(t, RunRecord(cmd, depsWithBasePath(t.TempDir(), nil), RecordOptions{Root: root, Mission: "m", Agent: "ranger", ClaimFile: "-"}))
	assert.Equal(t, 1, ledgerLines(t, root))
}
