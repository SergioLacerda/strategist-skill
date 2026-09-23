package eval

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realProjectRootStrategistPath returns <project_root>/.strategist without
// requiring that directory to actually exist on disk — testResolveRoot's
// explicit-root branch never stats its input, only derives projectRoot as
// the parent of the given path. This points eval run's -root flag at this
// real repository's root deterministically, purely from the Go module's own
// on-disk layout (cmd/strategist/eval is always three levels below the
// module root), without depending on .strategist/ being materialized by
// `strategist install`. That directory is gitignored and absent on a fresh
// checkout (e.g. CI), which is exactly the case this helper keeps hermetic.
func realProjectRootStrategistPath(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(cwd))) // eval -> strategist -> cmd -> module root
	return filepath.Join(projectRoot, ".strategist")
}

func testRunDeps() Dependencies {
	return Dependencies{RootFlag: "root", ResolveRoot: testResolveRoot, SilenceRun: noopSilenceRun}
}

func setEvalRunFlags(t *testing.T, cmd *cobra.Command, root string, race bool) {
	t.Helper()
	require.NoError(t, cmd.Flags().Set("root", root))
	require.NoError(t, cmd.Flags().Set("race", boolFlagString(race)))
}

func TestResolvePattern_DefaultsWhenNoArgs(t *testing.T) {
	assert.Equal(t, "./tests/evals/...", ResolvePattern(nil))
	assert.Equal(t, "./tests/evals/...", ResolvePattern([]string{}))
}

func TestResolvePattern_DefaultsOnEmptyFirstArg(t *testing.T) {
	assert.Equal(t, "./tests/evals/...", ResolvePattern([]string{""}))
}

func TestResolvePattern_UsesGivenPattern(t *testing.T) {
	assert.Equal(t, "./tests/evals/contracts/...", ResolvePattern([]string{"./tests/evals/contracts/..."}))
}

func TestBuildGoTestArgs_RaceOn(t *testing.T) {
	assert.Equal(t,
		[]string{"test", "-race", "-tags=eval", "./tests/evals/..."},
		BuildGoTestArgs("./tests/evals/...", true),
	)
}

func TestBuildGoTestArgs_RaceOff(t *testing.T) {
	assert.Equal(t,
		[]string{"test", "-tags=eval", "./tests/evals/contracts/..."},
		BuildGoTestArgs("./tests/evals/contracts/...", false),
	)
}

// TestRunCmd_EndToEnd exercises the full wiring — flag parsing, root
// resolution, and the actual go test subprocess — against this real
// repository's own tests/evals/contracts package, which is small and fast.
// It intentionally does not fabricate a temp Go module: scenario definitions
// live inside real _test.go files, so a temp-module fixture would need its
// own go.mod/build setup for little added confidence over running against
// real, already-passing content.
//
// It stays hermetic with respect to .strategist/ itself, though: that
// directory is gitignored and only exists on a machine that ran `strategist
// install`, never on a fresh checkout (e.g. CI). Passing an explicit -root
// computed from this module's own on-disk layout
// (realProjectRootStrategistPath) resolves the correct projectRoot without
// requiring .strategist/ to exist.
func TestRunCmd_EndToEnd(t *testing.T) {
	cmd := NewRun(testRunDeps())
	setEvalRunFlags(t, cmd, realProjectRootStrategistPath(t), false)

	err := cmd.RunE(cmd, []string{"./tests/evals/contracts/..."})
	require.NoError(t, err)
}

// TestRun_InvokesSilenceRun covers Run's own
// "if deps.SilenceRun != nil { deps.SilenceRun(cmd) }" branch. The telemetry
// MissionRun.SetSilent() logic itself lives in adapter_deps.go's SilenceRun
// closure (package main), outside this package's scope.
func TestRun_InvokesSilenceRun(t *testing.T) {
	deps := testRunDeps()
	called := false
	deps.SilenceRun = func(*cobra.Command) { called = true }
	cmd := NewRun(deps)
	setEvalRunFlags(t, cmd, realProjectRootStrategistPath(t), false)

	require.NoError(t, cmd.RunE(cmd, []string{"./tests/evals/contracts/..."}))
	assert.True(t, called)
}

func TestRunCmd_PropagatesTestFailureExitError(t *testing.T) {
	cmd := NewRun(testRunDeps())
	setEvalRunFlags(t, cmd, realProjectRootStrategistPath(t), false)

	// A pattern matching no packages under the eval tag is itself a `go
	// test` usage error (non-zero exit) without needing to fabricate a
	// failing test file in the real tree — sufficient to confirm this
	// command propagates exec.Command's error rather than swallowing it.
	err := cmd.RunE(cmd, []string{"./tests/evals/does-not-exist/..."})
	require.Error(t, err)
}
