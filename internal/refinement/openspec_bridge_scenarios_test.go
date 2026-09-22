package refinement

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The refined package keeps four files, but design.md now carries every
// requirement and scenario the provider wrote in specs/.
func TestNormalizeOpenSpecCarriesAcceptanceScenariosIntoDesign(t *testing.T) {
	project := t.TempDir()
	base := filepath.Join(project, ".analysis")
	runtime := filepath.Join(project, ".strategist", "openspec")
	pending := filepath.Join(base, "pending", "m-2-analysis.md")
	changeDir := filepath.Join(runtime, "changes", "two-caps")
	require.NoError(t, os.MkdirAll(filepath.Dir(pending), 0o755))
	require.NoError(t, os.WriteFile(pending, []byte("---\nmission_id: m-2\nmission_status: archivist_pending\n---\n"), 0o644))
	for _, name := range []string{"proposal.md", "design.md", "tasks.md"} {
		require.NoError(t, os.MkdirAll(changeDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(changeDir, name), []byte("# "+name+"\n"), 0o644))
	}
	for capability, requirement := range map[string]string{"zeta": "Zeta works", "alpha/beta": "Beta works"} {
		dir := filepath.Join(changeDir, "specs", filepath.FromSlash(capability))
		require.NoError(t, os.MkdirAll(dir, 0o755))
		spec := "## Purpose\n\nprose only\n\n## ADDED Requirements\n\n### Requirement: " + requirement + "\nIt SHALL work.\n\n#### Scenario: happy\n- **WHEN** used\n- **THEN** it works\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, "spec.md"), []byte(spec), 0o644))
	}

	result, err := NormalizeOpenSpec(OpenSpecInput{MissionID: "m-2", BasePath: base, RuntimeRoot: runtime, ChangeID: "two-caps", PendingAnalysisPath: pending})
	require.NoError(t, err)
	design, err := os.ReadFile(filepath.Join(result.RefinedPath, "design.md"))
	require.NoError(t, err)
	text := string(design)
	assert.Contains(t, text, "## Acceptance scenarios")
	assert.Contains(t, text, "### `alpha/beta`")
	assert.Contains(t, text, "##### Requirement: Beta works")
	assert.Contains(t, text, "###### Scenario: happy")
	assert.NotContains(t, text, "prose only", "the Purpose section is not copied")
	assert.Less(t, strings.Index(text, "`alpha/beta`"), strings.Index(text, "`zeta`"), "capabilities are in path order")

	entries, err := os.ReadDir(result.RefinedPath)
	require.NoError(t, err)
	assert.Len(t, entries, 4, "the package keeps exactly four files")
}
