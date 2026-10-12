package usagesummary

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func incWithRun(id, phase, runID string, in, out int64, at string) ur {
	r := inc(id, phase, "inv-"+runID, 1, n(in), n(out), at)
	r.PhaseRunID = runID
	return r
}

func TestSummarizeKeepsUsageInItsExplicitLifecycleRun(t *testing.T) {
	s := Summarize([]ur{
		incWithRun("a", "REFINEMENT", "r1", 100, 10, "2026-10-10T10:00:00Z"),
		incWithRun("b", "REFINEMENT", "r2", 200, 20, "2026-10-10T10:01:00Z"),
	}, nil)

	assert.Equal(t, int64(100), *s.ByRun["REFINEMENT|r1"].Tokens.Input)
	assert.Equal(t, int64(200), *s.ByRun["REFINEMENT|r2"].Tokens.Input)
	assert.Equal(t, int64(300), *s.ByPhase["REFINEMENT"].Tokens.Input)
}

func TestSummarizeCumulativeDeltaRetainsTheCurrentRun(t *testing.T) {
	base := snap("s1", "sess-1", "REFINEMENT", n(100), n(10), "2026-10-10T10:00:00Z")
	current := snap("s2", "sess-1", "REFINEMENT", n(250), n(25), "2026-10-10T10:01:00Z")
	current.PhaseRunID = "r2"

	s := Summarize([]ur{base, current}, nil)
	assert.Equal(t, int64(150), *s.ByRun["REFINEMENT|r2"].Tokens.Input)
	assert.NotContains(t, s.ByRun, "REFINEMENT|r1")
}

func TestSummarizeDoesNotInferRunsForUnkeyedUsage(t *testing.T) {
	s := Summarize([]ur{inc("a", "REFINEMENT", "inv-1", 1, n(100), n(10), "2026-10-10T10:00:00Z")}, nil)
	assert.Empty(t, s.ByRun)
	assert.Equal(t, int64(100), *s.ByPhase["REFINEMENT"].Tokens.Input)
}
