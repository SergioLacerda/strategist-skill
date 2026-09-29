package check

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportRefinedPortability_AdvisoryFindings(t *testing.T) {
	worktree := t.TempDir()
	root := filepath.Join(worktree, ".strategist")
	pkg := filepath.Join(worktree, ".analysis", "refined", "pkg")
	require.NoError(t, os.MkdirAll(pkg, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "analysis.md"), []byte("provider_runtime: /abs/path\n"), 0o600))

	var out bytes.Buffer
	reportRefinedPortability(&out, root, ".analysis")
	assert.Contains(t, out.String(), "package_portability")
	assert.Contains(t, out.String(), "absolute provider_runtime")
}

func TestReportRefinedPortability_NoRefinedDirIsSilent(t *testing.T) {
	var out bytes.Buffer
	reportRefinedPortability(&out, filepath.Join(t.TempDir(), ".strategist"), ".analysis")
	reportRefinedPortability(&out, filepath.Join(t.TempDir(), ".strategist"), "")
	assert.Empty(t, out.String())
}
