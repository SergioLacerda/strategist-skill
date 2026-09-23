package cliutil

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type runtimeContextKey string

func TestCommandContext_PreservesCobraContext(t *testing.T) {
	t.Parallel()

	key := runtimeContextKey("otel-context-contract")
	cmd := &cobra.Command{Use: "context-contract"}
	cmd.SetContext(context.WithValue(context.Background(), key, "mission-context"))

	got := CommandContext(cmd)

	require.NotNil(t, got)
	assert.Equal(t, "mission-context", got.Value(key))
}

func TestCommandContext_NilContextReturnsBackground(t *testing.T) {
	t.Parallel()

	ctx := CommandContext(&cobra.Command{Use: "no-context-cmd"})

	assert.NotNil(t, ctx)
}

func TestAddMissionLines_CountsOnTheMissionRun(t *testing.T) {
	t.Parallel()

	run := telemetry.NewMissionRun("cliutil-lines")
	ctx := telemetry.WithMissionRun(context.Background(), run)

	assert.NotPanics(t, func() { AddMissionLines(ctx, 3) })

	assert.Equal(t, int64(3), run.Snapshot().LinesEmitted)
}

func TestAddMissionLines_WithoutRunIsANoOp(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() { AddMissionLines(context.Background(), 3) })
}
