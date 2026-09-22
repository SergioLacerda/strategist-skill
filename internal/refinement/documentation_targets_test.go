package refinement_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/refinement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasDocumentationTargets(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]struct {
		body string
		want bool
	}{
		"checkbox tag":       {"- [ ] 1.1 [documentation_target] write the guide\n", true},
		"handoff field":      {"  - {id: \"1.1\", task_type: documentation_target}\n", true},
		"only handoffs":      {"- [ ] 1.1 [implementation_handoff] code\n  - {id: \"1.1\", task_type: implementation_handoff}\n", false},
		"prose mention only": {"There is no `documentation_target`: Sniper has nothing to do.\n", false},
	}
	for name, tc := range cases {
		path := filepath.Join(dir, name+".md")
		require.NoError(t, os.WriteFile(path, []byte(tc.body), 0o600))
		got, err := refinement.HasDocumentationTargets(path)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got, name)
	}
	got, err := refinement.HasDocumentationTargets(filepath.Join(dir, "missing.md"))
	require.NoError(t, err)
	assert.False(t, got, "a missing tasks.md declares no target")
}
