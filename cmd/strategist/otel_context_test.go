package main

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type contextKey string

func TestCommandContextPreservesCobraContext(t *testing.T) {
	t.Parallel()

	key := contextKey("otel-context-contract")
	ctx := context.WithValue(context.Background(), key, "mission-context")
	cmd := &cobra.Command{Use: "context-contract"}
	cmd.SetContext(ctx)

	got := commandContext(cmd)

	require.NotNil(t, got)
	assert.Equal(t, "mission-context", got.Value(key))
}

func TestCommandSpansPreserveMissionContext(t *testing.T) {
	t.Parallel()

	key := contextKey("mission-span-context")
	run := telemetry.NewMissionRun("otel-context-contract")
	ctx := telemetry.WithMissionRun(context.WithValue(context.Background(), key, "kept"), run)

	compileCtx, compileSpan := startCompileSpan(ctx)
	compileSpan.End()
	installCtx, installSpan := startInstallSpan(ctx)
	installSpan.End()

	assert.Equal(t, "kept", compileCtx.Value(key))
	assert.Same(t, run, telemetry.MissionRunFromContext(compileCtx))
	assert.Equal(t, "kept", installCtx.Value(key))
	assert.Same(t, run, telemetry.MissionRunFromContext(installCtx))
}
