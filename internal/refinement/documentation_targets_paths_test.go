package refinement_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/refinement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentationTargetPaths(t *testing.T) {
	dir := t.TempDir()
	body := "## 1. Documentation Targets\n\n" +
		"- [ ] 1.1 [documentation_target] Write `docs/adr/0057-mission-state-concurrency-and-atomicity.md` recording the decision.\n" +
		"- [ ] 1.2 [documentation_target] Write `docs/runbooks/corrupt-mission-state-recovery.md` and its sidecar.\n" +
		"- [ ] 2.1 [implementation_handoff] Edit `internal/mission/execution_boundary.go` to add the guard.\n" +
		"- [ ] 2.2 [implementation_handoff] Write a note with no path in backticks.\n" +
		"- [ ] 2.3 [documentation_target] Write `docs/adr/0057-mission-state-concurrency-and-atomicity.md` again (duplicate).\n"
	path := filepath.Join(dir, "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	got, err := refinement.DocumentationTargetPaths(path)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"docs/adr/0057-mission-state-concurrency-and-atomicity.md",
		"docs/runbooks/corrupt-mission-state-recovery.md",
	}, got, "only documentation_target lines contribute, duplicates are deduplicated, and an implementation_handoff line's path is excluded")
}

func TestDocumentationTargetPathsRejectsMissingOrUnsafePath(t *testing.T) {
	for name, body := range map[string]string{
		"missing path":  "- [ ] 1.1 [documentation_target] Write the guide\n",
		"absolute path": "- [ ] 1.1 [documentation_target] Write `/docs/guide.md`\n",
		"escaping path": "- [ ] 1.1 [documentation_target] Write `docs/../guide.md`\n",
		"windows path":  "- [ ] 1.1 [documentation_target] Write `docs\\guide.md`\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tasks.md")
			require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

			_, err := refinement.DocumentationTargetPaths(path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "documentation_target")
		})
	}
}

func TestDocumentationTargetPaths_MissingFile(t *testing.T) {
	got, err := refinement.DocumentationTargetPaths(filepath.Join(t.TempDir(), "missing.md"))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestDocumentationTargetPaths_NoTargets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.md")
	require.NoError(t, os.WriteFile(path, []byte("- [ ] 1.1 [implementation_handoff] code\n"), 0o600))

	got, err := refinement.DocumentationTargetPaths(path)
	require.NoError(t, err)
	assert.Empty(t, got)
}
