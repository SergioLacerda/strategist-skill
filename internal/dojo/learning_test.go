package dojo_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyFailures_MapsKnownLabels(t *testing.T) {
	items := []domain.DojoCheckItem{
		{Label: "files_created todo/geral.md", Passed: false},
		{Label: `must_contain "X" in todo/geral.md`, Passed: false},
		{Label: "emit ranger_start must NOT appear", Passed: false},
		{Label: "emit sniper_start", Passed: false},
		{Label: "manifest field \"canonical_role\" in skills/x/skill.yaml", Passed: false},
		{Label: "timing total_wall_time_ms", Passed: false},
		{Label: "pipeline slots_invoked execution", Passed: false},
		{Label: "files_created ok.md", Passed: true},
	}
	reasons := dojo.ClassifyFailures(items)
	assert.ElementsMatch(t, []dojo.FailureReason{
		dojo.FailureMissingArtifact,
		dojo.FailureMissingCanary,
		dojo.FailureForbiddenEmit,
		dojo.FailureMissingEmit,
		dojo.FailureManifestDrift,
		dojo.FailureTiming,
		dojo.FailurePipeline,
	}, reasons)
}

func TestClassifyFailures_Deduplicates(t *testing.T) {
	items := []domain.DojoCheckItem{
		{Label: "files_created a.md", Passed: false},
		{Label: "files_created b.md", Passed: false},
	}
	reasons := dojo.ClassifyFailures(items)
	assert.Equal(t, []dojo.FailureReason{dojo.FailureMissingArtifact}, reasons)
}

func TestClassifyFailures_AllPassed_Empty(t *testing.T) {
	items := []domain.DojoCheckItem{{Label: "files_created a.md", Passed: true}}
	assert.Empty(t, dojo.ClassifyFailures(items))
}

func TestGenerateLesson_IncludesFailuresAndNextAction(t *testing.T) {
	result := domain.DojoCheckResult{
		Scenario: "critical-hit",
		Items: []domain.DojoCheckItem{
			{Label: "timing total_wall_time_ms", Passed: false, Detail: "wall time 90000 ms exceeds max 60000 ms"},
			{Label: "files_created docs/dojo/critical-hit-fixture.md", Passed: true},
		},
	}
	reasons := dojo.ClassifyFailures(result.Items)
	lesson := dojo.GenerateLesson(result, reasons)
	assert.Contains(t, lesson, "critical-hit")
	assert.Contains(t, lesson, "timing total_wall_time_ms")
	assert.Contains(t, lesson, "timing_regression")
	assert.Contains(t, lesson, "wall-time regression")
}

func TestWriteLesson_SkipsPassingRun(t *testing.T) {
	base := t.TempDir()
	result := domain.DojoCheckResult{
		Scenario: "quick-draw",
		Items:    []domain.DojoCheckItem{{Label: "files_created x", Passed: true}},
	}
	require.NoError(t, dojo.WriteLesson(base, result))
	_, err := os.Stat(filepath.Join(base, "dojo", ".last-run", "quick-draw", "lesson.md"))
	assert.True(t, os.IsNotExist(err))
}

func TestWriteLesson_WritesForFailingRun(t *testing.T) {
	base := t.TempDir()
	result := domain.DojoCheckResult{
		Scenario: "critical-hit",
		Items:    []domain.DojoCheckItem{{Label: "timing total_wall_time_ms", Passed: false, Detail: "too slow"}},
	}
	require.NoError(t, dojo.WriteLesson(base, result))
	content, err := os.ReadFile(filepath.Join(base, "dojo", ".last-run", "critical-hit", "lesson.md"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "critical-hit")
	assert.Contains(t, string(content), "too slow")
}
