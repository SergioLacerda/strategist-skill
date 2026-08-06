package cliutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveStrategistRoot_ExplicitPathResolvesAbs(t *testing.T) {
	strategistDir, projectRoot, err := ResolveStrategistRoot("some/relative/path", "/unused/cwd")
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(strategistDir))
	assert.Equal(t, filepath.Dir(strategistDir), projectRoot)
}

func TestResolveStrategistRoot_EmptyExplicitFallsBackToFind(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".strategist"), 0o755))

	strategistDir, projectRoot, err := ResolveStrategistRoot("", dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, ".strategist"), strategistDir)
	assert.Equal(t, dir, projectRoot)
}

func TestResolveStrategistRoot_AbsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chdir-then-remove not reliable on windows")
	}
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	removed := t.TempDir()
	require.NoError(t, os.Chdir(removed))
	require.NoError(t, os.RemoveAll(removed))

	_, _, resolveErr := ResolveStrategistRoot("relative/explicit/path", "irrelevant")
	require.Error(t, resolveErr)
	assert.Contains(t, resolveErr.Error(), "resolve root")
}

func TestFindStrategistRoot_FoundInCWD(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".strategist"), 0o755))

	strategistDir, projectRoot, err := FindStrategistRoot(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, ".strategist"), strategistDir)
	assert.Equal(t, dir, projectRoot)
}

func TestFindStrategistRoot_FoundInParent(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".strategist"), 0o755))
	subdir := filepath.Join(root, "subproject", "src")
	require.NoError(t, os.MkdirAll(subdir, 0o755))

	strategistDir, projectRoot, err := FindStrategistRoot(subdir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, ".strategist"), strategistDir)
	assert.Equal(t, root, projectRoot)
}

func TestFindStrategistRoot_NotFound(t *testing.T) {
	dir := t.TempDir() // no .strategist/

	_, _, err := FindStrategistRoot(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStringFlag_NilCmdReturnsFallback(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "fallback", StringFlag(nil, "missing", "fallback"))
}

func TestStringFlag_NoFlagAnywhereReturnsFallback(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{Use: "no-such-flag"}
	assert.Equal(t, "fallback", StringFlag(cmd, "missing", "fallback"))
}

func TestStringFlag_ReadsOwnFlag(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{Use: "has-flag"}
	cmd.Flags().String("name", "default", "")
	require.NoError(t, cmd.Flags().Set("name", "explicit"))
	assert.Equal(t, "explicit", StringFlag(cmd, "name", "fallback"))
}

func TestStringFlag_ReadsInheritedFlag(t *testing.T) {
	t.Parallel()
	parent := &cobra.Command{Use: "parent"}
	parent.PersistentFlags().String("root", "from-parent", "")
	child := &cobra.Command{Use: "child"}
	parent.AddCommand(child)
	assert.Equal(t, "from-parent", StringFlag(child, "root", "fallback"))
}

func TestBoolFlag_NilCmdReturnsFallback(t *testing.T) {
	t.Parallel()
	assert.True(t, BoolFlag(nil, "missing", true))
}

func TestBoolFlag_NoFlagAnywhereReturnsFallback(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{Use: "no-such-flag"}
	assert.True(t, BoolFlag(cmd, "missing", true))
}

func TestBoolFlag_ReadsOwnFlag(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{Use: "has-flag"}
	cmd.Flags().Bool("enabled", false, "")
	require.NoError(t, cmd.Flags().Set("enabled", "true"))
	assert.True(t, BoolFlag(cmd, "enabled", false))
}

func TestBoolFlag_ReadsInheritedFlag(t *testing.T) {
	t.Parallel()
	parent := &cobra.Command{Use: "parent"}
	parent.PersistentFlags().Bool("enabled", true, "")
	child := &cobra.Command{Use: "child"}
	parent.AddCommand(child)
	assert.True(t, BoolFlag(child, "enabled", false))
}

func TestResolveActiveBasePath_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("base_path: .analysis\n"), 0o644))

	strategistRoot, basePath, err := ResolveActiveBasePath(dir)
	require.NoError(t, err)
	assert.Equal(t, dir, strategistRoot)
	assert.Equal(t, filepath.Join(filepath.Dir(dir), ".analysis"), basePath)
}

func TestResolveActiveBasePath_EmptyRootDefaultsToStrategist(t *testing.T) {
	// When root is empty, ResolveActiveBasePath sets strategistRoot = ".strategist".
	// Reading active.yaml from ".strategist/active.yaml" in a tmp CWD will fail
	// (file not found), but the assignment branch is exercised.
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(t.TempDir()))

	_, _, err = ResolveActiveBasePath("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active.yaml")
}

func TestResolveActiveBasePath_InvalidActiveYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	strategistRoot := filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "active.yaml"),
		[]byte(": not: valid: yaml:\n"), 0o644))

	_, _, err := ResolveActiveBasePath(strategistRoot)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse active.yaml")
}

func TestResolveActiveBasePath_EmptyBasePathErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: epic\n"), 0o644))

	_, _, err := ResolveActiveBasePath(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_path is empty")
}

func TestTelemetryRunFromCmd_NilCmd(t *testing.T) {
	t.Parallel()
	assert.Nil(t, TelemetryRunFromCmd(nil))
}

func TestTelemetryRunFromCmd_NilContext(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{Use: "no-context"}
	assert.Nil(t, TelemetryRunFromCmd(cmd))
}

func TestTelemetryRunFromCmd_WithNonNilContext(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	result := TelemetryRunFromCmd(cmd)
	// MissionRunFromContext returns nil when ctx has no embedded run.
	assert.Nil(t, result)
}
