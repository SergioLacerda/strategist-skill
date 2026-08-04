package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realProjectRootStrategistPath returns <project_root>/.strategist without
// requiring that directory to actually exist on disk. resolveStrategistRoot's
// explicit-root branch never stats its input — it only derives projectRoot as
// the parent of the given path — so this is enough to point eval run's -root
// flag at this real repository's root deterministically, purely from the
// Go module's own on-disk layout (cmd/strategist is always two levels below
// the module root), without depending on .strategist/ being materialized by
// `strategist install`. That directory is gitignored and absent on a fresh
// checkout (e.g. CI), which is exactly the case this helper keeps hermetic.
func realProjectRootStrategistPath(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Dir(filepath.Dir(cwd)) // cmd/strategist -> cmd -> module root
	return filepath.Join(projectRoot, ".strategist")
}

func setEvalRunFlags(t *testing.T, root string, race bool) {
	t.Helper()
	require.NoError(t, evalRunCmd.Flags().Set(flagRoot, root))
	require.NoError(t, evalRunCmd.Flags().Set("race", boolFlagString(race)))
}

func resetEvalRunFlags(t *testing.T) {
	t.Helper()
	setEvalRunFlags(t, "", true)
}

func TestResolveEvalRunPattern_DefaultsWhenNoArgs(t *testing.T) {
	assert.Equal(t, "./tests/evals/...", resolveEvalRunPattern(nil))
	assert.Equal(t, "./tests/evals/...", resolveEvalRunPattern([]string{}))
}

func TestResolveEvalRunPattern_DefaultsOnEmptyFirstArg(t *testing.T) {
	assert.Equal(t, "./tests/evals/...", resolveEvalRunPattern([]string{""}))
}

func TestResolveEvalRunPattern_UsesGivenPattern(t *testing.T) {
	assert.Equal(t, "./tests/evals/contracts/...", resolveEvalRunPattern([]string{"./tests/evals/contracts/..."}))
}

func TestBuildEvalRunGoTestArgs_RaceOn(t *testing.T) {
	assert.Equal(t,
		[]string{"test", "-race", "-tags=eval", "./tests/evals/..."},
		buildEvalRunGoTestArgs("./tests/evals/...", true),
	)
}

func TestBuildEvalRunGoTestArgs_RaceOff(t *testing.T) {
	assert.Equal(t,
		[]string{"test", "-tags=eval", "./tests/evals/contracts/..."},
		buildEvalRunGoTestArgs("./tests/evals/contracts/...", false),
	)
}

// TestEvalRunCmd_EndToEnd exercises the full wiring — flag parsing, root
// resolution reuse (resolveEvalActionRoot, shared with eval_harvest.go),
// and the actual go test subprocess — against this real repository's own
// tests/evals/contracts package, which is small and fast. It intentionally
// does not fabricate a temp Go module: scenario definitions live inside
// real _test.go files (see analysis.md known_facts), so a temp-module
// fixture would need its own go.mod/build setup for little added
// confidence over running against real, already-passing content.
//
// It stays hermetic with respect to .strategist/ itself, though: that
// directory is gitignored and only exists on a machine that ran `strategist
// install`, never on a fresh checkout (e.g. CI). Auto-discovery (empty
// -root) would fail there with ".strategist not found within 5 levels".
// Passing an explicit -root computed from this module's own on-disk layout
// (realProjectRootStrategistPath) resolves the correct projectRoot without
// requiring .strategist/ to exist.
func TestEvalRunCmd_EndToEnd(t *testing.T) {
	setEvalRunFlags(t, realProjectRootStrategistPath(t), false)
	t.Cleanup(func() { resetEvalRunFlags(t) })

	err := evalRunCmd.RunE(evalRunCmd, []string{"./tests/evals/contracts/..."})
	require.NoError(t, err)
}

func TestEvalRunCmd_PropagatesTestFailureExitError(t *testing.T) {
	setEvalRunFlags(t, realProjectRootStrategistPath(t), false)
	t.Cleanup(func() { resetEvalRunFlags(t) })

	// A pattern matching no packages under the eval tag is itself a `go
	// test` usage error (non-zero exit) without needing to fabricate a
	// failing test file in the real tree — sufficient to confirm this
	// command propagates exec.Command's error rather than swallowing it.
	err := evalRunCmd.RunE(evalRunCmd, []string{"./tests/evals/does-not-exist/..."})
	require.Error(t, err)
}
