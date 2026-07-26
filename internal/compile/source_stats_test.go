package compile_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertSourceStats(t *testing.T, artifact map[string]any) {
	t.Helper()
	stats, ok := artifact["source_stats"].(map[string]any)
	require.True(t, ok, "source_stats must be present")
	require.NotEmpty(t, stats, "source_stats must include compiled sources")
	for _, raw := range stats {
		entry, ok := raw.(map[string]any)
		require.True(t, ok, "source_stats entry must be an object")
		assert.NotZero(t, entry["mtime"])
		assert.NotZero(t, entry["mtime_ns"])
		assert.NotZero(t, entry["size"])
		assert.NotEmpty(t, entry["sha256"])
	}
}
