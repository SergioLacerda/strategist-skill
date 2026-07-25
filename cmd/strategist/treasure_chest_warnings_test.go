package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/stretchr/testify/assert"
)

// --- collectWarnings ---

func TestCollectWarnings_AllWarnings(t *testing.T) {
	t.Parallel()
	rows := []treasure.StatusRow{
		{ID: "drift-chest", Configured: true, Governed: false, Indexed: false,
			Drift: []string{"missing_governance", "missing_index"}},
		{ID: "historical-chest", TrustTier: "T2", LastReviewed: ""},
	}
	warnings := collectWarnings(rows,
		errors.New("gov load error"),
		errors.New("idx load error"),
		errors.New("corrupt"),
		0,
	)
	assert.Contains(t, strings.Join(warnings, "\n"), "treasure-chests.yaml unavailable")
	assert.Contains(t, strings.Join(warnings, "\n"), "knowledge.index.yaml unavailable")
	assert.Contains(t, strings.Join(warnings, "\n"), "corrupt")
	assert.Contains(t, strings.Join(warnings, "\n"), "drift detected")
	assert.Contains(t, strings.Join(warnings, "\n"), "historical")
}

func TestCollectWarnings_AbsentIndex(t *testing.T) {
	t.Parallel()
	warnings := collectWarnings(nil, nil, nil, nil, 0)
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "absent")
}

func TestCollectWarnings_DriftWithCompiledAt(t *testing.T) {
	t.Parallel()
	rows := []treasure.StatusRow{
		{ID: "chest-a", Configured: true, Governed: false,
			Drift: []string{"missing_governance"}},
	}
	warnings := collectWarnings(rows, nil, nil, nil, 1700000000)
	combined := strings.Join(warnings, "\n")
	assert.Contains(t, combined, "drift detected")
	assert.Contains(t, combined, "→ run: strategist treasure-chest --index to refresh")
}

// --- renderWarningsSection ---

func TestRenderWarningsSection_Empty(t *testing.T) {
	out := captureStdout(t, func() {
		renderWarningsSection(nil)
	})
	assert.Empty(t, out)
}

func TestRenderWarningsSection_WithWarnings(t *testing.T) {
	out := captureStdout(t, func() {
		renderWarningsSection([]string{"⚠ first warning", "⚠ second warning"})
	})
	assert.Contains(t, out, "WARNINGS")
	assert.Contains(t, out, "first warning")
	assert.Contains(t, out, "second warning")
}
