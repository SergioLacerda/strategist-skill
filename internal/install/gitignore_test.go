package install

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureGitignoreEntry_ResolvePathErrorPropagates(t *testing.T) {
	t.Parallel()

	err := ensureGitignoreEntry("", gitignoreEntry)
	require.ErrorContains(t, err, "resolve .gitignore path")
}
