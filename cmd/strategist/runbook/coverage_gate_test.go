package runbook

import (
	"bytes"
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/roles/jeweler"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunbookRenderingCoversEmptyAndPopulatedTables(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, render(cmd, jeweler.Projection{}, "table"))
	assert.Contains(t, out.String(), "no runbook matched")

	out.Reset()
	require.NoError(t, render(cmd, jeweler.Projection{Items: []jeweler.Item{{ID: "book", Source: jeweler.Source{Path: "runbooks/book.yaml"}, Reason: "matched"}}}, "table"))
	assert.Contains(t, out.String(), "RUNBOOK_ID")
}

func TestRunbookRenderingReportsEmptyAndRowWriteFailures(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{}
	cmd.SetOut(runbookFailWriter{})
	require.ErrorContains(t, render(cmd, jeweler.Projection{}, "table"), "render empty runbook selection")

	cmd.SetOut(&failAfterHeaderWriter{})
	projection := jeweler.Projection{Items: []jeweler.Item{{ID: "book", Source: jeweler.Source{Path: "book.md"}}}}
	require.ErrorContains(t, render(cmd, projection, "table"), "render runbook selection row")
}

type failAfterHeaderWriter struct{ writes int }

func (w *failAfterHeaderWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes > 1 {
		return 0, errors.New("row write failed")
	}
	return len(p), nil
}
