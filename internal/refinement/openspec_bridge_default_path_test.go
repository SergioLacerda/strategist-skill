package refinement

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin the default (non-amend) path of NormalizeOpenSpec before the
// amend mode exists (M016): the fail-closed conflict handling is what an amendment
// must never weaken.

type bridgeFixture struct {
	base, runtime, pending, refined string
	mission                         string
}

func newBridgeFixture(t *testing.T) bridgeFixture {
	t.Helper()
	project := t.TempDir()
	f := bridgeFixture{mission: "m-1", base: filepath.Join(project, ".analysis"), runtime: filepath.Join(project, ".strategist", "openspec")}
	f.pending = filepath.Join(f.base, "pending", f.mission+"-analysis.md")
	f.refined = filepath.Join(f.base, "refined", f.mission)
	return f
}

// change writes a complete OpenSpec change whose files hold body, and (re)creates the
// pending analysis.
func (f bridgeFixture) change(t *testing.T, id, body string) {
	t.Helper()
	dir := filepath.Join(f.runtime, "changes", id)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), canonicalTestArtifact(name, body), 0o644))
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(f.pending), 0o755))
	require.NoError(t, os.WriteFile(f.pending, []byte("---\nmission_id: "+f.mission+"\nmission_status: archivist_pending\n---\n\n# Analysis\n"), 0o644))
}

func (f bridgeFixture) normalize(id string) (OpenSpecResult, error) {
	return NormalizeOpenSpec(OpenSpecInput{MissionID: f.mission, BasePath: f.base, RuntimeRoot: f.runtime, ChangeID: id, PendingAnalysisPath: f.pending, RecordConfidence: noopConfidenceRecorder})
}

func readAll(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, name := range canonicalFiles {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		out[name] = string(raw)
	}
	return out
}

func TestNormalizeOpenSpecFailsClosedWhenTheExistingPackageDiffers(t *testing.T) {
	f := newBridgeFixture(t)
	f.change(t, "first", "v1")
	_, err := f.normalize("first")
	require.NoError(t, err)
	before := readAll(t, f.refined)
	f.change(t, "second", "v2")

	_, err = f.normalize("second")

	require.ErrorContains(t, err, "existing refined package conflicts with provider change")
	assert.Equal(t, before, readAll(t, f.refined), "the package bytes are unchanged")
	_, statErr := os.Stat(f.pending)
	require.NoError(t, statErr, "the pending analysis is still present")
	_, statErr = os.Stat(filepath.Join(f.runtime, "changes", "second"))
	assert.NoError(t, statErr, "the change is not archived")
}

func TestNormalizeOpenSpecRefusesAnExistingPackageMissingAFile(t *testing.T) {
	f := newBridgeFixture(t)
	f.change(t, "first", "v1")
	_, err := f.normalize("first")
	require.NoError(t, err)
	require.NoError(t, os.Remove(filepath.Join(f.refined, "tasks.md")))
	f.change(t, "second", "v1")

	_, err = f.normalize("second")

	require.ErrorContains(t, err, "conflicting refined package")
}

// Re-running the same change over its own published package: the package is
// identical, so the pending analysis is promoted (removed), but on the same day the
// archive target <date>-<id> already exists and the rename fails, leaving the change
// in the active list. Pinned as observed on 2026-09-26; it is a latent defect of the
// default path, reported as a side quest and deliberately not fixed here.
func TestNormalizeOpenSpecSameChangeAgainOverAnIdenticalPackageFailsAtTheArchiveStep(t *testing.T) {
	f := newBridgeFixture(t)
	f.change(t, "first", "v1")
	_, err := f.normalize("first")
	require.NoError(t, err)
	f.change(t, "first", "v1")

	_, err = f.normalize("first")

	require.ErrorContains(t, err, "archive change first")
	require.ErrorContains(t, err, "file exists")
	_, pendingErr := os.Stat(f.pending)
	require.ErrorIs(t, pendingErr, os.ErrNotExist, "the pending analysis was already promoted before the archive step failed")
	_, changeErr := os.Stat(filepath.Join(f.runtime, "changes", "first"))
	assert.NoError(t, changeErr, "the change stays in the active list")
}
