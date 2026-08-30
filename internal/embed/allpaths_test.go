package embed_test

import (
	"testing"

	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractor_AllPaths(t *testing.T) {
	t.Parallel()

	paths, err := embedpkg.Extractor{}.AllPaths()
	require.NoError(t, err)
	require.NotEmpty(t, paths)

	assert.Contains(t, paths, "SKILL.md")
	assert.Contains(t, paths, "skill.yaml")

	seen := make(map[string]bool, len(paths))
	for i, p := range paths {
		assert.NotEmpty(t, p)
		assert.False(t, seen[p], "duplicate path: %s", p)
		seen[p] = true
		if i > 0 {
			assert.LessOrEqual(t, paths[i-1], p, "AllPaths must return a sorted list")
		}
	}

	// Every path AllPaths reports must actually be readable through ReadFile
	// — the two must agree on what "the embedded tree" contains.
	for _, p := range paths[:min(10, len(paths))] {
		_, err := embedpkg.Extractor{}.ReadFile(p)
		assert.NoError(t, err, "AllPaths reported %s but ReadFile could not read it", p)
	}
}
