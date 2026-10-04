package handoffconsumer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func TestRangerSourceReadsSectionsAndMapsTheObjective(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.md", "---\nmission_id: m\napi_key: sk-frontmatter\n---\n\n## mission_objective\nEvaluate packages.\n## known_facts\n- fact one\n- fact two\n## uncertainties\nnone\n## recommended_refinement_focus\nPreserve verdicts.\n## handoff\nnot in the catalog\n")

	source, err := NewRangerSource(filepath.Join(dir, "a.md"))
	require.NoError(t, err)

	objective, ok := source.Field("objective")
	require.True(t, ok)
	require.Equal(t, "Evaluate packages.", objective)
	facts, _ := source.Field("known_facts")
	require.Equal(t, "- fact one\n- fact two", facts)
	_, ok = source.Field("api_key")
	require.False(t, ok, "frontmatter keys are not contract fields the source serves")
}

func TestPackageSourceReadsTasksAndFrontmatterScope(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "tasks.md", "---\napproved_scope:\n  allowed: [\"docs/**\"]\nacceptance_checks:\n  - go test passes\n---\n\n- [ ] 1.1 [documentation_target] write docs\n")
	write(t, dir, "analysis.md", "---\nmission_id: m\n---\n\n## affected_scope\n- docs/\n")

	source, err := NewPackageSource(dir)
	require.NoError(t, err)

	plan, ok := source.Field("implementation_plan")
	require.True(t, ok)
	require.Contains(t, plan, "[documentation_target] write docs")
	require.NotContains(t, plan, "approved_scope", "the frontmatter is not part of the plan text")
	scope, ok := source.Field("approved_scope")
	require.True(t, ok)
	require.Contains(t, scope, "docs/**")
	checks, ok := source.Field("acceptance_checks")
	require.True(t, ok)
	require.Contains(t, checks, "go test passes")
}

func TestPackageSourceFallsBackToTheAnalysisScopeSection(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "tasks.md", "- [ ] 1.1 [analysis_artifact] note\n")
	write(t, dir, "analysis.md", "## affected_scope\n- docs/\n")

	source, err := NewPackageSource(dir)
	require.NoError(t, err)
	scope, ok := source.Field("approved_scope")
	require.True(t, ok)
	require.Equal(t, "- docs/", scope)
}

func TestSourcesReportUnreadableArtifacts(t *testing.T) {
	_, err := NewRangerSource(filepath.Join(t.TempDir(), "missing.md"))
	require.Error(t, err)
	_, err = NewPackageSource(t.TempDir())
	require.Error(t, err)
}
