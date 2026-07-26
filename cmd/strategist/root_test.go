package main

import (
	"errors"
	"os"
	"os/exec"
	"testing"

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

// --- validate ---

// minimalValidateRoot creates a .strategist/-like tree suitable for validateCmd:
// active.yaml, personas/pragmatic.yaml, roles/default.yaml.
