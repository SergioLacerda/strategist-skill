package main

import (
	"context"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/assert"
)

type contextKey string

func TestCommandSpansPreserveMissionContext(t *testing.T) {
	t.Parallel()

	key := contextKey("mission-span-context")
	run := telemetry.NewMissionRun("otel-context-contract")
	ctx := telemetry.WithMissionRun(context.WithValue(context.Background(), key, "kept"), run)

	compileCtx, compileSpan := startCompileSpan(ctx)
	compileSpan.End()

	assert.Equal(t, "kept", compileCtx.Value(key))
	assert.Same(t, run, telemetry.MissionRunFromContext(compileCtx))
}
