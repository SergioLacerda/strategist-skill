package cliutil

import (
	"bytes"
	"context"
	"log"
	"log/slog"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultHandlerOutput captures what the default slog handler prints. It must
// not run in parallel: it swaps the global log writer and level.
func defaultHandlerOutput(t *testing.T) *bytes.Buffer {
	t.Helper()
	var out bytes.Buffer
	flags, writer := log.Flags(), log.Writer()
	log.SetFlags(0)
	log.SetOutput(&out)
	t.Cleanup(func() { log.SetFlags(flags); log.SetOutput(writer) })
	return &out
}

func commandWithVerbose(verbose bool) *cobra.Command {
	cmd := &cobra.Command{Use: "install"}
	cmd.Flags().Bool("verbose", false, "")
	if verbose {
		_ = cmd.Flags().Set("verbose", "true") //nolint:errcheck // the flag is declared above
	}
	return cmd
}

func emit() {
	slog.Info("info-line")
	slog.Warn("warn-line")
}

func TestApplyDisplayLevelHidesInfoButNotWarnForAQuietCommand(t *testing.T) {
	out := defaultHandlerOutput(t)
	cmd := commandWithVerbose(false)
	QuietByDefault(cmd)

	restore := ApplyDisplayLevel(cmd)
	emit()

	assert.NotContains(t, out.String(), "info-line")
	assert.Contains(t, out.String(), "warn-line")

	restore()
	restore() // idempotent
	out.Reset()
	emit()
	assert.Contains(t, out.String(), "info-line", "the level is restored")
}

func TestApplyDisplayLevelLeavesAVerboseCommandUntouched(t *testing.T) {
	out := defaultHandlerOutput(t)
	cmd := commandWithVerbose(true)
	QuietByDefault(cmd)

	restore := ApplyDisplayLevel(cmd)
	defer restore()
	emit()

	assert.Contains(t, out.String(), "info-line")
	assert.Contains(t, out.String(), "warn-line")
}

func TestApplyDisplayLevelLeavesUnannotatedCommandsUntouched(t *testing.T) {
	out := defaultHandlerOutput(t)
	cmd := commandWithVerbose(false)

	restore := ApplyDisplayLevel(cmd)
	defer restore()
	emit()

	assert.Contains(t, out.String(), "info-line", "a command that did not opt in keeps its INFO display, e.g. check-stale")
}

func TestApplyDisplayLevelDoesNotGovernAConfiguredTelemetryHandler(t *testing.T) {
	var captured []slog.Record
	previous := slog.Default()
	slog.SetDefault(slog.New(recordingHandler{records: &captured}))
	t.Cleanup(func() { slog.SetDefault(previous) })
	cmd := commandWithVerbose(false)
	QuietByDefault(cmd)

	restore := ApplyDisplayLevel(cmd)
	defer restore()
	emit()

	require.Len(t, captured, 2, "an OTel-style default handler still receives the INFO event")
	assert.Equal(t, "info-line", captured[0].Message)
}

type recordingHandler struct{ records *[]slog.Record }

func (recordingHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h recordingHandler) Handle(_ context.Context, record slog.Record) error {
	*h.records = append(*h.records, record)
	return nil
}
func (h recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h recordingHandler) WithGroup(string) slog.Handler      { return h }
