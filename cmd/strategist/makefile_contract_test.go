package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- go-file-size-report (Makefile contract) ---

func writeLines(t *testing.T, path string, count int) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()
	for i := 0; i < count; i++ {
		_, err = fmt.Fprintf(f, "// line %d\n", i+1)
		require.NoError(t, err)
	}
}

func copyFile(t *testing.T, srcRel, dstAbs string) {
	t.Helper()
	src, err := filepath.Abs(srcRel)
	require.NoError(t, err)
	data, err := os.ReadFile(src)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(dstAbs), 0o755))
	require.NoError(t, os.WriteFile(dstAbs, data, 0o644))
}

// copyMakefile mirrors the real repo's Makefile layout into an isolated test
// root: the root Makefile, every included make/*.mk domain file (go.mk
// includes go-file-size-report's home, quality.mk), and the
// go-file-size-report.sh script the quality.mk target now delegates to.
func copyMakefile(t *testing.T, dstRoot string) {
	t.Helper()
	copyFile(t, "../../Makefile", filepath.Join(dstRoot, "Makefile"))

	makeDir, err := filepath.Abs("../../make")
	require.NoError(t, err)
	entries, err := os.ReadDir(makeDir)
	require.NoError(t, err)
	for _, e := range entries {
		copyFile(t, filepath.Join("../../make", e.Name()), filepath.Join(dstRoot, "make", e.Name()))
	}

	copyFile(t, "../../scripts/go-file-size-report.sh", filepath.Join(dstRoot, "scripts", "go-file-size-report.sh"))
}

func TestMakeGoFileSizeReport_PrimarySourcesOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cmd", "app"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal", "pkg"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal", "embed", "defaults"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".strategist"), 0o755))

	writeLines(t, filepath.Join(root, "cmd", "app", "main.go"), 205)
	writeLines(t, filepath.Join(root, "internal", "pkg", "service.go"), 240)
	writeLines(t, filepath.Join(root, "internal", "pkg", "service_test.go"), 260)
	writeLines(t, filepath.Join(root, "internal", "embed", "defaults", "generated.go"), 300)
	writeLines(t, filepath.Join(root, ".strategist", "runtime.go"), 400)
	copyMakefile(t, root)

	cmd := exec.Command("make", "go-file-size-report")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	output := string(out)
	assert.Contains(t, output, "=== Go Files > 200 Lines ===")
	assert.Contains(t, output, "internal/pkg/service.go 240")
	assert.Contains(t, output, "cmd/app/main.go 205")
	assert.NotContains(t, output, "service_test.go")
	assert.NotContains(t, output, "internal/embed/defaults/generated.go")
	assert.NotContains(t, output, ".strategist/runtime.go")
}

func TestMakeGoFileSizeReport_PrintsNoneWhenNoLargeFilesExist(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cmd", "app"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "internal", "pkg"), 0o755))
	writeLines(t, filepath.Join(root, "cmd", "app", "main.go"), 40)
	writeLines(t, filepath.Join(root, "internal", "pkg", "service.go"), 120)
	copyMakefile(t, root)

	cmd := exec.Command("make", "go-file-size-report")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	output := string(out)
	assert.Contains(t, output, "=== Go Files > 200 Lines ===")
	assert.Contains(t, output, "none")
}

// --- dojoItemLine ---
