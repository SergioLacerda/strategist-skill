package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// --- telemetryRunFromCmd ---

func TestTelemetryRunFromCmd_NilCmd(t *testing.T) {
	t.Parallel()
	assert.Nil(t, telemetryRunFromCmd(nil))
}

func TestTelemetryRunFromCmd_NilContext(t *testing.T) {
	t.Parallel()
	// A cobra.Command with no context set → cmd.Context() returns nil.
	// Use the actual treasureChestCmd which starts without a context.
	assert.Nil(t, telemetryRunFromCmd(treasureChestCmd))
}

func TestTelemetryRunFromCmd_WithNonNilContext(t *testing.T) {
	t.Parallel()
	// Use a private *cobra.Command, not a shared package-level command var — other
	// parallel tests read package-level commands' Context() concurrently, and
	// SetContext on a shared command races with those reads.
	cmd := &cobra.Command{}
	// Set a background context so cmd.Context() returns non-nil.
	cmd.SetContext(t.Context())
	result := telemetryRunFromCmd(cmd)
	// MissionRunFromContext returns nil when ctx has no embedded run.
	assert.Nil(t, result)
}
