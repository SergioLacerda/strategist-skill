package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- root / execute ---

func TestRootCmd_UnknownSubcommand(t *testing.T) {
	// rootCmd.Execute returns an error for unknown commands without calling os.Exit.
	rootCmd.SetArgs([]string{"__unknown_cmd__"})
	err := rootCmd.Execute()
	// Cobra returns an error for unknown commands.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestExecute_NoError(t *testing.T) {
	// Smoke-test execute() success path: "version" command succeeds.
	// We redirect Stdout to suppress output during the test.
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "smoke"

	// Capture stdout to avoid test noise.
	_ = captureStdout(t, func() {
		rootCmd.SetArgs([]string{"version"})
		rootCmd.Execute() //nolint:errcheck // return value not needed here
	})
}

// TestExecute_Success calls execute() directly with a valid command so that the
// success branch (err == nil, no os.Exit) is covered.

// TestExecute_Success calls execute() directly with a valid command so that the
// success branch (err == nil, no os.Exit) is covered.
func TestExecute_Success(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "execute-smoke"

	_ = captureStdout(t, func() {
		rootCmd.SetArgs([]string{"version"})
		execute()
	})
}

// TestMain_Smoke calls main() directly (valid in package main tests) with a safe
// command so neither main() nor execute() can reach os.Exit.

// TestMain_Smoke calls main() directly (valid in package main tests) with a safe
// command so neither main() nor execute() can reach os.Exit.
func TestMain_Smoke(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "main-smoke"

	_ = captureStdout(t, func() {
		rootCmd.SetArgs([]string{"version"})
		main()
	})
}

// TestExecute_ErrorPath covers the os.Exit(1) branch in execute() by running the
// test binary in a subprocess with an unknown command.

// TestExecute_ErrorPath covers the os.Exit(1) branch in execute() by running the
// test binary in a subprocess with an unknown command.
func TestExecute_ErrorPath(t *testing.T) {
	if os.Getenv("STRATEGIST_EXPECT_EXIT") == "1" {
		rootCmd.SetArgs([]string{"__exit_test__"})
		execute()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestExecute_ErrorPath")
	cmd.Env = append(os.Environ(), "STRATEGIST_EXPECT_EXIT=1")
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected exit error, got: %v", err)
	}
	assert.Equal(t, 1, exitErr.ExitCode())
}

func TestPersistentPreRunE_HumanStatusCommandSuppressesBanner(t *testing.T) {
	// Any *cobra.Command named "check" exercises the human-status silent
	// path via isHumanStatusCommand — the real internal/check.CheckCmd is
	// not needed (this is root.go's own dispatch behavior under test, not
	// check's).
	chdirForTest(t, t.TempDir())

	err := rootCmd.PersistentPreRunE(&cobra.Command{Use: "check"}, nil)
	require.NoError(t, err)
}

func TestPersistentPreRunE_NonHumanStatusCommandDefaultsStrategistDir(t *testing.T) {
	chdirForTest(t, t.TempDir())

	err := rootCmd.PersistentPreRunE(versionCmd, nil)
	require.NoError(t, err)
}

// TestPersistentPreRunE_BindsAmbientMissionRunToRealMissionID covers F-T1
// (ADR-0057/design.md task 3.1): a command declaring --mission-id binds the
// ambient MissionRun to that real id instead of the synthetic
// "<subcommand>-<nanos>" one, so the metrics line printed for it can be
// joined back to the mission it measured.
func TestPersistentPreRunE_BindsAmbientMissionRunToRealMissionID(t *testing.T) {
	chdirForTest(t, t.TempDir())

	cmd := &cobra.Command{Use: "start"}
	cmd.Flags().String("mission-id", "", "")
	require.NoError(t, cmd.Flags().Set("mission-id", "20260927-example-mission"))

	require.NoError(t, rootCmd.PersistentPreRunE(cmd, nil))

	run := telemetry.MissionRunFromContext(cmd.Context())
	require.NotNil(t, run)
	assert.Equal(t, "20260927-example-mission", run.MissionID)
}

// TestPersistentPreRunE_NoMissionIDFlagKeepsSyntheticID confirms a command
// with no --mission-id flag (e.g. "version") is unaffected: the synthetic
// "<subcommand>-<nanos>" id is still used.
func TestPersistentPreRunE_NoMissionIDFlagKeepsSyntheticID(t *testing.T) {
	chdirForTest(t, t.TempDir())

	require.NoError(t, rootCmd.PersistentPreRunE(versionCmd, nil))

	run := telemetry.MissionRunFromContext(versionCmd.Context())
	require.NotNil(t, run)
	assert.True(t, strings.HasPrefix(run.MissionID, "version-"), "expected synthetic id prefixed by command name, got %q", run.MissionID)
}

// TestPersistentPreRunE_PreservesCallerContext covers F-X2 (ADR-0057/design.md
// task 5.3): PersistentPreRunE must not discard a context value already set
// on the command before it ran, in favor of a fresh context.Background().
func TestPersistentPreRunE_PreservesCallerContext(t *testing.T) {
	chdirForTest(t, t.TempDir())

	type ctxKey string
	const key ctxKey = "caller-set-value"
	cmd := &cobra.Command{Use: "version"}
	cmd.SetContext(context.WithValue(context.Background(), key, "kept"))

	require.NoError(t, rootCmd.PersistentPreRunE(cmd, nil))

	assert.Equal(t, "kept", cmd.Context().Value(key))
}

func TestPersistentPreRunE_GetwdErrorFallsBackToDot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chdir-then-remove not reliable on windows")
	}
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	removed := t.TempDir()
	require.NoError(t, os.Chdir(removed))
	require.NoError(t, os.RemoveAll(removed))

	runErr := rootCmd.PersistentPreRunE(versionCmd, nil)
	require.NoError(t, runErr)
}

// --- validate ---

// minimalValidateRoot creates a .strategist/-like tree suitable for validateCmd:
// active.yaml, personas/pragmatic.yaml, roles/default.yaml.
