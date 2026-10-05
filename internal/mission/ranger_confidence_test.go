package mission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

func rangerArtifact(t *testing.T, root, frontmatter string) string {
	t.Helper()
	path := filepath.Join(root, "analysis", "pending", "m-ranger-analysis.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("---\nmission_id: m-ranger\n"+frontmatter+"---\n\n## mission_objective\nx\n"), 0o644))
	return path
}

func rangerRecords(t *testing.T, root string) []telemetry.ConfidenceRecord {
	t.Helper()
	all, err := telemetry.ReadConfidenceRecords(telemetry.ConfidenceHistoryPath(root))
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	var mine []telemetry.ConfidenceRecord
	for _, record := range all {
		if record.Agent == telemetry.ConfidenceAgentRanger {
			mine = append(mine, record)
		}
	}
	return mine
}

const fullSources = `sources_consulted:
  - {source_path: a.md, content_fingerprint: sha256:1, coverage_status: full}
  - {source_path: b.md, content_fingerprint: sha256:2, coverage_status: full}
`

func TestTheRangerClaimTakesItsPercentFromTheArtifactScoreAsTheSingleSource(t *testing.T) {
	root := t.TempDir()
	artifact := rangerArtifact(t, root, "confidence_score: 0.82\nevaluation_verdict: partially_implemented\n"+fullSources)

	require.NoError(t, RecordRangerConfidence(root, artifact, "m-ranger"))
	got := rangerRecords(t, root)
	require.Len(t, got, 1)
	require.Equal(t, 82, got[0].ConfidencePercent, "the number exists once: the artifact score")
	require.Equal(t, domain.ClaimKindAssertion, got[0].ClaimKind)
	require.Equal(t, "reported", got[0].CoverageStatus)
	require.Equal(t, []string{"explicit", "explicit"}, got[0].EvidenceClasses, "fully read sources are explicit evidence")
}

func TestAnAbsentOrInvalidScoreLeavesAnExplicitMissingRecordNotAGuess(t *testing.T) {
	for name, frontmatter := range map[string]string{
		"absent":       fullSources,
		"out of range": "confidence_score: 1.4\n" + fullSources,
		"negative":     "confidence_score: -0.1\n" + fullSources,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, RecordRangerConfidence(root, rangerArtifact(t, root, frontmatter), "m-ranger"))
			got := rangerRecords(t, root)
			require.Len(t, got, 1)
			require.Equal(t, "missing", got[0].CoverageStatus)
			require.Contains(t, got[0].MissingReason, "confidence_score")
		})
	}
}

func TestAScoreWithoutSupportingEvidenceIsMissingNotAnUnsupportedAssertion(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, RecordRangerConfidence(root, rangerArtifact(t, root, "confidence_score: 0.9\nsources_consulted: []\n"), "m-ranger"))
	got := rangerRecords(t, root)
	require.Len(t, got, 1)
	require.Equal(t, "missing", got[0].CoverageStatus)
	require.Contains(t, got[0].MissingReason, "no_supporting_evidence")
}

func TestALowScoreIsAQuestion(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, RecordRangerConfidence(root, rangerArtifact(t, root, "confidence_score: 0.4\n"+fullSources), "m-ranger"))
	got := rangerRecords(t, root)
	require.Len(t, got, 1)
	require.Equal(t, domain.ClaimKindQuestion, got[0].ClaimKind, "a low-confidence claim must be a question")
}

func TestRecordingTwiceNeverDuplicatesTheClaim(t *testing.T) {
	root := t.TempDir()
	artifact := rangerArtifact(t, root, "confidence_score: 0.82\n"+fullSources)
	require.NoError(t, RecordRangerConfidence(root, artifact, "m-ranger"))
	require.NoError(t, RecordRangerConfidence(root, artifact, "m-ranger"))
	require.Len(t, rangerRecords(t, root), 1)
}

func TestAnUnreadableArtifactIsAnErrorNotASilentGap(t *testing.T) {
	require.Error(t, RecordRangerConfidence(t.TempDir(), filepath.Join(t.TempDir(), "absent.md"), "m-ranger"))
}
