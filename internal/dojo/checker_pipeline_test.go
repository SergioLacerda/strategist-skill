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

const sampleEmitLog = `2026-06-19T08:55:00Z INFO [Strategist] key=ranger_start scenario=sample-scenario strategist.phase=ranger strategist.status=start
2026-06-19T08:55:01Z INFO [Strategist] key=ranger_done scenario=sample-scenario strategist.phase=ranger strategist.status=done
2026-06-19T08:55:02Z INFO [Strategist] key=archivist_done scenario=sample-scenario strategist.phase=archivist strategist.status=done
2026-06-19T08:55:03Z INFO [Strategist] key=approval_prompt scenario=sample-scenario strategist.phase=approval_gate strategist.status=prompt
2026-06-19T08:55:03Z INFO [Strategist] phase=approval_gate status=auto_stopped scenario=sample-scenario strategist.phase=approval_gate strategist.status=auto_stopped
`

func writeEmitLog(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "emit.log")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestCheckPipeline_SampleScenario_AllPass(t *testing.T) {
	logPath := writeEmitLog(t, sampleEmitLog)
	criteria := domain.DojoCriteria{
		AutoStopAtGate: true,
		Pipeline: domain.DojoPipeline{
			MustStopAt:      "approval_gate",
			SlotsInvoked:    []string{"discovery", "refinement"},
			SlotsNotInvoked: []string{"execution"},
		},
	}
	items := dojo.CheckPipeline(criteria, logPath, false)
	require.NotEmpty(t, items)
	for _, it := range items {
		assert.True(t, it.Passed, "expected pass: %s — %s", it.Label, it.Detail)
	}
}

func TestCheckPipeline_SlotInvokedMissing(t *testing.T) {
	logPath := writeEmitLog(t, sampleEmitLog)
	criteria := domain.DojoCriteria{
		Pipeline: domain.DojoPipeline{SlotsInvoked: []string{"execution"}},
	}
	items := dojo.CheckPipeline(criteria, logPath, false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "execution")
}

func TestCheckPipeline_SlotNotInvokedViolated(t *testing.T) {
	logPath := writeEmitLog(t, sampleEmitLog)
	criteria := domain.DojoCriteria{
		Pipeline: domain.DojoPipeline{SlotsNotInvoked: []string{"discovery"}},
	}
	items := dojo.CheckPipeline(criteria, logPath, false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
}

func TestCheckPipeline_MustStopAtMismatch(t *testing.T) {
	logPath := writeEmitLog(t, sampleEmitLog)
	criteria := domain.DojoCriteria{
		Pipeline: domain.DojoPipeline{MustStopAt: "execution"},
	}
	items := dojo.CheckPipeline(criteria, logPath, false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "approval_gate")
}

func TestCheckPipeline_AutoStopAtGateMissing(t *testing.T) {
	logPath := writeEmitLog(t, "key=ranger_start strategist.phase=ranger strategist.status=start\n")
	criteria := domain.DojoCriteria{AutoStopAtGate: true}
	items := dojo.CheckPipeline(criteria, logPath, false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
}

func TestCheckPipeline_AutoStopAtGateFalse_NoAssertion(t *testing.T) {
	logPath := writeEmitLog(t, "key=sniper_start strategist.phase=sniper strategist.status=start\n")
	criteria := domain.DojoCriteria{AutoStopAtGate: false, Pipeline: domain.DojoPipeline{SlotsInvoked: []string{"execution"}}}
	items := dojo.CheckPipeline(criteria, logPath, false)
	for _, it := range items {
		assert.NotContains(t, it.Label, "auto_stop_at_gate")
	}
}

func TestCheckPipeline_NoCriteria_Skip(t *testing.T) {
	items := dojo.CheckPipeline(domain.DojoCriteria{}, filepath.Join(t.TempDir(), "emit.log"), false)
	assert.Empty(t, items)
}

func TestCheckPipeline_FilesOnly_Skip(t *testing.T) {
	criteria := domain.DojoCriteria{
		AutoStopAtGate: true,
		Pipeline:       domain.DojoPipeline{SlotsInvoked: []string{"discovery"}},
	}
	items := dojo.CheckPipeline(criteria, filepath.Join(t.TempDir(), "nonexistent.log"), true)
	assert.Empty(t, items)
}

func TestCheckPipeline_LogMissing(t *testing.T) {
	criteria := domain.DojoCriteria{AutoStopAtGate: true}
	items := dojo.CheckPipeline(criteria, filepath.Join(t.TempDir(), "nonexistent.log"), false)
	require.Len(t, items, 1)
	assert.False(t, items[0].Passed)
	assert.Contains(t, items[0].Detail, "emit.log not found")
}

func TestCheckPipeline_MustStopAtEmpty_NoAssertion(t *testing.T) {
	logPath := writeEmitLog(t, sampleEmitLog)
	criteria := domain.DojoCriteria{
		Pipeline: domain.DojoPipeline{MustStopAt: "", SlotsInvoked: []string{"discovery"}},
	}
	items := dojo.CheckPipeline(criteria, logPath, false)
	for _, it := range items {
		assert.NotContains(t, it.Label, "must_stop_at")
	}
}
