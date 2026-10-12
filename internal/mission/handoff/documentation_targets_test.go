package handoff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDocumentationTargetContent(t *testing.T) {
	paths, err := ValidateDocumentationTargetContent([]byte("- [ ] 1.1 [documentation_target] write `docs/adr/decision.md`\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"docs/adr/decision.md"}, paths)
}

func TestValidateDocumentationTargetContentRejectsDuplicateTarget(t *testing.T) {
	paths, err := ValidateDocumentationTargetContent([]byte("- [ ] 1.1 [documentation_target] write `docs/adr/decision.md`\n- [ ] 1.2 [documentation_target] repeat `docs/adr/decision.md`\n"))
	assert.Nil(t, paths)
	require.ErrorContains(t, err, "duplicates executable target")
}

func TestValidateArchivistPackageRejectsDocumentationTargetWithoutPath(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"analysis.md": "---\nmission_id: m1\nmission_status: archivist_done\n---\nanalysis\n",
		"proposal.md": "proposal\n",
		"design.md":   "design\n",
		"tasks.md":    "- [ ] 1.1 [documentation_target] write it\n",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
	require.ErrorContains(t, ValidateArchivistPackage(dir, "m1"), "explicit backtick-quoted repository-relative path")
}
