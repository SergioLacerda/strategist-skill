package refinement

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeOpenSpecPublishesCanonicalPackageAndPromotesPending(t *testing.T) {
	project := t.TempDir()
	base := filepath.Join(project, ".analysis")
	runtime := filepath.Join(project, ".strategist", "openspec")
	mission := "m-1"
	change := "20260917-change"
	pending := filepath.Join(base, "pending", mission+"-analysis.md")
	changeDir := filepath.Join(runtime, "changes", change)
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(changeDir, "specs", "private"), 0o755))
	require.NoError(t, os.WriteFile(pending, []byte("---\nmission_id: m-1\nmission_status: archivist_pending\n---\n\n# Analysis\n"), 0o644))
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(changeDir, name), []byte("# "+name+"\n"), 0o644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(changeDir, "specs", "private", "spec.md"), []byte("private\n"), 0o644))

	result, err := NormalizeOpenSpec(OpenSpecInput{MissionID: mission, BasePath: base, RuntimeRoot: runtime, ChangeID: change, PendingAnalysisPath: pending})
	require.NoError(t, err)
	assertFile(t, filepath.Join(result.RefinedPath, "analysis.md"), "mission_status: archivist_done", "provider_change_id: "+change)
	for _, name := range canonicalFiles[1:] {
		assertFile(t, filepath.Join(result.RefinedPath, name), "# "+name)
	}
	_, err = os.Stat(pending)
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(filepath.Join(result.RefinedPath, "specs"))
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(changeDir)
	require.ErrorIs(t, err, os.ErrNotExist, "a published change leaves the active list")
	archived, err := filepath.Glob(filepath.Join(runtime, "changes", "archive", "*-"+change))
	require.NoError(t, err)
	assert.Len(t, archived, 1, "it is moved to changes/archive/<date>-<id>")
}

func TestNormalizeOpenSpecRejectsEscapingAndPartialOutput(t *testing.T) {
	project := t.TempDir()
	base := filepath.Join(project, ".analysis")
	runtime := filepath.Join(project, ".strategist", "openspec")
	pending := filepath.Join(base, "pending", "m-1-analysis.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.WriteFile(pending, []byte("---\nmission_id: m-1\n---\n# analysis\n"), 0o644))
	_, err := NormalizeOpenSpec(OpenSpecInput{MissionID: "m-1", BasePath: base, RuntimeRoot: runtime, ChangeID: "../escape", PendingAnalysisPath: pending})
	require.Error(t, err)

	changeDir := filepath.Join(runtime, "changes", "change")
	require.NoError(t, os.MkdirAll(changeDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte("proposal"), 0o644))
	_, err = NormalizeOpenSpec(OpenSpecInput{MissionID: "m-1", BasePath: base, RuntimeRoot: runtime, ChangeID: "change", PendingAnalysisPath: pending})
	require.ErrorContains(t, err, "incomplete change")
	_, err = os.Stat(filepath.Join(base, "refined", "m-1"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestNormalizeOpenSpecRejectsMissionIdentityMismatch(t *testing.T) {
	project := t.TempDir()
	base := filepath.Join(project, ".analysis")
	runtime := filepath.Join(project, ".strategist", "openspec")
	pending := filepath.Join(base, "pending", "m-1-analysis.md")
	changeDir := filepath.Join(runtime, "changes", "change")
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.MkdirAll(changeDir, 0o755))
	require.NoError(t, os.WriteFile(pending, []byte("---\nmission_id: another\n---\n"), 0o644))
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(changeDir, name), []byte(name), 0o644))
	}
	_, err := NormalizeOpenSpec(OpenSpecInput{MissionID: "m-1", BasePath: base, RuntimeRoot: runtime, ChangeID: "change", PendingAnalysisPath: pending})
	require.ErrorContains(t, err, "mission_id does not match")
}

func assertFile(t *testing.T, path string, fragments ...string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	for _, fragment := range fragments {
		require.Contains(t, string(raw), fragment, "missing %q in %s", fragment, path)
	}
}
