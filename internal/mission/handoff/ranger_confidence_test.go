package handoff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeRanger(t *testing.T, frontmatter string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "m-analysis.md")
	require.NoError(t, os.WriteFile(path, []byte("---\nmission_id: m\n"+frontmatter+"---\n\n## mission_objective\nx\n"), 0o644))
	return path
}

func TestReadRangerConfidenceReturnsWhatTheArtifactDeclares(t *testing.T) {
	path := writeRanger(t, `confidence_score: 0.82
evaluation_verdict: partially_implemented
discovery_subtype: evaluation
sources_consulted:
  - {source_path: a.md, content_fingerprint: sha256:1, coverage_status: full}
  - {source_path: b.md, content_fingerprint: sha256:2, coverage_status: partial}
`)
	got, err := ReadRangerConfidence(path)
	require.NoError(t, err)
	require.NotNil(t, got.Score)
	require.InDelta(t, 0.82, *got.Score, 1e-9)
	require.Equal(t, "partially_implemented", got.Verdict)
	require.Equal(t, "evaluation", got.Subtype)
	require.Equal(t, []RangerSource{{Path: "a.md", Coverage: "full"}, {Path: "b.md", Coverage: "partial"}}, got.Sources)
}

func TestReadRangerConfidenceAcceptsAnIntegerScoreAndReportsAnAbsentOneAsNil(t *testing.T) {
	got, err := ReadRangerConfidence(writeRanger(t, "confidence_score: 1\n"))
	require.NoError(t, err)
	require.InDelta(t, 1.0, *got.Score, 1e-9)

	got, err = ReadRangerConfidence(writeRanger(t, ""))
	require.NoError(t, err)
	require.Nil(t, got.Score, "an absent score is nil, never zero")
}

func TestReadRangerConfidenceRejectsANonNumericScoreAndAMissingFile(t *testing.T) {
	_, err := ReadRangerConfidence(writeRanger(t, "confidence_score: high\n"))
	require.ErrorContains(t, err, "confidence_score")
	_, err = ReadRangerConfidence(filepath.Join(t.TempDir(), "absent.md"))
	require.Error(t, err)
}

func TestReadRangerConfidenceIgnoresMalformedSourcesAndRejectsAnArtifactWithoutFrontmatter(t *testing.T) {
	got, err := ReadRangerConfidence(writeRanger(t, "confidence_score: 0.5\nsources_consulted:\n  - not-a-map\n  - {coverage_status: full}\n  - {source_path: ok.md, coverage_status: partial}\n"))
	require.NoError(t, err)
	require.Equal(t, []RangerSource{{Path: "ok.md", Coverage: "partial"}}, got.Sources, "a malformed or pathless entry is skipped, never invented")

	bare := filepath.Join(t.TempDir(), "bare.md")
	require.NoError(t, os.WriteFile(bare, []byte("no frontmatter here\n"), 0o644))
	_, err = ReadRangerConfidence(bare)
	require.ErrorContains(t, err, "frontmatter")
}
