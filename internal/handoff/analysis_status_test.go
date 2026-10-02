package handoff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadAnalysisLifecycleReadsStructuredFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	require.NoError(t, os.WriteFile(path, []byte("---\nmission_id: m-1\nmission_status: gate_pending\nclaimed_by: archivist\n---\n\nbody\n"), 0o600))

	got, err := ReadAnalysisLifecycle(path)
	require.NoError(t, err)
	assert.Equal(t, AnalysisLifecycle{MissionID: "m-1", Status: "gate_pending", ClaimedBy: "archivist"}, got)

	require.NoError(t, os.WriteFile(path, []byte("---\nmission_id: m-1\nmission_status: gate_pending\n---\n\nbody\n"), 0o600))
	got, err = ReadAnalysisLifecycle(path)
	require.NoError(t, err)
	assert.Equal(t, AnalysisLifecycle{MissionID: "m-1", Status: "gate_pending"}, got)
}

func TestReadAnalysisLifecycleRejectsMissingAndMalformedFiles(t *testing.T) {
	_, err := ReadAnalysisLifecycle(filepath.Join(t.TempDir(), "missing.md"))
	require.ErrorContains(t, err, "read analysis lifecycle")

	path := filepath.Join(t.TempDir(), "analysis.md")
	require.NoError(t, os.WriteFile(path, []byte("not frontmatter\n"), 0o600))
	_, err = ReadAnalysisLifecycle(path)
	require.ErrorContains(t, err, "analysis lifecycle")
}

func TestAcceptAnalysisAtGateTransitionsAndCanRestore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	original := []byte("---\nmission_id: m-1\nmission_status: archivist_done\n---\n\nbody\n")
	require.NoError(t, os.WriteFile(path, original, 0o600))

	before, changed, err := AcceptAnalysisAtGate(path)
	require.NoError(t, err)
	require.True(t, changed)
	assert.Equal(t, original, before)
	updated, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(updated), "mission_status: gate_analysis_accepted")

	require.NoError(t, RestoreAnalysis(path, before))
	restored, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, original, restored)
}

func TestAcceptAnalysisAtGateIsIdempotentAndRejectsInvalidStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	require.NoError(t, os.WriteFile(path, []byte("---\nmission_id: m-1\nmission_status: gate_analysis_accepted\n---\n\nbody\n"), 0o600))
	_, changed, err := AcceptAnalysisAtGate(path)
	require.NoError(t, err)
	assert.False(t, changed)

	require.NoError(t, os.WriteFile(path, []byte("---\nmission_id: m-1\nmission_status: sniper_running\n---\n\nbody\n"), 0o600))
	_, _, err = AcceptAnalysisAtGate(path)
	require.ErrorContains(t, err, "cannot transition")
}

func TestAcceptAnalysisAtGateTransitionsGatePending(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	require.NoError(t, os.WriteFile(path, []byte("---\nmission_id: m-1\nmission_status: gate_pending\n---\n\nbody\n"), 0o600))

	_, changed, err := AcceptAnalysisAtGate(path)
	require.NoError(t, err)
	assert.True(t, changed)
	updated, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(updated), "mission_status: gate_analysis_accepted")
}

func TestAcceptAnalysisAtGateRejectsMissingStatusLineAndMissingFile(t *testing.T) {
	_, _, err := AcceptAnalysisAtGate(filepath.Join(t.TempDir(), "missing.md"))
	require.ErrorContains(t, err, "read approval-gate analysis")

	path := filepath.Join(t.TempDir(), "analysis.md")
	content := "---\nmission_id: m-1\nmission_status: archivist_done # inline comment\n---\n\nbody\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	_, _, err = AcceptAnalysisAtGate(path)
	require.ErrorContains(t, err, "mission_status line is missing or ambiguous")
}

func TestAcceptAnalysisAtGateRejectsMalformedFrontmatterAndNonStringStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analysis.md")
	require.NoError(t, os.WriteFile(path, []byte("not frontmatter\n"), 0o600))
	_, _, err := AcceptAnalysisAtGate(path)
	require.ErrorContains(t, err, "frontmatter is missing")

	require.NoError(t, os.WriteFile(path, []byte("---\nmission_id: m-1\nmission_status: []\n---\n\nbody\n"), 0o600))
	_, _, err = AcceptAnalysisAtGate(path)
	require.ErrorContains(t, err, "mission_status \"\" cannot transition")
}

func TestRestoreAnalysisReportsAtomicWriteFailure(t *testing.T) {
	err := RestoreAnalysis(filepath.Join(t.TempDir(), "missing", "analysis.md"), []byte("restored\n"))
	require.ErrorContains(t, err, "restore approval-gate analysis")
}
