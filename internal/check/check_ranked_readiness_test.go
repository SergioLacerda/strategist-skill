package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestValidateRankedRuntimeRootRequiresDirectOpenSpecConfig(t *testing.T) {
	runtimeRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(runtimeRoot, "config.yaml"), []byte("schema: spec-driven\n"), 0o644))

	result := validateRankedRuntimeRoot(runtimeRoot, "openspec-propose")

	require.True(t, result.Ready())
}

func TestValidateRankedRuntimeRootRejectsNestedOpenSpecConfig(t *testing.T) {
	runtimeRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(runtimeRoot, "openspec"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runtimeRoot, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644))

	result := validateRankedRuntimeRoot(runtimeRoot, "openspec-propose")

	require.Equal(t, domain.ReadinessBlocked, result.Status)
	require.Equal(t, "ranked_runtime_root_missing", result.ReasonCode)
}
