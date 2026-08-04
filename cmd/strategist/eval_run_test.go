package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
func TestEvalRunCmd_EndToEnd(t *testing.T) {
	// Empty root: resolveEvalActionRoot auto-discovers .strategist/ by
	// walking up from cwd (cmd/strategist/ during `go test`), landing on
	// this real repository's own root — the same auto-discovery every
	// other command in this package relies on by default.
	setEvalRunFlags(t, "", false)
	t.Cleanup(func() { resetEvalRunFlags(t) })

	err := evalRunCmd.RunE(evalRunCmd, []string{"./tests/evals/contracts/..."})
	require.NoError(t, err)
}

func TestEvalRunCmd_PropagatesTestFailureExitError(t *testing.T) {
	setEvalRunFlags(t, "", false)
	t.Cleanup(func() { resetEvalRunFlags(t) })

	// A pattern matching no packages under the eval tag is itself a `go
	// test` usage error (non-zero exit) without needing to fabricate a
	// failing test file in the real tree — sufficient to confirm this
	// command propagates exec.Command's error rather than swallowing it.
	err := evalRunCmd.RunE(evalRunCmd, []string{"./tests/evals/does-not-exist/..."})
	require.Error(t, err)
}
