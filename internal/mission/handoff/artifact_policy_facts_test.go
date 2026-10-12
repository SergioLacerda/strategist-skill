package handoff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateRefinedPackageForGateRejectsInvalidDeclaredHandoffFacts(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"analysis.md": "---\nmission_id: m1\nmission_status: archivist_done\nhandoff_policy_facts:\n  informational_only: true\n---\nanalysis\n",
		"proposal.md": "proposal\n",
		"design.md":   "design\n",
		"tasks.md":    "- [ ] 1.1 [analysis_artifact] record evidence\n",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
	require.ErrorContains(t, ValidateRefinedPackageForGate(dir, "m1"), "handoff_policy_facts_invalid")
}

func TestValidateRefinedPackageForGateRequiresFactsForDocumentationTargets(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"analysis.md": "---\nmission_id: m1\nmission_status: archivist_done\n---\nanalysis\n",
		"proposal.md": "proposal\n", "design.md": "design\n",
		"tasks.md": "- [ ] 1.1 [documentation_target] Write `docs/guide.md`\n",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
	require.ErrorContains(t, ValidateRefinedPackageForGate(dir, "m1"), "handoff_policy_facts_missing")
}

func TestValidateRefinedPackageForGateRejectsMalformedOrDuplicateSideQuests(t *testing.T) {
	for name, tc := range map[string]struct {
		tasks string
		want  string
	}{
		"malformed": {"```yaml\nside_quests_approved: malformed\n```\n- [ ] 1.1 [analysis_artifact] record evidence\n", "must be a list"},
		"duplicate": {"```yaml\nside_quests_approved:\n  - id: SQ-1\n    description: one\n    strategy: separate_mission\n    status: sq_backlog\n  - id: SQ-1\n    description: two\n    strategy: separate_mission\n    status: sq_backlog\n```\n- [ ] 1.1 [analysis_artifact] record evidence\n", "duplicates \"SQ-1\""},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			files := map[string]string{
				"analysis.md": "---\nmission_id: m1\nmission_status: archivist_done\n---\nanalysis\n",
				"proposal.md": "proposal\n", "design.md": "design\n", "tasks.md": tc.tasks,
			}
			for file, content := range files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, file), []byte(content), 0o600))
			}
			require.ErrorContains(t, ValidateRefinedPackageForGate(dir, "m1"), tc.want)
		})
	}
}
