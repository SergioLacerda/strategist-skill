package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootHelpers_HumanStatusAndPreRun(t *testing.T) {
	assert.False(t, isHumanStatusCommand(nil))
	parent := &cobra.Command{Use: "treasure-chest"}
	child := &cobra.Command{Use: "jewel"}
	parent.AddCommand(child)
	assert.True(t, isHumanStatusCommand(child))

	cmd := &cobra.Command{Use: "custom"}
	require.NoError(t, rootCmd.PersistentPreRunE(cmd, nil))
	assert.NotNil(t, cmd.Context())
	require.NoError(t, rootCmd.PersistentPostRunE(cmd, nil))
}

func TestRunCompile_CompileAllError(t *testing.T) {
	orig := compileRoot
	t.Cleanup(func() { compileRoot = orig })
	compileRoot = t.TempDir()

	err := runCompile(&cobra.Command{Use: "compile"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compile: compile all")
}

func TestRunInstall_RejectsConflictingShimFlags(t *testing.T) {
	origNoShim := installNoShim
	origShimPath := installShimPath
	t.Cleanup(func() {
		installNoShim = origNoShim
		installShimPath = origShimPath
	})
	installNoShim = true
	installShimPath = filepath.Join(t.TempDir(), "SKILL.md")

	err := runInstall(&cobra.Command{Use: "install"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func withClosedStdout(t *testing.T, fn func()) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "closed-stdout-*")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	old := os.Stdout
	os.Stdout = f
	t.Cleanup(func() { os.Stdout = old })
	fn()
}
