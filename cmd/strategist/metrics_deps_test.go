package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveMetricsBasePathReturnsResolvedPathWithoutError(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: epic\nbase_path: .analysis\n"), 0o600))

	basePath, err := resolveMetricsBasePath(root)

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(filepath.Dir(root), ".analysis"), basePath)
}
