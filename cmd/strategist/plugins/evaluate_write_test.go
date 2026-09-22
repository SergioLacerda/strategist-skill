package plugins

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunEvaluateWrite_Decisions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		target     string
		allowed    bool
		permission string
	}{
		{name: "under base_path", target: ".analysis/refined/example/tasks.md", allowed: true, permission: "permission=analysis.write"},
		{name: "under docs", target: "docs/example.md", permission: "permission=docs.write"},
		{name: "source file", target: "internal/foo/bar.go", permission: "permission=source.write"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := RunEvaluateWrite(&out, EvaluateWriteOptions{Root: minimalRoot(t), Target: tc.target})
			assert.Contains(t, out.String(), tc.permission)
			if tc.allowed {
				require.NoError(t, err)
				assert.Contains(t, out.String(), "write=allowed")
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), "denied")
			assert.Contains(t, out.String(), "write=denied")
		})
	}
}

func TestRunEvaluateWrite_MissingTargetErrors(t *testing.T) {
	t.Parallel()
	err := RunEvaluateWrite(&bytes.Buffer{}, EvaluateWriteOptions{Root: minimalRoot(t)})
	require.EqualError(t, err, "plugins evaluate-write: --target is required")
}

func TestRunEvaluateWrite_MissingActiveYAMLErrors(t *testing.T) {
	t.Parallel()
	err := RunEvaluateWrite(&bytes.Buffer{}, EvaluateWriteOptions{Root: t.TempDir(), Target: "docs/example.md"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plugins evaluate-write: read active.yaml")
}

func TestEvaluateWriteCmd_DispatchesThroughCobra(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, "plugins", "evaluate-write", "--root", minimalRoot(t), "--target", ".analysis/refined/example/tasks.md")
	require.NoError(t, err)
	assert.Contains(t, out, "write=allowed")
}
