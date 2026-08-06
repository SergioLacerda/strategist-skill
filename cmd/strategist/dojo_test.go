package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPersistDojoResult_WarnsOnPersistAndLessonErrors covers persistDojoResult's
// two stderr-warning branches. Both dojo.PersistResult and dojo.WriteLesson
// create basePath/dojo/.last-run/<scenario> via os.MkdirAll — pre-occupying
// basePath/dojo with a file blocks both MkdirAll calls in one shot, same
// black-box technique already used throughout this package (e.g.
// TestWriteFile_MkdirAllFailsWhenParentIsFile in internal/runtimefs). A
// failed result (Passed()==false) is required so WriteLesson actually
// attempts its write instead of returning nil immediately.
func TestPersistDojoResult_WarnsOnPersistAndLessonErrors(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "dojo")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	result := domain.DojoCheckResult{
		Scenario: "sample",
		Items:    []domain.DojoCheckItem{{Passed: false}},
	}

	errOut := captureStderr(t, func() {
		persistDojoResult(dir, result, time.Now(), time.Now())
	})
	assert.Contains(t, errOut, "failed to persist result")
	assert.Contains(t, errOut, "failed to write lesson")
}

// TestPersistDojoResult_PassedResultSkipsLesson covers the WriteLesson
// early-return branch (result.Passed()==true) separately from the
// persist-error branch above, so both are exercised under a condition
// that matches their real precondition rather than assuming a failed
// result also happens to cover the passed-result path.
func TestPersistDojoResult_PassedResultSkipsLesson(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "dojo")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	result := domain.DojoCheckResult{
		Scenario: "sample",
		Items:    []domain.DojoCheckItem{{Passed: true}},
	}

	errOut := captureStderr(t, func() {
		persistDojoResult(dir, result, time.Now(), time.Now())
	})
	assert.Contains(t, errOut, "failed to persist result")
	assert.NotContains(t, errOut, "failed to write lesson")
}

// TestPrintDojoResult_ClosedStdoutErrors covers printDojoResult's terminal
// w.Flush() error check — same withClosedStdout technique already used
// elsewhere in this package for other tabwriter-backed renderers.
func TestPrintDojoResult_ClosedStdoutErrors(t *testing.T) {
	result := domain.DojoCheckResult{
		Scenario: "sample",
		Items:    []domain.DojoCheckItem{{Passed: true, Label: "check-1"}},
	}

	withClosedStdout(t, func() {
		require.Error(t, printDojoResult(result))
	})
}
