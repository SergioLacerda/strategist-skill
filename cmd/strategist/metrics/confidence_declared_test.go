package metrics

import (
	"os"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func declaredClaimFor(id, kind string, percent int) domain.ConfidenceClaim {
	claim := domain.ConfidenceClaim{ID: id, Statement: "s " + id, Agent: "ranger", CorrelationKey: "k-" + id, ClaimKind: kind, ConfidencePercent: percent, CalibrationStatus: domain.CalibrationNoSample}
	if kind == domain.ClaimKindAssertion {
		claim.EvidenceIDs = []string{"E-1"}
		claim.EvidenceClasses = []string{domain.EvidenceClassExplicit}
	}
	return claim
}

// handoffBlock renders a handoff document: the confidence_summary next to keys only
// a handoff carries.
func handoffBlock(t *testing.T, claims ...domain.ConfidenceClaim) string {
	t.Helper()
	summary := domain.ConfidenceSummary{
		PolicyVersion: domain.ConfidencePolicyVersion, Claims: claims, SampleSize: 0, CalibrationStatus: domain.CalibrationNoSample,
		Evidence: []domain.Evidence{{ID: "E-1", SourceRef: "a.go", Class: domain.EvidenceClassExplicit, Confidence: domain.ConfidenceHigh}},
	}
	raw, err := yaml.Marshal(map[string]any{"mission_id": "m", "tasks_path": "tasks.md", "confidence_summary": summary})
	require.NoError(t, err)
	return string(raw)
}

func runDeclared(t *testing.T, root, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := NewConfidence(testDependencies())
	out := &strings.Builder{}
	cmd.SetOut(out)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(append([]string{"--root", root}, args...))
	err := cmd.Execute()
	return out.String(), err
}

// The history of mission m: C-1 and C-2 assertions and the question Q-1, all ranger.
func declaredM(t *testing.T, extra ...domain.ConfidenceClaim) string {
	t.Helper()
	claims := []domain.ConfidenceClaim{
		{ID: "C-1", Statement: "s1", Agent: "ranger", CorrelationKey: "k1", ClaimKind: "assertion", ConfidencePercent: 90, EvidenceIDs: []string{"E-1"}, EvidenceClasses: []string{"explicit"}, CalibrationStatus: domain.CalibrationNoSample},
		{ID: "C-2", Statement: "s2", Agent: "ranger", CorrelationKey: "k2", ClaimKind: "assertion", ConfidencePercent: 70, EvidenceIDs: []string{"E-1"}, EvidenceClasses: []string{"explicit"}, CalibrationStatus: domain.CalibrationNoSample},
		{ID: "Q-1", Statement: "is it so", Agent: "ranger", CorrelationKey: "k3", ClaimKind: "question", ConfidencePercent: 30, CalibrationStatus: domain.CalibrationNoSample},
	}
	return handoffBlock(t, append(claims, extra...)...)
}

func TestDeclaredAppendsTheComparisonAfterEveryExistingLine(t *testing.T) {
	root := fixedHistory(t)

	out, err := runDeclared(t, root, declaredM(t), "--mission", "m", "--declared", "-")

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(out, wantMissionM), "the existing output is a byte-identical prefix")
	assert.Equal(t, `declared_claims: 3
declared_assertions: 2
declared_questions: 1
persisted_claims: 3
declared_unpersisted: none
persisted_rejected: none
declared_mismatched: none
persisted_under_other_agent: none
question_preservation_declared: 1/1
declared_review: none
`, strings.TrimPrefix(out, wantMissionM))
}

func TestDeclaredNamesAClaimThatWasNeverRecorded(t *testing.T) {
	root := fixedHistory(t)

	out, err := runDeclared(t, root, declaredM(t, declaredClaimFor("C-9", "assertion", 90)), "--mission", "m", "--declared", "-")

	require.NoError(t, err)
	assert.Contains(t, out, "declared_claims: 4\n")
	assert.Contains(t, out, "persisted_claims: 3\n")
	assert.Contains(t, out, "declared_unpersisted: ranger/C-9\n")
	assert.Contains(t, out, "declared_review: recommended\n")
}

func TestDeclaredWithNoQuestionsSaysSo(t *testing.T) {
	root := fixedHistory(t)
	claim := declaredClaimFor("C-1", "assertion", 90)
	claim.CorrelationKey = "k1"

	out, err := runDeclared(t, root, handoffBlock(t, claim), "--mission", "z", "--declared", "-")

	require.NoError(t, err)
	assert.Contains(t, out, "question_preservation_declared: n/a (no questions declared)\n")
}

func TestDeclaredRequiresAMission(t *testing.T) {
	_, err := runDeclared(t, testRoot(t), declaredM(t), "--declared", "-")

	require.ErrorContains(t, err, "--declared requires --mission")
}

func TestDeclaredAcceptsOnlyStandardInput(t *testing.T) {
	_, err := runDeclared(t, testRoot(t), "", "--mission", "m", "--declared", "tasks.md")

	require.ErrorContains(t, err, `--declared accepts only "-"`)
}

func TestDeclaredFailsBeforeAnyOutputOnAnInvalidSummary(t *testing.T) {
	root := fixedHistory(t)
	claim := declaredClaimFor("C-1", "assertion", 90)
	claim.Agent = ""

	out, err := runDeclared(t, root, handoffBlock(t, claim), "--mission", "m", "--declared", "-")

	require.ErrorContains(t, err, "requires agent")
	assert.NotContains(t, out, "policy_version", "an invalid summary fails before the review is printed")
}

func TestDeclaredRejectsADocumentWithoutAConfidenceSummary(t *testing.T) {
	_, err := runDeclared(t, testRoot(t), "mission_id: m\n", "--mission", "m", "--declared", "-")

	require.ErrorContains(t, err, "confidence_summary")
}

func TestDeclaredWritesNothingToTheHistory(t *testing.T) {
	root := fixedHistory(t)
	path := telemetry.ConfidenceHistoryPath(root)
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	_, err = runDeclared(t, root, declaredM(t, declaredClaimFor("C-9", "assertion", 90)), "--mission", "m", "--declared", "-")
	require.NoError(t, err)

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func TestDeclaredAbsentKeepsTheExistingOutput(t *testing.T) {
	out, err := runDeclared(t, fixedHistory(t), "", "--mission", "m")

	require.NoError(t, err)
	assert.Equal(t, wantMissionM, out)
}
