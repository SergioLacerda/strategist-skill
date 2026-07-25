package main

import (
	"errors"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- renderIndexSection ---

func TestRenderIndexSection_CompError(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	err := renderIndexSection(w, t.TempDir(), 0, errors.New("decompress failed"))
	require.NoError(t, err)
	require.NoError(t, w.Flush())
	out := buf.String()
	assert.Contains(t, out, "corrupt")
	assert.Contains(t, out, "—")
}

func TestRenderIndexSection_Absent(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	err := renderIndexSection(w, t.TempDir(), 0, nil)
	require.NoError(t, err)
	require.NoError(t, w.Flush())
	out := buf.String()
	assert.Contains(t, out, "absent")
	assert.Contains(t, out, "—")
}

func TestRenderIndexSection_OK(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	err := renderIndexSection(w, t.TempDir(), 1700000000, nil)
	require.NoError(t, err)
	require.NoError(t, w.Flush())
	out := buf.String()
	assert.Contains(t, out, "ok")
	assert.Contains(t, out, "UTC")
}

// --- renderChestsSection ---

func TestRenderChestsSection_WithRows(t *testing.T) {
	t.Parallel()
	rows := []treasure.StatusRow{
		{ID: "chest-a", Path: "/path/a", Scope: []string{"discovery"}, TrustTier: "T1",
			Freshness: "fresh", Drift: nil},
		{ID: "chest-b", Scope: nil, TrustTier: "", Freshness: "unknown",
			Drift: []string{"missing_governance"}},
	}
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	err := renderChestsSection(w, rows)
	require.NoError(t, err)
	require.NoError(t, w.Flush())
	out := buf.String()
	assert.Contains(t, out, "chest-a")
	assert.Contains(t, out, "chest-b")
	assert.Contains(t, out, "T1")
	assert.Contains(t, out, "missing_governance")
	assert.Contains(t, out, "none") // chest-a has no drift
}
